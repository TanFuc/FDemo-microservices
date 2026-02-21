package impl

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"microservices/catalog/internal/model"
	"microservices/catalog/internal/repository"
	"microservices/catalog/internal/service"
)

var (
	ErrProductNotFound = errors.New("product not found")
	ErrProductExists   = errors.New("product with this slug already exists")
	ErrCategoryInvalid = errors.New("category not found")
	ErrBrandInvalid    = errors.New("brand not found")
	ErrInvalidID       = errors.New("invalid ID format")
	ErrInvalidSpecs    = errors.New("product specs do not match category attribute definitions")
)

type productService struct {
	productRepo  repository.ProductRepository
	categoryRepo repository.CategoryRepository
	brandRepo    repository.BrandRepository
	cache        repository.CacheRepository
	publisher    repository.EventPublisher
}

// NewProductService creates a new product service
func NewProductService(
	productRepo repository.ProductRepository,
	categoryRepo repository.CategoryRepository,
	brandRepo repository.BrandRepository,
	cache repository.CacheRepository,
	publisher repository.EventPublisher,
) service.ProductService {
	return &productService{
		productRepo:  productRepo,
		categoryRepo: categoryRepo,
		brandRepo:    brandRepo,
		cache:        cache,
		publisher:    publisher,
	}
}

func (s *productService) CreateProduct(ctx context.Context, req *model.CreateProductRequest) (*model.ProductResponse, error) {
	// Parse category ID
	categoryID, err := primitive.ObjectIDFromHex(req.CategoryID)
	if err != nil {
		return nil, ErrInvalidID
	}

	// Validate category exists
	category, err := s.categoryRepo.GetByID(ctx, categoryID)
	if err != nil {
		return nil, err
	}
	if category == nil {
		return nil, ErrCategoryInvalid
	}

	// Parse brand ID if provided
	var brandID primitive.ObjectID
	var brandName string
	if req.BrandID != "" {
		brandID, err = primitive.ObjectIDFromHex(req.BrandID)
		if err != nil {
			return nil, ErrInvalidID
		}
		brand, err := s.brandRepo.GetByID(ctx, brandID)
		if err != nil {
			return nil, err
		}
		if brand == nil {
			return nil, ErrBrandInvalid
		}
		brandName = brand.Name
	}

	// Generate slug
	slug := generateSlug(req.Name)

	// Check if product with slug already exists
	exists, err := s.productRepo.ExistsBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrProductExists
	}

	// Parse base price
	basePrice, err := decimal.NewFromString(req.BasePrice)
	if err != nil {
		basePrice = decimal.Zero
	}

	now := time.Now()
	product := &model.Product{
		ShopID:           req.ShopID,
		ShopName:         req.ShopName,
		Name:             req.Name,
		Slug:             slug,
		Description:      req.Description,
		ShortDescription: req.ShortDescription,
		CategoryID:       categoryID,
		CategoryPath:     category.Path,
		BrandID:          brandID,
		BrandName:        brandName,
		Thumbnail:        req.Thumbnail,
		Images:           req.Images,
		BasePrice:        basePrice,
		Currency:         req.Currency,
		Variations:       req.Variations,
		Attributes:       req.Attributes,
		Status:           req.Status,
		Visibility:       req.Visibility,
		CreatedBy:        req.CreatedBy,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	product.EnsureDefaults()
	product.CalculatePriceRange()
	product.CalculateTotalStock()

	if err := s.productRepo.Create(ctx, product); err != nil {
		return nil, err
	}

	// Cache the product
	if s.cache != nil {
		_ = s.cache.SetProduct(ctx, product)
	}

	// Publish event
	if s.publisher != nil {
		_ = s.publisher.PublishProductCreated(ctx, product)
	}

	return product.ToResponse(), nil
}

func (s *productService) GetProduct(ctx context.Context, id string) (*model.ProductResponse, error) {
	productID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrInvalidID
	}

	product, err := s.productRepo.GetByID(ctx, productID)
	if err != nil {
		return nil, err
	}
	if product == nil {
		return nil, ErrProductNotFound
	}

	return product.ToResponse(), nil
}

func (s *productService) GetProductBySlug(ctx context.Context, slug string) (*model.ProductResponse, error) {
	// Try cache first
	if s.cache != nil {
		product, err := s.cache.GetProduct(ctx, slug)
		if err == nil && product != nil {
			return product.ToResponse(), nil
		}
	}

	product, err := s.productRepo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if product == nil {
		return nil, ErrProductNotFound
	}

	// Cache it
	if s.cache != nil {
		_ = s.cache.SetProduct(ctx, product)
	}

	return product.ToResponse(), nil
}

