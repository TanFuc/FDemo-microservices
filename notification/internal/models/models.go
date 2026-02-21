package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// NotificationType represents the type of notification
type NotificationType string

const (
	NotificationTypeEmail NotificationType = "EMAIL"
	NotificationTypePush  NotificationType = "PUSH"
)

// NotificationStatus represents the status of a notification
type NotificationStatus string

const (
	NotificationStatusPending NotificationStatus = "PENDING"
	NotificationStatusSent    NotificationStatus = "SENT"
	NotificationStatusFailed  NotificationStatus = "FAILED"
)

// NotificationLog represents a notification audit log entry
type NotificationLog struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    string             `bson:"userId" json:"userId"`
	Type      NotificationType   `bson:"type" json:"type"`
	Status    NotificationStatus `bson:"status" json:"status"`
	Payload   interface{}        `bson:"payload" json:"payload"`
	Error     string             `bson:"error,omitempty" json:"error,omitempty"`
	CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time          `bson:"updatedAt" json:"updatedAt"`
}

// NotificationJob represents a notification job to be processed
type NotificationJob struct {
	Type      NotificationType       `json:"type"`
	Recipient string                 `json:"recipient"`
	Template  string                 `json:"template"`
	Data      map[string]interface{} `json:"data"`
	UserID    string                 `json:"userId,omitempty"`
}

// OrderCreatedEvent represents an order created event from NATS
type OrderCreatedEvent struct {
	OrderID   string  `json:"orderId"`
	UserID    string  `json:"userId"`
	UserEmail string  `json:"userEmail"`
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency,omitempty"`
	Items     []OrderItem `json:"items,omitempty"`
}

// OrderItem represents an item in an order
type OrderItem struct {
	Name     string  `json:"name"`
	Quantity int     `json:"quantity"`
	Price    float64 `json:"price"`
}

// Notification represents a notification to be sent
type Notification struct {
	ID          string                 `bson:"_id,omitempty" json:"id,omitempty"`
	UserID      string                 `bson:"userId" json:"userId"`
	Type        string                 `bson:"type" json:"type"`
	Title       string                 `bson:"title" json:"title"`
	Body        string                 `bson:"body" json:"body"`
	Data        map[string]interface{} `bson:"data,omitempty" json:"data,omitempty"`
	Channel     string                 `bson:"channel" json:"channel"` // email, push, websocket, all
	Priority    string                 `bson:"priority" json:"priority"` // low, normal, high, urgent
	Read        bool                   `bson:"read" json:"read"`
	ReadAt      *time.Time             `bson:"readAt,omitempty" json:"readAt,omitempty"`
	CreatedAt   time.Time              `bson:"createdAt" json:"createdAt"`
	UpdatedAt   time.Time              `bson:"updatedAt" json:"updatedAt"`
}

// NotificationListResponse represents a paginated list of notifications
type NotificationListResponse struct {
	Items       []*Notification `json:"items"`
	Total       int64           `json:"total"`
	UnreadCount int64           `json:"unread_count"`
	Page        int             `json:"page"`
	Limit       int             `json:"limit"`
	TotalPages  int             `json:"total_pages"`
}

// SendNotificationRequest represents a request to send a notification
type SendNotificationRequest struct {
	UserID   string                 `json:"user_id" validate:"required"`
	Type     string                 `json:"type" validate:"required"`
	Title    string                 `json:"title" validate:"required"`
	Body     string                 `json:"body" validate:"required"`
	Data     map[string]interface{} `json:"data,omitempty"`
	Channel  string                 `json:"channel,omitempty"` // email, push, websocket, all (default: all)
	Priority string                 `json:"priority,omitempty"` // low, normal, high, urgent (default: normal)
}

// BroadcastNotificationRequest represents a request to broadcast a notification
type BroadcastNotificationRequest struct {
	Type     string                 `json:"type" validate:"required"`
	Title    string                 `json:"title" validate:"required"`
	Body     string                 `json:"body" validate:"required"`
	Data     map[string]interface{} `json:"data,omitempty"`
	Priority string                 `json:"priority,omitempty"`
}
