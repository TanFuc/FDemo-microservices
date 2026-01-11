package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"tafu-logistic/logistics-service/internal/core/domain"
	"tafu-logistic/logistics-service/internal/core/ports"
)

// ShippingRepository implements ports.ShippingOrderRepository
type ShippingRepository struct {
	pool *pgxpool.Pool
}

// NewShippingRepository creates a new PostgreSQL shipping repository
func NewShippingRepository(pool *pgxpool.Pool) *ShippingRepository {
	return &ShippingRepository{pool: pool}
}

// Ensure interface implementation
var _ ports.ShippingOrderRepository = (*ShippingRepository)(nil)

// Create persists a new shipping order
func (r *ShippingRepository) Create(ctx context.Context, order *domain.ShippingOrder) error {
	query := `
		INSERT INTO shipping_orders (
			id, internal_order_id, provider, tracking_code, carrier_status,
			system_status, shipping_fee, cod_amount, label_url, metadata,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		)`

	_, err := r.pool.Exec(ctx, query,
		order.ID,
		order.InternalOrderID,
		order.Provider,
		order.TrackingCode,
		order.CarrierStatus,
		order.SystemStatus,
		order.ShippingFee,
		order.CODAmount,
		order.LabelURL,
		order.Metadata,
		order.CreatedAt,
		order.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create shipping order: %w", err)
	}

	return nil
}

// Update updates an existing shipping order
func (r *ShippingRepository) Update(ctx context.Context, order *domain.ShippingOrder) error {
	query := `
		UPDATE shipping_orders SET
			tracking_code = $2,
			carrier_status = $3,
			system_status = $4,
			shipping_fee = $5,
			cod_amount = $6,
			label_url = $7,
			metadata = $8,
			updated_at = NOW()
		WHERE id = $1`

	result, err := r.pool.Exec(ctx, query,
		order.ID,
		order.TrackingCode,
		order.CarrierStatus,
		order.SystemStatus,
		order.ShippingFee,
		order.CODAmount,
		order.LabelURL,
		order.Metadata,
	)
	if err != nil {
		return fmt.Errorf("failed to update shipping order: %w", err)
	}

	if result.RowsAffected() == 0 {
		return errors.New("shipping order not found")
	}

	return nil
}

// GetByID retrieves a shipping order by its ID
func (r *ShippingRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.ShippingOrder, error) {
	return r.getOne(ctx, "id = $1", id)
}

// GetByInternalOrderID retrieves a shipping order by internal order ID
func (r *ShippingRepository) GetByInternalOrderID(ctx context.Context, internalOrderID uuid.UUID) (*domain.ShippingOrder, error) {
	return r.getOne(ctx, "internal_order_id = $1", internalOrderID)
}

// GetByTrackingCode retrieves a shipping order by tracking code
func (r *ShippingRepository) GetByTrackingCode(ctx context.Context, trackingCode string) (*domain.ShippingOrder, error) {
	return r.getOne(ctx, "tracking_code = $1", trackingCode)
}

// getOne is a helper to retrieve a single shipping order
func (r *ShippingRepository) getOne(ctx context.Context, whereClause string, arg interface{}) (*domain.ShippingOrder, error) {
	query := fmt.Sprintf(`
		SELECT id, internal_order_id, provider, tracking_code, carrier_status,
			   system_status, shipping_fee, cod_amount, label_url, metadata,
			   created_at, updated_at
		FROM shipping_orders
		WHERE %s`, whereClause)

	row := r.pool.QueryRow(ctx, query, arg)

	var order domain.ShippingOrder
	var metadata []byte

	err := row.Scan(
		&order.ID,
		&order.InternalOrderID,
		&order.Provider,
		&order.TrackingCode,
		&order.CarrierStatus,
		&order.SystemStatus,
		&order.ShippingFee,
		&order.CODAmount,
		&order.LabelURL,
		&metadata,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get shipping order: %w", err)
	}

	if metadata != nil {
		order.Metadata = json.RawMessage(metadata)
	}

	return &order, nil
}

// ListByStatus retrieves shipping orders by system status
func (r *ShippingRepository) ListByStatus(ctx context.Context, status domain.SystemStatus, limit, offset int) ([]*domain.ShippingOrder, error) {
	query := `
		SELECT id, internal_order_id, provider, tracking_code, carrier_status,
			   system_status, shipping_fee, cod_amount, label_url, metadata,
			   created_at, updated_at
		FROM shipping_orders
		WHERE system_status = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, status, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list shipping orders: %w", err)
	}
	defer rows.Close()

	var orders []*domain.ShippingOrder
	for rows.Next() {
		var order domain.ShippingOrder
		var metadata []byte

		err := rows.Scan(
			&order.ID,
			&order.InternalOrderID,
			&order.Provider,
			&order.TrackingCode,
			&order.CarrierStatus,
			&order.SystemStatus,
			&order.ShippingFee,
			&order.CODAmount,
			&order.LabelURL,
			&metadata,
			&order.CreatedAt,
			&order.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan shipping order: %w", err)
		}

		if metadata != nil {
			order.Metadata = json.RawMessage(metadata)
		}

		orders = append(orders, &order)
	}

	return orders, nil
}

// UpdateStatus updates only the status fields of a shipping order
func (r *ShippingRepository) UpdateStatus(ctx context.Context, id uuid.UUID, carrierStatus string, systemStatus domain.SystemStatus) error {
	query := `
		UPDATE shipping_orders SET
			carrier_status = $2,
			system_status = $3,
			updated_at = NOW()
		WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id, carrierStatus, systemStatus)
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return errors.New("shipping order not found")
	}

	return nil
}
