package http

import (
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"

	"microservices/cart/internal/model"
	"microservices/cart/internal/service"
	"microservices/cart/internal/service/impl"
	"microservices/cart/pkg/response"
)

// CartHandler handles HTTP requests for cart operations.
type CartHandler struct {
	cartService service.CartService
	validate    *validator.Validate
}

// NewCartHandler creates a new cart handler.
func NewCartHandler(cartService service.CartService) *CartHandler {
	return &CartHandler{
		cartService: cartService,
		validate:    validator.New(),
	}
}

// GetCart retrieves the user's cart.
// GET /api/v1/cart/:userId
func (h *CartHandler) GetCart(c *fiber.Ctx) error {
	userID := c.Params("userId")
	if userID == "" {
		return response.BadRequest(c, "userId is required")
	}

	cart, err := h.cartService.GetCart(c.Context(), userID)
	if err != nil {
		return response.InternalError(c, "Failed to get cart")
	}

	return response.Success(c, cart)
}

// AddToCart adds an item to the user's cart.
// POST /api/v1/cart/:userId/items
func (h *CartHandler) AddToCart(c *fiber.Ctx) error {
	userID := c.Params("userId")
	if userID == "" {
		return response.BadRequest(c, "userId is required")
	}

	var req model.AddItemRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if err := h.validate.Struct(&req); err != nil {
		return response.ValidationError(c, formatValidationErrors(err))
	}

	if err := h.cartService.AddToCart(c.Context(), userID, &req); err != nil {
		if errors.Is(err, impl.ErrCartLimitExceed) {
			return response.BadRequest(c, err.Error())
		}
		return response.InternalError(c, "Failed to add item to cart")
	}

	return response.SuccessWithMessage(c, nil, "Item added to cart")
}

// RemoveItem removes an item from the user's cart.
// DELETE /api/v1/cart/:userId/items/:skuId
func (h *CartHandler) RemoveItem(c *fiber.Ctx) error {
	userID := c.Params("userId")
	skuID := c.Params("skuId")

	if userID == "" || skuID == "" {
		return response.BadRequest(c, "userId and skuId are required")
	}

	if err := h.cartService.RemoveItem(c.Context(), userID, skuID); err != nil {
		return response.InternalError(c, "Failed to remove item from cart")
	}

	return response.SuccessWithMessage(c, nil, "Item removed from cart")
}

// UpdateQuantity updates the quantity of an item.
// PUT /api/v1/cart/:userId/items/:skuId/quantity
func (h *CartHandler) UpdateQuantity(c *fiber.Ctx) error {
	userID := c.Params("userId")
	skuID := c.Params("skuId")

	if userID == "" || skuID == "" {
		return response.BadRequest(c, "userId and skuId are required")
	}

	var req model.UpdateQuantityRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if err := h.validate.Struct(&req); err != nil {
		return response.ValidationError(c, formatValidationErrors(err))
	}

	if err := h.cartService.UpdateQuantity(c.Context(), userID, skuID, req.Quantity); err != nil {
		if errors.Is(err, impl.ErrItemNotFound) {
			return response.NotFound(c, "Item not found in cart")
		}
		if errors.Is(err, impl.ErrInvalidQuantity) {
			return response.BadRequest(c, err.Error())
		}
		return response.InternalError(c, "Failed to update quantity")
	}

	return response.SuccessWithMessage(c, nil, "Quantity updated")
}

// UpdateSelection updates the selection status of an item.
// PUT /api/v1/cart/:userId/items/:skuId/selection
func (h *CartHandler) UpdateSelection(c *fiber.Ctx) error {
	userID := c.Params("userId")
	skuID := c.Params("skuId")

	if userID == "" || skuID == "" {
		return response.BadRequest(c, "userId and skuId are required")
	}

	var req model.UpdateSelectionRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if err := h.cartService.UpdateSelection(c.Context(), userID, skuID, req.Selected); err != nil {
		if errors.Is(err, impl.ErrItemNotFound) {
			return response.NotFound(c, "Item not found in cart")
		}
		return response.InternalError(c, "Failed to update selection")
	}

	return response.SuccessWithMessage(c, nil, "Selection updated")
}

// ClearCart removes all items from the user's cart.
// DELETE /api/v1/cart/:userId
func (h *CartHandler) ClearCart(c *fiber.Ctx) error {
	userID := c.Params("userId")

	if userID == "" {
		return response.BadRequest(c, "userId is required")
	}

	if err := h.cartService.ClearCart(c.Context(), userID); err != nil {
		return response.InternalError(c, "Failed to clear cart")
	}

	return response.SuccessWithMessage(c, nil, "Cart cleared")
}

// GetCartSummary retrieves cart summary with total price.
// GET /api/v1/cart/:userId/summary
func (h *CartHandler) GetCartSummary(c *fiber.Ctx) error {
	userID := c.Params("userId")
	if userID == "" {
		return response.BadRequest(c, "userId is required")
	}

	selectedOnly := c.QueryBool("selectedOnly", false)

	summary, err := h.cartService.GetCartSummary(c.Context(), userID, selectedOnly)
	if err != nil {
		return response.InternalError(c, "Failed to get cart summary")
	}

	return response.Success(c, summary)
}

// HealthCheck returns the health status of the service.
// GET /health
func (h *CartHandler) HealthCheck(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":  "healthy",
		"service": "cart-service",
	})
}

func formatValidationErrors(err error) []string {
	var errors []string
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, e := range validationErrors {
			errors = append(errors, formatFieldError(e))
		}
	}
	return errors
}

func formatFieldError(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return e.Field() + " is required"
	case "gt":
		return e.Field() + " must be greater than " + e.Param()
	case "min":
		return e.Field() + " must be at least " + e.Param()
	case "max":
		return e.Field() + " must be at most " + e.Param()
	default:
		return e.Field() + " is invalid"
	}
}
