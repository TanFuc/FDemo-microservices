package domain

import (
	"testing"

	"github.com/google/uuid"
)

func TestNewUserEvent(t *testing.T) {
	input := EventInput{
		UserID:    "user-888",
		EventType: "PRODUCT_VIEW",
		Metadata:  `{"product_id":"p-123"}`,
		URL:       "/products/p-123",
	}

	event := NewUserEvent(input, "192.168.1.1", "Mozilla/5.0")

	if event.UserID != "user-888" {
		t.Errorf("expected UserID 'user-888', got '%s'", event.UserID)
	}
	if event.EventType != "PRODUCT_VIEW" {
		t.Errorf("expected EventType 'PRODUCT_VIEW', got '%s'", event.EventType)
	}
	if event.IPAddress != "192.168.1.1" {
		t.Errorf("expected IPAddress '192.168.1.1', got '%s'", event.IPAddress)
	}
	if event.UserAgent != "Mozilla/5.0" {
		t.Errorf("expected UserAgent 'Mozilla/5.0', got '%s'", event.UserAgent)
	}
	if event.EventID == uuid.Nil {
		t.Error("expected non-nil EventID UUID")
	}
	if event.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt timestamp")
	}
}
