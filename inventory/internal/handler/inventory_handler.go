package handler

import (
	"errors"

	"microservices/inventory/internal/domain"
	"microservices/inventory/internal/usecase"

	"github.com/gofiber/fiber/v2"
)

type InventoryHandler struct {
	useCase usecase.InventoryUseCase
}

func NewInventoryHandler(useCase usecase.InventoryUseCase) *InventoryHandler {
	return &InventoryHandler{useCase: useCase}
}

type ReserveRequest struct {
	OrderID string                    `json:"order_id"`
	Items   []domain.ReservationItem  `json:"items"`
}

type OrderRequest struct {
	OrderID string `json:"order_id"`
}

type CreateInventoryRequest struct {
	SkuID      string `json:"sku_id"`
	TotalStock int    `json:"total_stock"`
}

type SyncRequest struct {
	SkuID string `json:"sku_id"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type SuccessResponse struct {
	Message string `json:"message"`
}

func (h *InventoryHandler) RegisterRoutes(app *fiber.App) {
	api := app.Group("/api/v1")

	api.Post("/inventory", h.CreateInventory)
	api.Get("/inventory/:sku_id", h.GetInventory)
	api.Post("/inventory/reserve", h.ReserveStock)
	api.Post("/inventory/confirm", h.ConfirmStock)
	api.Post("/inventory/release", h.ReleaseStock)
	api.Post("/inventory/sync", h.SyncRedisFromDB)
}

func (h *InventoryHandler) CreateInventory(c *fiber.Ctx) error {
	var req CreateInventoryRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "invalid request body"})
	}

	if req.SkuID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "sku_id is required"})
	}

	item := &domain.InventoryItem{
		SkuID:         req.SkuID,
		TotalStock:    req.TotalStock,
		ReservedStock: 0,
	}

	if err := h.useCase.CreateInventory(c.Context(), item); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(item)
}

func (h *InventoryHandler) GetInventory(c *fiber.Ctx) error {
	skuID := c.Params("sku_id")
	if skuID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "sku_id is required"})
	}

	item, err := h.useCase.GetInventory(c.Context(), skuID)
	if err != nil {
		if errors.Is(err, domain.ErrInventoryNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{Error: "inventory not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: err.Error()})
	}

	return c.JSON(item)
}

func (h *InventoryHandler) ReserveStock(c *fiber.Ctx) error {
	var req ReserveRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "invalid request body"})
	}

	if req.OrderID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "order_id is required"})
	}

	if len(req.Items) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "items are required"})
	}

	if err := h.useCase.ReserveStock(c.Context(), req.OrderID, req.Items); err != nil {
		if errors.Is(err, domain.ErrInsufficientStock) {
			return c.Status(fiber.StatusConflict).JSON(ErrorResponse{Error: "insufficient stock"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: err.Error()})
	}

	return c.JSON(SuccessResponse{Message: "stock reserved successfully"})
}

func (h *InventoryHandler) ConfirmStock(c *fiber.Ctx) error {
	var req OrderRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "invalid request body"})
	}

	if req.OrderID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "order_id is required"})
	}

	if err := h.useCase.ConfirmStock(c.Context(), req.OrderID); err != nil {
		if errors.Is(err, domain.ErrReservationNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{Error: "reservation not found"})
		}
		if errors.Is(err, domain.ErrAlreadyCancelled) {
			return c.Status(fiber.StatusConflict).JSON(ErrorResponse{Error: "reservation already cancelled"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: err.Error()})
	}

	return c.JSON(SuccessResponse{Message: "stock confirmed successfully"})
}

func (h *InventoryHandler) ReleaseStock(c *fiber.Ctx) error {
	var req OrderRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "invalid request body"})
	}

	if req.OrderID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "order_id is required"})
	}

	if err := h.useCase.ReleaseStock(c.Context(), req.OrderID); err != nil {
		if errors.Is(err, domain.ErrReservationNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{Error: "reservation not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: err.Error()})
	}

	return c.JSON(SuccessResponse{Message: "stock released successfully"})
}

func (h *InventoryHandler) SyncRedisFromDB(c *fiber.Ctx) error {
	var req SyncRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "invalid request body"})
	}

	if req.SkuID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{Error: "sku_id is required"})
	}

	if err := h.useCase.SyncRedisFromDB(c.Context(), req.SkuID); err != nil {
		if errors.Is(err, domain.ErrInventoryNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{Error: "inventory not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{Error: err.Error()})
	}

	return c.JSON(SuccessResponse{Message: "redis synced successfully"})
}
