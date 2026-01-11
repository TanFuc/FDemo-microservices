package usecase

import (
	"context"
	"errors"

	"catalog-service/internal/domain"
	"catalog-service/internal/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	ErrBrandNotFound = errors.New("brand not found")
	ErrBrandExists   = errors.New("brand with this slug already exists")
)

type BrandUsecase struct {
	repo repository.BrandRepository
}

func NewBrandUsecase(repo repository.BrandRepository) *BrandUsecase {
	return &BrandUsecase{repo: repo}
}

type CreateBrandDTO struct {
	Name    string
	LogoURL string
	Status  string
}

func (u *BrandUsecase) Create(ctx context.Context, dto CreateBrandDTO) (*domain.Brand, error) {
	slug := GenerateSlug(dto.Name)

	existing, err := u.repo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrBrandExists
	}

	brand := &domain.Brand{
		Name:    dto.Name,
		Slug:    slug,
		LogoURL: dto.LogoURL,
		Status:  dto.Status,
	}

	if err := u.repo.Create(ctx, brand); err != nil {
		return nil, err
	}

	return brand, nil
}

func (u *BrandUsecase) GetByID(ctx context.Context, id primitive.ObjectID) (*domain.Brand, error) {
	brand, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if brand == nil {
		return nil, ErrBrandNotFound
	}
	return brand, nil
}

func (u *BrandUsecase) GetBySlug(ctx context.Context, slug string) (*domain.Brand, error) {
	brand, err := u.repo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if brand == nil {
		return nil, ErrBrandNotFound
	}
	return brand, nil
}

func (u *BrandUsecase) GetAll(ctx context.Context) ([]*domain.Brand, error) {
	return u.repo.GetAll(ctx)
}

type UpdateBrandDTO struct {
	Name    string
	LogoURL string
	Status  string
}

func (u *BrandUsecase) Update(ctx context.Context, id primitive.ObjectID, dto UpdateBrandDTO) (*domain.Brand, error) {
	brand, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if brand == nil {
		return nil, ErrBrandNotFound
	}

	brand.Name = dto.Name
	brand.Slug = GenerateSlug(dto.Name)
	brand.LogoURL = dto.LogoURL
	brand.Status = dto.Status

	if err := u.repo.Update(ctx, brand); err != nil {
		return nil, err
	}

	return brand, nil
}

func (u *BrandUsecase) Delete(ctx context.Context, id primitive.ObjectID) error {
	brand, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if brand == nil {
		return ErrBrandNotFound
	}

	return u.repo.Delete(ctx, id)
}
