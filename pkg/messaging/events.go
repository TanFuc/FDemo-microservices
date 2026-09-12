package messaging

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Standard event subjects for NexusCommerce.
const (
	EventOrderCreated        = "order.created"
	EventOrderCancelled      = "order.cancelled"
	EventOrderConfirmed      = "order.confirmed"
	EventOrderCompleted      = "order.completed"

	EventStockReserved       = "inventory.stock.reserved"
	EventStockReserveFailed  = "inventory.stock.failed"
	EventStockReleased       = "inventory.stock.released"

	EventPaymentProcessed    = "payment.processed"
	EventPaymentFailed       = "payment.failed"
	EventPaymentRefunded     = "payment.refunded"

	EventShipmentCreated     = "logistic.shipment.created"
	EventShipmentDispatched  = "logistic.shipment.dispatched"
	EventShipmentDelivered   = "logistic.shipment.delivered"
)

// CloudEvent represents an industry standard CloudEvents v1.0 specification envelope.
type CloudEvent[T any] struct {
	ID              string    `json:"id"`
	Source          string    `json:"source"`
	SpecVersion     string    `json:"specversion"`
	Type            string    `json:"type"`
	Time            time.Time `json:"time"`
	CorrelationID   string    `json:"correlationid,omitempty"`
	DataContentType string    `json:"datacontenttype"`
	Data            T         `json:"data"`
}

// NewCloudEvent constructs a new CloudEvent with unique ID and timestamp.
func NewCloudEvent[T any](source string, eventType string, correlationID string, data T) CloudEvent[T] {
	return CloudEvent[T]{
		ID:              uuid.NewString(),
		Source:          source,
		SpecVersion:     "1.0",
		Type:            eventType,
		Time:            time.Now().UTC(),
		CorrelationID:   correlationID,
		DataContentType: "application/json",
		Data:            data,
	}
}

// OutboxRecord represents an event record stored in PostgreSQL for the Transactional Outbox Pattern.
type OutboxRecord struct {
	ID            string          `json:"id"`
	AggregateType string          `json:"aggregate_type"`
	AggregateID   string          `json:"aggregate_id"`
	EventType     string          `json:"event_type"`
	Payload       json.RawMessage `json:"payload"`
	Status        string          `json:"status"` // PENDING, PUBLISHED, FAILED
	RetryCount    int             `json:"retry_count"`
	CreatedAt     time.Time       `json:"created_at"`
	ProcessedAt   *time.Time      `json:"processed_at,omitempty"`
}

// Common Event Payloads

// OrderCreatedPayload represents the payload when a customer submits an order.
type OrderCreatedPayload struct {
	OrderID     string          `json:"order_id"`
	CustomerID  string          `json:"customer_id"`
	TotalAmount float64         `json:"total_amount"`
	Currency    string          `json:"currency"`
	Items       []OrderItemData `json:"items"`
}

// OrderItemData represents individual SKU and quantity in an order.
type OrderItemData struct {
	ProductID string  `json:"product_id"`
	SKU       string  `json:"sku"`
	Quantity  int     `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
}

// StockReservedPayload represents the inventory reservation result.
type StockReservedPayload struct {
	OrderID       string `json:"order_id"`
	ReservationID string `json:"reservation_id"`
	Status        string `json:"status"` // SUCCESS, OUT_OF_STOCK
}

// PaymentCompletedPayload represents payment capture outcome.
type PaymentCompletedPayload struct {
	OrderID       string  `json:"order_id"`
	PaymentID     string  `json:"payment_id"`
	TransactionID string  `json:"transaction_id"`
	Amount        float64 `json:"amount"`
	Currency      string  `json:"currency"`
	Status        string  `json:"status"` // SUCCESS, FAILED
}
