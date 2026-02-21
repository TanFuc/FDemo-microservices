package service

import (
	"context"

	"microservices/catalog/internal/model"
)

// ProductService defines the interface for product business logic
type ProductService interface {
	// CreateProduct creates a new product
	CreateProduct(ctx context.Context, req *model.CreateProductRequest) (*model.ProductResponse, error)

	// GetProduct retrieves a product by ID
	GetProduct(ctx context.Context, id string) (*model.ProductResponse, error)

	// GetProductBySlug retrieves a product by slug
	GetProductBySlug(ctx context.Context, slug string) (*model.ProductResponse, error)

	// UpdateProduct updates an existing product
	UpdateProduct(ctx context.Context, id string, req *model.UpdateProductRequest) (*model.ProductResponse, error)

	// UpdateProductMetadata updates only the product metadata
	UpdateProductMetadata(ctx context.Context, id string, metadata map[string]interface{}) (*model.ProductResponse, error)

	// DeleteProduct removes a product by ID
	DeleteProduct(ctx context.Context, id string) error

	// ListProducts retrieves products with filtering and pagination
	ListProducts(ctx context.Context, filter *model.ProductFilter) (*model.ListProductsResponse, error)
}

// CategoryService defines the interface for category business logic
type CategoryService interface {
	// CreateCategory creates a new category
	CreateCategory(ctx context.Context, req *model.CreateCategoryRequest) (*model.CategoryResponse, error)

	// GetCategory retrieves a category by ID
	GetCategory(ctx context.Context, id string) (*model.CategoryResponse, error)

	// GetCategoryBySlug retrieves a category by slug
	GetCategoryBySlug(ctx context.Context, slug string) (*model.CategoryResponse, error)

	// UpdateCategory updates an existing category
	UpdateCategory(ctx context.Context, id string, req *model.UpdateCategoryRequest) (*model.CategoryResponse, error)

	// DeleteCategory removes a category by ID
	DeleteCategory(ctx context.Context, id string) error

	// ListCategories retrieves all categories
	ListCategories(ctx context.Context) (*model.ListCategoriesResponse, error)
}

// BrandService defines the interface for brand business logic
type BrandService interface {
	// CreateBrand creates a new brand
	CreateBrand(ctx context.Context, req *model.CreateBrandRequest) (*model.BrandResponse, error)

	// GetBrand retrieves a brand by ID
	GetBrand(ctx context.Context, id string) (*model.BrandResponse, error)

	// GetBrandBySlug retrieves a brand by slug
	GetBrandBySlug(ctx context.Context, slug string) (*model.BrandResponse, error)

	// UpdateBrand updates an existing brand
	UpdateBrand(ctx context.Context, id string, req *model.UpdateBrandRequest) (*model.BrandResponse, error)

	// DeleteBrand removes a brand by ID
	DeleteBrand(ctx context.Context, id string) error

	// ListBrands retrieves all brands
	ListBrands(ctx context.Context) (*model.ListBrandsResponse, error)
}

// AuthService defines the interface for authentication operations
type AuthService interface {
	// VerifyToken validates an access token and returns user information
	VerifyToken(ctx context.Context, token string) (*AuthUser, error)

	// CheckPermission checks if a user has a specific permission
	CheckPermission(ctx context.Context, userID string, permission string) (bool, error)
}

// AuthUser represents authenticated user information
type AuthUser struct {
	UserID      string
	Email       string
	Role        string
	Permissions []string
}
