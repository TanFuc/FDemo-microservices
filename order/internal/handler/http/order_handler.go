package http

import (
	"errors"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"microservices/order/internal/domain"
	"microservices/order/internal/usecase"
	"microservices/pkg/authclient"
)

// OrderHandler handles HTTP requests for orders
type OrderHandler struct {
	createOrderUC       *usecase.CreateOrderUseCase
	cancelOrderUC       *usecase.CancelOrderUseCase
	getOrderUC          *usecase.GetOrderUseCase
	listOrdersUC        *usecase.ListOrdersUseCase
	markAsPaidUC        *usecase.MarkAsPaidUseCase
	markAsShippedUC     *usecase.MarkAsShippedUseCase
	markAsCompletedUC   *usecase.MarkAsCompletedUseCase
	createDraftOrderUC  *usecase.CreateDraftOrderUseCase
	confirmDraftOrderUC *usecase.ConfirmDraftOrderUseCase
	deleteDraftOrderUC  *usecase.DeleteDraftOrderUseCase
	authMiddleware      *authclient.FiberMiddleware
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
	createDraftOrderUC *usecase.CreateDraftOrderUseCase,
	confirmDraftOrderUC *usecase.ConfirmDraftOrderUseCase,
	deleteDraftOrderUC *usecase.DeleteDraftOrderUseCase,
	authMiddleware *authclient.FiberMiddleware,
) *OrderHandler {
	return &OrderHandler{
		createOrderUC:       createOrderUC,
		cancelOrderUC:       cancelOrderUC,
		getOrderUC:          getOrderUC,
		listOrdersUC:        listOrdersUC,
		markAsPaidUC:        markAsPaidUC,
		markAsShippedUC:     markAsShippedUC,
		markAsCompletedUC:   markAsCompletedUC,
		createDraftOrderUC:  createDraftOrderUC,
		confirmDraftOrderUC: confirmDraftOrderUC,
		deleteDraftOrderUC:  deleteDraftOrderUC,
		authMiddleware:      authMiddleware,
	}
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

// CreateDraftOrder handles POST /api/v1/orders/draft
func (h *OrderHandler) CreateDraftOrder(c *fiber.Ctx) error {
	var req usecase.CreateDraftOrderRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
		})
	}

	resp, err := h.createDraftOrderUC.Execute(c.Context(), &req)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.Status(http.StatusCreated).JSON(resp)
}

// GetDraftOrder handles GET /api/v1/orders/draft/:id
func (h *OrderHandler) GetDraftOrder(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Invalid order ID",
			Message: "Order ID must be a valid UUID",
		})
	}

	// Get order and verify it's a draft
	resp, err := h.getOrderUC.Execute(c.Context(), id)
	if err != nil {
		return h.handleError(c, err)
	}

	if resp.Status != string(domain.StatusDraft) {
		return c.Status(http.StatusNotFound).JSON(ErrorResponse{
			Error:   "Draft not found",
			Message: "Order is not a draft",
		})
	}

	return c.Status(http.StatusOK).JSON(resp)
}

// ConfirmDraftOrder handles POST /api/v1/orders/draft/:id/confirm
func (h *OrderHandler) ConfirmDraftOrder(c *fiber.Ctx) error {
	idStr := c.Params("id")
	orderID, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Invalid order ID",
			Message: "Order ID must be a valid UUID",
		})
	}

	// Get user ID from request body or header
	var req struct {
		UserID string `json:"userId"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
		})
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Invalid user ID",
			Message: "User ID must be a valid UUID",
		})
	}

	confirmReq := &usecase.ConfirmDraftOrderRequest{
		OrderID: orderID,
		UserID:  userID,
	}

	resp, err := h.confirmDraftOrderUC.Execute(c.Context(), confirmReq)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.Status(http.StatusOK).JSON(resp)
}

// ListDraftOrders handles GET /api/v1/orders/user/:userId/drafts
func (h *OrderHandler) ListDraftOrders(c *fiber.Ctx) error {
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

	// Use getOrderUC.ListDrafts (we need to add this)
	resp, err := h.listOrdersUC.ExecuteDrafts(c.Context(), userID, limit, offset)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.Status(http.StatusOK).JSON(ListOrdersResponse{
		Orders: resp,
		Limit:  limit,
		Offset: offset,
	})
}

// DeleteDraftOrder handles DELETE /api/v1/orders/draft/:id
func (h *OrderHandler) DeleteDraftOrder(c *fiber.Ctx) error {
	idStr := c.Params("id")
	orderID, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Invalid order ID",
			Message: "Order ID must be a valid UUID",
		})
	}

	// Extract userID from JWT locals (set by auth middleware)
	userIDStr, ok := c.Locals("userId").(string)
	if !ok || userIDStr == "" {
		return c.Status(http.StatusUnauthorized).JSON(ErrorResponse{
			Error:   "Unauthorized",
			Message: "Missing user context",
		})
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.Status(http.StatusUnauthorized).JSON(ErrorResponse{
			Error:   "Unauthorized",
			Message: "Invalid user context",
		})
	}

	if err := h.deleteDraftOrderUC.Execute(c.Context(), orderID, userID); err != nil {
		return h.handleError(c, err)
	}

	return c.SendStatus(http.StatusNoContent)
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

	case errors.Is(err, domain.ErrInvalidFinalAmount):
		return c.Status(http.StatusUnprocessableEntity).JSON(ErrorResponse{
			Error:   "Price mismatch",
			Message: "Final amount does not match item totals. Please refresh your cart.",
		})

	case errors.Is(err, domain.ErrDraftOrderExpired):
		return c.Status(http.StatusGone).JSON(ErrorResponse{
			Error:   "Draft expired",
			Message: "This draft order has expired. Please start a new checkout.",
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
