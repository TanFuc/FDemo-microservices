package hub

import (
	"testing"
)

func TestHub_RegisterAndUnregister(t *testing.T) {
	h := NewHub()

	c1 := NewClient("c-1", "user-100", nil, h)
	c2 := NewClient("c-2", "user-100", nil, h) // same user, different device
	c3 := NewClient("c-3", "user-200", nil, h)

	h.Register(c1)
	h.Register(c2)
	h.Register(c3)

	if h.TotalClients() != 3 {
		t.Errorf("expected 3 clients, got %d", h.TotalClients())
	}
	if h.TotalUsers() != 2 {
		t.Errorf("expected 2 unique users, got %d", h.TotalUsers())
	}

	// Test Room Subscription
	h.SubscribeRoom(c1, "order:ORD-999")
	h.SubscribeRoom(c3, "order:ORD-999")

	// Send to user
	delivered := h.SendToUser("user-100", []byte("hello user 100"))
	if delivered != 2 {
		t.Errorf("expected delivered to 2 clients for user-100, got %d", delivered)
	}

	// Send to room
	roomDelivered := h.SendToRoom("order:ORD-999", []byte("order updated"))
	if roomDelivered != 2 {
		t.Errorf("expected delivered to 2 clients in room, got %d", roomDelivered)
	}

	// Broadcast
	broadcastDelivered := h.Broadcast([]byte("broadcast event"))
	if broadcastDelivered != 3 {
		t.Errorf("expected broadcast to 3 clients, got %d", broadcastDelivered)
	}

	// Unregister c1
	h.Unregister(c1)
	if h.TotalClients() != 2 {
		t.Errorf("expected 2 clients after unregistering c1, got %d", h.TotalClients())
	}
	if h.TotalUsers() != 2 {
		t.Errorf("expected still 2 users since c2 belongs to user-100, got %d", h.TotalUsers())
	}

	// Unregister c2
	h.Unregister(c2)
	if h.TotalUsers() != 1 {
		t.Errorf("expected 1 user left after removing c2, got %d", h.TotalUsers())
	}
}
