package messaging

import (
	"encoding/json"
	"testing"
)

func TestNewCloudEvent(t *testing.T) {
	payload := OrderCreatedPayload{
		OrderID:     "ord-123",
		CustomerID:  "cust-456",
		TotalAmount: 99.99,
		Currency:    "USD",
	}

	evt := NewCloudEvent("nexus.order-service", EventOrderCreated, "corr-789", payload)

	if evt.ID == "" {
		t.Fatal("expected non-empty event ID")
	}
	if evt.Type != EventOrderCreated {
		t.Fatalf("expected type %s, got %s", EventOrderCreated, evt.Type)
	}
	if evt.CorrelationID != "corr-789" {
		t.Fatalf("expected correlation ID corr-789, got %s", evt.CorrelationID)
	}

	data, err := json.Marshal(evt)
	if err != nil {
		t.Fatalf("failed marshaling CloudEvent: %v", err)
	}

	var parsed CloudEvent[OrderCreatedPayload]
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed unmarshaling CloudEvent: %v", err)
	}

	if parsed.Data.OrderID != "ord-123" {
		t.Fatalf("expected order ID ord-123, got %s", parsed.Data.OrderID)
	}
}
