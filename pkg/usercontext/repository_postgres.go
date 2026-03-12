package usercontext

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PostgresRepository implements UserContextRepository for PostgreSQL using GORM
type PostgresRepository struct {
	db *gorm.DB
}

// NewPostgresRepository creates a new PostgreSQL-based user context repository
func NewPostgresRepository(db *gorm.DB) (*PostgresRepository, error) {
	// Auto-migrate the UserContext table
	if err := db.AutoMigrate(&UserContext{}); err != nil {
		return nil, err
	}

	return &PostgresRepository{
		db: db,
	}, nil
}

// FindByUserID retrieves user context from PostgreSQL
func (r *PostgresRepository) FindByUserID(ctx context.Context, userID string) (*UserContext, error) {
	var user UserContext
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

// Upsert inserts or updates user context in PostgreSQL
func (r *PostgresRepository) Upsert(ctx context.Context, user *UserContext) error {
	// Use ON CONFLICT DO UPDATE for PostgreSQL upsert
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"role", "shop_id", "display_name", "avatar_url", "email", "status", "version", "updated_at"}),
	}).Create(user).Error
}

// Delete removes user context from PostgreSQL (soft delete by setting status)
func (r *PostgresRepository) Delete(ctx context.Context, userID string) error {
	return r.db.WithContext(ctx).Model(&UserContext{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"status":     StatusInactive,
			"updated_at": time.Now(),
		}).Error
}

// FindByShopID retrieves user context by shop ID
func (r *PostgresRepository) FindByShopID(ctx context.Context, shopID string) (*UserContext, error) {
	var user UserContext
	err := r.db.WithContext(ctx).Where("shop_id = ?", shopID).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

// FindByEmail retrieves user context by email
func (r *PostgresRepository) FindByEmail(ctx context.Context, email string) (*UserContext, error) {
	var user UserContext
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &user, nil
}

// WithTx returns a new repository instance using the given transaction
func (r *PostgresRepository) WithTx(tx *gorm.DB) *PostgresRepository {
	return &PostgresRepository{db: tx}
}
