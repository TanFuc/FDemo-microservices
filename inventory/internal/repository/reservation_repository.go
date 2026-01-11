package repository

import (
	"context"

	"microservices/inventory/internal/domain"

	"gorm.io/gorm"
)

type ReservationRepository interface {
	Create(ctx context.Context, tx *gorm.DB, reservation *domain.StockReservation) error
	GetByOrderID(ctx context.Context, orderID string) ([]domain.StockReservation, error)
	GetPendingByOrderID(ctx context.Context, orderID string) ([]domain.StockReservation, error)
	UpdateStatus(ctx context.Context, tx *gorm.DB, orderID string, fromStatus, toStatus domain.ReservationStatus) error
}
