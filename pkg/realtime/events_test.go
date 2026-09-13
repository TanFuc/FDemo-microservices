package realtime

import (
	"context"
	"testing"
)

func TestNewUserEvent(t *testing.T) {
	payload := map[string]interface{}{
		"order_id": "ORD-123",
		"status":   "CONFIRMED",
	}

	event, err := NewUserEvent("user-456", EventOrderStatusUpdated, payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if event.UserID != "user-456" {
		t.Errorf("expected UserID 'user-456', got '%s'", event.UserID)
	}
	if event.Type != EventOrderStatusUpdated {
		t.Errorf("expected Type '%s', got '%s'", EventOrderStatusUpdated, event.Type)
	}
	if event.Topic != "user:user-456" {
		t.Errorf("expected Topic 'user:user-456', got '%s'", event.Topic)
	}
	if event.ID == "" {
		t.Error("expected non-empty event ID")
	}
	if event.Timestamp == 0 {
		t.Error("expected non-zero timestamp")
	}

	subject := NatsSubjectForEvent(event)
	if subject != "realtime.user.user-456" {
		t.Errorf("expected subject 'realtime.user.user-456', got '%s'", subject)
	}
}

func TestNewRoomEvent(t *testing.T) {
	event, err := NewRoomEvent("order-789", EventOrderStatusUpdated, "delivered")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if event.RoomID != "order-789" {
		t.Errorf("expected RoomID 'order-789', got '%s'", event.RoomID)
	}
	if event.Topic != "room:order-789" {
		t.Errorf("expected Topic 'room:order-789', got '%s'", event.Topic)
	}

	subject := NatsSubjectForEvent(event)
	if subject != "realtime.room.order-789" {
		t.Errorf("expected subject 'realtime.room.order-789', got '%s'", subject)
	}
}

func TestNewBroadcastEvent(t *testing.T) {
	event, err := NewBroadcastEvent(EventSystemBroadcast, "maintenance in 10m")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if event.Topic != "broadcast" {
		t.Errorf("expected Topic 'broadcast', got '%s'", event.Topic)
	}

	subject := NatsSubjectForEvent(event)
	if subject != "realtime.broadcast" {
		t.Errorf("expected subject 'realtime.broadcast', got '%s'", subject)
	}
}

func TestMemoryPublisher_Lifecycle(t *testing.T) {
	ctx := context.Background()
	pub := NewMemoryPublisher()

	if err := pub.Publish(ctx, nil); err != ErrNilEvent {
		t.Errorf("expected ErrNilEvent, got %v", err)
	}

	ev, _ := NewBroadcastEvent(EventFlashSaleStarted, "sale is live")
	if err := pub.Publish(ctx, ev); err != nil {
		t.Fatalf("unexpected publish error: %v", err)
	}

	events := pub.GetEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Type != EventFlashSaleStarted {
		t.Errorf("expected event type '%s', got '%s'", EventFlashSaleStarted, events[0].Type)
	}

	// Test Serialization
	data, err := SerializeEvent(ev)
	if err != nil {
		t.Fatalf("failed to serialize event: %v", err)
	}

	deserialized, err := DeserializeEvent(data)
	if err != nil {
		t.Fatalf("failed to deserialize event: %v", err)
	}
	if deserialized.ID != ev.ID {
		t.Errorf("expected ID '%s', got '%s'", ev.ID, deserialized.ID)
	}

	pub.Clear()
	if len(pub.GetEvents()) != 0 {
		t.Error("expected 0 events after clear")
	}

	if err := pub.Close(); err != nil {
		t.Fatalf("failed to close: %v", err)
	}
	if err := pub.Publish(ctx, ev); err != ErrClosed {
		t.Errorf("expected ErrClosed, got %v", err)
	}
}
