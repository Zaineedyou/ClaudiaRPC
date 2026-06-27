package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Gateway struct {
	Conn        *websocket.Conn
	Token       string
	Status      string // "Connected", "Disconnected", "Reconnecting..."
	LastRPCData *RPCData
	ctx         context.Context
	cancel      context.CancelFunc
	mu          sync.Mutex
}

func NewGateway(token string) *Gateway {
	ctx, cancel := context.WithCancel(context.Background())
	return &Gateway{
		Token:  token,
		Status: "Disconnected",
		ctx:    ctx,
		cancel: cancel,
	}
}

func sanitize(s string) string {
	return strings.TrimSpace(s)
}

func resolveImage(url string) string {
	url = sanitize(url)
	if url == "" {
		return ""
	}
	if strings.HasPrefix(url, "mp:") || strings.HasPrefix(url, "spotify:") {
		return url
	}
	return url
}

func (g *Gateway) Connect() error {
	// Initial connect — blocking, return error kalau gagal
	if err := g.connectOnce(); err != nil {
		return err
	}
	g.Status = "Connected"

	// Reconnect loop jalan di background
	go g.reconnectLoop()
	return nil
}

func (g *Gateway) reconnectLoop() {
	delay := 2 * time.Second
	maxDelay := 30 * time.Second

	// Tunggu koneksi drop dulu
	g.waitForDisconnect()

	for {
		select {
		case <-g.ctx.Done():
			g.Status = "Disconnected"
			return
		default:
		}

		g.Status = "Reconnecting..."
		log.Printf("Connection lost, reconnecting in %v...", delay)
		time.Sleep(delay)

		select {
		case <-g.ctx.Done():
			g.Status = "Disconnected"
			return
		default:
		}

		if err := g.connectOnce(); err != nil {
			log.Printf("Reconnect failed: %v", err)
			delay *= 2
			if delay > maxDelay {
				delay = maxDelay
			}
			continue
		}

		g.Status = "Connected"
		delay = 2 * time.Second

		if g.LastRPCData != nil {
			_ = g.UpdateRPC(*g.LastRPCData)
		}

		g.waitForDisconnect()
	}
}

func (g *Gateway) connectOnce() error {
	url := "wss://gateway.discord.gg/?v=10&encoding=json"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return err
	}
	g.Conn = conn

	_, message, err := g.Conn.ReadMessage()
	if err != nil {
		return err
	}

	var helloPayload Payload
	if err := json.Unmarshal(message, &helloPayload); err != nil {
		return err
	}

	if helloPayload.Op != 10 {
		return fmt.Errorf("expected opcode 10, got %d", helloPayload.Op)
	}

	helloDataMap, ok := helloPayload.D.(map[string]interface{})
	if !ok {
		return errors.New("invalid hello payload data")
	}
	interval := int(helloDataMap["heartbeat_interval"].(float64))

	go g.startHeartbeat(g.ctx, interval)

	identify := Payload{
		Op: 2,
		D: Identify{
			Token: g.Token,
			Properties: IdentifyProperties{
				OS:      "Windows",
				Browser: "Discord Client",
				Device:  "",
			},
			Intents: 0,
		},
	}

	if err := g.Conn.WriteJSON(identify); err != nil {
		return err
	}

	for {
		_, message, err := g.Conn.ReadMessage()
		if err != nil {
			return err
		}

		var p Payload
		if err := json.Unmarshal(message, &p); err != nil {
			continue
		}

		if p.Op == 0 && p.T != nil && *p.T == "READY" {
			break
		}
	}

	return nil
}

func (g *Gateway) waitForDisconnect() {
	for {
		_, _, err := g.Conn.ReadMessage()
		if err != nil {
			g.mu.Lock()
			g.Conn.Close()
			g.mu.Unlock()
			return
		}
	}
}

func (g *Gateway) UpdateRPC(data RPCData) error {
	g.LastRPCData = &data
	name := sanitize(data.AppName)
	if name == "" {
		name = "Discord RPC"
	}

	activity := Activity{
		Name:          name,
		Type:          data.Type,
		ApplicationID: sanitize(data.ClientID),
		Details:       sanitize(data.Details),
		State:         sanitize(data.State),
	}

	// Kizzy Approach for Streaming
	if data.Type == 1 { // Streaming
		foundURL := false
		if sanitize(data.Button1URL) != "" {
			activity.Metadata = &ActivityMetadata{ButtonURLs: []string{sanitize(data.Button1URL)}}
			foundURL = true
		}
		// If no URL found, hardcode hidden dummy Twitch URL to trigger purple status
		if !foundURL {
			activity.Metadata = &ActivityMetadata{ButtonURLs: []string{"https://twitch.tv/directory"}}
		}
	}

	if sanitize(data.TimestampStart) != "" || sanitize(data.TimestampEnd) != "" {
		ts := &ActivityTimestamp{}
		if sanitize(data.TimestampStart) != "" {
			if unixVal, err := strconv.ParseInt(sanitize(data.TimestampStart), 10, 64); err == nil {
				if unixVal < 1_000_000_000_000 {
					unixVal = unixVal * 1000
				}
				ts.Start = unixVal
			}
		}
		if sanitize(data.TimestampEnd) != "" {
			if unixVal, err := strconv.ParseInt(sanitize(data.TimestampEnd), 10, 64); err == nil {
				if unixVal < 1_000_000_000_000 {
					unixVal = unixVal * 1000
				}
				ts.End = unixVal
			}
		}
		activity.Timestamps = ts
	}

	largeImage := resolveImage(data.LargeImage)
	smallImage := resolveImage(data.SmallImage)
	if largeImage != "" || smallImage != "" {
		activity.Assets = &ActivityAssets{
			LargeImage: largeImage,
			LargeText:  sanitize(data.LargeText),
			SmallImage: smallImage,
			SmallText:  sanitize(data.SmallText),
		}
	}

	var labels []string
	var urls []string
	if sanitize(data.Button1Label) != "" && sanitize(data.Button1URL) != "" {
		labels = append(labels, sanitize(data.Button1Label))
		urls = append(urls, sanitize(data.Button1URL))
	}
	if sanitize(data.Button2Label) != "" && sanitize(data.Button2URL) != "" {
		labels = append(labels, sanitize(data.Button2Label))
		urls = append(urls, sanitize(data.Button2URL))
	}
	
	if len(labels) > 0 {
		activity.Buttons = labels
		activity.Metadata = &ActivityMetadata{ButtonURLs: urls}
	}

	status := UpdateStatus{
		Since:      nil,
		Activities: []Activity{activity},
		Status:     "online",
		AFK:        false,
	}

	payload := Payload{
		Op: 3,
		D:  status,
	}

	g.mu.Lock()
	defer g.mu.Unlock()
	if g.Conn == nil {
		return errors.New("not connected")
	}
	return g.Conn.WriteJSON(payload)
}

func (g *Gateway) ClearRPC() error {
	g.LastRPCData = nil
	status := UpdateStatus{
		Since:      nil,
		Activities: []Activity{},
		Status:     "online",
		AFK:        false,
	}
	payload := Payload{Op: 3, D: status}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.Conn == nil {
		return nil
	}
	return g.Conn.WriteJSON(payload)
}

func (g *Gateway) Close() {
	g.cancel()
	g.mu.Lock()
	if g.Conn != nil {
		g.Conn.Close()
	}
	g.mu.Unlock()
}
