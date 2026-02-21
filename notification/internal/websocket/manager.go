package websocket

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/gofiber/contrib/websocket"
	"microservices/notification/internal/config"
	"microservices/pkg/logger"
)

// Message represents a WebSocket message
type Message struct {
	Type      string      `json:"type"`
	Event     string      `json:"event,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// Client represents a WebSocket client connection
type Client struct {
	ID       string
	UserID   string
	Conn     *websocket.Conn
	Send     chan []byte
	Manager  *Manager
	mu       sync.Mutex
	isClosed bool
}

// Manager handles WebSocket connections and message broadcasting
type Manager struct {
	clients      map[string]*Client          // All clients by client ID
	userClients  map[string]map[string]bool  // User ID -> set of client IDs
	register     chan *Client
	unregister   chan *Client
	broadcast    chan *BroadcastMessage
	userMessage  chan *UserMessage
	cfg          config.WebSocketConfig
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
}

// BroadcastMessage represents a message to be broadcast to all clients
type BroadcastMessage struct {
	Message []byte
	Exclude map[string]bool // Client IDs to exclude
}

// UserMessage represents a message to be sent to a specific user
type UserMessage struct {
	UserID  string
	Message []byte
}

// NewManager creates a new WebSocket manager
func NewManager(cfg config.WebSocketConfig) *Manager {
	ctx, cancel := context.WithCancel(context.Background())

	return &Manager{
		clients:     make(map[string]*Client),
		userClients: make(map[string]map[string]bool),
		register:    make(chan *Client, 256),
		unregister:  make(chan *Client, 256),
		broadcast:   make(chan *BroadcastMessage, 256),
		userMessage: make(chan *UserMessage, 256),
		cfg:         cfg,
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start begins the manager's main loop
func (m *Manager) Start() {
	logger.Info().Msg("WebSocket manager started")

	for {
		select {
		case <-m.ctx.Done():
			logger.Info().Msg("WebSocket manager shutting down")
			return

		case client := <-m.register:
			m.registerClient(client)

		case client := <-m.unregister:
			m.unregisterClient(client)

		case msg := <-m.broadcast:
			m.broadcastMessage(msg)

		case msg := <-m.userMessage:
			m.sendToUser(msg)
		}
	}
}

// Stop gracefully shuts down the manager
func (m *Manager) Stop() {
	m.cancel()

	m.mu.Lock()
	defer m.mu.Unlock()

	// Close all client connections
	for _, client := range m.clients {
		client.Close()
	}
}

// Register adds a new client to the manager
func (m *Manager) Register(client *Client) {
	m.register <- client
}

// Unregister removes a client from the manager
func (m *Manager) Unregister(client *Client) {
	m.unregister <- client
}

// Broadcast sends a message to all connected clients
func (m *Manager) Broadcast(message *Message, exclude map[string]bool) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	m.broadcast <- &BroadcastMessage{
		Message: data,
		Exclude: exclude,
	}
	return nil
}

// SendToUser sends a message to all connections of a specific user
func (m *Manager) SendToUser(userID string, message *Message) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	m.userMessage <- &UserMessage{
		UserID:  userID,
		Message: data,
	}
	return nil
}

// SendNotification sends a notification to a specific user
func (m *Manager) SendNotification(userID string, notificationType string, data interface{}) error {
	return m.SendToUser(userID, &Message{
		Type:      "notification",
		Event:     notificationType,
		Data:      data,
		Timestamp: time.Now(),
	})
}

// GetOnlineUsers returns a list of currently online user IDs
func (m *Manager) GetOnlineUsers() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	users := make([]string, 0, len(m.userClients))
	for userID := range m.userClients {
		users = append(users, userID)
	}
	return users
}

// IsUserOnline checks if a user has any active connections
func (m *Manager) IsUserOnline(userID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	clients, exists := m.userClients[userID]
	return exists && len(clients) > 0
}

// GetConnectionCount returns the total number of active connections
func (m *Manager) GetConnectionCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.clients)
}

// GetUserConnectionCount returns the number of connections for a specific user
func (m *Manager) GetUserConnectionCount(userID string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if clients, exists := m.userClients[userID]; exists {
		return len(clients)
	}
	return 0
}

// Internal methods

func (m *Manager) registerClient(client *Client) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check max connections
	if len(m.clients) >= m.cfg.MaxConnections {
		logger.Warn().
			Str("clientId", client.ID).
			Int("maxConnections", m.cfg.MaxConnections).
			Msg("Max connections reached, rejecting client")
		client.Close()
		return
	}

	m.clients[client.ID] = client

	// Add to user's client set
	if _, exists := m.userClients[client.UserID]; !exists {
		m.userClients[client.UserID] = make(map[string]bool)
	}
	m.userClients[client.UserID][client.ID] = true

	logger.Info().
		Str("clientId", client.ID).
		Str("userId", client.UserID).
		Int("totalConnections", len(m.clients)).
		Msg("Client registered")
}

func (m *Manager) unregisterClient(client *Client) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.clients[client.ID]; !exists {
		return
	}

	delete(m.clients, client.ID)

	// Remove from user's client set
	if clients, exists := m.userClients[client.UserID]; exists {
		delete(clients, client.ID)
		if len(clients) == 0 {
			delete(m.userClients, client.UserID)
		}
	}

	client.Close()

	logger.Info().
		Str("clientId", client.ID).
		Str("userId", client.UserID).
		Int("totalConnections", len(m.clients)).
		Msg("Client unregistered")
}

func (m *Manager) broadcastMessage(msg *BroadcastMessage) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, client := range m.clients {
		if msg.Exclude != nil && msg.Exclude[client.ID] {
			continue
		}

		select {
		case client.Send <- msg.Message:
		default:
			// Client's buffer is full, skip
			logger.Warn().
				Str("clientId", client.ID).
				Msg("Client send buffer full, skipping message")
		}
	}
}

func (m *Manager) sendToUser(msg *UserMessage) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	clientIDs, exists := m.userClients[msg.UserID]
	if !exists {
		logger.Debug().
			Str("userId", msg.UserID).
			Msg("User not online, message not delivered")
		return
	}

	for clientID := range clientIDs {
		if client, ok := m.clients[clientID]; ok {
			select {
			case client.Send <- msg.Message:
			default:
				logger.Warn().
					Str("clientId", clientID).
					Msg("Client send buffer full, skipping message")
			}
		}
	}
}

// Client methods

// NewClient creates a new WebSocket client
func NewClient(id, userID string, conn *websocket.Conn, manager *Manager) *Client {
	return &Client{
		ID:      id,
		UserID:  userID,
		Conn:    conn,
		Send:    make(chan []byte, 256),
		Manager: manager,
	}
}

// Close closes the client connection
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isClosed {
		return
	}
	c.isClosed = true

	close(c.Send)
	c.Conn.Close()
}

// WritePump handles writing messages to the WebSocket connection
func (c *Client) WritePump() {
	ticker := time.NewTicker(c.Manager.cfg.PingInterval)
	defer func() {
		ticker.Stop()
		c.Manager.Unregister(c)
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(c.Manager.cfg.WriteWait))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				logger.Error().Err(err).Str("clientId", c.ID).Msg("Write error")
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(c.Manager.cfg.WriteWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// ReadPump handles reading messages from the WebSocket connection
func (c *Client) ReadPump() {
	defer func() {
		c.Manager.Unregister(c)
	}()

	c.Conn.SetReadLimit(c.Manager.cfg.MaxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(c.Manager.cfg.PongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(c.Manager.cfg.PongWait))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Error().Err(err).Str("clientId", c.ID).Msg("Unexpected close error")
			}
			return
		}

		// Handle incoming message
		c.handleMessage(message)
	}
}

func (c *Client) handleMessage(message []byte) {
	var msg Message
	if err := json.Unmarshal(message, &msg); err != nil {
		logger.Error().Err(err).Str("clientId", c.ID).Msg("Failed to unmarshal message")
		return
	}

	logger.Debug().
		Str("clientId", c.ID).
		Str("type", msg.Type).
		Str("event", msg.Event).
		Msg("Received message")

	// Handle different message types
	switch msg.Type {
	case "ping":
		// Respond with pong
		response, _ := json.Marshal(&Message{
			Type:      "pong",
			Timestamp: time.Now(),
		})
		c.Send <- response

	case "subscribe":
		// Handle subscription requests
		// This can be extended to support topic-based subscriptions

	case "unsubscribe":
		// Handle unsubscription requests

	default:
		logger.Debug().
			Str("clientId", c.ID).
			Str("type", msg.Type).
			Msg("Unknown message type")
	}
}
