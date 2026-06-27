package gateway

import (
	"context"
	"time"


)

func (g *Gateway) startHeartbeat(ctx context.Context, interval int) {
	ticker := time.NewTicker(time.Duration(interval) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			payload := Payload{
				Op: 1,
				D:  nil,
			}
			g.mu.Lock()
			err := g.Conn.WriteJSON(payload)
			g.mu.Unlock()
			if err != nil {
				return
			}
		}
	}
}

func (g *Gateway) sendHeartbeat() error {
	payload := Payload{
		Op: 1,
		D:  nil,
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.Conn.WriteJSON(payload)
}
