package domain

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestNewOrder_InitialState(t *testing.T) {
	userID := uuid.New()
	addr := ShippingAddress{
		ContactName: "Alice Smith",
		Phone:       "0912345678",
	}

	order := NewOrder(userID, "COD", addr)

	if order.UserID != userID {
		t.Errorf("expected UserID %v, got %v", userID, order.UserID)
	}
	if order.Status != StatusPending {
		t.Errorf("expected status 'PENDING', got '%s'", order.Status)
	}
	if order.PaymentStatus != PaymentPending {
		t.Errorf("expected payment status 'PENDING', got '%s'", order.PaymentStatus)
	}
	if !order.CanCancel() {
		t.Error("expected new PENDING order to be cancelable")
	}
}

func TestOrder_AddItemAndCalculateTotals(t *testing.T) {
	order := NewOrder(uuid.New(), "STRIPE", ShippingAddress{})

	item1 := OrderItem{
		SkuID:     "SKU-1",
		Quantity:  2,
		UnitPrice: decimal.NewFromInt(100),
		SubTotal:  decimal.NewFromInt(200),
	}
	item2 := OrderItem{
		SkuID:     "SKU-2",
		Quantity:  1,
		UnitPrice: decimal.NewFromInt(50),
		SubTotal:  decimal.NewFromInt(50),
	}

	order.AddItem(item1)
	order.AddItem(item2)

	if order.TotalItems != 2 {
		t.Errorf("expected 2 items, got %d", order.TotalItems)
	}
	if order.TotalQuantity != 3 {
		t.Errorf("expected 3 total quantity, got %d", order.TotalQuantity)
	}

	order.ShippingFee = decimal.NewFromInt(20)
	order.DiscountAmount = decimal.NewFromInt(30)
	order.CalculateTotals()

	// SubTotal = 200 + 50 = 250
	if !order.SubTotal.Equal(decimal.NewFromInt(250)) {
		t.Errorf("expected subtotal 250, got %v", order.SubTotal)
	}
	// FinalAmount = 250 + 20 (shipping) - 30 (discount) = 240
	if !order.FinalAmount.Equal(decimal.NewFromInt(240)) {
		t.Errorf("expected final amount 240, got %v", order.FinalAmount)
	}
}

func TestOrder_CancellationWorkflow(t *testing.T) {
	order := NewOrder(uuid.New(), "COD", ShippingAddress{})

	// Cancel when pending
	err := order.Cancel("changed mind", "CUSTOMER", "user-1")
	if err != nil {
		t.Fatalf("expected successful cancellation, got: %v", err)
	}
	if order.Status != StatusCancelled {
		t.Errorf("expected status 'CANCELLED', got '%s'", order.Status)
	}

	// Attempt to cancel again should fail
	err2 := order.Cancel("duplicate", "CUSTOMER", "user-1")
	if err2 == nil {
		t.Error("expected error when cancelling an already cancelled order")
	}
}
