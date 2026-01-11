package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"

	"tafu-logistic/logistics-service/internal/core/domain"
	"tafu-logistic/logistics-service/internal/core/ports"
)

// WebhookService handles webhook processing from providers
type WebhookService struct {
	providerFactory   ports.ProviderFactory
	repository        ports.ShippingOrderRepository
	webhookLogRepo    ports.WebhookLogRepository
	publisher         ports.EventPublisher
}

// NewWebhookService creates a new webhook service
func NewWebhookService(
	providerFactory ports.ProviderFactory,
	repository ports.ShippingOrderRepository,
	publisher ports.EventPublisher,
) *WebhookService {
	return &WebhookService{
		providerFactory: providerFactory,
		repository:      repository,
		publisher:       publisher,
	}
}

// NewWebhookServiceWithLogging creates a webhook service with logging support
func NewWebhookServiceWithLogging(
	providerFactory ports.ProviderFactory,
	repository ports.ShippingOrderRepository,
	webhookLogRepo ports.WebhookLogRepository,
	publisher ports.EventPublisher,
) *WebhookService {
	return &WebhookService{
		providerFactory:   providerFactory,
		repository:        repository,
		webhookLogRepo:    webhookLogRepo,
		publisher:         publisher,
	}
}

// HandleWebhookResult contains the result of processing a webhook
type HandleWebhookResult struct {
	TrackingCode  string              `json:"tracking_code"`
	OldStatus     domain.SystemStatus `json:"old_status"`
	NewStatus     domain.SystemStatus `json:"new_status"`
	CarrierStatus string              `json:"carrier_status"`
}

// HandleWebhook processes an incoming webhook from a provider
func (s *WebhookService) HandleWebhook(ctx context.Context, providerName domain.ProviderName, r *http.Request) (*HandleWebhookResult, error) {
	// Get the provider
	provider, err := s.providerFactory.GetProvider(providerName)
	if err != nil {
		return nil, fmt.Errorf("provider not found: %w", err)
	}

	// Parse the webhook
	payload, err := provider.ParseWebhook(r)
	if err != nil {
		s.logWebhookError(ctx, providerName, "", "", nil, http.StatusBadRequest, err)
		return nil, fmt.Errorf("failed to parse webhook: %w", err)
	}

	// Create webhook log entry
	webhookLog := domain.NewWebhookLog(
		providerName,
		payload.TrackingCode,
		payload.CarrierStatus,
		json.RawMessage(payload.RawPayload),
	)

	// Find the shipping order
	var order *domain.ShippingOrder

	// Try by tracking code first
	if payload.TrackingCode != "" {
		order, err = s.repository.GetByTrackingCode(ctx, payload.TrackingCode)
		if err != nil {
			s.saveWebhookLog(ctx, webhookLog, nil, http.StatusInternalServerError, err)
			return nil, fmt.Errorf("failed to get order by tracking code: %w", err)
		}
	}

	// Fall back to internal order ID if provided
	if order == nil && payload.InternalOrderID != "" {
		internalID, parseErr := uuid.Parse(payload.InternalOrderID)
		if parseErr == nil {
			order, err = s.repository.GetByInternalOrderID(ctx, internalID)
			if err != nil {
				s.saveWebhookLog(ctx, webhookLog, nil, http.StatusInternalServerError, err)
				return nil, fmt.Errorf("failed to get order by internal ID: %w", err)
			}
		}
	}

	if order == nil {
		err := fmt.Errorf("shipping order not found for tracking code: %s", payload.TrackingCode)
		s.saveWebhookLog(ctx, webhookLog, nil, http.StatusNotFound, err)
		return nil, err
	}

	// Get old status
	oldStatus := order.SystemStatus

	// Map carrier status to system status
	newStatus := domain.MapCarrierStatus(providerName, payload.CarrierStatus)

	// Update order status
	order.UpdateStatus(payload.CarrierStatus)

	// Store raw payload as metadata
	order.SetMetadata(payload.RawPayload)

	// Update in database
	if err := s.repository.Update(ctx, order); err != nil {
		s.saveWebhookLog(ctx, webhookLog, &order.ID, http.StatusInternalServerError, err)
		return nil, fmt.Errorf("failed to update order: %w", err)
	}

	// Log successful webhook
	s.saveWebhookLog(ctx, webhookLog, &order.ID, http.StatusOK, nil)

	// Publish status update event
	if s.publisher != nil {
		event := &ports.StatusUpdatedEvent{
			InternalOrderID: order.InternalOrderID,
			TrackingCode:    order.TrackingCode,
			Provider:        order.Provider,
			CarrierStatus:   payload.CarrierStatus,
			SystemStatus:    newStatus,
		}
		_ = s.publisher.PublishStatusUpdated(ctx, event)
	}

	return &HandleWebhookResult{
		TrackingCode:  order.TrackingCode,
		OldStatus:     oldStatus,
		NewStatus:     newStatus,
		CarrierStatus: payload.CarrierStatus,
	}, nil
}

// saveWebhookLog saves webhook log if repository is configured
func (s *WebhookService) saveWebhookLog(ctx context.Context, log *domain.WebhookLog, orderID *uuid.UUID, httpStatus int, err error) {
	if s.webhookLogRepo == nil {
		return
	}

	if err != nil {
		log.SetError(httpStatus, err.Error())
	} else if orderID != nil {
		log.SetSuccess(*orderID, httpStatus)
	}

	_ = s.webhookLogRepo.Create(ctx, log)
}

// logWebhookError logs a webhook error when payload parsing fails
func (s *WebhookService) logWebhookError(ctx context.Context, provider domain.ProviderName, trackingCode, carrierStatus string, rawPayload []byte, httpStatus int, err error) {
	if s.webhookLogRepo == nil {
		return
	}

	log := domain.NewWebhookLog(provider, trackingCode, carrierStatus, json.RawMessage(rawPayload))
	log.SetError(httpStatus, err.Error())
	_ = s.webhookLogRepo.Create(ctx, log)
}
