package model

import (
	"time"

	"github.com/google/uuid"
)

// ReservationStatus represents the status of a stock reservation
type ReservationStatus string

const (
	StatusPending   ReservationStatus = "PENDING"
	StatusConfirmed ReservationStatus = "CONFIRMED"
	StatusCancelled ReservationStatus = "CANCELLED"
)

// StockReservation represents a stock reservation record
type StockReservation struct {
	ID        uuid.UUID         `gorm:"type:uuid;primaryKey;column:id" json:"id"`
	OrderID   string            `gorm:"column:order_id;index;not null" json:"order_id"`
	SkuID     string            `gorm:"column:sku_id;not null" json:"sku_id"`
	Quantity  int               `gorm:"column:quantity;not null" json:"quantity"`
	Status    ReservationStatus `gorm:"column:status;type:varchar(20);not null;default:'PENDING'" json:"status"`
	ExpiresAt time.Time         `gorm:"column:expires_at" json:"expires_at"`
	CreatedAt time.Time         `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time         `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName returns the database table name
func (StockReservation) TableName() string {
	return "stock_reservations"
}

// NewStockReservation creates a new stock reservation
func NewStockReservation(orderID, skuID string, quantity int, expiresAt time.Time) *StockReservation {
	return &StockReservation{
		ID:        uuid.New(),
		OrderID:   orderID,
		SkuID:     skuID,
		Quantity:  quantity,
		Status:    StatusPending,
		ExpiresAt: expiresAt,
	}
}

// ReservationItem represents a single item in a reservation request
type ReservationItem struct {
	SkuID    string `json:"sku_id" validate:"required"`
	Quantity int    `json:"quantity" validate:"required,gt=0"`
}

// ReservationFilter for listing reservations
type ReservationFilter struct {
	Status  string `json:"status"`
	OrderID string `json:"order_id"`
	Page    int    `json:"page"`
	Limit   int    `json:"limit"`
}

// EnsureDefaults sets default values for filter
func (f *ReservationFilter) EnsureDefaults() {
	if f.Page <= 0 {
		f.Page = 1
	}
	if f.Limit <= 0 {
		f.Limit = 20
	}
	if f.Limit > 100 {
		f.Limit = 100
	}
}
