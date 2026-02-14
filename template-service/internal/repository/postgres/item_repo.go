package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"microservices/template-service/internal/model"
	"microservices/template-service/internal/repository"
)

type itemRepository struct {
	db *gorm.DB
}

// NewItemRepository creates a new item repository
func NewItemRepository(db *gorm.DB) repository.ItemRepository {
	return &itemRepository{db: db}
}

func (r *itemRepository) Create(ctx context.Context, item *model.Item) error {
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *itemRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Item, error) {
	var item model.Item
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (r *itemRepository) Update(ctx context.Context, item *model.Item) error {
	return r.db.WithContext(ctx).Save(item).Error
}

func (r *itemRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.Item{}).Error
}

func (r *itemRepository) List(ctx context.Context, offset, limit int) ([]model.Item, int64, error) {
	var items []model.Item
	var total int64

	if err := r.db.WithContext(ctx).Model(&model.Item{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := r.db.WithContext(ctx).
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&items).Error; err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r *itemRepository) ExistsByID(ctx context.Context, id uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Item{}).Where("id = ?", id).Count(&count).Error
	return count > 0, err
}
