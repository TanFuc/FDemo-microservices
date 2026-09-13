package domain

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestPaymentTransaction_Lifecycle(t *testing.T) {
	orderID := uuid.New()
	userID := uuid.New()
	amount := decimal.NewFromInt(500)

	tx := NewPaymentTransaction(orderID, userID, amount, CurrencyVND, ProviderStripe)

	if tx.OrderID != orderID {
		t.Errorf("expected OrderID %v, got %v", orderID, tx.OrderID)
	}
	if tx.UserID != userID {
		t.Errorf("expected UserID %v, got %v", userID, tx.UserID)
	}
	if !tx.Amount.Equal(amount) {
		t.Errorf("expected amount 500, got %v", tx.Amount)
	}
	if tx.Status != StatusPending {
		t.Errorf("expected Status 'PENDING', got '%s'", tx.Status)
	}

	// Update provider transaction ID
	tx.SetProviderTxID("ch_stripe_123456")
	if tx.ProviderTxID != "ch_stripe_123456" {
		t.Errorf("expected provider tx id 'ch_stripe_123456', got '%s'", tx.ProviderTxID)
	}

	// Update status
	tx.UpdateStatus(StatusSuccess)
	if tx.Status != StatusSuccess {
		t.Errorf("expected Status 'SUCCESS', got '%s'", tx.Status)
	}

	// Set metadata
	err := tx.SetMetadata(map[string]interface{}{"card_last4": "4242"})
	if err != nil {
		t.Fatalf("unexpected error setting metadata: %v", err)
	}
	if string(tx.Metadata) == "" || string(tx.Metadata) == "{}" {
		t.Error("expected non-empty updated metadata")
	}
}
