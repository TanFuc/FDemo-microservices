package http

import (
	"errors"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"microservices/order/internal/domain"
	"microservices/order/internal/usecase"
)

// OrderHandler handles HTTP requests for orders
type OrderHandler struct {
	createOrderUC      *usecase.CreateOrderUseCase
	cancelOrderUC      *usecase.CancelOrderUseCase
	getOrderUC         *usecase.GetOrderUseCase
	listOrdersUC       *usecase.ListOrdersUseCase
	markAsPaidUC       *usecase.MarkAsPaidUseCase
	markAsShippedUC    *usecase.MarkAsShippedUseCase
	markAsCompletedUC  *usecase.MarkAsCompletedUseCase
}

// NewOrderHandler creates a new OrderHandler
func NewOrderHandler(
	createOrderUC *usecase.CreateOrderUseCase,
	cancelOrderUC *usecase.CancelOrderUseCase,
	getOrderUC *usecase.GetOrderUseCase,
	listOrdersUC *usecase.ListOrdersUseCase,
	markAsPaidUC *usecase.MarkAsPaidUseCase,
	markAsShippedUC *usecase.MarkAsShippedUseCase,
	markAsCompletedUC *usecase.MarkAsCompletedUseCase,
) *OrderHandler {
	return &OrderHandler{
		createOrderUC:      createOrderUC,
		cancelOrderUC:      cancelOrderUC,
		getOrderUC:         getOrderUC,
		listOrdersUC:       listOrdersUC,
		markAsPaidUC:       markAsPaidUC,
		markAsShippedUC:    markAsShippedUC,
		markAsCompletedUC:  markAsCompletedUC,
	}
}

// RegisterRoutes registers all order routes
func (h *OrderHandler) RegisterRoutes(app *fiber.App) {
	orders := app.Group("/api/v1/orders")

	orders.Post("/", h.CreateOrder)
	orders.Get("/:id", h.GetOrder)
	orders.Post("/:id/cancel", h.CancelOrder)
	orders.Post("/:id/pay", h.MarkAsPaid)
	orders.Post("/:id/ship", h.MarkAsShipped)
	orders.Post("/:id/complete", h.MarkAsCompleted)
	orders.Get("/user/:userId", h.ListOrders)
}

// CreateOrder handles POST /api/v1/orders
func (h *OrderHandler) CreateOrder(c *fiber.Ctx) error {
	var req usecase.CreateOrderRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
		})
	}

	resp, err := h.createOrderUC.Execute(c.Context(), &req)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.Status(http.StatusCreated).JSON(resp)
}

// GetOrder handles GET /api/v1/orders/:id
func (h *OrderHandler) GetOrder(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Invalid order ID",
			Message: "Order ID must be a valid UUID",
		})
	}

	resp, err := h.getOrderUC.Execute(c.Context(), id)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.Status(http.StatusOK).JSON(resp)
}

// CancelOrder handles POST /api/v1/orders/:id/cancel
func (h *OrderHandler) CancelOrder(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Invalid order ID",
			Message: "Order ID must be a valid UUID",
		})
	}

	if err := h.cancelOrderUC.Execute(c.Context(), id); err != nil {
		return h.handleError(c, err)
	}

	return c.Status(http.StatusOK).JSON(SuccessResponse{
		Message: "Order cancelled successfully",
	})
}

// MarkAsPaid handles POST /api/v1/orders/:id/pay
func (h *OrderHandler) MarkAsPaid(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Invalid order ID",
			Message: "Order ID must be a valid UUID",
		})
	}

	if err := h.markAsPaidUC.Execute(c.Context(), id); err != nil {
		return h.handleError(c, err)
	}

	return c.Status(http.StatusOK).JSON(SuccessResponse{
		Message: "Order marked as paid successfully",
	})
}

// MarkAsShipped handles POST /api/v1/orders/:id/ship
func (h *OrderHandler) MarkAsShipped(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Invalid order ID",
			Message: "Order ID must be a valid UUID",
		})
	}

	if err := h.markAsShippedUC.Execute(c.Context(), id); err != nil {
		return h.handleError(c, err)
	}

	return c.Status(http.StatusOK).JSON(SuccessResponse{
		Message: "Order marked as shipped successfully",
	})
}

// MarkAsCompleted handles POST /api/v1/orders/:id/complete
func (h *OrderHandler) MarkAsCompleted(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Invalid order ID",
			Message: "Order ID must be a valid UUID",
		})
	}

	if err := h.markAsCompletedUC.Execute(c.Context(), id); err != nil {
		return h.handleError(c, err)
	}

	return c.Status(http.StatusOK).JSON(SuccessResponse{
		Message: "Order marked as completed successfully",
	})
}

// ListOrders handles GET /api/v1/orders/user/:userId
func (h *OrderHandler) ListOrders(c *fiber.Ctx) error {
	userIDStr := c.Params("userId")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Invalid user ID",
			Message: "User ID must be a valid UUID",
		})
	}

	limit := c.QueryInt("limit", 10)
	offset := c.QueryInt("offset", 0)

	resp, err := h.listOrdersUC.Execute(c.Context(), userID, limit, offset)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.Status(http.StatusOK).JSON(ListOrdersResponse{
		Orders: resp,
		Limit:  limit,
		Offset: offset,
	})
}

// handleError maps domain errors to HTTP responses
func (h *OrderHandler) handleError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrOrderNotFound):
		return c.Status(http.StatusNotFound).JSON(ErrorResponse{
			Error:   "Order not found",
			Message: err.Error(),
		})

	case errors.Is(err, domain.ErrOutOfStock):
		return c.Status(http.StatusConflict).JSON(ErrorResponse{
			Error:   "Out of stock",
			Message: err.Error(),
		})

	case errors.Is(err, domain.ErrOrderCannotBeCancelled):
		return c.Status(http.StatusConflict).JSON(ErrorResponse{
			Error:   "Cannot cancel order",
			Message: err.Error(),
		})

	case errors.Is(err, domain.ErrInvalidOrderStatusTransition):
		return c.Status(http.StatusConflict).JSON(ErrorResponse{
			Error:   "Invalid status transition",
			Message: err.Error(),
		})

	case errors.Is(err, domain.ErrInvalidUserID),
		errors.Is(err, domain.ErrInvalidPaymentMethod),
		errors.Is(err, domain.ErrInvalidAddress),
		errors.Is(err, domain.ErrEmptyOrderItems),
		errors.Is(err, domain.ErrInvalidQuantity),
		errors.Is(err, domain.ErrInvalidPrice):
		return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Validation error",
			Message: err.Error(),
		})

	case errors.Is(err, domain.ErrInventoryServiceUnavailable):
		return c.Status(http.StatusServiceUnavailable).JSON(ErrorResponse{
			Error:   "Service unavailable",
			Message: "Inventory service is currently unavailable",
		})

	default:
		return c.Status(http.StatusInternalServerError).JSON(ErrorResponse{
			Error:   "Internal server error",
			Message: "An unexpected error occurred",
		})
	}
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// SuccessResponse represents a success response
type SuccessResponse struct {
	Message string `json:"message"`
}

// ListOrdersResponse represents the response for listing orders
type ListOrdersResponse struct {
	Orders []*usecase.OrderResponse `json:"orders"`
	Limit  int                      `json:"limit"`
	Offset int                      `json:"offset"`
}
