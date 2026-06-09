//go:build legacy

package usecase

import (
	"context"
	"errors"
	"time"

	"microservices/catalog/internal/domain"
	"microservices/catalog/internal/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	ErrProductNotFound = errors.New("product not found")
	ErrProductExists   = errors.New("product with this slug already exists")
	ErrCategoryInvalid = errors.New("category not found")
	ErrBrandInvalid    = errors.New("brand not found")
	ErrInvalidSpecs    = errors.New("product specs do not match category attribute definitions")
)

type ProductUsecase struct {
	productRepo  repository.ProductRepository
	categoryRepo repository.CategoryRepository
	brandRepo    repository.BrandRepository
	cache        repository.CacheRepository
	publisher    repository.EventPublisher
}

func NewProductUsecase(
	productRepo repository.ProductRepository,
	categoryRepo repository.CategoryRepository,
	brandRepo repository.BrandRepository,
	cache repository.CacheRepository,
	publisher repository.EventPublisher,
) *ProductUsecase {
	return &ProductUsecase{
		productRepo:  productRepo,
		categoryRepo: categoryRepo,
		brandRepo:    brandRepo,
		cache:        cache,
		publisher:    publisher,
	}
}

type CreateProductDTO struct {
	Name        string
	CategoryID  primitive.ObjectID
	BrandID     primitive.ObjectID
	Thumbnail   string
	Images      []string
	VideoURL    string
	Description string
	Specs       map[string]interface{}
	Variations  []domain.Variation
	Metadata    map[string]interface{}
}

func (u *ProductUsecase) Create(ctx context.Context, dto CreateProductDTO) (*domain.Product, error) {
	// Validate category exists
	category, err := u.categoryRepo.GetByID(ctx, dto.CategoryID)
	if err != nil {
		return nil, err
	}
	if category == nil {
		return nil, ErrCategoryInvalid
	}

	// Validate brand exists
	brand, err := u.brandRepo.GetByID(ctx, dto.BrandID)
	if err != nil {
		return nil, err
	}
	if brand == nil {
		return nil, ErrBrandInvalid
	}

	// Validate specs against category attribute definitions
	if err := u.validateSpecs(dto.Specs, category.AttributeDefinitions); err != nil {
		return nil, err
	}

	// Generate slug
	slug := GenerateSlug(dto.Name)

	// Check if product with slug already exists
	existing, err := u.productRepo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrProductExists
	}

	now := time.Now()
	product := &domain.Product{
		Name:        dto.Name,
		Slug:        slug,
		CategoryID:  dto.CategoryID,
		BrandID:     dto.BrandID,
		Thumbnail:   dto.Thumbnail,
		Images:      dto.Images,
		VideoURL:    dto.VideoURL,
		Description: dto.Description,
		Specs:       dto.Specs,
		Variations:  dto.Variations,
		Metadata:    dto.Metadata,
		Status:      domain.ProductStatusDraft,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	product.EnsureDefaults()

	if err := u.productRepo.Create(ctx, product); err != nil {
		return nil, err
	}

	// Cache the product
	if u.cache != nil {
		_ = u.cache.SetProduct(ctx, product)
	}

	// Publish event
	if u.publisher != nil {
		_ = u.publisher.PublishProductCreated(ctx, product)
	}

	return product, nil
}

func (u *ProductUsecase) validateSpecs(specs map[string]interface{}, definitions []domain.AttributeDefinition) error {
	if specs == nil {
		specs = make(map[string]interface{})
	}

	for _, def := range definitions {
		if def.IsRequired {
			if _, exists := specs[def.Key]; !exists {
				return ErrInvalidSpecs
			}
		}
	}

	return nil
}

func (u *ProductUsecase) GetByID(ctx context.Context, id primitive.ObjectID) (*domain.Product, error) {
	product, err := u.productRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if product == nil {
		return nil, ErrProductNotFound
	}
	return product, nil
}

