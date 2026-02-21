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
	ErrBrandNotFound = errors.New("brand not found")
	ErrBrandExists   = errors.New("brand with this slug already exists")
)

type brandService struct {
	brandRepo repository.BrandRepository
	cache     repository.CacheRepository
}

// NewBrandService creates a new brand service
func NewBrandService(
	brandRepo repository.BrandRepository,
	cache repository.CacheRepository,
) service.BrandService {
	return &brandService{
		brandRepo: brandRepo,
		cache:     cache,
	}
}

func (s *brandService) CreateBrand(ctx context.Context, req *model.CreateBrandRequest) (*model.BrandResponse, error) {
	slug := generateSlug(req.Name)

	// Check if brand with slug already exists
	exists, err := s.brandRepo.ExistsBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrBrandExists
	}

	now := time.Now()
	brand := &model.Brand{
		Name:        req.Name,
		Slug:        slug,
		Description: req.Description,
		LogoURL:     req.LogoURL,
		BannerURL:   req.BannerURL,
		Website:     req.Website,
		Country:     req.Country,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	brand.EnsureDefaults()

	if err := s.brandRepo.Create(ctx, brand); err != nil {
		return nil, err
	}

	// Cache the brand
	if s.cache != nil {
		_ = s.cache.SetBrand(ctx, brand)
	}

	return brand.ToResponse(), nil
}

func (s *brandService) GetBrand(ctx context.Context, id string) (*model.BrandResponse, error) {
	brandID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrInvalidID
	}

	brand, err := s.brandRepo.GetByID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	if brand == nil {
		return nil, ErrBrandNotFound
	}

	return brand.ToResponse(), nil
}

func (s *brandService) GetBrandBySlug(ctx context.Context, slug string) (*model.BrandResponse, error) {
	// Try cache first
	if s.cache != nil {
		brand, err := s.cache.GetBrand(ctx, slug)
		if err == nil && brand != nil {
			return brand.ToResponse(), nil
		}
	}

	brand, err := s.brandRepo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if brand == nil {
		return nil, ErrBrandNotFound
	}

	// Cache it
	if s.cache != nil {
		_ = s.cache.SetBrand(ctx, brand)
	}

	return brand.ToResponse(), nil
}

func (s *brandService) UpdateBrand(ctx context.Context, id string, req *model.UpdateBrandRequest) (*model.BrandResponse, error) {
	brandID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrInvalidID
	}

	brand, err := s.brandRepo.GetByID(ctx, brandID)
	if err != nil {
		return nil, err
	}
	if brand == nil {
		return nil, ErrBrandNotFound
	}

	oldSlug := brand.Slug

	// Update fields if provided
	if req.Name != "" {
		brand.Name = req.Name
		brand.Slug = generateSlug(req.Name)
	}
	if req.Description != nil {
		brand.Description = req.Description
	}
	if req.LogoURL != "" {
		brand.LogoURL = req.LogoURL
	}
	if req.BannerURL != "" {
		brand.BannerURL = req.BannerURL
	}
	if req.Website != "" {
		brand.Website = req.Website
	}
	if req.Country != "" {
		brand.Country = req.Country
	}
	if req.Status != "" {
		brand.Status = req.Status
	}
	if req.IsVerified != nil {
		brand.IsVerified = *req.IsVerified
	}
	if req.IsFeatured != nil {
		brand.IsFeatured = *req.IsFeatured
	}

	brand.UpdatedAt = time.Now()

	if err := s.brandRepo.Update(ctx, brand); err != nil {
		return nil, err
	}

	// Invalidate old cache and set new
	if s.cache != nil {
		_ = s.cache.DeleteBrand(ctx, oldSlug)
		_ = s.cache.SetBrand(ctx, brand)
	}

	return brand.ToResponse(), nil
}

func (s *brandService) DeleteBrand(ctx context.Context, id string) error {
	brandID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return ErrInvalidID
	}

	brand, err := s.brandRepo.GetByID(ctx, brandID)
	if err != nil {
		return err
	}
	if brand == nil {
		return ErrBrandNotFound
	}

	if err := s.brandRepo.Delete(ctx, brandID); err != nil {
		return err
	}

	// Invalidate cache
	if s.cache != nil {
		_ = s.cache.DeleteBrand(ctx, brand.Slug)
	}

	return nil
}

func (s *brandService) ListBrands(ctx context.Context) (*model.ListBrandsResponse, error) {
	brands, err := s.brandRepo.List(ctx)
	if err != nil {
		return nil, err
	}

	responses := make([]model.BrandResponse, len(brands))
	for i, brand := range brands {
		responses[i] = *brand.ToResponse()
	}

	return &model.ListBrandsResponse{
		Brands: responses,
		Total:  int64(len(responses)),
		Page:   1,
		Limit:  len(responses),
	}, nil
}
