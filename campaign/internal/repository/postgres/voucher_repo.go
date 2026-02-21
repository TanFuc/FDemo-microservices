package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"microservices/campaign/internal/model"
)

// VoucherRepository implements the voucher repository interface using PostgreSQL.
type VoucherRepository struct {
	pool *pgxpool.Pool
}

// NewVoucherRepository creates a new voucher repository.
func NewVoucherRepository(pool *pgxpool.Pool) *VoucherRepository {
	return &VoucherRepository{pool: pool}
}

// Create creates a new voucher.
func (r *VoucherRepository) Create(ctx context.Context, voucher *model.Voucher) error {
	conditionsJSON, err := json.Marshal(voucher.Conditions)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO vouchers (id, code, campaign_id, total_count, used_count, type, value, conditions, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err = r.pool.Exec(ctx, query,
		voucher.ID,
		voucher.Code,
		voucher.CampaignID,
		voucher.TotalCount,
		voucher.UsedCount,
		voucher.Type,
		voucher.Value,
		conditionsJSON,
		voucher.Status,
		voucher.CreatedAt,
		voucher.UpdatedAt,
	)
	return err
}

// GetByID retrieves a voucher by ID.
func (r *VoucherRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Voucher, error) {
	query := `
		SELECT id, code, campaign_id, total_count, used_count, type, value, conditions, status, created_at, updated_at
		FROM vouchers
		WHERE id = $1
	`
	return r.scanVoucher(ctx, query, id)
}

// GetByCode retrieves a voucher by code.
func (r *VoucherRepository) GetByCode(ctx context.Context, code string) (*model.Voucher, error) {
	query := `
		SELECT id, code, campaign_id, total_count, used_count, type, value, conditions, status, created_at, updated_at
		FROM vouchers
		WHERE code = $1
	`
	return r.scanVoucher(ctx, query, code)
}

func (r *VoucherRepository) scanVoucher(ctx context.Context, query string, arg interface{}) (*model.Voucher, error) {
	row := r.pool.QueryRow(ctx, query, arg)

	var voucher model.Voucher
	var valueStr string
	var conditionsJSON []byte

	err := row.Scan(
		&voucher.ID,
		&voucher.Code,
		&voucher.CampaignID,
		&voucher.TotalCount,
		&voucher.UsedCount,
		&voucher.Type,
		&valueStr,
		&conditionsJSON,
		&voucher.Status,
		&voucher.CreatedAt,
		&voucher.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	voucher.Value, _ = decimal.NewFromString(valueStr)
	if err := json.Unmarshal(conditionsJSON, &voucher.Conditions); err != nil {
		return nil, err
	}

	return &voucher, nil
}

// Update updates a voucher.
func (r *VoucherRepository) Update(ctx context.Context, voucher *model.Voucher) error {
	conditionsJSON, err := json.Marshal(voucher.Conditions)
	if err != nil {
		return err
	}

	query := `
		UPDATE vouchers
		SET code = $2, campaign_id = $3, total_count = $4, used_count = $5, type = $6, value = $7, conditions = $8, status = $9, updated_at = $10
		WHERE id = $1
	`
	_, err = r.pool.Exec(ctx, query,
		voucher.ID,
		voucher.Code,
		voucher.CampaignID,
		voucher.TotalCount,
		voucher.UsedCount,
		voucher.Type,
		voucher.Value,
		conditionsJSON,
		voucher.Status,
		time.Now(),
	)
	return err
}

// IncrementUsedCount increments the used count of a voucher.
func (r *VoucherRepository) IncrementUsedCount(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE vouchers SET used_count = used_count + 1, updated_at = $2 WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id, time.Now())
	return err
}

// Delete deletes a voucher.
func (r *VoucherRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM vouchers WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

// ListByCampaignID retrieves all vouchers for a campaign.
func (r *VoucherRepository) ListByCampaignID(ctx context.Context, campaignID uuid.UUID) ([]*model.Voucher, error) {
	query := `
		SELECT id, code, campaign_id, total_count, used_count, type, value, conditions, status, created_at, updated_at
		FROM vouchers
		WHERE campaign_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, query, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vouchers []*model.Voucher
	for rows.Next() {
		var voucher model.Voucher
		var valueStr string
		var conditionsJSON []byte

		if err := rows.Scan(
			&voucher.ID,
			&voucher.Code,
			&voucher.CampaignID,
			&voucher.TotalCount,
			&voucher.UsedCount,
			&voucher.Type,
			&valueStr,
			&conditionsJSON,
			&voucher.Status,
			&voucher.CreatedAt,
			&voucher.UpdatedAt,
		); err != nil {
			return nil, err
		}

		voucher.Value, _ = decimal.NewFromString(valueStr)
		if err := json.Unmarshal(conditionsJSON, &voucher.Conditions); err != nil {
			return nil, err
		}

		vouchers = append(vouchers, &voucher)
	}

	return vouchers, nil
}
