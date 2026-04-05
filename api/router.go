package api

import (
	"net/http"

	"chatgpt2api/internal/auth"
	"chatgpt2api/internal/config"
	"chatgpt2api/internal/middleware"
)

func SetupRouter(cfg *config.Config) http.Handler {
	appKey := cfg.GetString("app.app_key", "chatgpt2api")
	apiKeyFn := func() string { return cfg.GetString("app.api_key", "") }

	mux := http.NewServeMux()

	mux.Handle("POST /v1/images/generations", auth.VerifyAPIKeyFunc(apiKeyFn)(handleImageGenerations()))
	mux.Handle("POST /v1/images/edits", auth.VerifyAPIKeyFunc(apiKeyFn)(handleImageEdits()))
	mux.Handle("GET /v1/models", auth.VerifyAPIKeyFunc(apiKeyFn)(handleModels()))

	mux.Handle("/v1/admin/tokens", auth.VerifyAppKey(appKey)(handleTokens()))
	mux.Handle("/v1/admin/config", auth.VerifyAppKey(appKey)(handleConfig(cfg)))
	mux.Handle("POST /v1/admin/verify", auth.VerifyAppKey(appKey)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
	})))

	// Static file serving (cached images)
	mux.Handle("GET /v1/files/image/", handleImageFile())

	// Page routes
	pages := NewPageHandler()
	mux.HandleFunc("GET /{$}", pages.Root())
	mux.HandleFunc("GET /admin", pages.AdminRoot())
	mux.HandleFunc("GET /admin/login", pages.ServePage("static/admin/pages/login.html"))
	mux.HandleFunc("GET /admin/token", pages.ServePage("static/admin/pages/token.html"))
	mux.Handle("GET /static/", pages.Static())

	mux.HandleFunc("GET /health", handleHealth)

	return middleware.CORS(middleware.RequestID(middleware.Logger(mux)))
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}
