package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"microservices/order/internal/domain"
)

// ReturnRepository implements domain.ReturnRepository
type ReturnRepository struct {
	db *gorm.DB
}

// NewReturnRepository creates a new return repository
func NewReturnRepository(db *gorm.DB) *ReturnRepository {
	return &ReturnRepository{db: db}
}

// Create creates a new return request
func (r *ReturnRepository) Create(ctx context.Context, req *domain.ReturnRequest) error {
	return r.db.WithContext(ctx).Create(req).Error
}

// GetByID retrieves a return request by its ID
func (r *ReturnRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.ReturnRequest, error) {
	var req domain.ReturnRequest
	err := r.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&req).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrReturnNotFound
		}
		return nil, err
	}
	return &req, nil
}

// GetByOrderID retrieves all return requests for an order
func (r *ReturnRepository) GetByOrderID(ctx context.Context, orderID uuid.UUID) ([]*domain.ReturnRequest, error) {
	var requests []*domain.ReturnRequest
	err := r.db.WithContext(ctx).
		Where("order_id = ? AND deleted_at IS NULL", orderID).
		Order("created_at DESC").
		Find(&requests).Error
	return requests, err
}

// Update updates an existing return request
func (r *ReturnRepository) Update(ctx context.Context, req *domain.ReturnRequest) error {
	return r.db.WithContext(ctx).Save(req).Error
}

// List retrieves return requests with filtering
func (r *ReturnRepository) List(ctx context.Context, filter domain.ReturnFilter) ([]*domain.ReturnRequest, int64, error) {
	var requests []*domain.ReturnRequest
	var total int64

	query := r.db.WithContext(ctx).Model(&domain.ReturnRequest{}).Where("deleted_at IS NULL")

	if filter.UserID != uuid.Nil {
		query = query.Where("user_id = ?", filter.UserID)
	}
	if filter.OrderID != uuid.Nil {
		query = query.Where("order_id = ?", filter.OrderID)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	offset := (filter.Page - 1) * filter.Limit

	err := query.
		Order("created_at DESC").
		Offset(offset).
		Limit(filter.Limit).
		Find(&requests).Error

	return requests, total, err
}

// Ensure ReturnRepository implements domain.ReturnRepository
var _ domain.ReturnRepository = (*ReturnRepository)(nil)
