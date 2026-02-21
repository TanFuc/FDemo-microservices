package repository

import (
	"context"

	"microservices/inventory/internal/model"

	"gorm.io/gorm"
)

// InventoryRepository defines inventory data access operations
type InventoryRepository interface {
	// Create creates a new inventory item
	Create(ctx context.Context, item *model.InventoryItem) error

	// GetBySkuID retrieves inventory by SKU ID
	GetBySkuID(ctx context.Context, skuID string) (*model.InventoryItem, error)

	// Update updates an inventory item
	Update(ctx context.Context, item *model.InventoryItem) error

	// Delete deletes an inventory item by SKU ID
	Delete(ctx context.Context, skuID string) error

	// List lists inventory items with pagination
	List(ctx context.Context, filter *model.InventoryFilter) ([]model.InventoryItem, int64, error)

	// UpdateReservedStock updates the reserved stock amount
	UpdateReservedStock(ctx context.Context, tx *gorm.DB, skuID string, delta int) error

	// DeductStock deducts from both total and reserved stock (on confirmation)
	DeductStock(ctx context.Context, tx *gorm.DB, skuID string, quantity int) error

	// SyncFromDB synchronizes inventory from DB to cache
	SyncFromDB(ctx context.Context, skuID string) error

	// ExistsBySkuID checks if inventory exists
	ExistsBySkuID(ctx context.Context, skuID string) (bool, error)

	// GetDB returns the database instance for transactions
	GetDB() *gorm.DB
}

// ReservationRepository defines stock reservation data access operations
type ReservationRepository interface {
	// Create creates a new stock reservation
	Create(ctx context.Context, tx *gorm.DB, reservation *model.StockReservation) error

	// GetByID retrieves a reservation by ID
	GetByID(ctx context.Context, id string) (*model.StockReservation, error)

	// GetByOrderID retrieves all reservations for an order
	GetByOrderID(ctx context.Context, orderID string) ([]model.StockReservation, error)

	// GetPendingByOrderID retrieves only pending reservations for an order
	GetPendingByOrderID(ctx context.Context, orderID string) ([]model.StockReservation, error)

	// UpdateStatus updates the status of reservations for an order
	UpdateStatus(ctx context.Context, tx *gorm.DB, orderID string, fromStatus, toStatus model.ReservationStatus) error

	// List lists reservations with filtering and pagination
	List(ctx context.Context, filter *model.ReservationFilter) ([]model.StockReservation, int64, error)

	// DeleteExpired deletes expired pending reservations
	DeleteExpired(ctx context.Context) (int64, error)
}

// CacheRepository defines cache operations for inventory
type CacheRepository interface {
	// Reserve atomically reserves stock in cache
	Reserve(ctx context.Context, skuID string, quantity int) (bool, error)

	// Release releases reserved stock in cache
	Release(ctx context.Context, skuID string, quantity int) error

	// SetInventory sets inventory values in cache
	SetInventory(ctx context.Context, skuID string, total, reserved int) error

	// GetInventory gets inventory values from cache
	GetInventory(ctx context.Context, skuID string) (total int, reserved int, err error)

	// DeleteInventory removes inventory from cache
	DeleteInventory(ctx context.Context, skuID string) error
}
