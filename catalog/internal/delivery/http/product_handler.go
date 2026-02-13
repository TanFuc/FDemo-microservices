package http

import (
	"strconv"

	"microservices/catalog/internal/delivery/http/middleware"
	"microservices/catalog/internal/domain"
	"microservices/catalog/internal/infrastructure/grpc"
	"microservices/catalog/internal/repository"
	"microservices/catalog/internal/usecase"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ProductHandler struct {
	usecase    *usecase.ProductUsecase
	authClient *grpc.AuthClient
}

func NewProductHandler(uc *usecase.ProductUsecase, authClient *grpc.AuthClient) *ProductHandler {
	return &ProductHandler{
		usecase:    uc,
		authClient: authClient,
	}
}

func (h *ProductHandler) RegisterRoutes(app fiber.Router, authMiddleware *middleware.AuthMiddleware) {
	products := app.Group("/products")

	// Public routes
	products.Get("/", h.GetAll)
	products.Get("/:id", h.GetByID)
	products.Get("/slug/:slug", h.GetBySlug)

	// Protected routes - require authentication and permission
	products.Post("/",
		authMiddleware.RequireAuth(),
		authMiddleware.RequirePermission("products", "create"),
		h.Create,
	)
	products.Put("/:id",
		authMiddleware.RequireAuth(),
		authMiddleware.RequirePermission("products", "update"),
		h.Update,
	)
	products.Patch("/:id/metadata",
		authMiddleware.RequireAuth(),
		authMiddleware.RequirePermission("products", "update"),
		h.UpdateMetadata,
	)
	products.Delete("/:id",
		authMiddleware.RequireAuth(),
		authMiddleware.RequirePermission("products", "delete"),
		h.Delete,
	)
}

type CreateProductRequest struct {
	Name        string                 `json:"name"`
	CategoryID  string                 `json:"categoryId"`
	BrandID     string                 `json:"brandId"`
	Thumbnail   string                 `json:"thumbnail"`
	Images      []string               `json:"images"`
	VideoURL    string                 `json:"videoUrl"`
	Description string                 `json:"description"`
	Specs       map[string]interface{} `json:"specs"`
	Variations  []domain.Variation     `json:"variations"`
	Metadata    map[string]interface{} `json:"metadata"`
}

func (h *ProductHandler) Create(c *fiber.Ctx) error {
	var req CreateProductRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Name is required",
		})
	}

	categoryID, err := primitive.ObjectIDFromHex(req.CategoryID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid category ID",
		})
	}

	brandID, err := primitive.ObjectIDFromHex(req.BrandID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid brand ID",
		})
	}

	// Get user info from context (set by middleware)
	userID, _ := c.Locals("userId").(string)
	email, _ := c.Locals("email").(string)

	dto := usecase.CreateProductDTO{
		Name:        req.Name,
		CategoryID:  categoryID,
		BrandID:     brandID,
		Thumbnail:   req.Thumbnail,
		Images:      req.Images,
		VideoURL:    req.VideoURL,
		Description: req.Description,
		Specs:       req.Specs,
		Variations:  req.Variations,
		Metadata:    req.Metadata,
		CreatedBy:   userID,
		CreatedByEmail: email,
	}

	product, err := h.usecase.Create(c.Context(), dto)
	if err != nil {
		switch err {
		case usecase.ErrProductExists:
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": err.Error(),
			})
		case usecase.ErrCategoryInvalid, usecase.ErrBrandInvalid, usecase.ErrInvalidSpecs:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    product,
	})
}

func (h *ProductHandler) GetAll(c *fiber.Ctx) error {
	filter := repository.ProductFilter{}

	if categoryID := c.Query("categoryId"); categoryID != "" {
		id, err := primitive.ObjectIDFromHex(categoryID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid category ID",
			})
		}
		filter.CategoryID = &id
	}

	if brandID := c.Query("brandId"); brandID != "" {
		id, err := primitive.ObjectIDFromHex(brandID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid brand ID",
			})
		}
		filter.BrandID = &id
	}

	if status := c.Query("status"); status != "" {
		filter.Status = status
	}

	if limit := c.Query("limit"); limit != "" {
		l, err := strconv.ParseInt(limit, 10, 64)
		if err == nil && l > 0 {
			filter.Limit = l
		}
	}

	if offset := c.Query("offset"); offset != "" {
		o, err := strconv.ParseInt(offset, 10, 64)
		if err == nil && o >= 0 {
			filter.Offset = o
		}
	}

	products, err := h.usecase.GetAll(c.Context(), filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(products)
}

func (h *ProductHandler) GetByID(c *fiber.Ctx) error {
	id, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid ID",
		})
	}

	product, err := h.usecase.GetByID(c.Context(), id)
	if err != nil {
		if err == usecase.ErrProductNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(product)
}

func (h *ProductHandler) GetBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Slug is required",
		})
	}

	product, err := h.usecase.GetBySlug(c.Context(), slug)
	if err != nil {
		if err == usecase.ErrProductNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(product)
}

type UpdateProductRequest struct {
	Name        string                 `json:"name"`
	CategoryID  string                 `json:"categoryId"`
	BrandID     string                 `json:"brandId"`
	Thumbnail   string                 `json:"thumbnail"`
	Images      []string               `json:"images"`
	VideoURL    string                 `json:"videoUrl"`
	Description string                 `json:"description"`
	Specs       map[string]interface{} `json:"specs"`
	Variations  []domain.Variation     `json:"variations"`
	Metadata    map[string]interface{} `json:"metadata"`
	Status      string                 `json:"status"`
}

func (h *ProductHandler) Update(c *fiber.Ctx) error {
	id, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid ID",
		})
	}

	var req UpdateProductRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	categoryID, err := primitive.ObjectIDFromHex(req.CategoryID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid category ID",
		})
	}

	brandID, err := primitive.ObjectIDFromHex(req.BrandID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid brand ID",
		})
	}

	dto := usecase.UpdateProductDTO{
		Name:        req.Name,
		CategoryID:  categoryID,
		BrandID:     brandID,
		Thumbnail:   req.Thumbnail,
		Images:      req.Images,
		VideoURL:    req.VideoURL,
		Description: req.Description,
		Specs:       req.Specs,
		Variations:  req.Variations,
		Metadata:    req.Metadata,
		Status:      req.Status,
	}

	product, err := h.usecase.Update(c.Context(), id, dto)
	if err != nil {
		switch err {
		case usecase.ErrProductNotFound:
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": err.Error(),
			})
		case usecase.ErrCategoryInvalid, usecase.ErrBrandInvalid:
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
	}

	return c.JSON(product)
}

type UpdateMetadataRequest struct {
	Metadata map[string]interface{} `json:"metadata"`
}

func (h *ProductHandler) UpdateMetadata(c *fiber.Ctx) error {
	id, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid ID",
		})
	}

	var req UpdateMetadataRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	product, err := h.usecase.UpdateMetadata(c.Context(), id, req.Metadata)
	if err != nil {
		if err == usecase.ErrProductNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(product)
}

func (h *ProductHandler) Delete(c *fiber.Ctx) error {
	id, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid ID",
		})
	}

	if err := h.usecase.Delete(c.Context(), id); err != nil {
		if err == usecase.ErrProductNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}
