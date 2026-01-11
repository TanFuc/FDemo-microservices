package repository

import (
	"context"

	"inventory-service/internal/domain"

	"gorm.io/gorm"
)

type InventoryRepository interface {
	GetBySkuID(ctx context.Context, skuID string) (*domain.InventoryItem, error)
	UpdateReservedStock(ctx context.Context, tx *gorm.DB, skuID string, delta int) error
	DeductStock(ctx context.Context, tx *gorm.DB, skuID string, quantity int) error
	Create(ctx context.Context, item *domain.InventoryItem) error
	SyncFromDB(ctx context.Context, skuID string) error
}
