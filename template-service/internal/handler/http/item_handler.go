package http

import (
	"errors"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"

	"microservices/template-service/internal/model"
	"microservices/template-service/internal/service"
	"microservices/template-service/internal/service/impl"
	"microservices/template-service/pkg/response"
)

type ItemHandler struct {
	itemService service.ItemService
	validate    *validator.Validate
}

func NewItemHandler(itemService service.ItemService) *ItemHandler {
	return &ItemHandler{
		itemService: itemService,
		validate:    validator.New(),
	}
}

// CreateItem godoc
// @Summary Create a new item
// @Tags items
// @Accept json
// @Produce json
// @Param request body model.CreateItemRequest true "Create Item Request"
// @Success 201 {object} response.Response{data=model.ItemResponse}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /items [post]
func (h *ItemHandler) CreateItem(c *fiber.Ctx) error {
	var req model.CreateItemRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if err := h.validate.Struct(&req); err != nil {
		return response.ValidationError(c, formatValidationErrors(err))
	}

	item, err := h.itemService.CreateItem(c.Context(), &req)
	if err != nil {
		return response.InternalError(c, "Failed to create item")
	}

	return response.Created(c, item)
}

// GetItem godoc
// @Summary Get an item by ID
// @Tags items
// @Produce json
// @Param id path string true "Item ID"
// @Success 200 {object} response.Response{data=model.ItemResponse}
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /items/{id} [get]
func (h *ItemHandler) GetItem(c *fiber.Ctx) error {
	id := c.Params("id")

	item, err := h.itemService.GetItem(c.Context(), id)
	if err != nil {
		if errors.Is(err, impl.ErrItemNotFound) {
			return response.NotFound(c, "Item not found")
		}
		if errors.Is(err, impl.ErrInvalidID) {
			return response.BadRequest(c, "Invalid item ID")
		}
		return response.InternalError(c, "Failed to get item")
	}

	return response.Success(c, item)
}

// UpdateItem godoc
// @Summary Update an item
// @Tags items
// @Accept json
// @Produce json
// @Param id path string true "Item ID"
// @Param request body model.UpdateItemRequest true "Update Item Request"
// @Success 200 {object} response.Response{data=model.ItemResponse}
// @Failure 400 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /items/{id} [put]
func (h *ItemHandler) UpdateItem(c *fiber.Ctx) error {
	id := c.Params("id")

	var req model.UpdateItemRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if err := h.validate.Struct(&req); err != nil {
		return response.ValidationError(c, formatValidationErrors(err))
	}

	item, err := h.itemService.UpdateItem(c.Context(), id, &req)
	if err != nil {
		if errors.Is(err, impl.ErrItemNotFound) {
			return response.NotFound(c, "Item not found")
		}
		if errors.Is(err, impl.ErrInvalidID) {
			return response.BadRequest(c, "Invalid item ID")
		}
		return response.InternalError(c, "Failed to update item")
	}

	return response.Success(c, item)
}

// DeleteItem godoc
// @Summary Delete an item
// @Tags items
// @Produce json
// @Param id path string true "Item ID"
// @Success 200 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /items/{id} [delete]
func (h *ItemHandler) DeleteItem(c *fiber.Ctx) error {
	id := c.Params("id")

	err := h.itemService.DeleteItem(c.Context(), id)
	if err != nil {
		if errors.Is(err, impl.ErrItemNotFound) {
			return response.NotFound(c, "Item not found")
		}
		if errors.Is(err, impl.ErrInvalidID) {
			return response.BadRequest(c, "Invalid item ID")
		}
		return response.InternalError(c, "Failed to delete item")
	}

	return response.SuccessWithMessage(c, nil, "Item deleted successfully")
}

// ListItems godoc
// @Summary List items with pagination
// @Tags items
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Success 200 {object} response.Response{data=model.ListItemsResponse}
// @Failure 500 {object} response.Response
// @Router /items [get]
func (h *ItemHandler) ListItems(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	result, err := h.itemService.ListItems(c.Context(), page, limit)
	if err != nil {
		return response.InternalError(c, "Failed to list items")
	}

	return response.Success(c, result)
}

// HealthCheck godoc
// @Summary Health check endpoint
// @Tags health
// @Produce json
// @Success 200 {object} map[string]string
// @Router /health [get]
func (h *ItemHandler) HealthCheck(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status": "ok",
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
	case "min":
		return e.Field() + " must be at least " + e.Param() + " characters"
	case "max":
		return e.Field() + " must be at most " + e.Param() + " characters"
	default:
		return e.Field() + " is invalid"
	}
}
