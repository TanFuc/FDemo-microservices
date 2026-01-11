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

// WebhookLogRepository implements ports.WebhookLogRepository
type WebhookLogRepository struct {
	pool *pgxpool.Pool
}

// NewWebhookLogRepository creates a new PostgreSQL webhook log repository
func NewWebhookLogRepository(pool *pgxpool.Pool) *WebhookLogRepository {
	return &WebhookLogRepository{pool: pool}
}

// Ensure interface implementation
var _ ports.WebhookLogRepository = (*WebhookLogRepository)(nil)

// Create persists a new webhook log entry
func (r *WebhookLogRepository) Create(ctx context.Context, log *domain.WebhookLog) error {
	query := `
		INSERT INTO webhook_logs (
			id, shipping_order_id, provider, tracking_code, carrier_status,
			system_status, raw_payload, http_status, error_message,
			processed_at, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)`

	_, err := r.pool.Exec(ctx, query,
		log.ID,
		log.ShippingOrderID,
		log.Provider,
		log.TrackingCode,
		log.CarrierStatus,
		log.SystemStatus,
		log.RawPayload,
		log.HTTPStatus,
		log.ErrorMessage,
		log.ProcessedAt,
		log.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create webhook log: %w", err)
	}

	return nil
}

// GetByID retrieves a webhook log by its ID
func (r *WebhookLogRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.WebhookLog, error) {
	query := `
		SELECT id, shipping_order_id, provider, tracking_code, carrier_status,
			   system_status, raw_payload, http_status, error_message,
			   processed_at, created_at
		FROM webhook_logs
		WHERE id = $1`

	row := r.pool.QueryRow(ctx, query, id)

	var log domain.WebhookLog
	var rawPayload []byte

	err := row.Scan(
		&log.ID,
		&log.ShippingOrderID,
		&log.Provider,
		&log.TrackingCode,
		&log.CarrierStatus,
		&log.SystemStatus,
		&rawPayload,
		&log.HTTPStatus,
		&log.ErrorMessage,
		&log.ProcessedAt,
		&log.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get webhook log: %w", err)
	}

	if rawPayload != nil {
		log.RawPayload = json.RawMessage(rawPayload)
	}

	return &log, nil
}

// ListByTrackingCode retrieves webhook logs for a tracking code
func (r *WebhookLogRepository) ListByTrackingCode(ctx context.Context, trackingCode string, limit, offset int) ([]*domain.WebhookLog, error) {
	query := `
		SELECT id, shipping_order_id, provider, tracking_code, carrier_status,
			   system_status, raw_payload, http_status, error_message,
			   processed_at, created_at
		FROM webhook_logs
		WHERE tracking_code = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	return r.listLogs(ctx, query, trackingCode, limit, offset)
}

// ListByShippingOrderID retrieves webhook logs for a shipping order
func (r *WebhookLogRepository) ListByShippingOrderID(ctx context.Context, shippingOrderID uuid.UUID, limit, offset int) ([]*domain.WebhookLog, error) {
	query := `
		SELECT id, shipping_order_id, provider, tracking_code, carrier_status,
			   system_status, raw_payload, http_status, error_message,
			   processed_at, created_at
		FROM webhook_logs
		WHERE shipping_order_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	return r.listLogs(ctx, query, shippingOrderID, limit, offset)
}

func (r *WebhookLogRepository) listLogs(ctx context.Context, query string, arg interface{}, limit, offset int) ([]*domain.WebhookLog, error) {
	rows, err := r.pool.Query(ctx, query, arg, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list webhook logs: %w", err)
	}
	defer rows.Close()

	var logs []*domain.WebhookLog
	for rows.Next() {
		var log domain.WebhookLog
		var rawPayload []byte

		err := rows.Scan(
			&log.ID,
			&log.ShippingOrderID,
			&log.Provider,
			&log.TrackingCode,
			&log.CarrierStatus,
			&log.SystemStatus,
			&rawPayload,
			&log.HTTPStatus,
			&log.ErrorMessage,
			&log.ProcessedAt,
			&log.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan webhook log: %w", err)
		}

		if rawPayload != nil {
			log.RawPayload = json.RawMessage(rawPayload)
		}

		logs = append(logs, &log)
	}

	return logs, nil
}
