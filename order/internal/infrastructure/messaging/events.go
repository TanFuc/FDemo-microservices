package messaging

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Event types
const (
	EventOrderCreated   = "order.created"
	EventOrderCancelled = "order.cancelled"
	EventOrderPaid      = "order.paid"
	EventOrderShipped   = "order.shipped"
	EventOrderCompleted = "order.completed"
)

// OrderCreatedEvent is published when a new order is created
type OrderCreatedEvent struct {
	OrderID       uuid.UUID       `json:"order_id"`
	UserID        uuid.UUID       `json:"user_id"`
	FinalAmount   decimal.Decimal `json:"final_amount"`
	PaymentMethod string          `json:"payment_method"`
	ItemCount     int             `json:"item_count"`
	CreatedAt     time.Time       `json:"created_at"`
}

// OrderCancelledEvent is published when an order is cancelled
type OrderCancelledEvent struct {
	OrderID        uuid.UUID   `json:"order_id"`
	UserID         uuid.UUID   `json:"user_id"`
	ReservationIDs []string    `json:"reservation_ids"`
	CancelledAt    time.Time   `json:"cancelled_at"`
}

// OrderPaidEvent is published when an order is paid
type OrderPaidEvent struct {
	OrderID   uuid.UUID `json:"order_id"`
	UserID    uuid.UUID `json:"user_id"`
	PaidAt    time.Time `json:"paid_at"`
}

// OrderShippedEvent is published when an order is shipped
type OrderShippedEvent struct {
	OrderID   uuid.UUID `json:"order_id"`
	UserID    uuid.UUID `json:"user_id"`
	ShippedAt time.Time `json:"shipped_at"`
}

// OrderCompletedEvent is published when an order is completed
type OrderCompletedEvent struct {
	OrderID     uuid.UUID `json:"order_id"`
	UserID      uuid.UUID `json:"user_id"`
	CompletedAt time.Time `json:"completed_at"`
}
