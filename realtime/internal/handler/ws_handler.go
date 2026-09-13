package handler

import (
	"strings"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"microservices/realtime/internal/hub"
)

type WSHandler struct {
	hub       *hub.Hub
	jwtSecret []byte
}

func NewWSHandler(h *hub.Hub, jwtSecret string) *WSHandler {
	return &WSHandler{
		hub:       h,
		jwtSecret: []byte(jwtSecret),
	}
}

// UpgradeMiddleware validates websocket upgrade and extracts JWT credentials.
func (h *WSHandler) UpgradeMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !websocket.IsWebSocketUpgrade(c) {
			return fiber.ErrUpgradeRequired
		}

		// Extract token from query or Authorization header
		tokenStr := c.Query("token")
		if tokenStr == "" {
			authHeader := c.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		userID := ""
		if tokenStr != "" {
			token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
				return h.jwtSecret, nil
			})
			if err == nil && token.Valid {
				if claims, ok := token.Claims.(jwt.MapClaims); ok {
					if sub, exists := claims["sub"].(string); exists {
						userID = sub
					} else if uid, exists := claims["userId"].(string); exists {
						userID = uid
					}
				}
			}
		}

		// Store in locals for the websocket handler
		c.Locals("userID", userID)
		c.Locals("clientID", uuid.New().String())

		return c.Next()
	}
}

// HandleWebSocket manages the websocket connection lifecycle.
func (h *WSHandler) HandleWebSocket() fiber.Handler {
	return websocket.New(func(c *websocket.Conn) {
		clientID, _ := c.Locals("clientID").(string)
		userID, _ := c.Locals("userID").(string)

		client := hub.NewClient(clientID, userID, c, h.hub)
		h.hub.Register(client)

		// Run write pump in goroutine
		go client.WritePump()

		// Read pump blocks until socket is closed
		client.ReadPump()
	})
}
