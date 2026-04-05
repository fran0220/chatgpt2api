package auth

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"
)

// VerifyAPIKey returns middleware that validates Bearer tokens against a
// comma-separated list of allowed API keys. If apiKeys is empty, all
// requests are allowed (no auth required).
func VerifyAPIKey(apiKeys string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			keys := parseKeys(apiKeys)
			if len(keys) == 0 {
				next.ServeHTTP(w, r)
				return
			}

			token, ok := bearerToken(r)
			if !ok {
				writeUnauthorized(w, "Missing authentication token")
				return
			}
			for _, key := range keys {
				if secureCompare(token, key) {
					next.ServeHTTP(w, r)
					return
				}
			}
			writeUnauthorized(w, "Invalid authentication token")
		})
	}
}

// VerifyAPIKeyFunc is like VerifyAPIKey but reads the key list dynamically
// on each request via the provided function.
func VerifyAPIKeyFunc(keysFn func() string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			keys := parseKeys(keysFn())
			if len(keys) == 0 {
				next.ServeHTTP(w, r)
				return
			}

			token, ok := bearerToken(r)
			if !ok {
				writeUnauthorized(w, "Missing authentication token")
				return
			}
			for _, key := range keys {
				if secureCompare(token, key) {
					next.ServeHTTP(w, r)
					return
				}
			}
			writeUnauthorized(w, "Invalid authentication token")
		})
	}
}

// VerifyAppKey returns middleware that validates Bearer tokens against a
// single admin app key.
func VerifyAppKey(appKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := strings.TrimSpace(appKey)
			if key == "" {
				writeUnauthorized(w, "App key is not configured")
				return
			}

			token, ok := bearerToken(r)
			if !ok {
				writeUnauthorized(w, "Missing authentication token")
				return
			}
			if !secureCompare(token, key) {
				writeUnauthorized(w, "Invalid authentication token")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func parseKeys(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	keys := make([]string, 0, len(parts))
	for _, p := range parts {
		if k := strings.TrimSpace(p); k != "" {
			keys = append(keys, k)
		}
	}
	return keys
}

func bearerToken(r *http.Request) (string, bool) {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if header == "" {
		return "", false
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}
	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", false
	}
	return token, true
}

func secureCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func writeUnauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("WWW-Authenticate", "Bearer")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
