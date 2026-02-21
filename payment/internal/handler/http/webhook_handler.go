package http

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp/fasthttpadaptor"

	"microservices/payment/internal/domain"
	"microservices/payment/internal/usecase"
	"microservices/pkg/logger"
)

// WebhookHandler handles webhook HTTP requests from payment providers
type WebhookHandler struct {
	uc *usecase.PaymentUseCase
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(uc *usecase.PaymentUseCase) *WebhookHandler {
	return &WebhookHandler{
		uc: uc,
	}
}

// HandleWebhook handles POST /api/v1/webhooks/:provider
func (h *WebhookHandler) HandleWebhook(c *fiber.Ctx) error {
	providerStr := c.Params("provider")
	provider := domain.Provider(strings.ToUpper(providerStr))

	if !provider.IsValid() {
		logger.Warn().
			Str("provider", providerStr).
			Str("ip", getClientIP(c)).
			Msg("Webhook received for invalid provider")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid provider",
		})
	}

	// Read body once
	body := c.Body()
	if len(body) == 0 {
		logger.Error().
			Str("provider", string(provider)).
			Msg("Failed to read webhook body")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Failed to read body",
		})
	}

	// Log raw webhook for debugging
	logger.Debug().
		Str("provider", string(provider)).
		Int("body_length", len(body)).
		Str("ip", getClientIP(c)).
		Msg("Webhook received")

	// Convert Fiber request to http.Request for use case
	httpReq, err := convertToHTTPRequest(c, body)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to convert request")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Internal error",
		})
	}

	clientIP := getClientIP(c)

	result, err := h.uc.HandleWebhook(c.Context(), provider, httpReq, clientIP)
	if err != nil {
		if errors.Is(err, usecase.ErrWebhookVerification) {
			logger.Warn().
				Str("provider", string(provider)).
				Str("ip", clientIP).
				Msg("Webhook signature verification failed")
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Signature verification failed",
			})
		}

		if errors.Is(err, usecase.ErrTransactionNotFound) {
			logger.Warn().
				Str("provider", string(provider)).
				Str("ip", clientIP).
				Msg("Webhook for unknown transaction")
			// Return 200 to prevent retries for unknown transactions
			return c.JSON(fiber.Map{
				"success":  true,
				"received": true,
			})
		}

		logger.Error().
			Err(err).
			Str("provider", string(provider)).
			Str("ip", clientIP).
			Msg("Failed to handle webhook")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Internal error",
		})
	}

	// Always return 200 OK to acknowledge receipt
	response := fiber.Map{
		"success":  true,
		"received": true,
	}

	if result != nil && result.Processed {
		response["transaction_id"] = result.TransactionID.String()
		response["status"] = result.Status.String()
		response["idempotent"] = result.IsIdempotent
	}

	return c.JSON(response)
}

// convertToHTTPRequest converts a Fiber context to an http.Request
func convertToHTTPRequest(c *fiber.Ctx, body []byte) (*http.Request, error) {
	var httpReq http.Request
	fasthttpadaptor.ConvertRequest(c.Context(), &httpReq, true)

	// Reset body since it was already read
	httpReq.Body = io.NopCloser(bytes.NewReader(body))

	return &httpReq, nil
}

// getClientIP extracts the client IP from the Fiber context
func getClientIP(c *fiber.Ctx) string {
	// Check X-Forwarded-For header
	xff := c.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	xri := c.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to remote IP
	return c.IP()
}
