package http

import (
	"microservices/catalog/internal/usecase"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type BrandHandler struct {
	usecase *usecase.BrandUsecase
}

func NewBrandHandler(uc *usecase.BrandUsecase) *BrandHandler {
	return &BrandHandler{usecase: uc}
}

func (h *BrandHandler) RegisterRoutes(app fiber.Router) {
	brands := app.Group("/brands")
	brands.Post("/", h.Create)
	brands.Get("/", h.GetAll)
	brands.Get("/:id", h.GetByID)
	brands.Get("/slug/:slug", h.GetBySlug)
	brands.Put("/:id", h.Update)
	brands.Delete("/:id", h.Delete)
}

type CreateBrandRequest struct {
	Name    string `json:"name"`
	LogoURL string `json:"logoUrl"`
	Status  string `json:"status"`
}

func (h *BrandHandler) Create(c *fiber.Ctx) error {
	var req CreateBrandRequest
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

	dto := usecase.CreateBrandDTO{
		Name:    req.Name,
		LogoURL: req.LogoURL,
		Status:  req.Status,
	}

	brand, err := h.usecase.Create(c.Context(), dto)
	if err != nil {
		if err == usecase.ErrBrandExists {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(brand)
}

func (h *BrandHandler) GetAll(c *fiber.Ctx) error {
	brands, err := h.usecase.GetAll(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(brands)
}

func (h *BrandHandler) GetByID(c *fiber.Ctx) error {
	id, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid ID",
		})
	}

	brand, err := h.usecase.GetByID(c.Context(), id)
	if err != nil {
		if err == usecase.ErrBrandNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(brand)
}

func (h *BrandHandler) GetBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Slug is required",
		})
	}

	brand, err := h.usecase.GetBySlug(c.Context(), slug)
	if err != nil {
		if err == usecase.ErrBrandNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(brand)
}

type UpdateBrandRequest struct {
	Name    string `json:"name"`
	LogoURL string `json:"logoUrl"`
	Status  string `json:"status"`
}

func (h *BrandHandler) Update(c *fiber.Ctx) error {
	id, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid ID",
		})
	}

	var req UpdateBrandRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	dto := usecase.UpdateBrandDTO{
		Name:    req.Name,
		LogoURL: req.LogoURL,
		Status:  req.Status,
	}

	brand, err := h.usecase.Update(c.Context(), id, dto)
	if err != nil {
		if err == usecase.ErrBrandNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(brand)
}

func (h *BrandHandler) Delete(c *fiber.Ctx) error {
	id, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid ID",
		})
	}

	if err := h.usecase.Delete(c.Context(), id); err != nil {
		if err == usecase.ErrBrandNotFound {
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
