package http

import (
	"microservices/catalog/internal/domain"
	"microservices/catalog/internal/usecase"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CategoryHandler struct {
	usecase *usecase.CategoryUsecase
}

func NewCategoryHandler(uc *usecase.CategoryUsecase) *CategoryHandler {
	return &CategoryHandler{usecase: uc}
}

func (h *CategoryHandler) RegisterRoutes(app fiber.Router) {
	categories := app.Group("/categories")
	categories.Post("/", h.Create)
	categories.Get("/", h.GetAll)
	categories.Get("/:id", h.GetByID)
	categories.Get("/slug/:slug", h.GetBySlug)
	categories.Put("/:id", h.Update)
	categories.Delete("/:id", h.Delete)
}

type CreateCategoryRequest struct {
	Name                 string                       `json:"name"`
	ImageURL             string                       `json:"imageUrl"`
	AttributeDefinitions []domain.AttributeDefinition `json:"attributeDefinitions"`
	ParentID             string                       `json:"parentId,omitempty"`
}

func (h *CategoryHandler) Create(c *fiber.Ctx) error {
	var req CreateCategoryRequest
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

	dto := usecase.CreateCategoryDTO{
		Name:                 req.Name,
		ImageURL:             req.ImageURL,
		AttributeDefinitions: req.AttributeDefinitions,
	}

	if req.ParentID != "" {
		parentID, err := primitive.ObjectIDFromHex(req.ParentID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid parent ID",
			})
		}
		dto.ParentID = &parentID
	}

	category, err := h.usecase.Create(c.Context(), dto)
	if err != nil {
		if err == usecase.ErrCategoryExists {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(category)
}

func (h *CategoryHandler) GetAll(c *fiber.Ctx) error {
	categories, err := h.usecase.GetAll(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(categories)
}

func (h *CategoryHandler) GetByID(c *fiber.Ctx) error {
	id, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid ID",
		})
	}

	category, err := h.usecase.GetByID(c.Context(), id)
	if err != nil {
		if err == usecase.ErrCategoryNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(category)
}

func (h *CategoryHandler) GetBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Slug is required",
		})
	}

	category, err := h.usecase.GetBySlug(c.Context(), slug)
	if err != nil {
		if err == usecase.ErrCategoryNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(category)
}

type UpdateCategoryRequest struct {
	Name                 string                       `json:"name"`
	ImageURL             string                       `json:"imageUrl"`
	AttributeDefinitions []domain.AttributeDefinition `json:"attributeDefinitions"`
	ParentID             string                       `json:"parentId,omitempty"`
}

func (h *CategoryHandler) Update(c *fiber.Ctx) error {
	id, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid ID",
		})
	}

	var req UpdateCategoryRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	dto := usecase.UpdateCategoryDTO{
		Name:                 req.Name,
		ImageURL:             req.ImageURL,
		AttributeDefinitions: req.AttributeDefinitions,
	}

	if req.ParentID != "" {
		parentID, err := primitive.ObjectIDFromHex(req.ParentID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid parent ID",
			})
		}
		dto.ParentID = &parentID
	}

	category, err := h.usecase.Update(c.Context(), id, dto)
	if err != nil {
		if err == usecase.ErrCategoryNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(category)
}

func (h *CategoryHandler) Delete(c *fiber.Ctx) error {
	id, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid ID",
		})
	}

	if err := h.usecase.Delete(c.Context(), id); err != nil {
		if err == usecase.ErrCategoryNotFound {
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
