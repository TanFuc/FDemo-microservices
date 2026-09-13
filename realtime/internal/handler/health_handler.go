package handler

import (
	"github.com/gofiber/fiber/v2"
	"microservices/realtime/internal/hub"
)

type HealthHandler struct {
	hub *hub.Hub
}

func NewHealthHandler(h *hub.Hub) *HealthHandler {
	return &HealthHandler{hub: h}
}

func (h *HealthHandler) Health(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":      "healthy",
		"service":     "realtime",
		"connections": h.hub.TotalClients(),
		"users":       h.hub.TotalUsers(),
	})
}
