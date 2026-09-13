package domain

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestVoucher_AvailabilityAndStock(t *testing.T) {
	campaignID := uuid.New()
	v := NewVoucher(
		"SUMMER50",
		campaignID,
		100,
		VoucherTypePercentage,
		decimal.NewFromInt(50),
		VoucherConditions{MinOrderValue: decimal.NewFromInt(200)},
	)

	if v.Code != "SUMMER50" {
		t.Errorf("expected code 'SUMMER50', got '%s'", v.Code)
	}
	if !v.IsAvailable() {
		t.Error("expected new voucher to be available")
	}
	if v.RemainingStock() != 100 {
		t.Errorf("expected remaining stock 100, got %d", v.RemainingStock())
	}

	// Increment used count
	v.UsedCount = 99
	if !v.IsAvailable() || v.RemainingStock() != 1 {
		t.Errorf("expected 1 remaining, got %d", v.RemainingStock())
	}

	// Fully used
	v.UsedCount = 100
	if v.IsAvailable() {
		t.Error("expected fully used voucher to be unavailable")
	}
	if v.RemainingStock() != 0 {
		t.Errorf("expected 0 remaining, got %d", v.RemainingStock())
	}

	// Inactive status
	v.UsedCount = 50
	v.Status = VoucherStatusInactive
	if v.IsAvailable() {
		t.Error("expected inactive voucher to be unavailable")
	}
}