func (s *productService) UpdateProduct(ctx context.Context, id string, req *model.UpdateProductRequest) (*model.ProductResponse, error) {
	productID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrInvalidID
	}

	product, err := s.productRepo.GetByID(ctx, productID)
	if err != nil {
		return nil, err
	}
	if product == nil {
		return nil, ErrProductNotFound
	}

	oldSlug := product.Slug

	// Update fields if provided
	if req.Name != "" {
		product.Name = req.Name
		product.Slug = generateSlug(req.Name)
	}
	if req.Description != "" {
		product.Description = req.Description
	}
	if req.ShortDescription != "" {
		product.ShortDescription = req.ShortDescription
	}
	if req.CategoryID != "" {
		categoryID, err := primitive.ObjectIDFromHex(req.CategoryID)
		if err != nil {
			return nil, ErrInvalidID
		}
		category, err := s.categoryRepo.GetByID(ctx, categoryID)
		if err != nil {
			return nil, err
		}
		if category == nil {
			return nil, ErrCategoryInvalid
		}
		product.CategoryID = categoryID
		product.CategoryPath = category.Path
	}
	if req.BrandID != "" {
		brandID, err := primitive.ObjectIDFromHex(req.BrandID)
		if err != nil {
			return nil, ErrInvalidID
		}
		brand, err := s.brandRepo.GetByID(ctx, brandID)
		if err != nil {
			return nil, err
		}
		if brand == nil {
			return nil, ErrBrandInvalid
		}
		product.BrandID = brandID
		product.BrandName = brand.Name
	}
	if req.Thumbnail != "" {
		product.Thumbnail = req.Thumbnail
	}
	if req.Images != nil {
		product.Images = req.Images
	}
	if req.BasePrice != "" {
		basePrice, err := decimal.NewFromString(req.BasePrice)
		if err == nil {
			product.BasePrice = basePrice
		}
	}
	if req.Variations != nil {
		product.Variations = req.Variations
	}
	if req.Attributes != nil {
		product.Attributes = req.Attributes
	}
	if req.Status != "" {
		product.Status = req.Status
	}
	if req.Visibility != "" {
		product.Visibility = req.Visibility
	}
	if req.UpdatedBy != "" {
		product.UpdatedBy = req.UpdatedBy
	}

	product.UpdatedAt = time.Now()
	product.EnsureDefaults()
	product.CalculatePriceRange()
	product.CalculateTotalStock()

	if err := s.productRepo.Update(ctx, product); err != nil {
		return nil, err
	}

	// Invalidate old cache and set new
	if s.cache != nil {
		_ = s.cache.DeleteProduct(ctx, oldSlug)
		_ = s.cache.SetProduct(ctx, product)
	}

	// Publish event
	if s.publisher != nil {
		_ = s.publisher.PublishProductUpdated(ctx, product)
	}

	return product.ToResponse(), nil
}

func (s *productService) UpdateProductMetadata(ctx context.Context, id string, metadata map[string]interface{}) (*model.ProductResponse, error) {
	productID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrInvalidID
	}

	product, err := s.productRepo.GetByID(ctx, productID)
	if err != nil {
		return nil, err
	}
	if product == nil {
		return nil, ErrProductNotFound
	}

	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	if err := s.productRepo.UpdateMetadata(ctx, productID, metadata); err != nil {
		return nil, err
	}

	product.Metadata = metadata
	product.UpdatedAt = time.Now()

	// Update cache
	if s.cache != nil {
		_ = s.cache.SetProduct(ctx, product)
	}

	// Publish event
	if s.publisher != nil {
		_ = s.publisher.PublishProductUpdated(ctx, product)
	}

	return product.ToResponse(), nil
}

func (s *productService) DeleteProduct(ctx context.Context, id string) error {
	productID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return ErrInvalidID
	}

	product, err := s.productRepo.GetByID(ctx, productID)
	if err != nil {
		return err
	}
	if product == nil {
		return ErrProductNotFound
	}

	if err := s.productRepo.Delete(ctx, productID); err != nil {
		return err
	}

	// Invalidate cache
	if s.cache != nil {
		_ = s.cache.DeleteProduct(ctx, product.Slug)
	}

	// Publish delete event
	if s.publisher != nil {
		_ = s.publisher.PublishProductDeleted(ctx, id)
	}

	return nil
}

func (s *productService) ListProducts(ctx context.Context, filter *model.ProductFilter) (*model.ListProductsResponse, error) {
	if filter == nil {
		filter = &model.ProductFilter{}
	}
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 || filter.Limit > 100 {
		filter.Limit = 10
	}

	products, total, err := s.productRepo.List(ctx, filter)
	if err != nil {
		return nil, err
	}

	responses := make([]model.ProductResponse, len(products))
	for i, product := range products {
		responses[i] = *product.ToResponse()
	}

	totalPages := int(total) / filter.Limit
	if int(total)%filter.Limit > 0 {
		totalPages++
	}

	return &model.ListProductsResponse{
		Products:   responses,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}

// generateSlug generates a URL-friendly slug from text
func generateSlug(text string) string {
	slug := strings.ToLower(text)
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	slug = reg.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	return slug
}
