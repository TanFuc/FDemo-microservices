package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"microservices/campaign/internal/model"
)

// UserVoucherRepository implements the user voucher repository interface using PostgreSQL.
type UserVoucherRepository struct {
	pool *pgxpool.Pool
}

// NewUserVoucherRepository creates a new user voucher repository.
func NewUserVoucherRepository(pool *pgxpool.Pool) *UserVoucherRepository {
	return &UserVoucherRepository{pool: pool}
}

// Create creates a new user voucher.
func (r *UserVoucherRepository) Create(ctx context.Context, userVoucher *model.UserVoucher) error {
	query := `
		INSERT INTO user_vouchers (id, user_id, voucher_id, is_used, claimed_at, used_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.pool.Exec(ctx, query,
		userVoucher.ID,
		userVoucher.UserID,
		userVoucher.VoucherID,
		userVoucher.IsUsed,
		userVoucher.ClaimedAt,
		userVoucher.UsedAt,
	)
	return err
}

// GetByUserAndVoucher retrieves a user voucher by user ID and voucher ID.
func (r *UserVoucherRepository) GetByUserAndVoucher(ctx context.Context, userID, voucherID uuid.UUID) (*model.UserVoucher, error) {
	query := `
		SELECT id, user_id, voucher_id, is_used, claimed_at, used_at
		FROM user_vouchers
		WHERE user_id = $1 AND voucher_id = $2
	`
	row := r.pool.QueryRow(ctx, query, userID, voucherID)

	var userVoucher model.UserVoucher
	err := row.Scan(
		&userVoucher.ID,
		&userVoucher.UserID,
		&userVoucher.VoucherID,
		&userVoucher.IsUsed,
		&userVoucher.ClaimedAt,
		&userVoucher.UsedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &userVoucher, nil
}

// GetByUserID retrieves all user vouchers by user ID.
func (r *UserVoucherRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*model.UserVoucher, error) {
	query := `
		SELECT id, user_id, voucher_id, is_used, claimed_at, used_at
		FROM user_vouchers
		WHERE user_id = $1
		ORDER BY claimed_at DESC
	`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var userVouchers []*model.UserVoucher
	for rows.Next() {
		var userVoucher model.UserVoucher
		if err := rows.Scan(
			&userVoucher.ID,
			&userVoucher.UserID,
			&userVoucher.VoucherID,
			&userVoucher.IsUsed,
			&userVoucher.ClaimedAt,
			&userVoucher.UsedAt,
		); err != nil {
			return nil, err
		}
		userVouchers = append(userVouchers, &userVoucher)
	}

	return userVouchers, nil
}

// MarkUsed marks a user voucher as used.
func (r *UserVoucherRepository) MarkUsed(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE user_vouchers SET is_used = true, used_at = $2 WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id, time.Now())
	return err
}

// Exists checks if a user voucher exists.
func (r *UserVoucherRepository) Exists(ctx context.Context, userID, voucherID uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM user_vouchers WHERE user_id = $1 AND voucher_id = $2)`
	var exists bool
	err := r.pool.QueryRow(ctx, query, userID, voucherID).Scan(&exists)
	return exists, err
}
