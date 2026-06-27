package main

import (
	"ClaudiaRPC/internal/handler"
	"ClaudiaRPC/internal/session"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// silentRoutes adalah prefix route yang tidak perlu di-log
var silentRoutes = []string{
	"/api/rpc/status",
	"/api/proxy-image",
}

func isSilent(path string) bool {
	for _, p := range silentRoutes {
		if strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

func filteredLogger(next http.Handler) http.Handler {
	logger := middleware.Logger
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isSilent(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		logger(next).ServeHTTP(w, r)
	})
}

func main() {
	sm := session.NewSessionManager()
	h := handler.NewHandler(sm)

	r := chi.NewRouter()
	r.Use(filteredLogger)
	r.Use(middleware.Recoverer)

	r.Route("/api/rpc", func(r chi.Router) {
		r.Post("/start", h.StartRPC)
		r.Post("/stop", h.StopRPC)
		r.Get("/status", h.GetStatus)
	})

	r.Post("/api/image", h.UploadImage)
	r.Get("/api/proxy-image", h.ProxyImage)
	r.Get("/api/profiles/last", h.GetLastProfile)
	r.Post("/api/profiles/last", h.SetLastProfile)

	r.Route("/api/profiles", func(r chi.Router) {
		r.Get("/", h.GetProfiles)
		r.Post("/", h.SaveProfile)
		r.Delete("/{name}", h.DeleteProfile)
	})

	workDir, _ := os.Getwd()
	filesDir := http.Dir(filepath.Join(workDir, "static"))
	FileServer(r, "/", filesDir)

	log.Println("Server starting on :8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatal(err)
	}
}

func FileServer(r chi.Router, path string, root http.FileSystem) {
	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, http.RedirectHandler(path+"/", http.StatusMovedPermanently).ServeHTTP)
		path += "/"
	}
	path += "*"

	r.Get(path, func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.RouteContext(r.Context())
		pathPrefix := rctx.RoutePattern()
		if pathPrefix[len(pathPrefix)-1] == '*' {
			pathPrefix = pathPrefix[:len(pathPrefix)-1]
		}
		fs := http.StripPrefix(pathPrefix, http.FileServer(root))
		fs.ServeHTTP(w, r)
	})
}
