package clickhouse

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"microservices/analytic/internal/domain"
)

type EventRepository struct {
	conn driver.Conn
}

func NewEventRepository(conn driver.Conn) *EventRepository {
	return &EventRepository{conn: conn}
}

// BatchInsert inserts multiple events in a single batch operation.
func (r *EventRepository) BatchInsert(ctx context.Context, events []domain.UserEvent) error {
	if len(events) == 0 {
		return nil
	}

	batch, err := r.conn.PrepareBatch(ctx, `
		INSERT INTO analytics.user_events (
			event_id, user_id, event_type, metadata, url, ip_address, user_agent, created_at
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare batch: %w", err)
	}

	for _, event := range events {
		if err := batch.Append(
			event.EventID,
			event.UserID,
			event.EventType,
			event.Metadata,
			event.URL,
			event.IPAddress,
			event.UserAgent,
			event.CreatedAt,
		); err != nil {
			return fmt.Errorf("failed to append event to batch: %w", err)
		}
	}

	if err := batch.Send(); err != nil {
		return fmt.Errorf("failed to send batch: %w", err)
	}

	return nil
}

// ProductViewsResult represents hourly view counts.
type ProductViewsResult struct {
	Time  time.Time `json:"time"`
	Views uint64    `json:"views"`
}

// GetProductViews returns hourly view counts for a specific product.
func (r *EventRepository) GetProductViews(ctx context.Context, skuID string, start, end time.Time) ([]ProductViewsResult, error) {
	query := `
		SELECT
			toStartOfHour(created_at) as time,
			count(*) as views
		FROM analytics.user_events
		WHERE event_type = 'view_item'
			AND JSONExtractString(metadata, 'sku_id') = $1
			AND created_at >= $2
			AND created_at <= $3
		GROUP BY time
		ORDER BY time
	`

	rows, err := r.conn.Query(ctx, query, skuID, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query product views: %w", err)
	}
	defer rows.Close()

	var results []ProductViewsResult
	for rows.Next() {
		var result ProductViewsResult
		if err := rows.Scan(&result.Time, &result.Views); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		results = append(results, result)
	}

	return results, nil
}

// FunnelResult represents conversion funnel metrics.
type FunnelResult struct {
	Views         uint64  `json:"views"`
	AddToCarts    uint64  `json:"add_to_carts"`
	Checkouts     uint64  `json:"checkouts"`
	ViewToCart    float64 `json:"view_to_cart_rate"`
	CartToCheckout float64 `json:"cart_to_checkout_rate"`
}

// GetConversionRate calculates the conversion funnel metrics.
func (r *EventRepository) GetConversionRate(ctx context.Context, start, end time.Time) (*FunnelResult, error) {
	query := `
		SELECT
			countIf(event_type = 'view_item') as views,
			countIf(event_type = 'add_to_cart') as add_to_carts,
			countIf(event_type = 'checkout_start') as checkouts
		FROM analytics.user_events
		WHERE created_at >= $1 AND created_at <= $2
	`

	var result FunnelResult
	row := r.conn.QueryRow(ctx, query, start, end)
	if err := row.Scan(&result.Views, &result.AddToCarts, &result.Checkouts); err != nil {
		return nil, fmt.Errorf("failed to query conversion rate: %w", err)
	}

	// Calculate conversion rates
	if result.Views > 0 {
		result.ViewToCart = float64(result.AddToCarts) / float64(result.Views) * 100
	}
	if result.AddToCarts > 0 {
		result.CartToCheckout = float64(result.Checkouts) / float64(result.AddToCarts) * 100
	}

	return &result, nil
}

// GetEventCount returns the total count of events.
func (r *EventRepository) GetEventCount(ctx context.Context) (uint64, error) {
	var count uint64
	row := r.conn.QueryRow(ctx, `SELECT count(*) FROM analytics.user_events`)
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count events: %w", err)
	}
	return count, nil
}
