package dto

import (
	"time"

	"github.com/google/uuid"

	"tafu-logistic/logistics-service/internal/core/domain"
)

// APIResponse is a generic API response wrapper
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
}

// APIError represents an API error
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// CalculateFeeResponse represents the fee calculation response
type CalculateFeeResponse struct {
	Provider  string  `json:"provider"`
	Fee       float64 `json:"fee"`
	FromCache bool    `json:"from_cache"`
}

// CreateShipmentResponse represents the shipment creation response
type CreateShipmentResponse struct {
	ID           uuid.UUID `json:"id"`
	TrackingCode string    `json:"tracking_code"`
	LabelURL     string    `json:"label_url"`
	ShippingFee  float64   `json:"shipping_fee"`
	Provider     string    `json:"provider"`
}

// ShippingOrderResponse represents a shipping order in response
type ShippingOrderResponse struct {
	ID              uuid.UUID `json:"id"`
	InternalOrderID uuid.UUID `json:"internal_order_id"`
	Provider        string    `json:"provider"`
	TrackingCode    string    `json:"tracking_code"`
	CarrierStatus   string    `json:"carrier_status"`
	SystemStatus    string    `json:"system_status"`
	ShippingFee     float64   `json:"shipping_fee"`
	CODAmount       float64   `json:"cod_amount"`
	LabelURL        string    `json:"label_url"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// WebhookResponse represents the webhook processing response
type WebhookResponse struct {
	TrackingCode  string `json:"tracking_code"`
	OldStatus     string `json:"old_status"`
	NewStatus     string `json:"new_status"`
	CarrierStatus string `json:"carrier_status"`
}

// ProvidersResponse represents available providers
type ProvidersResponse struct {
	Providers []string `json:"providers"`
}

// FromShippingOrder converts domain ShippingOrder to response DTO
func FromShippingOrder(order *domain.ShippingOrder) *ShippingOrderResponse {
	return &ShippingOrderResponse{
		ID:              order.ID,
		InternalOrderID: order.InternalOrderID,
		Provider:        string(order.Provider),
		TrackingCode:    order.TrackingCode,
		CarrierStatus:   order.CarrierStatus,
		SystemStatus:    string(order.SystemStatus),
		ShippingFee:     order.ShippingFee,
		CODAmount:       order.CODAmount,
		LabelURL:        order.LabelURL,
		CreatedAt:       order.CreatedAt,
		UpdatedAt:       order.UpdatedAt,
	}
}

// NewSuccessResponse creates a success response
func NewSuccessResponse(data interface{}) *APIResponse {
	return &APIResponse{
		Success: true,
		Data:    data,
	}
}

// NewErrorResponse creates an error response
func NewErrorResponse(code, message string) *APIResponse {
	return &APIResponse{
		Success: false,
		Error: &APIError{
			Code:    code,
			Message: message,
		},
	}
}
