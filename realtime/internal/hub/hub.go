package hub

import (
	"sync"
)

// Hub maintains the set of active clients, room subscriptions, and broadcasts messages.
type Hub struct {
	clients     map[string]*Client         // clientID -> *Client
	userClients map[string]map[string]bool // userID -> set of clientIDs
	roomClients map[string]map[string]bool // roomID -> set of clientIDs
	mu          sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		clients:     make(map[string]*Client),
		userClients: make(map[string]map[string]bool),
		roomClients: make(map[string]map[string]bool),
	}
}

// Register adds a new client to the hub.
func (h *Hub) Register(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.clients[client.ID] = client

	if client.UserID != "" {
		if _, ok := h.userClients[client.UserID]; !ok {
			h.userClients[client.UserID] = make(map[string]bool)
		}
		h.userClients[client.UserID][client.ID] = true
	}
}

// Unregister removes a client from the hub and cleans up room/user mappings.
func (h *Hub) Unregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[client.ID]; !ok {
		return
	}

	delete(h.clients, client.ID)

	// Clean up user mapping
	if client.UserID != "" {
		if uMap, ok := h.userClients[client.UserID]; ok {
			delete(uMap, client.ID)
			if len(uMap) == 0 {
				delete(h.userClients, client.UserID)
			}
		}
	}

	// Clean up room mappings
	client.mu.Lock()
	for room := range client.rooms {
		if rMap, ok := h.roomClients[room]; ok {
			delete(rMap, client.ID)
			if len(rMap) == 0 {
				delete(h.roomClients, room)
			}
		}
	}
	client.mu.Unlock()
}

// SubscribeRoom subscribes a client to a room.
func (h *Hub) SubscribeRoom(client *Client, roomID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.roomClients[roomID]; !ok {
		h.roomClients[roomID] = make(map[string]bool)
	}
	h.roomClients[roomID][client.ID] = true

	client.mu.Lock()
	client.rooms[roomID] = true
	client.mu.Unlock()
}

// UnsubscribeRoom unsubscribes a client from a room.
func (h *Hub) UnsubscribeRoom(client *Client, roomID string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if rMap, ok := h.roomClients[roomID]; ok {
		delete(rMap, client.ID)
		if len(rMap) == 0 {
			delete(h.roomClients, roomID)
		}
	}

	client.mu.Lock()
	delete(client.rooms, roomID)
	client.mu.Unlock()
}

// SendToUser sends a message to all active connections belonging to the specified user.
func (h *Hub) SendToUser(userID string, msg []byte) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	delivered := 0
	clientIDs, ok := h.userClients[userID]
	if !ok {
		return 0
	}

	for id := range clientIDs {
		if c, exists := h.clients[id]; exists {
			select {
			case c.Send <- msg:
				delivered++
			default:
				// Buffer full
			}
		}
	}
	return delivered
}

// SendToRoom sends a message to all clients subscribed to a room.
func (h *Hub) SendToRoom(roomID string, msg []byte) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	delivered := 0
	clientIDs, ok := h.roomClients[roomID]
	if !ok {
		return 0
	}

	for id := range clientIDs {
		if c, exists := h.clients[id]; exists {
			select {
			case c.Send <- msg:
				delivered++
			default:
			}
		}
	}
	return delivered
}

// Broadcast sends a message to all connected clients.
func (h *Hub) Broadcast(msg []byte) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	delivered := 0
	for _, c := range h.clients {
		select {
		case c.Send <- msg:
			delivered++
		default:
		}
	}
	return delivered
}

// TotalClients returns current active socket connections.
func (h *Hub) TotalClients() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// TotalUsers returns current unique active online users.
func (h *Hub) TotalUsers() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.userClients)
}
