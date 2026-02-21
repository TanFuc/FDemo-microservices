package websocket

import (
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	ws "microservices/notification/internal/websocket"
	"microservices/pkg/logger"
)

type Handler struct {
	manager *ws.Manager
}

func NewHandler(manager *ws.Manager) *Handler {
	return &Handler{manager: manager}
}

// WebSocketUpgrade middleware checks if the request is a WebSocket upgrade request
func (h *Handler) WebSocketUpgrade() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	}
}

// HandleConnection handles WebSocket connections
func (h *Handler) HandleConnection() fiber.Handler {
	return websocket.New(func(c *websocket.Conn) {
		// Get user ID from context (set by auth middleware)
		userID := c.Locals("userId")
		if userID == nil {
			// Try to get from query parameter for testing
			userID = c.Query("user_id")
		}

		userIDStr, ok := userID.(string)
		if !ok || userIDStr == "" {
			logger.Error().Msg("WebSocket connection rejected: no user ID")
			c.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "Authentication required"))
			c.Close()
			return
		}

		// Create client
		clientID := uuid.New().String()
		client := ws.NewClient(clientID, userIDStr, c, h.manager)

		logger.Info().
			Str("clientId", clientID).
			Str("userId", userIDStr).
			Msg("New WebSocket connection")

		// Register client
		h.manager.Register(client)

		// Send welcome message
		welcomeMsg := &ws.Message{
			Type: "connected",
			Data: map[string]interface{}{
				"client_id": clientID,
				"user_id":   userIDStr,
				"message":   "Connected to notification service",
			},
		}
		if err := h.manager.SendToUser(userIDStr, welcomeMsg); err != nil {
			logger.Error().Err(err).Msg("Failed to send welcome message")
		}

		// Start read and write pumps
		go client.WritePump()
		client.ReadPump()
	})
}

// GetStats returns WebSocket statistics
func (h *Handler) GetStats(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"total_connections": h.manager.GetConnectionCount(),
			"online_users":      len(h.manager.GetOnlineUsers()),
		},
	})
}
