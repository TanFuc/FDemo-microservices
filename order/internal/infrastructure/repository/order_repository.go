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

// DraftTTL is the time-to-live for draft orders before cleanup
const DraftTTL = 24 * time.Hour

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

// GetDraftsByUserID retrieves all draft orders for a user with pagination
func (r *OrderRepository) GetDraftsByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Order, error) {
	db := r.db.GetDB(ctx)

	if limit <= 0 {
		limit = 10
	}

	var orders []*domain.Order
	err := db.Preload("Items").
		Where("user_id = ? AND status = ?", userID, domain.StatusDraft).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&orders).Error

	if err != nil {
		return nil, err
	}

	return orders, nil
}

// GetByIDAndUserID retrieves an order by ID while verifying ownership
func (r *OrderRepository) GetByIDAndUserID(ctx context.Context, orderID, userID uuid.UUID) (*domain.Order, error) {
	db := r.db.GetDB(ctx)

	var order domain.Order
	err := db.Preload("Items").
		Where("id = ? AND user_id = ?", orderID, userID).
		First(&order).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrOrderNotFound
		}
		return nil, err
	}

	return &order, nil
}

// DeleteExpiredDrafts soft-deletes expired DRAFT orders in batches.
func (r *OrderRepository) DeleteExpiredDrafts(ctx context.Context, cutoff time.Time, limit int) (int, error) {
	db := r.db.GetDB(ctx)

	// Fetch IDs first to handle cache invalidation
	var orderIDs []uuid.UUID
	err := db.Model(&domain.Order{}).
		Select("id").
		Where("status = ? AND created_at < ? AND deleted_at IS NULL", domain.StatusDraft, cutoff).
		Limit(limit).
		Pluck("id", &orderIDs).Error
	if err != nil {
		return 0, fmt.Errorf("failed to query expired drafts: %w", err)
	}

	if len(orderIDs) == 0 {
		return 0, nil
	}

	// Soft-delete in a transaction
	err = db.Transaction(func(tx *gorm.DB) error {
		// Delete child items first
		if err := tx.Where("order_id IN ?", orderIDs).
			Delete(&domain.OrderItem{}).Error; err != nil {
			return fmt.Errorf("failed to delete order items: %w", err)
		}
		// Delete parent orders
		if err := tx.Where("id IN ?", orderIDs).
			Delete(&domain.Order{}).Error; err != nil {
			return fmt.Errorf("failed to delete orders: %w", err)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}

	// Invalidate Redis cache for deleted orders
	if r.cache != nil {
		for _, id := range orderIDs {
			_ = r.cache.Delete(ctx, orderCacheKey(id))
		}
	}

	return len(orderIDs), nil
}

// SoftDeleteDraft soft-deletes a single draft order and its items.
func (r *OrderRepository) SoftDeleteDraft(ctx context.Context, orderID uuid.UUID) error {
	db := r.db.GetDB(ctx)

	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("order_id = ?", orderID).Delete(&domain.OrderItem{}).Error; err != nil {
			return err
		}
		return tx.Where("id = ? AND status = ?", orderID, domain.StatusDraft).Delete(&domain.Order{}).Error
	})
	if err != nil {
		return err
	}

	if r.cache != nil {
		_ = r.cache.Delete(ctx, orderCacheKey(orderID))
	}
	return nil
}

// GetDraftCountByUserID counts active draft orders for a user.
func (r *OrderRepository) GetDraftCountByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	db := r.db.GetDB(ctx)
	var count int64
	err := db.Model(&domain.Order{}).
		Where("user_id = ? AND status = ?", userID, domain.StatusDraft).
		Count(&count).Error
	return count, err
}

// Ensure OrderRepository implements domain.OrderRepository
var _ domain.OrderRepository = (*OrderRepository)(nil)
