package http

import (
	"microservices/pkg/authclient"
	"microservices/review/internal/core/domain"
	"microservices/review/internal/core/dto"
	"microservices/review/internal/core/service"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type ReviewHandler struct {
	reviewService *service.ReviewService
	validate      *validator.Validate
}

func NewReviewHandler(reviewService *service.ReviewService) *ReviewHandler {
	return &ReviewHandler{
		reviewService: reviewService,
		validate:      validator.New(),
	}
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

type SuccessResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
}

// CreateReview handles POST /api/reviews
func (h *ReviewHandler) CreateReview(c *fiber.Ctx) error {
	var req dto.CreateReviewRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "INVALID_REQUEST",
			Message: "Invalid request body",
		})
	}

	if err := h.validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "VALIDATION_ERROR",
			Message: err.Error(),
		})
	}

	review, err := h.reviewService.CreateReview(c.Context(), &req)
	if err != nil {
		return h.handleServiceError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(SuccessResponse{
		Success: true,
		Data:    review,
		Message: "Review created successfully",
	})
}

// GetProductReviews handles GET /api/products/:productId/reviews
func (h *ReviewHandler) GetProductReviews(c *fiber.Ctx) error {
	productID := c.Params("productId")
	if productID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "INVALID_REQUEST",
			Message: "Product ID is required",
		})
	}

	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)
	sortBy := c.Query("sortBy", "createdAt")

	req := &dto.GetReviewsRequest{
		ProductID: productID,
		Page:      page,
		Limit:     limit,
		SortBy:    sortBy,
	}

	result, err := h.reviewService.GetProductReviews(c.Context(), req)
	if err != nil {
		return h.handleServiceError(c, err)
	}

	return c.JSON(result)
}

// GetRatingSummary handles GET /api/products/:productId/rating
func (h *ReviewHandler) GetRatingSummary(c *fiber.Ctx) error {
	productID := c.Params("productId")
	if productID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "INVALID_REQUEST",
			Message: "Product ID is required",
		})
	}

	summary, err := h.reviewService.GetRatingSummary(c.Context(), productID)
	if err != nil {
		return h.handleServiceError(c, err)
	}

	return c.JSON(SuccessResponse{
		Success: true,
		Data:    summary,
	})
}

// ReplyReview handles POST /api/reviews/:reviewId/reply
func (h *ReviewHandler) ReplyReview(c *fiber.Ctx) error {
	reviewID := c.Params("reviewId")
	if reviewID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "INVALID_REQUEST",
			Message: "Review ID is required",
		})
	}

	var body struct {
		Content string `json:"content"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "INVALID_REQUEST",
			Message: "Invalid request body",
		})
	}

	req := &dto.ReplyReviewRequest{
		ReviewID: reviewID,
		Content:  body.Content,
	}

	if err := h.validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "VALIDATION_ERROR",
			Message: err.Error(),
		})
	}

	if err := h.reviewService.ReplyReview(c.Context(), req); err != nil {
		return h.handleServiceError(c, err)
	}

	return c.JSON(SuccessResponse{
		Success: true,
		Message: "Reply added successfully",
	})
}

// GetReviewByID handles GET /api/reviews/:reviewId
func (h *ReviewHandler) GetReviewByID(c *fiber.Ctx) error {
	reviewID := c.Params("reviewId")
	if reviewID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "INVALID_REQUEST",
			Message: "Review ID is required",
		})
	}

	review, err := h.reviewService.GetReviewByID(c.Context(), reviewID)
	if err != nil {
		return h.handleServiceError(c, err)
	}

	return c.JSON(SuccessResponse{
		Success: true,
		Data:    review,
	})
}

// GetUserReviews handles GET /api/users/:userId/reviews
func (h *ReviewHandler) GetUserReviews(c *fiber.Ctx) error {
	userID := c.Params("userId")
	if userID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "INVALID_REQUEST",
			Message: "User ID is required",
		})
	}

	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)

	result, err := h.reviewService.GetUserReviews(c.Context(), userID, page, limit)
	if err != nil {
		return h.handleServiceError(c, err)
	}

	return c.JSON(result)
}

// UpdateReviewStatus handles PATCH /api/reviews/:reviewId/status
func (h *ReviewHandler) UpdateReviewStatus(c *fiber.Ctx) error {
	reviewID := c.Params("reviewId")
	if reviewID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "INVALID_REQUEST",
			Message: "Review ID is required",
		})
	}

	var body struct {
		Status string `json:"status"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "INVALID_REQUEST",
			Message: "Invalid request body",
		})
	}

	status := domain.ReviewStatus(body.Status)
	if status != domain.ReviewStatusVisible && status != domain.ReviewStatusHidden {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "VALIDATION_ERROR",
			Message: "Status must be VISIBLE or HIDDEN",
		})
	}

	if err := h.reviewService.UpdateReviewStatus(c.Context(), reviewID, status); err != nil {
		return h.handleServiceError(c, err)
	}

	return c.JSON(SuccessResponse{
		Success: true,
		Message: "Review status updated successfully",
	})
}

func (h *ReviewHandler) handleServiceError(c *fiber.Ctx, err error) error {
	switch err {
	case service.ErrInvalidRating:
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "INVALID_RATING",
			Message: err.Error(),
		})
	case service.ErrEmptyContent:
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Error:   "EMPTY_CONTENT",
			Message: err.Error(),
		})
	case service.ErrDuplicateReview:
		return c.Status(fiber.StatusConflict).JSON(ErrorResponse{
			Error:   "DUPLICATE_REVIEW",
			Message: err.Error(),
		})
	case service.ErrVerificationFailed:
		return c.Status(fiber.StatusServiceUnavailable).JSON(ErrorResponse{
			Error:   "VERIFICATION_FAILED",
			Message: "Unable to verify purchase. Please try again later.",
		})
	case service.ErrReviewNotFound:
		return c.Status(fiber.StatusNotFound).JSON(ErrorResponse{
			Error:   "NOT_FOUND",
			Message: err.Error(),
		})
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Error:   "INTERNAL_ERROR",
			Message: "An unexpected error occurred",
		})
	}
}
