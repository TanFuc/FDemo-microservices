package database

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"microservices/auth/internal/domain/entity"
	"microservices/auth/internal/domain/repository"
)

type loginHistoryRepository struct {
	db *gorm.DB
}

func NewLoginHistoryRepository(db *gorm.DB) repository.LoginHistoryRepository {
	return &loginHistoryRepository{db: db}
}

func (r *loginHistoryRepository) Create(ctx context.Context, history *entity.LoginHistory) error {
	return r.db.WithContext(ctx).Create(history).Error
}

func (r *loginHistoryRepository) FindByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]entity.LoginHistory, error) {
	var histories []entity.LoginHistory
	query := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&histories).Error
	return histories, err
}

func (r *loginHistoryRepository) FindRecentByUserID(ctx context.Context, userID uuid.UUID, since interface{}) ([]entity.LoginHistory, error) {
	var histories []entity.LoginHistory
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND created_at > ?", userID, since).
		Order("created_at DESC").
		Find(&histories).Error
	return histories, err
}

func (r *loginHistoryRepository) CountFailedAttempts(ctx context.Context, userID uuid.UUID, since interface{}) (int64, error) {
	var count int64
	var sinceTime time.Time

	switch v := since.(type) {
	case time.Time:
		sinceTime = v
	case time.Duration:
		sinceTime = time.Now().Add(-v)
	default:
		sinceTime = time.Now().Add(-30 * time.Minute)
	}

	err := r.db.WithContext(ctx).
		Model(&entity.LoginHistory{}).
		Where("user_id = ? AND status = ? AND created_at > ?", userID, entity.LoginStatusFailed, sinceTime).
		Count(&count).Error
	return count, err
}
