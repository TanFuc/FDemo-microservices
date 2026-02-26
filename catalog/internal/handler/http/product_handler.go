package http

import (
	"errors"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"

	"microservices/catalog/internal/middleware"
	"microservices/catalog/internal/model"
	"microservices/catalog/internal/service"
	"microservices/catalog/internal/service/impl"
	"microservices/catalog/pkg/response"
)

type ProductHandler struct {
	productService service.ProductService
	validate       *validator.Validate
}

func NewProductHandler(productService service.ProductService) *ProductHandler {
	return &ProductHandler{
		productService: productService,
		validate:       validator.New(),
	}
}

// CreateProduct godoc
// @Summary Create a new product
// @Tags products
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body model.CreateProductRequest true "Create Product Request"
// @Success 201 {object} response.Response{data=model.ProductResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /products [post]
func (h *ProductHandler) CreateProduct(c *fiber.Ctx) error {
	var req model.CreateProductRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	// Set created by from authenticated user
	userID := middleware.GetUserID(c)
	if userID != "" {
		req.CreatedBy = userID
	}

	if err := h.validate.Struct(&req); err != nil {
		return response.ValidationError(c, formatValidationErrors(err))
	}

	product, err := h.productService.CreateProduct(c.Context(), &req)
	if err != nil {
		if errors.Is(err, impl.ErrProductExists) {
			return response.Conflict(c, "Product with this name already exists")
		}
		if errors.Is(err, impl.ErrCategoryInvalid) {
			return response.BadRequest(c, "Invalid category")
		}
		if errors.Is(err, impl.ErrBrandInvalid) {
			return response.BadRequest(c, "Invalid brand")
		}
		return response.InternalError(c, "Failed to create product")
	}

	return response.Created(c, product)
}

// GetProduct godoc
// @Summary Get a product by ID
// @Tags products
// @Produce json
// @Param id path string true "Product ID"
// @Success 200 {object} response.Response{data=model.ProductResponse}
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /products/{id} [get]
func (h *ProductHandler) GetProduct(c *fiber.Ctx) error {
	id := c.Params("id")

	product, err := h.productService.GetProduct(c.Context(), id)
	if err != nil {
		if errors.Is(err, impl.ErrProductNotFound) {
			return response.NotFound(c, "Product not found")
		}
		if errors.Is(err, impl.ErrInvalidID) {
			return response.BadRequest(c, "Invalid product ID")
		}
		return response.InternalError(c, "Failed to get product")
	}

	return response.Success(c, product)
}

// GetProductBySlug godoc
// @Summary Get a product by slug
// @Tags products
// @Produce json
// @Param slug path string true "Product Slug"
// @Success 200 {object} response.Response{data=model.ProductResponse}
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /products/slug/{slug} [get]
func (h *ProductHandler) GetProductBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")

	product, err := h.productService.GetProductBySlug(c.Context(), slug)
	if err != nil {
		if errors.Is(err, impl.ErrProductNotFound) {
			return response.NotFound(c, "Product not found")
		}
		return response.InternalError(c, "Failed to get product")
	}

	return response.Success(c, product)
}

// UpdateProduct godoc
// @Summary Update a product
// @Tags products
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Product ID"
// @Param request body model.UpdateProductRequest true "Update Product Request"
// @Success 200 {object} response.Response{data=model.ProductResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /products/{id} [put]
func (h *ProductHandler) UpdateProduct(c *fiber.Ctx) error {
	id := c.Params("id")

	var req model.UpdateProductRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	// Set updated by from authenticated user
	userID := middleware.GetUserID(c)
	if userID != "" {
		req.UpdatedBy = userID
	}

	product, err := h.productService.UpdateProduct(c.Context(), id, &req)
	if err != nil {
		if errors.Is(err, impl.ErrProductNotFound) {
			return response.NotFound(c, "Product not found")
		}
		if errors.Is(err, impl.ErrInvalidID) {
			return response.BadRequest(c, "Invalid product ID")
		}
		if errors.Is(err, impl.ErrCategoryInvalid) {
			return response.BadRequest(c, "Invalid category")
		}
		if errors.Is(err, impl.ErrBrandInvalid) {
			return response.BadRequest(c, "Invalid brand")
		}
		return response.InternalError(c, "Failed to update product")
	}

	return response.Success(c, product)
}

