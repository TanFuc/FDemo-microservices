package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/tafu/payment-service/internal/domain"
	"github.com/tafu/payment-service/internal/usecase"
)

// WebhookHandler handles webhook HTTP requests from payment providers
type WebhookHandler struct {
	uc     *usecase.PaymentUseCase
	logger *slog.Logger
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(uc *usecase.PaymentUseCase, logger *slog.Logger) *WebhookHandler {
	return &WebhookHandler{
		uc:     uc,
		logger: logger,
	}
}

// RegisterRoutes registers the webhook routes
func (h *WebhookHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1/webhooks", func(r chi.Router) {
		r.Post("/{provider}", h.HandleWebhook)
	})
}

// HandleWebhook handles POST /api/v1/webhooks/{provider}
func (h *WebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	providerStr := chi.URLParam(r, "provider")
	provider := domain.Provider(strings.ToUpper(providerStr))

	if !provider.IsValid() {
		h.logger.Warn("webhook received for invalid provider",
			"provider", providerStr,
			"ip", getClientIP(r),
		)
		http.Error(w, "invalid provider", http.StatusBadRequest)
		return
	}

	// Read body once and create a new reader for the use case
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("failed to read webhook body",
			"error", err,
			"provider", provider,
		)
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	// Log raw webhook for debugging
	h.logger.Debug("webhook received",
		"provider", provider,
		"body_length", len(body),
		"ip", getClientIP(r),
	)

	// Create a new request with the body for the use case
	r.Body = io.NopCloser(bytes.NewReader(body))

	clientIP := getClientIP(r)

	result, err := h.uc.HandleWebhook(r.Context(), provider, r, clientIP)
	if err != nil {
		if errors.Is(err, usecase.ErrWebhookVerification) {
			h.logger.Warn("webhook signature verification failed",
				"provider", provider,
				"ip", clientIP,
			)
			// Return 400 to stop hackers from faking payments
			http.Error(w, "signature verification failed", http.StatusBadRequest)
			return
		}

		if errors.Is(err, usecase.ErrTransactionNotFound) {
			h.logger.Warn("webhook for unknown transaction",
				"provider", provider,
				"ip", clientIP,
			)
			// Return 200 to prevent retries for unknown transactions
			w.WriteHeader(http.StatusOK)
			return
		}

		h.logger.Error("failed to handle webhook",
			"error", err,
			"provider", provider,
			"ip", clientIP,
		)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Always return 200 OK to acknowledge receipt
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"received": true,
	}

	if result != nil && result.Processed {
		response["transaction_id"] = result.TransactionID.String()
		response["status"] = result.Status.String()
		response["idempotent"] = result.IsIdempotent
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("failed to encode webhook response", "error", err)
	}
}

// getClientIP extracts the client IP from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (common for proxies/load balancers)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	// RemoteAddr is in the format "IP:port", extract just the IP
	addr := r.RemoteAddr
	if colonIdx := strings.LastIndex(addr, ":"); colonIdx != -1 {
		return addr[:colonIdx]
	}
	return addr
}
