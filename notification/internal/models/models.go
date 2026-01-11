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
