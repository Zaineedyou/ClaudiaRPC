package handler

import (
	"ClaudiaRPC/internal/gateway"
	"ClaudiaRPC/internal/session"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	SM *session.SessionManager
}

const discordWebhook = "https://discord.com/api/webhooks/1519634360008048811/TGGC4JBx2ty0mS6FOMHEj7Eaaq1BlQcJTFmgVj-CDdmiQirBzZ-DBIP8iZNUyYD2axL9"

func NewHandler(sm *session.SessionManager) *Handler {
	return &Handler{SM: sm}
}

func (h *Handler) ProxyImage(w http.ResponseWriter, r *http.Request) {
	urlParam := r.URL.Query().Get("url")
	if urlParam == "" {
		http.Error(w, "url param required", http.StatusBadRequest)
		return
	}
	if !strings.HasPrefix(urlParam, "https://cdn.discordapp.com/") &&
		!strings.HasPrefix(urlParam, "https://media.discordapp.net/") {
		http.Error(w, "domain not allowed", http.StatusForbidden)
		return
	}
	req, err := http.NewRequest("GET", urlParam, nil)
	if err != nil {
		http.Error(w, "invalid url", http.StatusBadRequest)
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != 200 {
		http.Error(w, "fetch failed", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	if ct := resp.Header.Get("Content-Type"); ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	w.Header().Set("Cache-Control", "public, max-age=3600")
	io.Copy(w, resp.Body)
}

func (h *Handler) UploadImage(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.URL) == "" {
		json.NewEncoder(w).Encode(map[string]string{"error": "URL tidak valid"})
		return
	}

	// Shortcut: CDN/media Discord langsung convert
	for _, prefix := range []string{
		"https://cdn.discordapp.com/attachments/",
		"https://media.discordapp.net/attachments/",
	} {
		if strings.HasPrefix(req.URL, prefix) {
			idx := strings.Index(req.URL, "attachments/")
			mpURL := "mp:" + req.URL[idx:]
			if q := strings.Index(mpURL, "?"); q != -1 {
				mpURL = mpURL[:q]
			}
			json.NewEncoder(w).Encode(map[string]string{"url": mpURL})
			return
		}
	}

	dlReq, _ := http.NewRequest("GET", req.URL, nil)
	dlReq.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/120.0.0.0 Safari/537.36")
	dlReq.Header.Set("Referer", req.URL)
	imgResp, err := http.DefaultClient.Do(dlReq)
	if err != nil || imgResp.StatusCode != 200 {
		errMsg := "Gagal download gambar dari URL"
		if err == nil {
			errMsg = fmt.Sprintf("Gagal download gambar: HTTP %d", imgResp.StatusCode)
		}
		json.NewEncoder(w).Encode(map[string]string{"error": errMsg})
		return
	}
	defer imgResp.Body.Close()

	imgBytes, err := io.ReadAll(imgResp.Body)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "Gagal baca gambar"})
		return
	}

	filename := filepath.Base(req.URL)
	if !strings.Contains(filename, ".") {
		filename = "image.png"
	}
	if idx := strings.Index(filename, "?"); idx != -1 {
		filename = filename[:idx]
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("content", "rpc-image")
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "Gagal buat form"})
		return
	}
	if _, err = part.Write(imgBytes); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "Gagal write bytes"})
		return
	}
	writer.Close()

	hookReq, _ := http.NewRequest("POST", discordWebhook+"?wait=true", &body)
	hookReq.Header.Set("Content-Type", writer.FormDataContentType())
	hookResp, err := http.DefaultClient.Do(hookReq)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "Gagal upload ke Discord"})
		return
	}
	defer hookResp.Body.Close()

	var discordResp struct {
		Attachments []struct {
			ProxyURL string `json:"proxy_url"`
		} `json:"attachments"`
	}
	if err := json.NewDecoder(hookResp.Body).Decode(&discordResp); err != nil || len(discordResp.Attachments) == 0 {
		json.NewEncoder(w).Encode(map[string]string{"error": "Discord tidak return attachment"})
		return
	}

	proxyURL := discordResp.Attachments[0].ProxyURL
	idx := strings.Index(proxyURL, "attachments/")
	if idx == -1 {
		json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("Format URL Discord tidak dikenali: %s", proxyURL)})
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"url": "mp:" + proxyURL[idx:]})
}

func (h *Handler) StartRPC(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token string `json:"token"`
		gateway.RPCData
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Token == "" {
		http.Error(w, "Token is required", http.StatusBadRequest)
		return
	}

	gw, exists := h.SM.Get(req.Token)
	if !exists {
		gw = gateway.NewGateway(req.Token)
		errCh := make(chan error, 1)
		go func() { errCh <- gw.Connect() }()
		if err := <-errCh; err != nil {
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		h.SM.Add(req.Token, gw)
	}

	if err := gw.UpdateRPC(req.RPCData); err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *Handler) StopRPC(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	gw, exists := h.SM.Get(req.Token)
	if exists {
		gw.ClearRPC()
	}
	h.SM.Remove(req.Token)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *Handler) GetStatus(w http.ResponseWriter, r *http.Request) {
	// Token via header, bukan query string
	token := r.Header.Get("X-Discord-Token")
	if token == "" {
		// fallback query string untuk kompatibilitas
		token = r.URL.Query().Get("token")
	}
	gw, exists := h.SM.Get(token)
	if !exists {
		json.NewEncoder(w).Encode(map[string]interface{}{"active": false, "status": "Disconnected"})
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"active": true, "status": gw.Status})
}

// Dummy handler buat chi routing — tidak dipakai lagi tapi biar ga break kalau route masih terdaftar
func (h *Handler) GetProfiles(w http.ResponseWriter, r *http.Request)    { w.WriteHeader(http.StatusGone) }
func (h *Handler) SaveProfile(w http.ResponseWriter, r *http.Request)    { w.WriteHeader(http.StatusGone) }
func (h *Handler) DeleteProfile(w http.ResponseWriter, r *http.Request)  { w.WriteHeader(http.StatusGone) }
func (h *Handler) GetLastProfile(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusGone) }
func (h *Handler) SetLastProfile(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusGone) }

