package repository

import (
	"context"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"microservices/catalog/internal/model"
)

// ProductRepository defines the interface for product data access
type ProductRepository interface {
	// Create creates a new product
	Create(ctx context.Context, product *model.Product) error

	// GetByID retrieves a product by ID
	GetByID(ctx context.Context, id primitive.ObjectID) (*model.Product, error)

	// GetBySlug retrieves a product by slug
	GetBySlug(ctx context.Context, slug string) (*model.Product, error)

	// Update updates an existing product
	Update(ctx context.Context, product *model.Product) error

	// UpdateMetadata updates only the metadata field
	UpdateMetadata(ctx context.Context, id primitive.ObjectID, metadata map[string]interface{}) error

	// Delete removes a product by ID
	Delete(ctx context.Context, id primitive.ObjectID) error

	// List retrieves products with filtering and pagination
	List(ctx context.Context, filter *model.ProductFilter) ([]model.Product, int64, error)

	// ExistsByID checks if a product exists
	ExistsByID(ctx context.Context, id primitive.ObjectID) (bool, error)

	// ExistsBySlug checks if a product with the given slug exists
	ExistsBySlug(ctx context.Context, slug string) (bool, error)
}

// CategoryRepository defines the interface for category data access
type CategoryRepository interface {
	// Create creates a new category
	Create(ctx context.Context, category *model.Category) error

	// GetByID retrieves a category by ID
	GetByID(ctx context.Context, id primitive.ObjectID) (*model.Category, error)

	// GetBySlug retrieves a category by slug
	GetBySlug(ctx context.Context, slug string) (*model.Category, error)

	// Update updates an existing category
	Update(ctx context.Context, category *model.Category) error

	// Delete removes a category by ID
	Delete(ctx context.Context, id primitive.ObjectID) error

	// List retrieves all categories
	List(ctx context.Context) ([]model.Category, error)

	// ExistsByID checks if a category exists
	ExistsByID(ctx context.Context, id primitive.ObjectID) (bool, error)

	// ExistsBySlug checks if a category with the given slug exists
	ExistsBySlug(ctx context.Context, slug string) (bool, error)
}

// BrandRepository defines the interface for brand data access
type BrandRepository interface {
	// Create creates a new brand
	Create(ctx context.Context, brand *model.Brand) error

	// GetByID retrieves a brand by ID
	GetByID(ctx context.Context, id primitive.ObjectID) (*model.Brand, error)

	// GetBySlug retrieves a brand by slug
	GetBySlug(ctx context.Context, slug string) (*model.Brand, error)

	// Update updates an existing brand
	Update(ctx context.Context, brand *model.Brand) error

	// Delete removes a brand by ID
	Delete(ctx context.Context, id primitive.ObjectID) error

	// List retrieves all brands
	List(ctx context.Context) ([]model.Brand, error)

	// ExistsByID checks if a brand exists
	ExistsByID(ctx context.Context, id primitive.ObjectID) (bool, error)

	// ExistsBySlug checks if a brand with the given slug exists
	ExistsBySlug(ctx context.Context, slug string) (bool, error)
}

// CacheRepository defines the interface for caching operations
type CacheRepository interface {
	// Product cache operations
	GetProduct(ctx context.Context, slug string) (*model.Product, error)
	SetProduct(ctx context.Context, product *model.Product) error
	DeleteProduct(ctx context.Context, slug string) error

	// Category cache operations
	GetCategory(ctx context.Context, slug string) (*model.Category, error)
	SetCategory(ctx context.Context, category *model.Category) error
	DeleteCategory(ctx context.Context, slug string) error

	// Brand cache operations
	GetBrand(ctx context.Context, slug string) (*model.Brand, error)
	SetBrand(ctx context.Context, brand *model.Brand) error
	DeleteBrand(ctx context.Context, slug string) error
}

// EventPublisher defines the interface for publishing events
type EventPublisher interface {
	// PublishProductCreated publishes a product created event
	PublishProductCreated(ctx context.Context, product *model.Product) error

	// PublishProductUpdated publishes a product updated event
	PublishProductUpdated(ctx context.Context, product *model.Product) error

	// PublishProductDeleted publishes a product deleted event
	PublishProductDeleted(ctx context.Context, productID string) error

	// Close closes the publisher connection
	Close() error
}
