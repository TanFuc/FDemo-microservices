package ports

import (
	"context"
	"net/http"

	"microservices/logistic/internal/core/domain"
)

// RateRequest contains parameters for calculating shipping fee
type RateRequest struct {
	FromDistrictID int `json:"from_district_id"`
	ToDistrictID   int `json:"to_district_id"`
	WeightGram     int `json:"weight_gram"`
	InsuranceValue int `json:"insurance_value"` // Value of goods in VND
}

// ShipRequest contains parameters for creating a shipment
type ShipRequest struct {
	InternalOrderID string             `json:"internal_order_id"`
	Sender          domain.ContactInfo `json:"sender"`
	Receiver        domain.ContactInfo `json:"receiver"`
	Parcels         []domain.Parcel    `json:"parcels"`
	IsCOD           bool               `json:"is_cod"`
	CODAmount       float64            `json:"cod_amount"`
	Note            string             `json:"note"`
}

// ShipResponse contains the result of creating a shipment
type ShipResponse struct {
	TrackingCode string  `json:"tracking_code"`
	LabelURL     string  `json:"label_url"`
	ShippingFee  float64 `json:"shipping_fee"`
}

// WebhookPayload contains parsed webhook data
type WebhookPayload struct {
	TrackingCode    string `json:"tracking_code"`
	InternalOrderID string `json:"internal_order_id"`
	CarrierStatus   string `json:"carrier_status"`
	RawPayload      []byte `json:"raw_payload"`
}

// Provider defines the interface for shipping providers
type Provider interface {
	// GetName returns the provider identifier
	GetName() domain.ProviderName

	// CalculateFee calculates shipping fee for given parameters
	CalculateFee(ctx context.Context, req *RateRequest) (float64, error)

	// CreateOrder creates a shipping order and returns tracking info
	CreateOrder(ctx context.Context, req *ShipRequest) (*ShipResponse, error)

	// CancelOrder cancels a shipment at the carrier level
	CancelOrder(ctx context.Context, trackingCode string) error

	// ParseWebhook parses incoming webhook from provider
	ParseWebhook(r *http.Request) (*WebhookPayload, error)
}

// ProviderFactory creates provider instances by name
type ProviderFactory interface {
	GetProvider(name domain.ProviderName) (Provider, error)
	ListProviders() []domain.ProviderName
}
