package repository

import (
	"context"

	"microservices/catalog/internal/domain"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CategoryRepository defines the interface for category data access
type CategoryRepository interface {
	Create(ctx context.Context, category *domain.Category) error
	GetByID(ctx context.Context, id primitive.ObjectID) (*domain.Category, error)
	GetBySlug(ctx context.Context, slug string) (*domain.Category, error)
	GetAll(ctx context.Context) ([]*domain.Category, error)
	Update(ctx context.Context, category *domain.Category) error
	Delete(ctx context.Context, id primitive.ObjectID) error
}

// BrandRepository defines the interface for brand data access
type BrandRepository interface {
	Create(ctx context.Context, brand *domain.Brand) error
	GetByID(ctx context.Context, id primitive.ObjectID) (*domain.Brand, error)
	GetBySlug(ctx context.Context, slug string) (*domain.Brand, error)
	GetAll(ctx context.Context) ([]*domain.Brand, error)
	Update(ctx context.Context, brand *domain.Brand) error
	Delete(ctx context.Context, id primitive.ObjectID) error
}

// ProductRepository defines the interface for product data access
type ProductRepository interface {
	Create(ctx context.Context, product *domain.Product) error
	GetByID(ctx context.Context, id primitive.ObjectID) (*domain.Product, error)
	GetBySlug(ctx context.Context, slug string) (*domain.Product, error)
	GetAll(ctx context.Context, filter ProductFilter) ([]*domain.Product, error)
	Update(ctx context.Context, product *domain.Product) error
	UpdateMetadata(ctx context.Context, id primitive.ObjectID, metadata map[string]interface{}) error
	Delete(ctx context.Context, id primitive.ObjectID) error
}

// ProductFilter for querying products
type ProductFilter struct {
	CategoryID *primitive.ObjectID
	BrandID    *primitive.ObjectID
	Status     string
	Limit      int64
	Offset     int64
}

// CacheRepository defines the interface for cache operations
type CacheRepository interface {
	GetCategory(ctx context.Context, slug string) (*domain.Category, error)
	SetCategory(ctx context.Context, category *domain.Category) error
	DeleteCategory(ctx context.Context, slug string) error
	GetProduct(ctx context.Context, slug string) (*domain.Product, error)
	SetProduct(ctx context.Context, product *domain.Product) error
	DeleteProduct(ctx context.Context, slug string) error
}

// EventPublisher defines the interface for publishing events
type EventPublisher interface {
	PublishProductCreated(ctx context.Context, product *domain.Product) error
	PublishProductUpdated(ctx context.Context, product *domain.Product) error
	Close() error
}
