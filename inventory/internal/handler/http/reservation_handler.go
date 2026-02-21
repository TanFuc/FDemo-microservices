package http

import (
	"errors"

	"microservices/inventory/internal/model"
	"microservices/inventory/internal/service"
	"microservices/inventory/pkg/response"

	"github.com/gofiber/fiber/v2"
)

type ReservationHandler struct {
	reservationService service.ReservationService
}

func NewReservationHandler(reservationService service.ReservationService) *ReservationHandler {
	return &ReservationHandler{reservationService: reservationService}
}

// ReserveStock reserves stock for an order
// @Summary Reserve stock
// @Description Reserve stock items for an order
// @Tags Reservations
// @Accept json
// @Produce json
// @Param request body model.ReserveStockRequest true "Reserve stock request"
// @Success 200 {object} response.Response{data=model.ReserveStockResponse}
// @Failure 400 {object} response.Response
// @Failure 409 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/reservations/reserve [post]
func (h *ReservationHandler) ReserveStock(c *fiber.Ctx) error {
	var req model.ReserveStockRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}

	reservations, err := h.reservationService.ReserveStock(c.Context(), &req)
	if err != nil {
		if errors.Is(err, model.ErrInvalidOrderID) || errors.Is(err, model.ErrInvalidSkuID) || errors.Is(err, model.ErrInvalidQuantity) {
			return response.BadRequest(c, err.Error())
		}
		if errors.Is(err, model.ErrInsufficientStock) {
			return response.Conflict(c, err.Error())
		}
		return response.InternalError(c, err)
	}

	resps := make([]*model.ReservationResponse, len(reservations))
	for i := range reservations {
		resps[i] = reservations[i].ToResponse()
	}

	return response.Success(c, &model.ReserveStockResponse{
		Success:      true,
		Message:      "stock reserved successfully",
		Reservations: resps,
	})
}

// ConfirmStock confirms a stock reservation
// @Summary Confirm stock
// @Description Confirm stock reservation (deducts from total stock)
// @Tags Reservations
// @Accept json
// @Produce json
// @Param request body model.OrderRequest true "Confirm stock request"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 409 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/reservations/confirm [post]
func (h *ReservationHandler) ConfirmStock(c *fiber.Ctx) error {
	var req model.OrderRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}

	err := h.reservationService.ConfirmStock(c.Context(), req.OrderID)
	if err != nil {
		if errors.Is(err, model.ErrInvalidOrderID) {
			return response.BadRequest(c, err.Error())
		}
		if errors.Is(err, model.ErrReservationNotFound) {
			return response.NotFound(c, err.Error())
		}
		if errors.Is(err, model.ErrAlreadyCancelled) {
			return response.Conflict(c, err.Error())
		}
		return response.InternalError(c, err)
	}

	return response.SuccessWithMessage(c, "stock confirmed successfully", nil)
}

// ReleaseStock releases a stock reservation
// @Summary Release stock
// @Description Release stock reservation (cancels without deducting)
// @Tags Reservations
// @Accept json
// @Produce json
// @Param request body model.OrderRequest true "Release stock request"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/reservations/release [post]
func (h *ReservationHandler) ReleaseStock(c *fiber.Ctx) error {
	var req model.OrderRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "invalid request body")
	}

	err := h.reservationService.ReleaseStock(c.Context(), req.OrderID)
	if err != nil {
		if errors.Is(err, model.ErrInvalidOrderID) {
			return response.BadRequest(c, err.Error())
		}
		if errors.Is(err, model.ErrReservationNotFound) {
			return response.NotFound(c, err.Error())
		}
		return response.InternalError(c, err)
	}

	return response.SuccessWithMessage(c, "stock released successfully", nil)
}

// GetReservation retrieves reservations for an order
// @Summary Get reservation
// @Description Get stock reservations by order ID
// @Tags Reservations
// @Accept json
// @Produce json
// @Param order_id path string true "Order ID"
// @Success 200 {object} response.Response{data=[]model.ReservationResponse}
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/reservations/{order_id} [get]
func (h *ReservationHandler) GetReservation(c *fiber.Ctx) error {
	orderID := c.Params("order_id")
	if orderID == "" {
		return response.BadRequest(c, "order_id is required")
	}

	reservations, err := h.reservationService.GetReservation(c.Context(), orderID)
	if err != nil {
		if errors.Is(err, model.ErrReservationNotFound) {
			return response.NotFound(c, err.Error())
		}
		return response.InternalError(c, err)
	}

	resps := make([]*model.ReservationResponse, len(reservations))
	for i := range reservations {
		resps[i] = reservations[i].ToResponse()
	}

	return response.Success(c, resps)
}

// ListReservations lists reservations with filtering
// @Summary List reservations
// @Description List stock reservations with filtering and pagination
// @Tags Reservations
// @Accept json
// @Produce json
// @Param status query string false "Filter by status (PENDING, CONFIRMED, CANCELLED)"
// @Param order_id query string false "Filter by order ID"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {object} response.PaginatedResponse{data=[]model.ReservationResponse}
// @Failure 500 {object} response.Response
// @Router /api/v1/reservations [get]
func (h *ReservationHandler) ListReservations(c *fiber.Ctx) error {
	filter := &model.ReservationFilter{
		Status:  c.Query("status"),
		OrderID: c.Query("order_id"),
		Page:    c.QueryInt("page", 1),
		Limit:   c.QueryInt("limit", 20),
	}

	reservations, total, err := h.reservationService.ListReservations(c.Context(), filter)
	if err != nil {
		return response.InternalError(c, err)
	}

	resps := make([]*model.ReservationResponse, len(reservations))
	for i := range reservations {
		resps[i] = reservations[i].ToResponse()
	}

	return response.Paginated(c, resps, total, filter.Page, filter.Limit)
}
