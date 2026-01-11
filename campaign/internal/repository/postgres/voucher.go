package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/tafu/campaign-service/internal/domain"
)

type VoucherRepository struct {
	pool *pgxpool.Pool
}

func NewVoucherRepository(pool *pgxpool.Pool) *VoucherRepository {
	return &VoucherRepository{pool: pool}
}

func (r *VoucherRepository) Create(ctx context.Context, voucher *domain.Voucher) error {
	conditionsJSON, err := json.Marshal(voucher.Conditions)
	if err != nil {
		return fmt.Errorf("failed to marshal conditions: %w", err)
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
	if err != nil {
		return fmt.Errorf("failed to create voucher: %w", err)
	}
	return nil
}

func (r *VoucherRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Voucher, error) {
	query := `
		SELECT id, code, campaign_id, total_count, used_count, type, value, conditions, status, created_at, updated_at
		FROM vouchers
		WHERE id = $1
	`
	return r.scanVoucher(ctx, query, id)
}

func (r *VoucherRepository) GetByCode(ctx context.Context, code string) (*domain.Voucher, error) {
	query := `
		SELECT id, code, campaign_id, total_count, used_count, type, value, conditions, status, created_at, updated_at
		FROM vouchers
		WHERE code = $1
	`
	return r.scanVoucher(ctx, query, code)
}

func (r *VoucherRepository) scanVoucher(ctx context.Context, query string, arg interface{}) (*domain.Voucher, error) {
	voucher := &domain.Voucher{}
	var conditionsJSON []byte

	err := r.pool.QueryRow(ctx, query, arg).Scan(
		&voucher.ID,
		&voucher.Code,
		&voucher.CampaignID,
		&voucher.TotalCount,
		&voucher.UsedCount,
		&voucher.Type,
		&voucher.Value,
		&conditionsJSON,
		&voucher.Status,
		&voucher.CreatedAt,
		&voucher.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrVoucherNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get voucher: %w", err)
	}

	if err := json.Unmarshal(conditionsJSON, &voucher.Conditions); err != nil {
		return nil, fmt.Errorf("failed to unmarshal conditions: %w", err)
	}

	return voucher, nil
}

func (r *VoucherRepository) Update(ctx context.Context, voucher *domain.Voucher) error {
	conditionsJSON, err := json.Marshal(voucher.Conditions)
	if err != nil {
		return fmt.Errorf("failed to marshal conditions: %w", err)
	}

	query := `
		UPDATE vouchers
		SET code = $2, total_count = $3, used_count = $4, type = $5, value = $6, conditions = $7, status = $8
		WHERE id = $1
	`
	_, err = r.pool.Exec(ctx, query,
		voucher.ID,
		voucher.Code,
		voucher.TotalCount,
		voucher.UsedCount,
		voucher.Type,
		voucher.Value,
		conditionsJSON,
		voucher.Status,
	)
	if err != nil {
		return fmt.Errorf("failed to update voucher: %w", err)
	}
	return nil
}

func (r *VoucherRepository) IncrementUsedCount(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE vouchers
		SET used_count = used_count + 1
		WHERE id = $1 AND used_count < total_count
	`
	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to increment used count: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrVoucherOutOfStock
	}
	return nil
}

func (r *VoucherRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM vouchers WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete voucher: %w", err)
	}
	return nil
}

func (r *VoucherRepository) ListByCampaignID(ctx context.Context, campaignID uuid.UUID) ([]*domain.Voucher, error) {
	query := `
		SELECT id, code, campaign_id, total_count, used_count, type, value, conditions, status, created_at, updated_at
		FROM vouchers
		WHERE campaign_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, query, campaignID)
	if err != nil {
		return nil, fmt.Errorf("failed to list vouchers: %w", err)
	}
	defer rows.Close()

	var vouchers []*domain.Voucher
	for rows.Next() {
		voucher := &domain.Voucher{}
		var conditionsJSON []byte

		if err := rows.Scan(
			&voucher.ID,
			&voucher.Code,
			&voucher.CampaignID,
			&voucher.TotalCount,
			&voucher.UsedCount,
			&voucher.Type,
			&voucher.Value,
			&conditionsJSON,
			&voucher.Status,
			&voucher.CreatedAt,
			&voucher.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan voucher: %w", err)
		}

		if err := json.Unmarshal(conditionsJSON, &voucher.Conditions); err != nil {
			return nil, fmt.Errorf("failed to unmarshal conditions: %w", err)
		}

		vouchers = append(vouchers, voucher)
	}
	return vouchers, nil
}

var _ domain.VoucherRepository = (*VoucherRepository)(nil)
