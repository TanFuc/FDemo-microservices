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

// ReturnHandler handles HTTP requests for returns
type ReturnHandler struct {
	processReturnUC *usecase.ProcessReturnUseCase
	authMiddleware  *authclient.FiberMiddleware
}

// NewReturnHandler creates a new ReturnHandler
func NewReturnHandler(
	processReturnUC *usecase.ProcessReturnUseCase,
	authMiddleware *authclient.FiberMiddleware,
) *ReturnHandler {
	return &ReturnHandler{
		processReturnUC: processReturnUC,
		authMiddleware:  authMiddleware,
	}
}

// CreateReturn handles POST /api/v1/returns
func (h *ReturnHandler) CreateReturn(c *fiber.Ctx) error {
	var req usecase.CreateReturnRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
		})
	}

	// Get user ID from auth context if available
	if h.authMiddleware != nil {
		userIDStr := authclient.GetUserID(c)
		if userIDStr != "" {
			if userID, err := uuid.Parse(userIDStr); err == nil {
				req.UserID = userID
			}
		}
	}

	resp, err := h.processReturnUC.CreateReturn(c.Context(), &req)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.Status(http.StatusCreated).JSON(resp)
}

// GetReturn handles GET /api/v1/returns/:id
func (h *ReturnHandler) GetReturn(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Invalid return ID",
			Message: "Return ID must be a valid UUID",
		})
	}

	// Get user ID from auth context
	userID := uuid.Nil
	if h.authMiddleware != nil {
		if userIDStr := authclient.GetUserID(c); userIDStr != "" {
			userID, _ = uuid.Parse(userIDStr)
		}
	}

	resp, err := h.processReturnUC.GetReturn(c.Context(), id, userID)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.Status(http.StatusOK).JSON(resp)
}

// ListReturns handles GET /api/v1/returns
func (h *ReturnHandler) ListReturns(c *fiber.Ctx) error {
	req := usecase.ListReturnsRequest{
		Status: c.Query("status"),
		Page:   c.QueryInt("page", 1),
		Limit:  c.QueryInt("limit", 20),
	}

	// Parse optional order_id
	if orderIDStr := c.Query("order_id"); orderIDStr != "" {
		orderID, err := uuid.Parse(orderIDStr)
		if err != nil {
			return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
				Error:   "Invalid order ID",
				Message: "Order ID must be a valid UUID",
			})
		}
		req.OrderID = orderID
	}

	// Get user ID from auth context
	if h.authMiddleware != nil {
		if userIDStr := authclient.GetUserID(c); userIDStr != "" {
			if userID, err := uuid.Parse(userIDStr); err == nil {
				req.UserID = userID
			}
		}
	}

	resp, err := h.processReturnUC.ListReturns(c.Context(), &req)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.Status(http.StatusOK).JSON(resp)
}

// GetReturnsByOrder handles GET /api/v1/orders/:id/returns
func (h *ReturnHandler) GetReturnsByOrder(c *fiber.Ctx) error {
	orderIDStr := c.Params("id")
	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Invalid order ID",
			Message: "Order ID must be a valid UUID",
		})
	}

	// Get user ID from auth context
	userID := uuid.Nil
	if h.authMiddleware != nil {
		if userIDStr := authclient.GetUserID(c); userIDStr != "" {
			userID, _ = uuid.Parse(userIDStr)
		}
	}

	resp, err := h.processReturnUC.GetReturnsByOrderID(c.Context(), orderID, userID)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"returns": resp,
	})
}

// ApproveReturn handles POST /api/v1/returns/:id/approve
func (h *ReturnHandler) ApproveReturn(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Invalid return ID",
			Message: "Return ID must be a valid UUID",
		})
	}

	var req usecase.ApproveReturnRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
		})
	}
	req.ReturnID = id

	// Get approver info from auth context if available
	if h.authMiddleware != nil {
		if userIDStr := authclient.GetUserID(c); userIDStr != "" && req.ApprovedByID == "" {
			req.ApprovedByID = userIDStr
		}
	}

	resp, err := h.processReturnUC.ApproveReturn(c.Context(), &req)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.Status(http.StatusOK).JSON(resp)
}

// RejectReturn handles POST /api/v1/returns/:id/reject
func (h *ReturnHandler) RejectReturn(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Invalid return ID",
			Message: "Return ID must be a valid UUID",
		})
	}

	var req usecase.RejectReturnRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Invalid request body",
			Message: err.Error(),
		})
	}
	req.ReturnID = id

	// Get rejector info from auth context if available
	if h.authMiddleware != nil {
		if userIDStr := authclient.GetUserID(c); userIDStr != "" && req.RejectedBy == "" {
			req.RejectedBy = userIDStr
		}
	}

	resp, err := h.processReturnUC.RejectReturn(c.Context(), &req)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.Status(http.StatusOK).JSON(resp)
}

// CompleteReturn handles POST /api/v1/returns/:id/complete
func (h *ReturnHandler) CompleteReturn(c *fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Invalid return ID",
			Message: "Return ID must be a valid UUID",
		})
	}

	req := usecase.CompleteReturnRequest{
		ReturnID: id,
	}

	resp, err := h.processReturnUC.CompleteReturn(c.Context(), &req)
	if err != nil {
		return h.handleError(c, err)
	}

	return c.Status(http.StatusOK).JSON(resp)
}

// handleError maps domain errors to HTTP responses
func (h *ReturnHandler) handleError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, domain.ErrReturnNotFound):
		return c.Status(http.StatusNotFound).JSON(ErrorResponse{
			Error:   "Return not found",
			Message: err.Error(),
		})

	case errors.Is(err, domain.ErrOrderNotFound):
		return c.Status(http.StatusNotFound).JSON(ErrorResponse{
			Error:   "Order not found",
			Message: err.Error(),
		})

	case errors.Is(err, domain.ErrUnauthorized):
		return c.Status(http.StatusForbidden).JSON(ErrorResponse{
			Error:   "Unauthorized",
			Message: "You don't have permission to access this resource",
		})

	case errors.Is(err, domain.ErrOrderCannotBeReturned):
		return c.Status(http.StatusConflict).JSON(ErrorResponse{
			Error:   "Cannot create return",
			Message: err.Error(),
		})

	case errors.Is(err, domain.ErrInvalidReturnStatus):
		return c.Status(http.StatusConflict).JSON(ErrorResponse{
			Error:   "Invalid return status",
			Message: err.Error(),
		})

	case errors.Is(err, domain.ErrInvalidRefundQuantity):
		return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
			Error:   "Invalid quantity",
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
