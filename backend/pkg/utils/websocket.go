package utils

import (
	"net/http"
	"os"
	"strings"
)

// defaultOrigins used when WEBSOCKET_ALLOWED_ORIGINS is not set.
var defaultOrigins = map[string]bool{
	"http://localhost:3000":  true,
	"http://localhost:5173":  true,
	"http://localhost:8080":  true,
	"https://localhost:3000": true,
	"https://localhost:5173": true,
	"https://localhost:8080": true,
	"https://app.waas.dev":   true,
}

// GetAllowedOrigins returns allowed WebSocket origins from the
// WEBSOCKET_ALLOWED_ORIGINS env var (comma-separated). Falls back to
// development defaults when unset.
func GetAllowedOrigins() map[string]bool {
	origins := os.Getenv("WEBSOCKET_ALLOWED_ORIGINS")
	if origins == "" {
		return defaultOrigins
	}
	allowed := make(map[string]bool)
	for _, o := range strings.Split(origins, ",") {
		o = strings.TrimSpace(o)
		if o != "" {
			allowed[o] = true
		}
	}
	return allowed
}

// CheckWebSocketOrigin returns a function suitable for websocket.Upgrader.CheckOrigin
// that validates origins against WEBSOCKET_ALLOWED_ORIGINS.
func CheckWebSocketOrigin() func(r *http.Request) bool {
	allowed := GetAllowedOrigins()
	return func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true // Same-origin / non-browser clients
		}
		return allowed[origin]
	}
}
