package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"microservices/logistic/internal/api/dto"
	"microservices/logistic/internal/core/domain"
	"microservices/logistic/internal/core/services"
)

// WebhookHandler handles webhook requests from providers
type WebhookHandler struct {
	webhookService *services.WebhookService
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(webhookService *services.WebhookService) *WebhookHandler {
	return &WebhookHandler{
		webhookService: webhookService,
	}
}

// HandleWebhook processes incoming webhooks from providers
// POST /api/v1/webhooks/:provider
func (h *WebhookHandler) HandleWebhook(c *gin.Context) {
	providerStr := c.Param("provider")
	providerName := domain.ProviderName(providerStr)

	if !providerName.IsValid() {
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse("INVALID_PROVIDER", "Invalid provider name"))
		return
	}

	result, err := h.webhookService.HandleWebhook(c.Request.Context(), providerName, c.Request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.NewErrorResponse("WEBHOOK_ERROR", err.Error()))
		return
	}

	c.JSON(http.StatusOK, dto.NewSuccessResponse(&dto.WebhookResponse{
		TrackingCode:  result.TrackingCode,
		OldStatus:     string(result.OldStatus),
		NewStatus:     string(result.NewStatus),
		CarrierStatus: result.CarrierStatus,
	}))
}
