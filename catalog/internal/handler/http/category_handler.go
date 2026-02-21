package http

import (
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"

	"microservices/catalog/internal/model"
	"microservices/catalog/internal/service"
	"microservices/catalog/internal/service/impl"
	"microservices/catalog/pkg/response"
)

type CategoryHandler struct {
	categoryService service.CategoryService
	validate        *validator.Validate
}

func NewCategoryHandler(categoryService service.CategoryService) *CategoryHandler {
	return &CategoryHandler{
		categoryService: categoryService,
		validate:        validator.New(),
	}
}

// CreateCategory godoc
// @Summary Create a new category
// @Tags categories
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body model.CreateCategoryRequest true "Create Category Request"
// @Success 201 {object} response.Response{data=model.CategoryResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 409 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /categories [post]
func (h *CategoryHandler) CreateCategory(c *fiber.Ctx) error {
	var req model.CreateCategoryRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if err := h.validate.Struct(&req); err != nil {
		return response.ValidationError(c, formatValidationErrors(err))
	}

	category, err := h.categoryService.CreateCategory(c.Context(), &req)
	if err != nil {
		if errors.Is(err, impl.ErrCategoryExists) {
			return response.Conflict(c, "Category with this name already exists")
		}
		if errors.Is(err, impl.ErrCategoryNotFound) {
			return response.BadRequest(c, "Parent category not found")
		}
		return response.InternalError(c, "Failed to create category")
	}

	return response.Created(c, category)
}

// GetCategory godoc
// @Summary Get a category by ID
// @Tags categories
// @Produce json
// @Param id path string true "Category ID"
// @Success 200 {object} response.Response{data=model.CategoryResponse}
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /categories/{id} [get]
func (h *CategoryHandler) GetCategory(c *fiber.Ctx) error {
	id := c.Params("id")

	category, err := h.categoryService.GetCategory(c.Context(), id)
	if err != nil {
		if errors.Is(err, impl.ErrCategoryNotFound) {
			return response.NotFound(c, "Category not found")
		}
		if errors.Is(err, impl.ErrInvalidID) {
			return response.BadRequest(c, "Invalid category ID")
		}
		return response.InternalError(c, "Failed to get category")
	}

	return response.Success(c, category)
}

// GetCategoryBySlug godoc
// @Summary Get a category by slug
// @Tags categories
// @Produce json
// @Param slug path string true "Category Slug"
// @Success 200 {object} response.Response{data=model.CategoryResponse}
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /categories/slug/{slug} [get]
func (h *CategoryHandler) GetCategoryBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")

	category, err := h.categoryService.GetCategoryBySlug(c.Context(), slug)
	if err != nil {
		if errors.Is(err, impl.ErrCategoryNotFound) {
			return response.NotFound(c, "Category not found")
		}
		return response.InternalError(c, "Failed to get category")
	}

	return response.Success(c, category)
}

// UpdateCategory godoc
// @Summary Update a category
// @Tags categories
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Category ID"
// @Param request body model.UpdateCategoryRequest true "Update Category Request"
// @Success 200 {object} response.Response{data=model.CategoryResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /categories/{id} [put]
func (h *CategoryHandler) UpdateCategory(c *fiber.Ctx) error {
	id := c.Params("id")

	var req model.UpdateCategoryRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	category, err := h.categoryService.UpdateCategory(c.Context(), id, &req)
	if err != nil {
		if errors.Is(err, impl.ErrCategoryNotFound) {
			return response.NotFound(c, "Category not found")
		}
		if errors.Is(err, impl.ErrInvalidID) {
			return response.BadRequest(c, "Invalid category ID")
		}
		return response.InternalError(c, "Failed to update category")
	}

	return response.Success(c, category)
}

// DeleteCategory godoc
// @Summary Delete a category
// @Tags categories
// @Produce json
// @Security BearerAuth
// @Param id path string true "Category ID"
// @Success 200 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /categories/{id} [delete]
func (h *CategoryHandler) DeleteCategory(c *fiber.Ctx) error {
	id := c.Params("id")

	err := h.categoryService.DeleteCategory(c.Context(), id)
	if err != nil {
		if errors.Is(err, impl.ErrCategoryNotFound) {
			return response.NotFound(c, "Category not found")
		}
		if errors.Is(err, impl.ErrInvalidID) {
			return response.BadRequest(c, "Invalid category ID")
		}
		return response.InternalError(c, "Failed to delete category")
	}

	return response.SuccessWithMessage(c, nil, "Category deleted successfully")
}

// ListCategories godoc
// @Summary List all categories
// @Tags categories
// @Produce json
// @Success 200 {object} response.Response{data=model.ListCategoriesResponse}
// @Failure 500 {object} response.Response
// @Router /categories [get]
func (h *CategoryHandler) ListCategories(c *fiber.Ctx) error {
	result, err := h.categoryService.ListCategories(c.Context())
	if err != nil {
		return response.InternalError(c, "Failed to list categories")
	}

	return response.Success(c, result)
}
