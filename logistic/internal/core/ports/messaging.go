package ports

import (
	"context"

	"github.com/google/uuid"

	"tafu-logistic/logistics-service/internal/core/domain"
)

// ShipmentCreatedEvent is published when a shipment is created
type ShipmentCreatedEvent struct {
	InternalOrderID uuid.UUID            `json:"internal_order_id"`
	TrackingCode    string               `json:"tracking_code"`
	Provider        domain.ProviderName  `json:"provider"`
	ShippingFee     float64              `json:"shipping_fee"`
	LabelURL        string               `json:"label_url"`
}

// StatusUpdatedEvent is published when shipment status changes
type StatusUpdatedEvent struct {
	InternalOrderID uuid.UUID            `json:"internal_order_id"`
	TrackingCode    string               `json:"tracking_code"`
	Provider        domain.ProviderName  `json:"provider"`
	CarrierStatus   string               `json:"carrier_status"`
	SystemStatus    domain.SystemStatus  `json:"system_status"`
}

// EventPublisher defines the interface for publishing events
type EventPublisher interface {
	// PublishShipmentCreated publishes a shipment created event
	PublishShipmentCreated(ctx context.Context, event *ShipmentCreatedEvent) error

	// PublishStatusUpdated publishes a status updated event
	PublishStatusUpdated(ctx context.Context, event *StatusUpdatedEvent) error

	// Close closes the publisher connection
	Close() error
}

// Event subjects (NATS topics)
const (
	SubjectShipmentCreated = "logistics.shipment.created"
	SubjectStatusUpdated   = "logistics.status.updated"
)
