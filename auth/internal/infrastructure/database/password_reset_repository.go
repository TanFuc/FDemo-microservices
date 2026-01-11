package database

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"tafu-auth/internal/domain/entity"
	"tafu-auth/internal/domain/repository"
)

type passwordResetRepository struct {
	db *gorm.DB
}

func NewPasswordResetRepository(db *gorm.DB) repository.PasswordResetRepository {
	return &passwordResetRepository{db: db}
}

func (r *passwordResetRepository) Create(ctx context.Context, reset *entity.PasswordReset) error {
	return r.db.WithContext(ctx).Create(reset).Error
}

func (r *passwordResetRepository) FindByID(ctx context.Context, id uuid.UUID) (*entity.PasswordReset, error) {
	var reset entity.PasswordReset
	err := r.db.WithContext(ctx).First(&reset, id).Error
	if err != nil {
		return nil, err
	}
	return &reset, nil
}

func (r *passwordResetRepository) FindByTokenHash(ctx context.Context, tokenHash string) (*entity.PasswordReset, error) {
	var reset entity.PasswordReset
	err := r.db.WithContext(ctx).
		Where("token_hash = ?", tokenHash).
		First(&reset).Error
	if err != nil {
		return nil, err
	}
	return &reset, nil
}

func (r *passwordResetRepository) FindActiveByUserID(ctx context.Context, userID uuid.UUID) (*entity.PasswordReset, error) {
	var reset entity.PasswordReset
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND used_at IS NULL AND expires_at > ?", userID, time.Now()).
		Order("created_at DESC").
		First(&reset).Error
	if err != nil {
		return nil, err
	}
	return &reset, nil
}

func (r *passwordResetRepository) MarkUsed(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&entity.PasswordReset{}).
		Where("id = ?", id).
		Update("used_at", time.Now()).Error
}

func (r *passwordResetRepository) DeleteExpired(ctx context.Context) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("expires_at < ?", time.Now()).
		Delete(&entity.PasswordReset{})
	return result.RowsAffected, result.Error
}
