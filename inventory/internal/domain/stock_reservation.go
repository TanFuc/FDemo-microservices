package domain

import (
	"time"

	"github.com/google/uuid"
)

type ReservationStatus string

const (
	StatusPending   ReservationStatus = "PENDING"
	StatusConfirmed ReservationStatus = "CONFIRMED"
	StatusCancelled ReservationStatus = "CANCELLED"
)

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

func (StockReservation) TableName() string {
	return "stock_reservations"
}

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
