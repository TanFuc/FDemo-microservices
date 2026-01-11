package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// WebhookLog represents a webhook request log entry
type WebhookLog struct {
	ID              uuid.UUID       `json:"id" db:"id"`
	ShippingOrderID *uuid.UUID      `json:"shipping_order_id" db:"shipping_order_id"`
	Provider        ProviderName    `json:"provider" db:"provider"`
	TrackingCode    string          `json:"tracking_code" db:"tracking_code"`
	CarrierStatus   string          `json:"carrier_status" db:"carrier_status"`
	SystemStatus    SystemStatus    `json:"system_status" db:"system_status"`
	RawPayload      json.RawMessage `json:"raw_payload" db:"raw_payload"`
	HTTPStatus      int             `json:"http_status" db:"http_status"`
	ErrorMessage    string          `json:"error_message" db:"error_message"`
	ProcessedAt     time.Time       `json:"processed_at" db:"processed_at"`
	CreatedAt       time.Time       `json:"created_at" db:"created_at"`
}

// NewWebhookLog creates a new webhook log entry
func NewWebhookLog(provider ProviderName, trackingCode, carrierStatus string, rawPayload json.RawMessage) *WebhookLog {
	now := time.Now()
	return &WebhookLog{
		ID:            uuid.New(),
		Provider:      provider,
		TrackingCode:  trackingCode,
		CarrierStatus: carrierStatus,
		SystemStatus:  MapCarrierStatus(provider, carrierStatus),
		RawPayload:    rawPayload,
		ProcessedAt:   now,
		CreatedAt:     now,
	}
}

// SetSuccess marks the webhook as successfully processed
func (w *WebhookLog) SetSuccess(shippingOrderID uuid.UUID, httpStatus int) {
	w.ShippingOrderID = &shippingOrderID
	w.HTTPStatus = httpStatus
}

// SetError marks the webhook as failed
func (w *WebhookLog) SetError(httpStatus int, errorMessage string) {
	w.HTTPStatus = httpStatus
	w.ErrorMessage = errorMessage
}
