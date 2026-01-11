package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"microservices/order/internal/domain"
	"microservices/order/internal/infrastructure/database"
	"gorm.io/gorm"
)

// OrderRepository implements domain.OrderRepository
type OrderRepository struct {
	db *database.Database
}

// NewOrderRepository creates a new OrderRepository
func NewOrderRepository(db *database.Database) *OrderRepository {
	return &OrderRepository{db: db}
}

// Create creates a new order with its items in a single transaction
func (r *OrderRepository) Create(ctx context.Context, order *domain.Order) error {
	db := r.db.GetDB(ctx)

	// Create order and items in a single transaction
	return db.Transaction(func(tx *gorm.DB) error {
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
}

// GetByID retrieves an order by its ID with items
func (r *OrderRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Order, error) {
	db := r.db.GetDB(ctx)

	var order domain.Order
	err := db.Preload("Items").First(&order, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrOrderNotFound
		}
		return nil, err
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
	return db.Save(order).Error
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

	return nil
}

// Ensure OrderRepository implements domain.OrderRepository
var _ domain.OrderRepository = (*OrderRepository)(nil)
