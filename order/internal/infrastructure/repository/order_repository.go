package repository

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"microservices/order/internal/domain"
	"microservices/order/internal/infrastructure/database"
	"microservices/pkg/cache"
)

const (
	orderCacheTTL     = 5 * time.Minute
	orderListCacheTTL = 10 * time.Minute
)

// OrderRepository implements domain.OrderRepository
type OrderRepository struct {
	db    *database.Database
	cache cache.Cache
}

// NewOrderRepository creates a new OrderRepository
func NewOrderRepository(db *database.Database, cacheClient cache.Cache) *OrderRepository {
	return &OrderRepository{
		db:    db,
		cache: cacheClient,
	}
}

// orderCacheKey generates a cache key for an order
func orderCacheKey(id uuid.UUID) string {
	return fmt.Sprintf("order:%s", id.String())
}

// userOrdersListKey generates a cache key for user orders list
func userOrdersListKey(userID uuid.UUID) string {
	return fmt.Sprintf("user:%s:orders", userID.String())
}

// Create creates a new order with its items in a single transaction
func (r *OrderRepository) Create(ctx context.Context, order *domain.Order) error {
	db := r.db.GetDB(ctx)

	// Create order and items in a single transaction
	err := db.Transaction(func(tx *gorm.DB) error {
		// Create the order first
		if err := tx.Create(order).Error; err != nil {
			return err
		}

		// Create order items
		for i := range order.Items {
			order.Items[i].OrderID = order.ID
			if err := tx.Create(&order.Items[i]).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	// Cache the newly created order
	if r.cache != nil {
		cacheKey := orderCacheKey(order.ID)
		if cacheErr := r.cache.Set(ctx, cacheKey, order, orderCacheTTL); cacheErr != nil {
			log.Printf("Warning: Failed to cache order %s: %v", order.ID, cacheErr)
		}

		// Invalidate user's order list cache
		listKey := userOrdersListKey(order.UserID)
		if cacheErr := r.cache.Delete(ctx, listKey); cacheErr != nil {
			log.Printf("Warning: Failed to invalidate user orders list cache: %v", cacheErr)
		}
	}

	return nil
}

// GetByID retrieves an order by its ID with items
func (r *OrderRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Order, error) {
	// Try cache first
	if r.cache != nil {
		cacheKey := orderCacheKey(id)
		var order domain.Order
		if err := r.cache.Get(ctx, cacheKey, &order); err == nil {
			log.Printf("Cache hit for order %s", id)
			return &order, nil
		}
		// Cache miss, continue to database
	}

	db := r.db.GetDB(ctx)

	var order domain.Order
	err := db.Preload("Items").First(&order, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrOrderNotFound
		}
		return nil, err
	}

	// Store in cache
	if r.cache != nil {
		cacheKey := orderCacheKey(id)
		if cacheErr := r.cache.Set(ctx, cacheKey, &order, orderCacheTTL); cacheErr != nil {
			log.Printf("Warning: Failed to cache order %s: %v", id, cacheErr)
		}
	}

	return &order, nil
}

// GetByUserID retrieves all orders for a user with pagination
func (r *OrderRepository) GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Order, error) {
	db := r.db.GetDB(ctx)

	var orders []*domain.Order
	err := db.Preload("Items").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&orders).Error

	if err != nil {
		return nil, err
	}

	return orders, nil
}

// Update updates an existing order
func (r *OrderRepository) Update(ctx context.Context, order *domain.Order) error {
	db := r.db.GetDB(ctx)
	if err := db.Save(order).Error; err != nil {
		return err
	}

	// Update cache
	if r.cache != nil {
		cacheKey := orderCacheKey(order.ID)
		if cacheErr := r.cache.Set(ctx, cacheKey, order, orderCacheTTL); cacheErr != nil {
			log.Printf("Warning: Failed to update cache for order %s: %v", order.ID, cacheErr)
		}

		// Invalidate user's order list cache
		listKey := userOrdersListKey(order.UserID)
		if cacheErr := r.cache.Delete(ctx, listKey); cacheErr != nil {
			log.Printf("Warning: Failed to invalidate user orders list cache: %v", cacheErr)
		}
	}

	return nil
}

// UpdateStatus updates only the order status
func (r *OrderRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.OrderStatus) error {
	db := r.db.GetDB(ctx)

	result := db.Model(&domain.Order{}).
		Where("id = ?", id).
		Update("status", status)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return domain.ErrOrderNotFound
	}

	// Invalidate cache for this order
	if r.cache != nil {
		cacheKey := orderCacheKey(id)
		if cacheErr := r.cache.Delete(ctx, cacheKey); cacheErr != nil {
			log.Printf("Warning: Failed to invalidate cache for order %s: %v", id, cacheErr)
		}
	}

	return nil
}

// Ensure OrderRepository implements domain.OrderRepository
var _ domain.OrderRepository = (*OrderRepository)(nil)
