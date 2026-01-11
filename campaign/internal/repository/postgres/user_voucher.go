package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"microservices/campaign/internal/domain"
)

type UserVoucherRepository struct {
	pool *pgxpool.Pool
}

func NewUserVoucherRepository(pool *pgxpool.Pool) *UserVoucherRepository {
	return &UserVoucherRepository{pool: pool}
}

func (r *UserVoucherRepository) Create(ctx context.Context, uv *domain.UserVoucher) error {
	query := `
		INSERT INTO user_vouchers (id, user_id, voucher_id, is_used, claimed_at, used_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.pool.Exec(ctx, query,
		uv.ID,
		uv.UserID,
		uv.VoucherID,
		uv.IsUsed,
		uv.ClaimedAt,
		uv.UsedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create user voucher: %w", err)
	}
	return nil
}

func (r *UserVoucherRepository) GetByUserAndVoucher(ctx context.Context, userID, voucherID uuid.UUID) (*domain.UserVoucher, error) {
	query := `
		SELECT id, user_id, voucher_id, is_used, claimed_at, used_at
		FROM user_vouchers
		WHERE user_id = $1 AND voucher_id = $2
	`
	uv := &domain.UserVoucher{}
	err := r.pool.QueryRow(ctx, query, userID, voucherID).Scan(
		&uv.ID,
		&uv.UserID,
		&uv.VoucherID,
		&uv.IsUsed,
		&uv.ClaimedAt,
		&uv.UsedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user voucher: %w", err)
	}
	return uv, nil
}

func (r *UserVoucherRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*domain.UserVoucher, error) {
	query := `
		SELECT id, user_id, voucher_id, is_used, claimed_at, used_at
		FROM user_vouchers
		WHERE user_id = $1
		ORDER BY claimed_at DESC
	`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list user vouchers: %w", err)
	}
	defer rows.Close()

	var uvs []*domain.UserVoucher
	for rows.Next() {
		uv := &domain.UserVoucher{}
		if err := rows.Scan(
			&uv.ID,
			&uv.UserID,
			&uv.VoucherID,
			&uv.IsUsed,
			&uv.ClaimedAt,
			&uv.UsedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan user voucher: %w", err)
		}
		uvs = append(uvs, uv)
	}
	return uvs, nil
}

func (r *UserVoucherRepository) MarkUsed(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE user_vouchers
		SET is_used = true, used_at = NOW()
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to mark voucher as used: %w", err)
	}
	return nil
}

func (r *UserVoucherRepository) Exists(ctx context.Context, userID, voucherID uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM user_vouchers
			WHERE user_id = $1 AND voucher_id = $2
		)
	`
	var exists bool
	err := r.pool.QueryRow(ctx, query, userID, voucherID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check existence: %w", err)
	}
	return exists, nil
}

var _ domain.UserVoucherRepository = (*UserVoucherRepository)(nil)
