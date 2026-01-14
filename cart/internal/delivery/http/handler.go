package http

import (
	"microservices/cart/internal/domain"
	"microservices/cart/internal/usecase"

	"github.com/gofiber/fiber/v2"
)

// CartHandler handles HTTP requests for cart operations.
type CartHandler struct {
	usecase *usecase.CartUsecase
}

// NewCartHandler creates a new cart handler.
func NewCartHandler(uc *usecase.CartUsecase) *CartHandler {
	return &CartHandler{usecase: uc}
}

// RegisterRoutes registers all cart routes.
func (h *CartHandler) RegisterRoutes(app *fiber.App) {
	cart := app.Group("/api/v1/cart")

	cart.Get("/:userId", h.GetCart)
	cart.Post("/:userId/items", h.AddToCart)
	cart.Delete("/:userId/items/:skuId", h.RemoveItem)
	cart.Put("/:userId/items/:skuId/quantity", h.UpdateQuantity)
	cart.Put("/:userId/items/:skuId/selection", h.UpdateSelection)
	cart.Delete("/:userId", h.ClearCart)
}

// GetCart retrieves the user's cart.
// GET /api/v1/cart/:userId
func (h *CartHandler) GetCart(c *fiber.Ctx) error {
	userID := c.Params("userId")
	if userID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "invalid_request",
			Message: "userId is required",
		})
	}

	cart, err := h.usecase.GetCart(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(domain.ErrorResponse{
			Error:   "internal_error",
			Message: err.Error(),
		})
	}

	return c.JSON(cart)
}

// AddToCart adds an item to the user's cart.
// POST /api/v1/cart/:userId/items
func (h *CartHandler) AddToCart(c *fiber.Ctx) error {
	userID := c.Params("userId")
	if userID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "invalid_request",
			Message: "userId is required",
		})
	}

	var req domain.AddItemRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "invalid_request",
			Message: "invalid request body",
		})
	}

	// Basic validation
	if req.SkuID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "invalid_request",
			Message: "skuId is required",
		})
	}
	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "invalid_request",
			Message: "name is required",
		})
	}
	if req.Price <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "invalid_request",
			Message: "price must be greater than 0",
		})
	}
	if req.Quantity <= 0 {
		req.Quantity = 1 // Default to 1 if not specified
	}

	if err := h.usecase.AddToCart(c.Context(), userID, req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(domain.ErrorResponse{
			Error:   "internal_error",
			Message: err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "item added to cart",
	})
}

// RemoveItem removes an item from the user's cart.
// DELETE /api/v1/cart/:userId/items/:skuId
func (h *CartHandler) RemoveItem(c *fiber.Ctx) error {
	userID := c.Params("userId")
	skuID := c.Params("skuId")

	if userID == "" || skuID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "invalid_request",
			Message: "userId and skuId are required",
		})
	}

	if err := h.usecase.RemoveItem(c.Context(), userID, skuID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(domain.ErrorResponse{
			Error:   "internal_error",
			Message: err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "item removed from cart",
	})
}

// UpdateQuantity updates the quantity of an item.
// PUT /api/v1/cart/:userId/items/:skuId/quantity
func (h *CartHandler) UpdateQuantity(c *fiber.Ctx) error {
	userID := c.Params("userId")
	skuID := c.Params("skuId")

	if userID == "" || skuID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "invalid_request",
			Message: "userId and skuId are required",
		})
	}

	var req domain.UpdateQuantityRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "invalid_request",
			Message: "invalid request body",
		})
	}

	if req.Quantity <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "invalid_request",
			Message: "quantity must be greater than 0",
		})
	}

	if err := h.usecase.UpdateQuantity(c.Context(), userID, skuID, req.Quantity); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(domain.ErrorResponse{
			Error:   "internal_error",
			Message: err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "quantity updated",
	})
}

// UpdateSelection updates the selection status of an item.
// PUT /api/v1/cart/:userId/items/:skuId/selection
func (h *CartHandler) UpdateSelection(c *fiber.Ctx) error {
	userID := c.Params("userId")
	skuID := c.Params("skuId")

	if userID == "" || skuID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "invalid_request",
			Message: "userId and skuId are required",
		})
	}

	var req struct {
		Selected bool `json:"selected"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "invalid_request",
			Message: "invalid request body",
		})
	}

	if err := h.usecase.UpdateItemSelection(c.Context(), userID, skuID, req.Selected); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(domain.ErrorResponse{
			Error:   "internal_error",
			Message: err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "selection updated",
	})
}

// ClearCart removes all items from the user's cart.
// DELETE /api/v1/cart/:userId
func (h *CartHandler) ClearCart(c *fiber.Ctx) error {
	userID := c.Params("userId")

	if userID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(domain.ErrorResponse{
			Error:   "invalid_request",
			Message: "userId is required",
		})
	}

	if err := h.usecase.ClearCart(c.Context(), userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(domain.ErrorResponse{
			Error:   "internal_error",
			Message: err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "cart cleared",
	})
}
