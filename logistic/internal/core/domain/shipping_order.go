package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ShippingOrder represents a shipping order record
type ShippingOrder struct {
	ID              uuid.UUID       `json:"id" db:"id"`
	InternalOrderID uuid.UUID       `json:"internal_order_id" db:"internal_order_id"`
	Provider        ProviderName    `json:"provider" db:"provider"`
	TrackingCode    string          `json:"tracking_code" db:"tracking_code"`
	CarrierStatus   string          `json:"carrier_status" db:"carrier_status"`
	SystemStatus    SystemStatus    `json:"system_status" db:"system_status"`
	ShippingFee     float64         `json:"shipping_fee" db:"shipping_fee"`
	CODAmount       float64         `json:"cod_amount" db:"cod_amount"`
	LabelURL        string          `json:"label_url" db:"label_url"`
	Metadata        json.RawMessage `json:"metadata" db:"metadata"`
	CreatedAt       time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at" db:"updated_at"`
}

// NewShippingOrder creates a new shipping order
func NewShippingOrder(internalOrderID uuid.UUID, provider ProviderName) *ShippingOrder {
	now := time.Now()
	return &ShippingOrder{
		ID:              uuid.New(),
		InternalOrderID: internalOrderID,
		Provider:        provider,
		SystemStatus:    StatusPending,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

// UpdateStatus updates the shipping order status
func (so *ShippingOrder) UpdateStatus(carrierStatus string) {
	so.CarrierStatus = carrierStatus
	so.SystemStatus = MapCarrierStatus(so.Provider, carrierStatus)
	so.UpdatedAt = time.Now()
}

// SetTrackingInfo sets tracking code and label URL
func (so *ShippingOrder) SetTrackingInfo(trackingCode, labelURL string) {
	so.TrackingCode = trackingCode
	so.LabelURL = labelURL
	so.UpdatedAt = time.Now()
}

// SetFees sets shipping fee and COD amount
func (so *ShippingOrder) SetFees(shippingFee, codAmount float64) {
	so.ShippingFee = shippingFee
	so.CODAmount = codAmount
	so.UpdatedAt = time.Now()
}

// SetMetadata sets raw metadata from provider response
func (so *ShippingOrder) SetMetadata(metadata json.RawMessage) {
	so.Metadata = metadata
	so.UpdatedAt = time.Now()
}
