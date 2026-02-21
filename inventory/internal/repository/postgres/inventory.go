package postgres

import (
	"context"
	"errors"

	"microservices/inventory/internal/model"
	"microservices/inventory/internal/repository"

	"gorm.io/gorm"
)

var _ repository.InventoryRepository = (*InventoryRepository)(nil)

type InventoryRepository struct {
	db    *gorm.DB
	cache repository.CacheRepository
}

func NewInventoryRepository(db *gorm.DB, cache repository.CacheRepository) *InventoryRepository {
	return &InventoryRepository{
		db:    db,
		cache: cache,
	}
}

func (r *InventoryRepository) Create(ctx context.Context, item *model.InventoryItem) error {
	if err := r.db.WithContext(ctx).Create(item).Error; err != nil {
		return err
	}

	// Sync to cache
	if r.cache != nil {
		if err := r.cache.SetInventory(ctx, item.SkuID, item.TotalStock, item.ReservedStock); err != nil {
			return err
		}
	}

	return nil
}

func (r *InventoryRepository) GetBySkuID(ctx context.Context, skuID string) (*model.InventoryItem, error) {
	var item model.InventoryItem
	if err := r.db.WithContext(ctx).Where("sku_id = ?", skuID).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (r *InventoryRepository) Update(ctx context.Context, item *model.InventoryItem) error {
	result := r.db.WithContext(ctx).
		Model(&model.InventoryItem{}).
		Where("sku_id = ?", item.SkuID).
		Updates(map[string]interface{}{
			"total_stock":    item.TotalStock,
			"reserved_stock": item.ReservedStock,
		})

	if result.Error != nil {
		return result.Error
	}

	// Sync to cache
	if r.cache != nil {
		if err := r.cache.SetInventory(ctx, item.SkuID, item.TotalStock, item.ReservedStock); err != nil {
			return err
		}
	}

	return nil
}

func (r *InventoryRepository) Delete(ctx context.Context, skuID string) error {
	result := r.db.WithContext(ctx).Where("sku_id = ?", skuID).Delete(&model.InventoryItem{})
	if result.Error != nil {
		return result.Error
	}

	// Remove from cache
	if r.cache != nil {
		if err := r.cache.DeleteInventory(ctx, skuID); err != nil {
			return err
		}
	}

	return nil
}

func (r *InventoryRepository) List(ctx context.Context, filter *model.InventoryFilter) ([]model.InventoryItem, int64, error) {
	filter.EnsureDefaults()

	var total int64
	if err := r.db.WithContext(ctx).Model(&model.InventoryItem{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []model.InventoryItem
	offset := (filter.Page - 1) * filter.Limit
	if err := r.db.WithContext(ctx).
		Order("sku_id ASC").
		Offset(offset).
		Limit(filter.Limit).
		Find(&items).Error; err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r *InventoryRepository) UpdateReservedStock(ctx context.Context, tx *gorm.DB, skuID string, delta int) error {
	db := tx
	if db == nil {
		db = r.db
	}

	result := db.WithContext(ctx).
		Model(&model.InventoryItem{}).
		Where("sku_id = ?", skuID).
		Update("reserved_stock", gorm.Expr("reserved_stock + ?", delta))

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return model.ErrInventoryNotFound
	}
	return nil
}

func (r *InventoryRepository) DeductStock(ctx context.Context, tx *gorm.DB, skuID string, quantity int) error {
	db := tx
	if db == nil {
		db = r.db
	}

	result := db.WithContext(ctx).
		Model(&model.InventoryItem{}).
		Where("sku_id = ?", skuID).
		Updates(map[string]interface{}{
			"total_stock":    gorm.Expr("total_stock - ?", quantity),
			"reserved_stock": gorm.Expr("reserved_stock - ?", quantity),
		})

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return model.ErrInventoryNotFound
	}
	return nil
}

func (r *InventoryRepository) SyncFromDB(ctx context.Context, skuID string) error {
	item, err := r.GetBySkuID(ctx, skuID)
	if err != nil {
		return err
	}
	if item == nil {
		return model.ErrInventoryNotFound
	}

	if r.cache != nil {
		return r.cache.SetInventory(ctx, skuID, item.TotalStock, item.ReservedStock)
	}
	return nil
}

func (r *InventoryRepository) ExistsBySkuID(ctx context.Context, skuID string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&model.InventoryItem{}).
		Where("sku_id = ?", skuID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *InventoryRepository) GetDB() *gorm.DB {
	return r.db
}
