package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/tafu/analytics-service/internal/domain"
	"github.com/tafu/analytics-service/internal/infrastructure/clickhouse"
	natsClient "github.com/tafu/analytics-service/internal/infrastructure/nats"
)

type Handler struct {
	nats *natsClient.Client
	repo *clickhouse.EventRepository
}

func NewHandler(nats *natsClient.Client, repo *clickhouse.EventRepository) *Handler {
	return &Handler{
		nats: nats,
		repo: repo,
	}
}

// CollectEvent handles POST /analytics/collect
// Receives events from frontend, validates, and publishes to NATS.
func (h *Handler) CollectEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var input domain.EventInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	// Basic validation
	if input.EventType == "" {
		http.Error(w, "event_type is required", http.StatusBadRequest)
		return
	}

	// Validate metadata is valid JSON if provided
	if input.Metadata != "" && !isValidJSON(input.Metadata) {
		http.Error(w, "metadata must be valid JSON", http.StatusBadRequest)
		return
	}

	// Get client IP
	ipAddress := getClientIP(r)

	// Get User Agent
	userAgent := r.UserAgent()

	// Create event with server-side fields
	event := domain.NewUserEvent(input, ipAddress, userAgent)

	// Publish to NATS (async - non-blocking)
	if err := h.nats.Publish(r.Context(), event); err != nil {
		log.Printf("Failed to publish event: %v", err)
		http.Error(w, "Failed to queue event", http.StatusInternalServerError)
		return
	}

	// Return 202 Accepted immediately
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"status":   "accepted",
		"event_id": event.EventID.String(),
	})
}

// GetProductViews handles GET /analytics/products/{sku_id}/views
func (h *Handler) GetProductViews(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract sku_id from path
	path := strings.TrimPrefix(r.URL.Path, "/analytics/products/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[1] != "views" {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	skuID := parts[0]

	// Parse time range from query params
	start, end := parseTimeRange(r)

	results, err := h.repo.GetProductViews(r.Context(), skuID, start, end)
	if err != nil {
		log.Printf("Failed to get product views: %v", err)
		http.Error(w, "Failed to get product views", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// GetConversionRate handles GET /analytics/conversion
func (h *Handler) GetConversionRate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	start, end := parseTimeRange(r)

	result, err := h.repo.GetConversionRate(r.Context(), start, end)
	if err != nil {
		log.Printf("Failed to get conversion rate: %v", err)
		http.Error(w, "Failed to get conversion rate", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// HealthCheck handles GET /health
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
	})
}

// Helper functions

func isValidJSON(s string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(s), &js) == nil
}

func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxied requests)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	if colonIdx := strings.LastIndex(ip, ":"); colonIdx != -1 {
		ip = ip[:colonIdx]
	}
	return ip
}

func parseTimeRange(r *http.Request) (start, end time.Time) {
	// Default: last 24 hours
	end = time.Now().UTC()
	start = end.Add(-24 * time.Hour)

	if startStr := r.URL.Query().Get("start"); startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			start = t
		}
	}

	if endStr := r.URL.Query().Get("end"); endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			end = t
		}
	}

	return start, end
}
