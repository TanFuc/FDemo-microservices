package service

import (
	"context"

	"github.com/google/uuid"

	"microservices/logistic/internal/core/domain"
)

// ShippingService defines the interface for shipping business logic
type ShippingService interface {
	// CalculateFee calculates shipping fee for a provider
	CalculateFee(ctx context.Context, req *CalculateFeeRequest) (*CalculateFeeResponse, error)

	// CreateShipment creates a new shipment with a provider
	CreateShipment(ctx context.Context, req *CreateShipmentRequest) (*CreateShipmentResponse, error)

	// GetShipment retrieves a shipment by ID
	GetShipment(ctx context.Context, id uuid.UUID) (*domain.ShippingOrder, error)

	// GetShipmentByTrackingCode retrieves a shipment by tracking code
	GetShipmentByTrackingCode(ctx context.Context, trackingCode string) (*domain.ShippingOrder, error)

	// ListProviders returns all available providers
	ListProviders() []domain.ProviderName
}

// WebhookService defines the interface for webhook processing
type WebhookService interface {
	// ProcessWebhook processes a webhook from a shipping provider
	ProcessWebhook(ctx context.Context, provider domain.ProviderName, payload []byte) (*WebhookResult, error)
}

// HealthService defines the interface for health checks
type HealthService interface {
	// CheckHealth performs a health check on all dependencies
	CheckHealth(ctx context.Context) (*HealthStatus, error)

	// GetDBStats returns database pool statistics
	GetDBStats() interface{}
}

// CalculateFeeRequest contains parameters for fee calculation
type CalculateFeeRequest struct {
	Provider       domain.ProviderName `json:"provider"`
	FromDistrictID int                 `json:"from_district_id"`
	ToDistrictID   int                 `json:"to_district_id"`
	WeightGram     int                 `json:"weight_gram"`
	InsuranceValue int                 `json:"insurance_value"`
}

// CalculateFeeResponse contains the calculated fee
type CalculateFeeResponse struct {
	Provider  domain.ProviderName `json:"provider"`
	Fee       float64             `json:"fee"`
	FromCache bool                `json:"from_cache"`
}

// CreateShipmentRequest contains parameters for creating a shipment
type CreateShipmentRequest struct {
	InternalOrderID uuid.UUID           `json:"internal_order_id"`
	Provider        domain.ProviderName `json:"provider"`
	Sender          domain.ContactInfo  `json:"sender"`
	Receiver        domain.ContactInfo  `json:"receiver"`
	Parcels         []domain.Parcel     `json:"parcels"`
	IsCOD           bool                `json:"is_cod"`
	CODAmount       float64             `json:"cod_amount"`
	Note            string              `json:"note"`
}

// CreateShipmentResponse contains the created shipment details
type CreateShipmentResponse struct {
	ID           uuid.UUID           `json:"id"`
	TrackingCode string              `json:"tracking_code"`
	LabelURL     string              `json:"label_url"`
	ShippingFee  float64             `json:"shipping_fee"`
	Provider     domain.ProviderName `json:"provider"`
}

// WebhookResult contains the result of webhook processing
type WebhookResult struct {
	TrackingCode  string `json:"tracking_code"`
	OldStatus     string `json:"old_status"`
	NewStatus     string `json:"new_status"`
	CarrierStatus string `json:"carrier_status"`
}

// HealthStatus contains health check results
type HealthStatus struct {
	Status       string                  `json:"status"`
	Dependencies map[string]HealthDetail `json:"dependencies"`
}

// HealthDetail contains details for a single dependency health check
type HealthDetail struct {
	Status  string `json:"status"`
	Latency string `json:"latency,omitempty"`
	Error   string `json:"error,omitempty"`
}
