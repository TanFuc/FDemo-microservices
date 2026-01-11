package api

import (
	"net/http"
	"strings"
)

func NewRouter(h *Handler) http.Handler {
	mux := http.NewServeMux()

	// Health check
	mux.HandleFunc("/health", h.HealthCheck)

	// Ingestion endpoint
	mux.HandleFunc("/analytics/collect", h.CollectEvent)

	// Analytics read endpoints
	mux.HandleFunc("/analytics/products/", h.GetProductViews)
	mux.HandleFunc("/analytics/conversion", h.GetConversionRate)

	// Wrap with middleware
	return corsMiddleware(loggingMiddleware(mux))
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip logging for health checks
		if r.URL.Path != "/health" {
			// Log is handled by the handler itself or you can add logging here
		}
		next.ServeHTTP(w, r)
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Max-Age", "86400")
		}

		// Handle preflight
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// ExtractPathParam extracts a parameter from the URL path.
func ExtractPathParam(path, prefix, suffix string) string {
	path = strings.TrimPrefix(path, prefix)
	if suffix != "" {
		if idx := strings.Index(path, suffix); idx != -1 {
			path = path[:idx]
		}
	}
	return strings.Trim(path, "/")
}