func (u *ProductUsecase) GetBySlug(ctx context.Context, slug string) (*domain.Product, error) {
	// Try cache first
	if u.cache != nil {
		product, err := u.cache.GetProduct(ctx, slug)
		if err == nil && product != nil {
			return product, nil
		}
	}

	product, err := u.productRepo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if product == nil {
		return nil, ErrProductNotFound
	}

	// Cache it
	if u.cache != nil {
		_ = u.cache.SetProduct(ctx, product)
	}

	return product, nil
}

func (u *ProductUsecase) GetAll(ctx context.Context, filter repository.ProductFilter) ([]*domain.Product, error) {
	return u.productRepo.GetAll(ctx, filter)
}

type UpdateProductDTO struct {
	Name        string
	CategoryID  primitive.ObjectID
	BrandID     primitive.ObjectID
	Thumbnail   string
	Images      []string
	VideoURL    string
	Description string
	Specs       map[string]interface{}
	Variations  []domain.Variation
	Metadata    map[string]interface{}
	Status      string
}

func (u *ProductUsecase) Update(ctx context.Context, id primitive.ObjectID, dto UpdateProductDTO) (*domain.Product, error) {
	product, err := u.productRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if product == nil {
		return nil, ErrProductNotFound
	}

	// Validate category if changed
	if dto.CategoryID != product.CategoryID {
		category, err := u.categoryRepo.GetByID(ctx, dto.CategoryID)
		if err != nil {
			return nil, err
		}
		if category == nil {
			return nil, ErrCategoryInvalid
		}
	}

	// Validate brand if changed
	if dto.BrandID != product.BrandID {
		brand, err := u.brandRepo.GetByID(ctx, dto.BrandID)
		if err != nil {
			return nil, err
		}
		if brand == nil {
			return nil, ErrBrandInvalid
		}
	}

	oldSlug := product.Slug

	product.Name = dto.Name
	product.Slug = GenerateSlug(dto.Name)
	product.CategoryID = dto.CategoryID
	product.BrandID = dto.BrandID
	product.Thumbnail = dto.Thumbnail
	product.Images = dto.Images
	product.VideoURL = dto.VideoURL
	product.Description = dto.Description
	product.Specs = dto.Specs
	product.Variations = dto.Variations
	product.Metadata = dto.Metadata
	product.Status = dto.Status
	product.UpdatedAt = time.Now()

	product.EnsureDefaults()

	if err := u.productRepo.Update(ctx, product); err != nil {
		return nil, err
	}

	// Invalidate old cache and set new
	if u.cache != nil {
		_ = u.cache.DeleteProduct(ctx, oldSlug)
		_ = u.cache.SetProduct(ctx, product)
	}

	// Publish event
	if u.publisher != nil {
		_ = u.publisher.PublishProductUpdated(ctx, product)
	}

	return product, nil
}

// UpdateMetadata updates only the metadata field of a product (PATCH operation)
func (u *ProductUsecase) UpdateMetadata(ctx context.Context, id primitive.ObjectID, metadata map[string]interface{}) (*domain.Product, error) {
	product, err := u.productRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if product == nil {
		return nil, ErrProductNotFound
	}

	// Initialize metadata if nil
	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	if err := u.productRepo.UpdateMetadata(ctx, id, metadata); err != nil {
		return nil, err
	}

	// Update product in memory for return and cache
	product.Metadata = metadata
	product.UpdatedAt = time.Now()

	// Update cache
	if u.cache != nil {
		_ = u.cache.SetProduct(ctx, product)
	}

	// Publish event
	if u.publisher != nil {
		_ = u.publisher.PublishProductUpdated(ctx, product)
	}

	return product, nil
}

func (u *ProductUsecase) Delete(ctx context.Context, id primitive.ObjectID) error {
	product, err := u.productRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if product == nil {
		return ErrProductNotFound
	}

	if err := u.productRepo.Delete(ctx, id); err != nil {
		return err
	}

	// Invalidate cache
	if u.cache != nil {
		_ = u.cache.DeleteProduct(ctx, product.Slug)
	}

	return nil
}
