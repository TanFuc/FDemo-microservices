package http

import (
	"errors"

	"microservices/inventory/internal/model"
	"microservices/inventory/internal/service"
	"microservices/inventory/pkg/response"

	"github.com/gofiber/fiber/v2"
)

type InventoryHandler struct {
	inventoryService service.InventoryService
}

func NewInventoryHandler(inventoryService service.InventoryService) *InventoryHandler {
	return &InventoryHandler{inventoryService: inventoryService}
}

// CreateInventory creates a new inventory item
// @Summary Create inventory
// @Description Create a new inventory item for a SKU
// @Tags Inventory
// @Accept json
// @Produce json
// @Param request body model.CreateInventoryRequest true "Create inventory request"
// @Success 201 {object} response.Response{data=model.InventoryResponse}
// @Failure 400 {object} response.Response
// @Failure 409 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/inventory [post]
func (h *InventoryHandler) CreateInventory(c *fiber.Ctx) error {
	var req model.CreateInventoryRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}

	item, err := h.inventoryService.CreateInventory(c.Context(), &req)
	if err != nil {
		if errors.Is(err, model.ErrInvalidSkuID) || errors.Is(err, model.ErrInvalidQuantity) {
			return response.BadRequest(c, err.Error())
		}
		if errors.Is(err, model.ErrInventoryAlreadyExists) {
			return response.Conflict(c, err.Error())
		}
		return response.InternalError(c, err)
	}

	return response.Created(c, item.ToResponse())
}

// GetInventory retrieves an inventory item
// @Summary Get inventory
// @Description Get inventory by SKU ID
// @Tags Inventory
// @Accept json
// @Produce json
// @Param sku_id path string true "SKU ID"
// @Success 200 {object} response.Response{data=model.InventoryResponse}
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/inventory/{sku_id} [get]
func (h *InventoryHandler) GetInventory(c *fiber.Ctx) error {
	skuID := c.Params("sku_id")
	if skuID == "" {
		return response.BadRequest(c, "sku_id is required")
	}

	item, err := h.inventoryService.GetInventory(c.Context(), skuID)
	if err != nil {
		if errors.Is(err, model.ErrInventoryNotFound) {
			return response.NotFound(c, err.Error())
		}
		return response.InternalError(c, err)
	}

	return response.Success(c, item.ToResponse())
}

// UpdateInventory updates an inventory item
// @Summary Update inventory
// @Description Update inventory total stock
// @Tags Inventory
// @Accept json
// @Produce json
// @Param sku_id path string true "SKU ID"
// @Param request body model.UpdateInventoryRequest true "Update inventory request"
// @Success 200 {object} response.Response{data=model.InventoryResponse}
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 409 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/inventory/{sku_id} [put]
func (h *InventoryHandler) UpdateInventory(c *fiber.Ctx) error {
	skuID := c.Params("sku_id")
	if skuID == "" {
		return response.BadRequest(c, "sku_id is required")
	}

	var req model.UpdateInventoryRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}

	item, err := h.inventoryService.UpdateInventory(c.Context(), skuID, &req)
	if err != nil {
		if errors.Is(err, model.ErrInventoryNotFound) {
			return response.NotFound(c, err.Error())
		}
		if errors.Is(err, model.ErrInsufficientStock) {
			return response.Conflict(c, "total stock cannot be less than reserved stock")
		}
		if errors.Is(err, model.ErrInvalidQuantity) {
			return response.BadRequest(c, err.Error())
		}
		return response.InternalError(c, err)
	}

	return response.Success(c, item.ToResponse())
}

// DeleteInventory deletes an inventory item
// @Summary Delete inventory
// @Description Delete inventory by SKU ID
// @Tags Inventory
// @Accept json
// @Produce json
// @Param sku_id path string true "SKU ID"
// @Success 204 "No Content"
// @Failure 404 {object} response.Response
// @Failure 409 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/inventory/{sku_id} [delete]
func (h *InventoryHandler) DeleteInventory(c *fiber.Ctx) error {
	skuID := c.Params("sku_id")
	if skuID == "" {
		return response.BadRequest(c, "sku_id is required")
	}

	err := h.inventoryService.DeleteInventory(c.Context(), skuID)
	if err != nil {
		if errors.Is(err, model.ErrInventoryNotFound) {
			return response.NotFound(c, err.Error())
		}
		if errors.Is(err, model.ErrInsufficientStock) {
			return response.Conflict(c, "cannot delete inventory with reserved stock")
		}
		return response.InternalError(c, err)
	}

	return response.NoContent(c)
}

// ListInventory lists inventory items
// @Summary List inventory
// @Description List all inventory items with pagination
// @Tags Inventory
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {object} response.PaginatedResponse{data=[]model.InventoryResponse}
// @Failure 500 {object} response.Response
// @Router /api/v1/inventory [get]
func (h *InventoryHandler) ListInventory(c *fiber.Ctx) error {
	filter := &model.InventoryFilter{
		Page:  c.QueryInt("page", 1),
		Limit: c.QueryInt("limit", 20),
	}

	items, total, err := h.inventoryService.ListInventory(c.Context(), filter)
	if err != nil {
		return response.InternalError(c, err)
	}

	responses := make([]*model.InventoryResponse, len(items))
	for i := range items {
		responses[i] = items[i].ToResponse()
	}

	return response.Paginated(c, responses, total, filter.Page, filter.Limit)
}

// SyncInventory synchronizes inventory from DB to cache
// @Summary Sync inventory
// @Description Synchronize inventory from database to Redis cache
// @Tags Inventory
// @Accept json
// @Produce json
// @Param request body model.SyncInventoryRequest true "Sync inventory request"
// @Success 200 {object} response.Response{data=model.InventoryResponse}
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/inventory/sync [post]
func (h *InventoryHandler) SyncInventory(c *fiber.Ctx) error {
	var req model.SyncInventoryRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}

	item, err := h.inventoryService.SyncInventory(c.Context(), req.SkuID)
	if err != nil {
		if errors.Is(err, model.ErrInventoryNotFound) {
			return response.NotFound(c, err.Error())
		}
		if errors.Is(err, model.ErrInvalidSkuID) {
			return response.BadRequest(c, err.Error())
		}
		return response.InternalError(c, err)
	}

	return response.SuccessWithMessage(c, "inventory synced successfully", item.ToResponse())
}
