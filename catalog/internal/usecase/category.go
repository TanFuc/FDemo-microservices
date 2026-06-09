//go:build legacy

package usecase

import (
	"context"
	"errors"

	"microservices/catalog/internal/domain"
	"microservices/catalog/internal/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	ErrCategoryNotFound = errors.New("category not found")
	ErrCategoryExists   = errors.New("category with this slug already exists")
)

type CategoryUsecase struct {
	repo  repository.CategoryRepository
	cache repository.CacheRepository
}

func NewCategoryUsecase(repo repository.CategoryRepository, cache repository.CacheRepository) *CategoryUsecase {
	return &CategoryUsecase{
		repo:  repo,
		cache: cache,
	}
}

type CreateCategoryDTO struct {
	Name                 string
	ImageURL             string
	AttributeDefinitions []domain.AttributeDefinition
	ParentID             *primitive.ObjectID
}

func (u *CategoryUsecase) Create(ctx context.Context, dto CreateCategoryDTO) (*domain.Category, error) {
	slug := GenerateSlug(dto.Name)

	existing, err := u.repo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrCategoryExists
	}

	category := &domain.Category{
		Name:                 dto.Name,
		Slug:                 slug,
		ImageURL:             dto.ImageURL,
		AttributeDefinitions: dto.AttributeDefinitions,
		ParentID:             dto.ParentID,
	}

	if err := u.repo.Create(ctx, category); err != nil {
		return nil, err
	}

	// Cache the category
	if u.cache != nil {
		_ = u.cache.SetCategory(ctx, category)
	}

	return category, nil
}

func (u *CategoryUsecase) GetByID(ctx context.Context, id primitive.ObjectID) (*domain.Category, error) {
	category, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if category == nil {
		return nil, ErrCategoryNotFound
	}
	return category, nil
}

func (u *CategoryUsecase) GetBySlug(ctx context.Context, slug string) (*domain.Category, error) {
	// Try cache first
	if u.cache != nil {
		category, err := u.cache.GetCategory(ctx, slug)
		if err == nil && category != nil {
			return category, nil
		}
	}

	category, err := u.repo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if category == nil {
		return nil, ErrCategoryNotFound
	}

	// Cache it
	if u.cache != nil {
		_ = u.cache.SetCategory(ctx, category)
	}

	return category, nil
}

func (u *CategoryUsecase) GetAll(ctx context.Context) ([]*domain.Category, error) {
	return u.repo.GetAll(ctx)
}

type UpdateCategoryDTO struct {
	Name                 string
	ImageURL             string
	AttributeDefinitions []domain.AttributeDefinition
	ParentID             *primitive.ObjectID
}

func (u *CategoryUsecase) Update(ctx context.Context, id primitive.ObjectID, dto UpdateCategoryDTO) (*domain.Category, error) {
	category, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if category == nil {
		return nil, ErrCategoryNotFound
	}

	oldSlug := category.Slug

	category.Name = dto.Name
	category.Slug = GenerateSlug(dto.Name)
	category.ImageURL = dto.ImageURL
	category.AttributeDefinitions = dto.AttributeDefinitions
	category.ParentID = dto.ParentID

	if err := u.repo.Update(ctx, category); err != nil {
		return nil, err
	}

	// Invalidate old cache and set new
	if u.cache != nil {
		_ = u.cache.DeleteCategory(ctx, oldSlug)
		_ = u.cache.SetCategory(ctx, category)
	}

	return category, nil
}

func (u *CategoryUsecase) Delete(ctx context.Context, id primitive.ObjectID) error {
	category, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if category == nil {
		return ErrCategoryNotFound
	}

	if err := u.repo.Delete(ctx, id); err != nil {
		return err
	}

	// Invalidate cache
	if u.cache != nil {
		_ = u.cache.DeleteCategory(ctx, category.Slug)
	}

	return nil
}
