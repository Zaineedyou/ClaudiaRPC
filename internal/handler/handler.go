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
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	SM       *session.SessionManager
	profiles map[string]gateway.RPCData
	pmu      sync.RWMutex
}

const discordWebhook = "https://discord.com/api/webhooks/1519634360008048811/TGGC4JBx2ty0mS6FOMHEj7Eaaq1BlQcJTFmgVj-CDdmiQirBzZ-DBIP8iZNUyYD2axL9"

const profilesFile = "profiles.json"

func NewHandler(sm *session.SessionManager) *Handler {
	h := &Handler{
		SM:       sm,
		profiles: make(map[string]gateway.RPCData),
	}
	h.loadProfiles()
	return h
}

func (h *Handler) loadProfiles() {
	data, err := os.ReadFile(profilesFile)
	if err != nil {
		return // file belum ada, skip
	}
	h.pmu.Lock()
	defer h.pmu.Unlock()
	_ = json.Unmarshal(data, &h.profiles)
}

func (h *Handler) saveProfiles() {
	h.pmu.RLock()
	data, err := json.MarshalIndent(h.profiles, "", "  ")
	h.pmu.RUnlock()
	if err != nil {
		return
	}
	_ = os.WriteFile(profilesFile, data, 0644)
}

func (h *Handler) ProxyImage(w http.ResponseWriter, r *http.Request) {
	urlParam := r.URL.Query().Get("url")
	if urlParam == "" {
		http.Error(w, "url param required", http.StatusBadRequest)
		return
	}

	// Hanya izinkan domain Discord
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

	ct := resp.Header.Get("Content-Type")
	if ct != "" {
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

	// 1. Download gambar dari URL yang dikasih user
	imgResp, err := http.Get(req.URL)
	if err != nil || imgResp.StatusCode != 200 {
		json.NewEncoder(w).Encode(map[string]string{"error": "Gagal download gambar dari URL"})
		return
	}
	defer imgResp.Body.Close()

	imgBytes, err := io.ReadAll(imgResp.Body)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "Gagal baca gambar"})
		return
	}

	// Tentukan nama file dari URL
	urlPath := req.URL
	filename := filepath.Base(urlPath)
	if !strings.Contains(filename, ".") {
		filename = "image.png"
	}
	// Buang query string kalau ada
	if idx := strings.Index(filename, "?"); idx != -1 {
		filename = filename[:idx]
	}

	// 2. Upload ke Discord webhook sebagai multipart form
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Form field "content" wajib ada
	_ = writer.WriteField("content", "rpc-image")

	// Form field "file" dengan bytes gambar
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

	// 3. POST ke webhook dengan ?wait=true supaya dapat response JSON
	hookURL := discordWebhook + "?wait=true"
	hookReq, _ := http.NewRequest("POST", hookURL, &body)
	hookReq.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	hookResp, err := client.Do(hookReq)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"error": "Gagal upload ke Discord"})
		return
	}
	defer hookResp.Body.Close()

	// 4. Parse response, ambil proxy_url dari attachments[0]
	var discordResp struct {
		Attachments []struct {
			ProxyURL string `json:"proxy_url"`
			URL      string `json:"url"`
		} `json:"attachments"`
	}
	if err := json.NewDecoder(hookResp.Body).Decode(&discordResp); err != nil || len(discordResp.Attachments) == 0 {
		json.NewEncoder(w).Encode(map[string]string{"error": "Discord tidak return attachment"})
		return
	}

	// 5. Convert proxy_url ke format mp:attachments/...
	proxyURL := discordResp.Attachments[0].ProxyURL
	idx := strings.Index(proxyURL, "attachments/")
	if idx == -1 {
		json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("Format URL Discord tidak dikenali: %s", proxyURL)})
		return
	}
	mpURL := "mp:" + proxyURL[idx:]

	json.NewEncoder(w).Encode(map[string]string{"url": mpURL})
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
		// Connect() blocking sekali, reconnect loop jalan di background
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
		// ClearRPC dulu (kirim payload kosong ke Discord), baru disconnect
		gw.ClearRPC()
	}
	// Remove() akan close websocket — karena Close() sekarang acquire mu,
	// dia nunggu ClearRPC selesai write baru nutup koneksi
	h.SM.Remove(req.Token)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *Handler) GetStatus(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	gw, exists := h.SM.Get(token)
	if !exists {
		json.NewEncoder(w).Encode(map[string]interface{}{"active": false, "status": "Disconnected"})
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"active": true, "status": gw.Status})
}

const lastProfileFile = "last_profile.txt"

func (h *Handler) GetLastProfile(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile(lastProfileFile)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]string{"name": ""})
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"name": strings.TrimSpace(string(data))})
}

func (h *Handler) SetLastProfile(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_ = os.WriteFile(lastProfileFile, []byte(req.Name), 0644)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *Handler) GetProfiles(w http.ResponseWriter, r *http.Request) {
	h.pmu.RLock()
	defer h.pmu.RUnlock()
	json.NewEncoder(w).Encode(h.profiles)
}

func (h *Handler) SaveProfile(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string          `json:"name"`
		Data gateway.RPCData `json:"data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	h.pmu.Lock()
	h.profiles[req.Name] = req.Data
	h.pmu.Unlock()
	h.saveProfiles()

	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *Handler) DeleteProfile(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	h.pmu.Lock()
	delete(h.profiles, name)
	h.pmu.Unlock()
	h.saveProfiles()
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
