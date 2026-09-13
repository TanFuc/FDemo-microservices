package realtime

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// EventType identifies the nature of the real-time event
type EventType string

const (
	// Order event types
	EventOrderCreated       EventType = "order.created"
	EventOrderStatusUpdated EventType = "order.status_updated"
	EventOrderCancelled     EventType = "order.cancelled"
	EventOrderDelivered     EventType = "order.delivered"

	// Payment event types
	EventPaymentPending   EventType = "payment.pending"
	EventPaymentSucceeded EventType = "payment.succeeded"
	EventPaymentFailed    EventType = "payment.failed"

	// Inventory event types
	EventStockLow         EventType = "inventory.stock_low"
	EventStockOutOfStock  EventType = "inventory.out_of_stock"

	// Campaign / Flash Sale event types
	EventFlashSaleStarted EventType = "campaign.flash_sale_started"
	EventVoucherClaimed   EventType = "campaign.voucher_claimed"

	// Notification event types
	EventNotificationNew  EventType = "notification.new"
	EventSystemBroadcast  EventType = "system.broadcast"
)

// EventEnvelope is the canonical wire format for all real-time events transmitted
// across the distributed backplane and down to client WebSocket frames.
type EventEnvelope struct {
	ID        string          `json:"id"`
	Type      EventType       `json:"type"`
	Topic     string          `json:"topic"`
	UserID    string          `json:"userId,omitempty"`
	RoomID    string          `json:"roomId,omitempty"`
	Payload   json.RawMessage `json:"payload"`
	Timestamp int64           `json:"timestamp"`
}

// NewUserEvent constructs an event addressed to all active sessions of a specific user.
func NewUserEvent(userID string, eventType EventType, payload interface{}) (*EventEnvelope, error) {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return &EventEnvelope{
		ID:        uuid.New().String(),
		Type:      eventType,
		Topic:     "user:" + userID,
		UserID:    userID,
		Payload:   bytes,
		Timestamp: time.Now().UTC().UnixMilli(),
	}, nil
}

// NewRoomEvent constructs an event addressed to all subscribers of a room (e.g. order tracking).
func NewRoomEvent(roomID string, eventType EventType, payload interface{}) (*EventEnvelope, error) {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return &EventEnvelope{
		ID:        uuid.New().String(),
		Type:      eventType,
		Topic:     "room:" + roomID,
		RoomID:    roomID,
		Payload:   bytes,
		Timestamp: time.Now().UTC().UnixMilli(),
	}, nil
}

// NewBroadcastEvent constructs an event addressed to all connected WebSocket clients.
func NewBroadcastEvent(eventType EventType, payload interface{}) (*EventEnvelope, error) {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return &EventEnvelope{
		ID:        uuid.New().String(),
		Type:      eventType,
		Topic:     "broadcast",
		Payload:   bytes,
		Timestamp: time.Now().UTC().UnixMilli(),
	}, nil
}
