package domain

import (
	"testing"
	"time"
)

func TestInventoryItem_AvailableStock(t *testing.T) {
	item := InventoryItem{
		SkuID:         "SKU-IPHONE-15",
		TotalStock:    100,
		ReservedStock: 25,
	}

	if item.AvailableStock() != 75 {
		t.Errorf("expected available stock 75, got %d", item.AvailableStock())
	}

	// Boundary test: fully reserved
	item.ReservedStock = 100
	if item.AvailableStock() != 0 {
		t.Errorf("expected available stock 0, got %d", item.AvailableStock())
	}
}

func TestNewStockReservation(t *testing.T) {
	expires := time.Now().Add(15 * time.Minute)
	res := NewStockReservation("ORD-999", "SKU-IPHONE-15", 2, expires)

	if res.OrderID != "ORD-999" {
		t.Errorf("expected OrderID 'ORD-999', got '%s'", res.OrderID)
	}
	if res.SkuID != "SKU-IPHONE-15" {
		t.Errorf("expected SkuID 'SKU-IPHONE-15', got '%s'", res.SkuID)
	}
	if res.Quantity != 2 {
		t.Errorf("expected quantity 2, got %d", res.Quantity)
	}
	if res.Status != StatusPending {
		t.Errorf("expected status 'PENDING', got '%s'", res.Status)
	}
	if res.ID.String() == "" {
		t.Error("expected non-empty reservation UUID")
	}
}
