package service

import (
	"context"

	"microservices/inventory/internal/model"
)

// InventoryService defines inventory management operations
type InventoryService interface {
	// CreateInventory creates a new inventory item
	CreateInventory(ctx context.Context, req *model.CreateInventoryRequest) (*model.InventoryItem, error)

	// GetInventory retrieves an inventory item by SKU ID
	GetInventory(ctx context.Context, skuID string) (*model.InventoryItem, error)

	// UpdateInventory updates an existing inventory item
	UpdateInventory(ctx context.Context, skuID string, req *model.UpdateInventoryRequest) (*model.InventoryItem, error)

	// DeleteInventory deletes an inventory item
	DeleteInventory(ctx context.Context, skuID string) error

	// ListInventory lists inventory items with pagination
	ListInventory(ctx context.Context, filter *model.InventoryFilter) ([]model.InventoryItem, int64, error)

	// SyncInventory synchronizes inventory from DB to cache
	SyncInventory(ctx context.Context, skuID string) (*model.InventoryItem, error)
}

// ReservationService defines stock reservation operations
type ReservationService interface {
	// ReserveStock reserves stock for an order
	ReserveStock(ctx context.Context, req *model.ReserveStockRequest) ([]model.StockReservation, error)

	// ConfirmStock confirms a stock reservation (deducts from total stock)
	ConfirmStock(ctx context.Context, orderID string) error

	// ReleaseStock releases a stock reservation (cancels without deducting)
	ReleaseStock(ctx context.Context, orderID string) error

	// GetReservation retrieves reservations for an order
	GetReservation(ctx context.Context, orderID string) ([]model.StockReservation, error)

	// ListReservations lists reservations with filtering
	ListReservations(ctx context.Context, filter *model.ReservationFilter) ([]model.StockReservation, int64, error)
}

// AuthUser represents an authenticated user
type AuthUser struct {
	UserID      string
	Email       string
	Role        string
	Permissions []string
}

// AuthService defines authentication service operations
type AuthService interface {
	// VerifyToken verifies a JWT token and returns user info
	VerifyToken(ctx context.Context, token string) (*AuthUser, error)

	// CheckPermission checks if a user has a specific permission
	CheckPermission(ctx context.Context, userID string, permission string) (bool, error)
}
