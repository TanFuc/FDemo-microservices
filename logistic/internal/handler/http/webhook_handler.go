package http

import (
	"github.com/gofiber/fiber/v2"

	"microservices/logistic/internal/api/dto"
	"microservices/logistic/internal/core/domain"
	"microservices/logistic/internal/core/services"
	"microservices/pkg/response"
)

// WebhookHandler handles webhook-related HTTP requests
type WebhookHandler struct {
	webhookService *services.WebhookService
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(webhookService *services.WebhookService) *WebhookHandler {
	return &WebhookHandler{
		webhookService: webhookService,
	}
}

// HandleWebhook processes webhooks from shipping providers
// @Summary Handle webhook
// @Description Process a webhook from a shipping provider
// @Tags Webhooks
// @Accept json
// @Produce json
// @Param provider path string true "Provider name"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /webhooks/{provider} [post]
func (h *WebhookHandler) HandleWebhook(c *fiber.Ctx) error {
	providerStr := c.Params("provider")
	providerName := domain.ProviderName(providerStr)
	if !providerName.IsValid() {
		return response.BadRequest(c, "Invalid provider name")
	}

	// Get raw body
	body := c.Body()
	if len(body) == 0 {
		return response.BadRequest(c, "Empty request body")
	}

	// Process webhook
	result, err := h.webhookService.ProcessWebhook(c.Context(), providerName, body)
	if err != nil {
		return response.InternalError(c, err.Error())
	}

	return response.Success(c, &dto.WebhookResponse{
		TrackingCode:  result.TrackingCode,
		OldStatus:     string(result.OldStatus),
		NewStatus:     string(result.NewStatus),
		CarrierStatus: result.CarrierStatus,
	})
}
