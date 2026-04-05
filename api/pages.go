package api

import (
	"errors"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
)

type PageHandler struct {
	publicFS fs.FS
	staticFS fs.FS
}

func NewPageHandler() *PageHandler {
	publicDir := resolvePublicDir()
	var publicFS fs.FS
	var staticFS fs.FS
	if publicDir != "" {
		publicFS = os.DirFS(publicDir)
		if sub, err := fs.Sub(publicFS, "static"); err == nil {
			staticFS = sub
		}
	}
	return &PageHandler{publicFS: publicFS, staticFS: staticFS}
}

func (h *PageHandler) Static() http.Handler {
	if h.staticFS == nil {
		return http.NotFoundHandler()
	}
	return http.StripPrefix("/static/", http.FileServer(http.FS(h.staticFS)))
}

func (h *PageHandler) Root() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/admin/login", http.StatusTemporaryRedirect)
	}
}

func (h *PageHandler) AdminRoot() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/admin/login", http.StatusTemporaryRedirect)
	}
}

func (h *PageHandler) ServePage(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if h.publicFS == nil {
			http.NotFound(w, r)
			return
		}
		if _, err := fs.Stat(h.publicFS, name); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				http.NotFound(w, r)
				return
			}
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		http.ServeFileFS(w, r, h.publicFS, name)
	}
}

func resolvePublicDir() string {
	workingDir, _ := os.Getwd()
	exePath, _ := os.Executable()
	candidates := []string{
		filepath.Join(workingDir, "_public"),
		filepath.Join(filepath.Dir(exePath), "_public"),
	}
	for _, c := range candidates {
		if c == "" {
			continue
		}
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			return c
		}
	}
	return ""
}
