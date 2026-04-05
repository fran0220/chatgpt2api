package api

import (
	"crypto/sha256"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"chatgpt2api/handler"
)

const imageDir = "data/tmp/image"

// downloadAndCache downloads an upstream image using the ChatGPT client's transport
// (Chrome TLS fingerprint), saves to local disk, and returns the local filename.
func downloadAndCache(client *handler.ChatGPTClient, upstreamURL string) (string, error) {
	// Generate a stable filename from the URL
	hash := sha256.Sum256([]byte(upstreamURL))
	filename := fmt.Sprintf("%x.png", hash[:12])
	localPath := filepath.Join(imageDir, filename)

	// Check cache
	if _, err := os.Stat(localPath); err == nil {
		return filename, nil
	}

	if err := os.MkdirAll(imageDir, 0o755); err != nil {
		return "", err
	}

	data, err := client.DownloadBytes(upstreamURL)
	if err != nil {
		return "", fmt.Errorf("download upstream image: %w", err)
	}

	tmpFile := localPath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0o644); err != nil {
		return "", err
	}
	if err := os.Rename(tmpFile, localPath); err != nil {
		return "", err
	}

	slog.Info("cached image", "file", filename, "size", len(data))
	return filename, nil
}

// gatewayImageURL builds the public URL for a cached image.
func gatewayImageURL(r *http.Request, filename string) string {
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	host := r.Host
	return fmt.Sprintf("%s://%s/v1/files/image/%s", scheme, host, filename)
}

// handleImageFile serves cached images from data/tmp/image/.
func handleImageFile() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/v1/files/image/")
		name = strings.ReplaceAll(name, "/", "-")
		if name == "" {
			writeError(w, http.StatusNotFound, "image not found")
			return
		}

		path := filepath.Join(imageDir, filepath.Base(name))
		info, err := os.Stat(path)
		if err != nil || !info.Mode().IsRegular() {
			writeError(w, http.StatusNotFound, "image not found")
			return
		}

		ext := strings.ToLower(filepath.Ext(path))
		contentTypes := map[string]string{
			".png":  "image/png",
			".jpg":  "image/jpeg",
			".jpeg": "image/jpeg",
			".webp": "image/webp",
		}
		ct := contentTypes[ext]
		if ct == "" {
			ct = "image/png"
		}

		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		w.Header().Set("Content-Type", ct)
		http.ServeFile(w, r, path)
	}
}
