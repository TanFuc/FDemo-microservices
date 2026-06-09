//go:build legacy

package infrastructure

import (
	"context"
	"errors"

	"microservices/inventory/internal/domain"
	"microservices/inventory/internal/repository"

	"gorm.io/gorm"
)

type inventoryRepositoryImpl struct {
	db    *gorm.DB
	cache repository.CacheRepository
}

func NewInventoryRepository(db *gorm.DB, cache repository.CacheRepository) repository.InventoryRepository {
	return &inventoryRepositoryImpl{
		db:    db,
		cache: cache,
	}
}

func (r *inventoryRepositoryImpl) GetBySkuID(ctx context.Context, skuID string) (*domain.InventoryItem, error) {
	var item domain.InventoryItem
	if err := r.db.WithContext(ctx).Where("sku_id = ?", skuID).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrInventoryNotFound
		}
		return nil, err
	}
	return &item, nil
}

func (r *inventoryRepositoryImpl) UpdateReservedStock(ctx context.Context, tx *gorm.DB, skuID string, delta int) error {
	db := tx
	if db == nil {
		db = r.db
	}

	result := db.WithContext(ctx).
		Model(&domain.InventoryItem{}).
		Where("sku_id = ?", skuID).
		Update("reserved_stock", gorm.Expr("reserved_stock + ?", delta))

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrInventoryNotFound
	}
	return nil
}

func (r *inventoryRepositoryImpl) DeductStock(ctx context.Context, tx *gorm.DB, skuID string, quantity int) error {
	db := tx
	if db == nil {
		db = r.db
	}

	result := db.WithContext(ctx).
		Model(&domain.InventoryItem{}).
		Where("sku_id = ?", skuID).
		Updates(map[string]interface{}{
			"total_stock":    gorm.Expr("total_stock - ?", quantity),
			"reserved_stock": gorm.Expr("reserved_stock - ?", quantity),
		})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrInventoryNotFound
	}
	return nil
}

func (r *inventoryRepositoryImpl) Create(ctx context.Context, item *domain.InventoryItem) error {
	if err := r.db.WithContext(ctx).Create(item).Error; err != nil {
		return err
	}

	if err := r.cache.SetInventory(ctx, item.SkuID, item.TotalStock, item.ReservedStock); err != nil {
		return err
	}

	return nil
}

func (r *inventoryRepositoryImpl) SyncFromDB(ctx context.Context, skuID string) error {
	item, err := r.GetBySkuID(ctx, skuID)
	if err != nil {
		return err
	}

	return r.cache.SetInventory(ctx, skuID, item.TotalStock, item.ReservedStock)
}