// UpdateProductMetadata godoc
// @Summary Update product metadata (PATCH)
// @Tags products
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Product ID"
// @Param request body map[string]interface{} true "Metadata"
// @Success 200 {object} response.Response{data=model.ProductResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /products/{id}/metadata [patch]
func (h *ProductHandler) UpdateProductMetadata(c *fiber.Ctx) error {
	id := c.Params("id")

	var metadata map[string]interface{}
	if err := c.BodyParser(&metadata); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	product, err := h.productService.UpdateProductMetadata(c.Context(), id, metadata)
	if err != nil {
		if errors.Is(err, impl.ErrProductNotFound) {
			return response.NotFound(c, "Product not found")
		}
		if errors.Is(err, impl.ErrInvalidID) {
			return response.BadRequest(c, "Invalid product ID")
		}
		return response.InternalError(c, "Failed to update product metadata")
	}

	return response.Success(c, product)
}

// DeleteProduct godoc
// @Summary Delete a product
// @Tags products
// @Produce json
// @Security BearerAuth
// @Param id path string true "Product ID"
// @Success 200 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /products/{id} [delete]
func (h *ProductHandler) DeleteProduct(c *fiber.Ctx) error {
	id := c.Params("id")

	err := h.productService.DeleteProduct(c.Context(), id)
	if err != nil {
		if errors.Is(err, impl.ErrProductNotFound) {
			return response.NotFound(c, "Product not found")
		}
		if errors.Is(err, impl.ErrInvalidID) {
			return response.BadRequest(c, "Invalid product ID")
		}
		return response.InternalError(c, "Failed to delete product")
	}

	return response.SuccessWithMessage(c, nil, "Product deleted successfully")
}

// ListProducts godoc
// @Summary List products with pagination
// @Tags products
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Param category_id query string false "Filter by category ID"
// @Param brand_id query string false "Filter by brand ID"
// @Param status query string false "Filter by status"
// @Param shop_id query string false "Filter by shop ID"
// @Success 200 {object} response.Response{data=model.ListProductsResponse}
// @Failure 500 {object} response.Response
// @Router /products [get]
func (h *ProductHandler) ListProducts(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	filter := &model.ProductFilter{
		CategoryID: c.Query("category_id"),
		BrandID:    c.Query("brand_id"),
		Status:     c.Query("status"),
		ShopID:     c.Query("shop_id"),
		Page:       page,
		Limit:      limit,
	}

	result, err := h.productService.ListProducts(c.Context(), filter)
	if err != nil {
		return response.InternalError(c, "Failed to list products")
	}

	return response.Success(c, result)
}

// GetProductSeller returns seller information for a product (internal endpoint)
// @Summary Get product seller info
// @Tags internal
// @Produce json
// @Param productId path string true "Product ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} response.Response
// @Router /internal/products/{productId}/seller [get]
func (h *ProductHandler) GetProductSeller(c *fiber.Ctx) error {
	productID := c.Params("productId")

	product, err := h.productService.GetProduct(c.Context(), productID)
	if err != nil {
		if errors.Is(err, impl.ErrProductNotFound) {
			return response.NotFound(c, "Product not found")
		}
		if errors.Is(err, impl.ErrInvalidID) {
			return response.BadRequest(c, "Invalid product ID")
		}
		return response.InternalError(c, "Failed to get product")
	}

	// Return seller info from product
	sellerInfo := fiber.Map{
		"productId":   productID,
		"productName": product.Name,
		"shopId":      product.ShopID,
		"sellerId":    product.ShopID, // ShopID is typically the seller's user ID
		"shopName":    product.ShopName,
	}

	return c.JSON(sellerInfo)
}

// UpdateProductRating updates the product rating (internal endpoint for rating sync)
// @Summary Update product rating
// @Tags internal
// @Accept json
// @Produce json
// @Param productId path string true "Product ID"
// @Param request body map[string]interface{} true "Rating data"
// @Success 200 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /internal/products/{productId}/rating [put]
func (h *ProductHandler) UpdateProductRating(c *fiber.Ctx) error {
	productID := c.Params("productId")

	var req struct {
		AverageRating float64 `json:"averageRating"`
		TotalReviews  int     `json:"totalReviews"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	// Update the product's rating fields
	metadata := map[string]interface{}{
		"rating":      req.AverageRating,
		"reviewCount": req.TotalReviews,
	}

	_, err := h.productService.UpdateProductMetadata(c.Context(), productID, metadata)
	if err != nil {
		if errors.Is(err, impl.ErrProductNotFound) {
			return response.NotFound(c, "Product not found")
		}
		if errors.Is(err, impl.ErrInvalidID) {
			return response.BadRequest(c, "Invalid product ID")
		}
		return response.InternalError(c, "Failed to update product rating")
	}

	return response.SuccessWithMessage(c, nil, "Product rating updated successfully")
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
