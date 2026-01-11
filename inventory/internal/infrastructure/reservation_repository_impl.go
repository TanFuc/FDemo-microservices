package infrastructure

import (
	"context"

	"inventory-service/internal/domain"
	"inventory-service/internal/repository"

	"gorm.io/gorm"
)

type reservationRepositoryImpl struct {
	db *gorm.DB
}

func NewReservationRepository(db *gorm.DB) repository.ReservationRepository {
	return &reservationRepositoryImpl{db: db}
}

func (r *reservationRepositoryImpl) Create(ctx context.Context, tx *gorm.DB, reservation *domain.StockReservation) error {
	db := tx
	if db == nil {
		db = r.db
	}
	return db.WithContext(ctx).Create(reservation).Error
}

func (r *reservationRepositoryImpl) GetByOrderID(ctx context.Context, orderID string) ([]domain.StockReservation, error) {
	var reservations []domain.StockReservation
	if err := r.db.WithContext(ctx).Where("order_id = ?", orderID).Find(&reservations).Error; err != nil {
		return nil, err
	}
	return reservations, nil
}

func (r *reservationRepositoryImpl) GetPendingByOrderID(ctx context.Context, orderID string) ([]domain.StockReservation, error) {
	var reservations []domain.StockReservation
	if err := r.db.WithContext(ctx).
		Where("order_id = ? AND status = ?", orderID, domain.StatusPending).
		Find(&reservations).Error; err != nil {
		return nil, err
	}
	return reservations, nil
}

func (r *reservationRepositoryImpl) UpdateStatus(ctx context.Context, tx *gorm.DB, orderID string, fromStatus, toStatus domain.ReservationStatus) error {
	db := tx
	if db == nil {
		db = r.db
	}

	result := db.WithContext(ctx).
		Model(&domain.StockReservation{}).
		Where("order_id = ? AND status = ?", orderID, fromStatus).
		Update("status", toStatus)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrReservationNotFound
	}
	return nil
}
