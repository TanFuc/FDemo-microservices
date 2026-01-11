package handler

import (
	"log/slog"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/tafu/search-service/internal/domain"
	"github.com/tafu/search-service/internal/usecase"
)

// SearchHandler handles search API requests
type SearchHandler struct {
	searchUsecase *usecase.SearchUsecase
	logger        *slog.Logger
}

// NewSearchHandler creates a new search handler
func NewSearchHandler(searchUsecase *usecase.SearchUsecase, logger *slog.Logger) *SearchHandler {
	return &SearchHandler{
		searchUsecase: searchUsecase,
		logger:        logger,
	}
}

// SearchRequest represents the search API request body
type SearchRequest struct {
	Keyword    string                 `json:"keyword"`
	CategoryID string                 `json:"categoryId"`
	BrandID    string                 `json:"brandId"`
	PriceMin   *float64               `json:"priceMin"`
	PriceMax   *float64               `json:"priceMax"`
	Specs      map[string]interface{} `json:"specs"`
	Metadata   map[string]interface{} `json:"metadata"`
	SortBy     string                 `json:"sortBy"`
	SortOrder  string                 `json:"sortOrder"`
	Page       int                    `json:"page"`
	Limit      int                    `json:"limit"`
}

// SearchResponse represents the search API response
type SearchResponse struct {
	Success bool                `json:"success"`
	Data    *domain.SearchResult `json:"data,omitempty"`
	Error   string              `json:"error,omitempty"`
}

// Search handles POST /api/search
func (h *SearchHandler) Search(c *fiber.Ctx) error {
	var req SearchRequest
	if err := c.BodyParser(&req); err != nil {
		h.logger.Warn("invalid request body", "error", err)
		return c.Status(fiber.StatusBadRequest).JSON(SearchResponse{
			Success: false,
			Error:   "Invalid request body",
		})
	}

	params := &domain.SearchParams{
		Keyword:    req.Keyword,
		CategoryID: req.CategoryID,
		BrandID:    req.BrandID,
		PriceMin:   req.PriceMin,
		PriceMax:   req.PriceMax,
		Specs:      req.Specs,
		Metadata:   req.Metadata,
		SortBy:     req.SortBy,
		SortOrder:  req.SortOrder,
		Page:       req.Page,
		Limit:      req.Limit,
	}

	result, err := h.searchUsecase.SearchProducts(c.Context(), params)
	if err != nil {
		h.logger.Error("search failed", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(SearchResponse{
			Success: false,
			Error:   "Search failed",
		})
	}

	return c.JSON(SearchResponse{
		Success: true,
		Data:    result,
	})
}

// SearchGet handles GET /api/search with query parameters
func (h *SearchHandler) SearchGet(c *fiber.Ctx) error {
	params := &domain.SearchParams{
		Keyword:    c.Query("keyword"),
		CategoryID: c.Query("categoryId"),
		BrandID:    c.Query("brandId"),
		SortBy:     c.Query("sortBy"),
		SortOrder:  c.Query("sortOrder"),
		Page:       parseIntDefault(c.Query("page"), 1),
		Limit:      parseIntDefault(c.Query("limit"), 20),
	}

	// Parse price range
	if priceMin := c.Query("priceMin"); priceMin != "" {
		if val, err := strconv.ParseFloat(priceMin, 64); err == nil {
			params.PriceMin = &val
		}
	}
	if priceMax := c.Query("priceMax"); priceMax != "" {
		if val, err := strconv.ParseFloat(priceMax, 64); err == nil {
			params.PriceMax = &val
		}
	}

	result, err := h.searchUsecase.SearchProducts(c.Context(), params)
	if err != nil {
		h.logger.Error("search failed", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(SearchResponse{
			Success: false,
			Error:   "Search failed",
		})
	}

	return c.JSON(SearchResponse{
		Success: true,
		Data:    result,
	})
}

// Health handles GET /health
func (h *SearchHandler) Health(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status": "healthy",
	})
}

func parseIntDefault(s string, defaultValue int) int {
	if s == "" {
		return defaultValue
	}
	val, err := strconv.Atoi(s)
	if err != nil {
		return defaultValue
	}
	return val
}
