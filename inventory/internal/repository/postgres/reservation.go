package postgres

import (
	"context"
	"errors"
	"time"

	"microservices/inventory/internal/model"
	"microservices/inventory/internal/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var _ repository.ReservationRepository = (*ReservationRepository)(nil)

type ReservationRepository struct {
	db *gorm.DB
}

func NewReservationRepository(db *gorm.DB) *ReservationRepository {
	return &ReservationRepository{db: db}
}

func (r *ReservationRepository) Create(ctx context.Context, tx *gorm.DB, reservation *model.StockReservation) error {
	db := tx
	if db == nil {
		db = r.db
	}
	return db.WithContext(ctx).Create(reservation).Error
}

func (r *ReservationRepository) GetByID(ctx context.Context, id string) (*model.StockReservation, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	var reservation model.StockReservation
	if err := r.db.WithContext(ctx).Where("id = ?", uid).First(&reservation).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &reservation, nil
}

func (r *ReservationRepository) GetByOrderID(ctx context.Context, orderID string) ([]model.StockReservation, error) {
	var reservations []model.StockReservation
	if err := r.db.WithContext(ctx).
		Where("order_id = ?", orderID).
		Order("created_at ASC").
		Find(&reservations).Error; err != nil {
		return nil, err
	}
	return reservations, nil
}

func (r *ReservationRepository) GetPendingByOrderID(ctx context.Context, orderID string) ([]model.StockReservation, error) {
	var reservations []model.StockReservation
	if err := r.db.WithContext(ctx).
		Where("order_id = ? AND status = ?", orderID, model.StatusPending).
		Order("created_at ASC").
		Find(&reservations).Error; err != nil {
		return nil, err
	}
	return reservations, nil
}

func (r *ReservationRepository) UpdateStatus(ctx context.Context, tx *gorm.DB, orderID string, fromStatus, toStatus model.ReservationStatus) error {
	db := tx
	if db == nil {
		db = r.db
	}

	result := db.WithContext(ctx).
		Model(&model.StockReservation{}).
		Where("order_id = ? AND status = ?", orderID, fromStatus).
		Update("status", toStatus)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return model.ErrReservationNotFound
	}
	return nil
}

func (r *ReservationRepository) List(ctx context.Context, filter *model.ReservationFilter) ([]model.StockReservation, int64, error) {
	filter.EnsureDefaults()

	query := r.db.WithContext(ctx).Model(&model.StockReservation{})

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.OrderID != "" {
		query = query.Where("order_id = ?", filter.OrderID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var reservations []model.StockReservation
	offset := (filter.Page - 1) * filter.Limit
	if err := query.
		Order("created_at DESC").
		Offset(offset).
		Limit(filter.Limit).
		Find(&reservations).Error; err != nil {
		return nil, 0, err
	}

	return reservations, total, nil
}

func (r *ReservationRepository) DeleteExpired(ctx context.Context) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("status = ? AND expires_at < ?", model.StatusPending, time.Now()).
		Delete(&model.StockReservation{})

	return result.RowsAffected, result.Error
}
