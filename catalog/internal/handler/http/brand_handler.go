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

type BrandHandler struct {
	brandService service.BrandService
	validate     *validator.Validate
}

func NewBrandHandler(brandService service.BrandService) *BrandHandler {
	return &BrandHandler{
		brandService: brandService,
		validate:     validator.New(),
	}
}

// CreateBrand godoc
// @Summary Create a new brand
// @Tags brands
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body model.CreateBrandRequest true "Create Brand Request"
// @Success 201 {object} response.Response{data=model.BrandResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 409 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /brands [post]
func (h *BrandHandler) CreateBrand(c *fiber.Ctx) error {
	var req model.CreateBrandRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if err := h.validate.Struct(&req); err != nil {
		return response.ValidationError(c, formatValidationErrors(err))
	}

	brand, err := h.brandService.CreateBrand(c.Context(), &req)
	if err != nil {
		if errors.Is(err, impl.ErrBrandExists) {
			return response.Conflict(c, "Brand with this name already exists")
		}
		return response.InternalError(c, "Failed to create brand")
	}

	return response.Created(c, brand)
}

// GetBrand godoc
// @Summary Get a brand by ID
// @Tags brands
// @Produce json
// @Param id path string true "Brand ID"
// @Success 200 {object} response.Response{data=model.BrandResponse}
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /brands/{id} [get]
func (h *BrandHandler) GetBrand(c *fiber.Ctx) error {
	id := c.Params("id")

	brand, err := h.brandService.GetBrand(c.Context(), id)
	if err != nil {
		if errors.Is(err, impl.ErrBrandNotFound) {
			return response.NotFound(c, "Brand not found")
		}
		if errors.Is(err, impl.ErrInvalidID) {
			return response.BadRequest(c, "Invalid brand ID")
		}
		return response.InternalError(c, "Failed to get brand")
	}

	return response.Success(c, brand)
}

// GetBrandBySlug godoc
// @Summary Get a brand by slug
// @Tags brands
// @Produce json
// @Param slug path string true "Brand Slug"
// @Success 200 {object} response.Response{data=model.BrandResponse}
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /brands/slug/{slug} [get]
func (h *BrandHandler) GetBrandBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")

	brand, err := h.brandService.GetBrandBySlug(c.Context(), slug)
	if err != nil {
		if errors.Is(err, impl.ErrBrandNotFound) {
			return response.NotFound(c, "Brand not found")
		}
		return response.InternalError(c, "Failed to get brand")
	}

	return response.Success(c, brand)
}

// UpdateBrand godoc
// @Summary Update a brand
// @Tags brands
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Brand ID"
// @Param request body model.UpdateBrandRequest true "Update Brand Request"
// @Success 200 {object} response.Response{data=model.BrandResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /brands/{id} [put]
func (h *BrandHandler) UpdateBrand(c *fiber.Ctx) error {
	id := c.Params("id")

	var req model.UpdateBrandRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	brand, err := h.brandService.UpdateBrand(c.Context(), id, &req)
	if err != nil {
		if errors.Is(err, impl.ErrBrandNotFound) {
			return response.NotFound(c, "Brand not found")
		}
		if errors.Is(err, impl.ErrInvalidID) {
			return response.BadRequest(c, "Invalid brand ID")
		}
		return response.InternalError(c, "Failed to update brand")
	}

	return response.Success(c, brand)
}

// DeleteBrand godoc
// @Summary Delete a brand
// @Tags brands
// @Produce json
// @Security BearerAuth
// @Param id path string true "Brand ID"
// @Success 200 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /brands/{id} [delete]
func (h *BrandHandler) DeleteBrand(c *fiber.Ctx) error {
	id := c.Params("id")

	err := h.brandService.DeleteBrand(c.Context(), id)
	if err != nil {
		if errors.Is(err, impl.ErrBrandNotFound) {
			return response.NotFound(c, "Brand not found")
		}
		if errors.Is(err, impl.ErrInvalidID) {
			return response.BadRequest(c, "Invalid brand ID")
		}
		return response.InternalError(c, "Failed to delete brand")
	}

	return response.SuccessWithMessage(c, nil, "Brand deleted successfully")
}

// ListBrands godoc
// @Summary List all brands
// @Tags brands
// @Produce json
// @Success 200 {object} response.Response{data=model.ListBrandsResponse}
// @Failure 500 {object} response.Response
// @Router /brands [get]
func (h *BrandHandler) ListBrands(c *fiber.Ctx) error {
	result, err := h.brandService.ListBrands(c.Context())
	if err != nil {
		return response.InternalError(c, "Failed to list brands")
	}

	return response.Success(c, result)
}
