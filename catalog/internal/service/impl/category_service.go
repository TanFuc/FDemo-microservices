package impl

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"microservices/catalog/internal/model"
	"microservices/catalog/internal/repository"
	"microservices/catalog/internal/service"
)

var (
	ErrCategoryNotFound = errors.New("category not found")
	ErrCategoryExists   = errors.New("category with this slug already exists")
)

type categoryService struct {
	categoryRepo repository.CategoryRepository
	cache        repository.CacheRepository
}

// NewCategoryService creates a new category service
func NewCategoryService(
	categoryRepo repository.CategoryRepository,
	cache repository.CacheRepository,
) service.CategoryService {
	return &categoryService{
		categoryRepo: categoryRepo,
		cache:        cache,
	}
}

func (s *categoryService) CreateCategory(ctx context.Context, req *model.CreateCategoryRequest) (*model.CategoryResponse, error) {
	// Get default name for slug generation
	name := ""
	if req.Name != nil {
		if n, ok := req.Name["en"]; ok {
			name = n
		} else {
			for _, v := range req.Name {
				name = v
				break
			}
		}
	}

	slug := generateSlug(name)

	// Check if category with slug already exists
	exists, err := s.categoryRepo.ExistsBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrCategoryExists
	}

	now := time.Now()
	category := &model.Category{
		Name:        req.Name,
		Slug:        slug,
		Description: req.Description,
		ImageURL:    req.ImageURL,
		IconURL:     req.IconURL,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Parse parent ID if provided
	if req.ParentID != "" {
		parentID, err := primitive.ObjectIDFromHex(req.ParentID)
		if err != nil {
			return nil, ErrInvalidID
		}
		parent, err := s.categoryRepo.GetByID(ctx, parentID)
		if err != nil {
			return nil, err
		}
		if parent == nil {
			return nil, ErrCategoryNotFound
		}
		category.ParentID = &parentID
		category.Level = parent.Level + 1
		category.BuildPath(parent.Path)
	} else {
		category.Level = 0
		category.BuildPath("")
	}

	category.EnsureDefaults()

	if err := s.categoryRepo.Create(ctx, category); err != nil {
		return nil, err
	}

	// Cache the category
	if s.cache != nil {
		_ = s.cache.SetCategory(ctx, category)
	}

	return category.ToResponse(), nil
}

func (s *categoryService) GetCategory(ctx context.Context, id string) (*model.CategoryResponse, error) {
	categoryID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrInvalidID
	}

	category, err := s.categoryRepo.GetByID(ctx, categoryID)
	if err != nil {
		return nil, err
	}
	if category == nil {
		return nil, ErrCategoryNotFound
	}

	return category.ToResponse(), nil
}

func (s *categoryService) GetCategoryBySlug(ctx context.Context, slug string) (*model.CategoryResponse, error) {
	// Try cache first
	if s.cache != nil {
		category, err := s.cache.GetCategory(ctx, slug)
		if err == nil && category != nil {
			return category.ToResponse(), nil
		}
	}

	category, err := s.categoryRepo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if category == nil {
		return nil, ErrCategoryNotFound
	}

	// Cache it
	if s.cache != nil {
		_ = s.cache.SetCategory(ctx, category)
	}

	return category.ToResponse(), nil
}

func (s *categoryService) UpdateCategory(ctx context.Context, id string, req *model.UpdateCategoryRequest) (*model.CategoryResponse, error) {
	categoryID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrInvalidID
	}

	category, err := s.categoryRepo.GetByID(ctx, categoryID)
	if err != nil {
		return nil, err
	}
	if category == nil {
		return nil, ErrCategoryNotFound
	}

	oldSlug := category.Slug

	// Update fields if provided
	if req.Name != nil {
		category.Name = req.Name
		// Regenerate slug from the name
		name := ""
		if n, ok := req.Name["en"]; ok {
			name = n
		} else {
			for _, v := range req.Name {
				name = v
				break
			}
		}
		category.Slug = generateSlug(name)
	}
	if req.Description != nil {
		category.Description = req.Description
	}
	if req.ImageURL != "" {
		category.ImageURL = req.ImageURL
	}
	if req.IconURL != "" {
		category.IconURL = req.IconURL
	}
	if req.Status != "" {
		category.Status = req.Status
	}
	if req.ParentID != "" {
		parentID, err := primitive.ObjectIDFromHex(req.ParentID)
		if err != nil {
			return nil, ErrInvalidID
		}
		parent, err := s.categoryRepo.GetByID(ctx, parentID)
		if err != nil {
			return nil, err
		}
		if parent == nil {
			return nil, ErrCategoryNotFound
		}
		category.ParentID = &parentID
		category.Level = parent.Level + 1
		category.BuildPath(parent.Path)
	}

	category.UpdatedAt = time.Now()

	if err := s.categoryRepo.Update(ctx, category); err != nil {
		return nil, err
	}

	// Invalidate old cache and set new
	if s.cache != nil {
		_ = s.cache.DeleteCategory(ctx, oldSlug)
		_ = s.cache.SetCategory(ctx, category)
	}

	return category.ToResponse(), nil
}

func (s *categoryService) DeleteCategory(ctx context.Context, id string) error {
	categoryID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return ErrInvalidID
	}

	category, err := s.categoryRepo.GetByID(ctx, categoryID)
	if err != nil {
		return err
	}
	if category == nil {
		return ErrCategoryNotFound
	}

	if err := s.categoryRepo.Delete(ctx, categoryID); err != nil {
		return err
	}

	// Invalidate cache
	if s.cache != nil {
		_ = s.cache.DeleteCategory(ctx, category.Slug)
	}

	return nil
}

func (s *categoryService) ListCategories(ctx context.Context) (*model.ListCategoriesResponse, error) {
	categories, err := s.categoryRepo.List(ctx)
	if err != nil {
		return nil, err
	}

	responses := make([]model.CategoryResponse, len(categories))
	for i, category := range categories {
		responses[i] = *category.ToResponse()
	}

	return &model.ListCategoriesResponse{
		Categories: responses,
		Total:      int64(len(responses)),
		Page:       1,
		Limit:      len(responses),
	}, nil
}
