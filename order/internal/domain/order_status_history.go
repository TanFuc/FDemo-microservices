package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// OrderStatusHistory tracks all status changes for an order
type OrderStatusHistory struct {
	ID      uuid.UUID `gorm:"type:uuid;primary_key"`
	OrderID uuid.UUID `gorm:"type:uuid;index;not null"`

	// Status change
	FromStatus OrderStatus `gorm:"type:varchar(20)"`
	ToStatus   OrderStatus `gorm:"type:varchar(20);not null"`

	// Change details
	Reason  string `gorm:"type:text"`
	Comment string `gorm:"type:text"`

	// Who made the change
	ChangedBy     string `gorm:"type:varchar(100)"` // user, seller, system, admin
	ChangedByID   string `gorm:"type:varchar(100)"`
	ChangedByName string `gorm:"type:varchar(255)"`

	// Additional context
	IPAddress string          `gorm:"type:varchar(45)"`
	UserAgent string          `gorm:"type:text"`
	Metadata  json.RawMessage `gorm:"type:jsonb"`

	// Timestamps
	CreatedAt time.Time `gorm:"type:timestamptz;not null"`
}

// TableName specifies the table name for OrderStatusHistory
func (OrderStatusHistory) TableName() string {
	return "order_status_history"
}

// NewOrderStatusHistory creates a new status history entry
func NewOrderStatusHistory(
	orderID uuid.UUID,
	fromStatus, toStatus OrderStatus,
	reason, comment string,
	changedBy, changedByID, changedByName string,
) *OrderStatusHistory {
	return &OrderStatusHistory{
		ID:            uuid.New(),
		OrderID:       orderID,
		FromStatus:    fromStatus,
		ToStatus:      toStatus,
		Reason:        reason,
		Comment:       comment,
		ChangedBy:     changedBy,
		ChangedByID:   changedByID,
		ChangedByName: changedByName,
		CreatedAt:     time.Now(),
	}
}
