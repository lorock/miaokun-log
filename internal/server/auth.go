package server

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"strings"
)

// AuthConfig holds authentication configuration
type AuthConfig struct {
	Enabled  bool
	Username string
	Password string
}

// SimpleAuthMiddleware provides basic HTTP authentication
func SimpleAuthMiddleware(auth AuthConfig, next http.HandlerFunc) http.HandlerFunc {
	if !auth.Enabled || (auth.Username == "" && auth.Password == "") {
		return next
	}

	return func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if !ok {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Use constant-time comparison to prevent timing attacks
		userMatch := subtle.ConstantTimeCompare([]byte(username), []byte(auth.Username)) == 1
		passMatch := subtle.ConstantTimeCompare([]byte(password), []byte(auth.Password)) == 1

		if !userMatch || !passMatch {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}

// APIKeyAuthMiddleware provides API key based authentication
func APIKeyAuthMiddleware(validKeys []string, next http.HandlerFunc) http.HandlerFunc {
	if len(validKeys) == 0 {
		return next
	}

	return func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			apiKey = r.URL.Query().Get("api_key")
		}

		if apiKey == "" {
			http.Error(w, `{"error": "API key required", "code": "MISSING_API_KEY"}`, http.StatusUnauthorized)
			return
		}

		valid := false
		for _, key := range validKeys {
			if subtle.ConstantTimeCompare([]byte(apiKey), []byte(key)) == 1 {
				valid = true
				break
			}
		}

		if !valid {
			http.Error(w, `{"error": "Invalid API key", "code": "INVALID_API_KEY"}`, http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}

// sanitizePath prevents path traversal attacks
func sanitizePath(path string) (string, error) {
	// Clean the path
	path = strings.TrimSpace(path)

	// Reject empty paths
	if path == "" {
		return "", nil
	}

	// Reject paths containing traversal sequences
	if strings.Contains(path, "..") {
		return "", fmt.Errorf("path contains traversal sequences")
	}

	// Normalize path separators
	path = strings.ReplaceAll(path, "\\", "/")

	// Ensure path starts with /
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return path, nil
}

// isPathAllowed checks if the requested path is within allowed directories
func isPathAllowed(path string, allowedPaths []string) bool {
	if len(allowedPaths) == 0 {
		return true
	}

	for _, allowed := range allowedPaths {
		if strings.HasPrefix(path, allowed) {
			return true
		}
	}
	return false
}
