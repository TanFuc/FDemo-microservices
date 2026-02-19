package model

import "time"

// CreateInventoryRequest represents a request to create inventory
type CreateInventoryRequest struct {
	SkuID      string `json:"sku_id" validate:"required"`
	TotalStock int    `json:"total_stock" validate:"required,gte=0"`
}

// UpdateInventoryRequest represents a request to update inventory
type UpdateInventoryRequest struct {
	TotalStock int `json:"total_stock" validate:"required,gte=0"`
}

// ReserveStockRequest represents a request to reserve stock
type ReserveStockRequest struct {
	OrderID string            `json:"order_id" validate:"required"`
	Items   []ReservationItem `json:"items" validate:"required,min=1,dive"`
}

// OrderRequest represents a request with just order ID
type OrderRequest struct {
	OrderID string `json:"order_id" validate:"required"`
}

// SyncInventoryRequest represents a request to sync inventory cache
type SyncInventoryRequest struct {
	SkuID string `json:"sku_id" validate:"required"`
}

// InventoryResponse represents inventory item response
type InventoryResponse struct {
	SkuID          string    `json:"sku_id"`
	TotalStock     int       `json:"total_stock"`
	ReservedStock  int       `json:"reserved_stock"`
	AvailableStock int       `json:"available_stock"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ToResponse converts InventoryItem to InventoryResponse
func (i *InventoryItem) ToResponse() *InventoryResponse {
	return &InventoryResponse{
		SkuID:          i.SkuID,
		TotalStock:     i.TotalStock,
		ReservedStock:  i.ReservedStock,
		AvailableStock: i.AvailableStock(),
		UpdatedAt:      i.UpdatedAt,
	}
}

// ReservationResponse represents stock reservation response
type ReservationResponse struct {
	ID        string    `json:"id"`
	OrderID   string    `json:"order_id"`
	SkuID     string    `json:"sku_id"`
	Quantity  int       `json:"quantity"`
	Status    string    `json:"status"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ToResponse converts StockReservation to ReservationResponse
func (r *StockReservation) ToResponse() *ReservationResponse {
	return &ReservationResponse{
		ID:        r.ID.String(),
		OrderID:   r.OrderID,
		SkuID:     r.SkuID,
		Quantity:  r.Quantity,
		Status:    string(r.Status),
		ExpiresAt: r.ExpiresAt,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

// ReserveStockResponse represents the response after reserving stock
type ReserveStockResponse struct {
	Success      bool                   `json:"success"`
	Message      string                 `json:"message"`
	Reservations []*ReservationResponse `json:"reservations,omitempty"`
}

// ListInventoryResponse represents paginated inventory list response
type ListInventoryResponse struct {
	Items []*InventoryResponse `json:"items"`
	Total int64                `json:"total"`
	Page  int                  `json:"page"`
	Limit int                  `json:"limit"`
}

// ListReservationsResponse represents paginated reservations list response
type ListReservationsResponse struct {
	Reservations []*ReservationResponse `json:"reservations"`
	Total        int64                  `json:"total"`
	Page         int                    `json:"page"`
	Limit        int                    `json:"limit"`
}
