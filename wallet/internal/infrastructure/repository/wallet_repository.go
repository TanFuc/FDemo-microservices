package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"microservices/wallet/internal/domain"
	"microservices/wallet/internal/port"
)

// PostgresWalletRepository implements port.WalletRepository using PostgreSQL
type PostgresWalletRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresWalletRepository creates a new PostgreSQL wallet repository
func NewPostgresWalletRepository(pool *pgxpool.Pool) *PostgresWalletRepository {
	return &PostgresWalletRepository{pool: pool}
}

// Ensure PostgresWalletRepository implements port.WalletRepository
var _ port.WalletRepository = (*PostgresWalletRepository)(nil)

// CreateWallet creates a new wallet
func (r *PostgresWalletRepository) CreateWallet(ctx context.Context, w *domain.Wallet) error {
	query := `
		INSERT INTO wallets (id, user_id, balance, currency, status, version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.pool.Exec(ctx, query,
		w.ID,
		w.UserID,
		w.Balance,
		w.Currency,
		w.Status,
		w.Version,
		w.CreatedAt,
		w.UpdatedAt,
	)
	if err != nil && isDuplicateKeyError(err) {
		return domain.ErrWalletAlreadyExists
	}
	return err
}

// GetByUserID retrieves a wallet by user ID
func (r *PostgresWalletRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.Wallet, error) {
	query := `
		SELECT id, user_id, balance, currency, status, version, created_at, updated_at
		FROM wallets
		WHERE user_id = $1
	`
	return r.scanWallet(r.pool.QueryRow(ctx, query, userID))
}

// GetByID retrieves a wallet by wallet ID
func (r *PostgresWalletRepository) GetByID(ctx context.Context, walletID uuid.UUID) (*domain.Wallet, error) {
	query := `
		SELECT id, user_id, balance, currency, status, version, created_at, updated_at
		FROM wallets
		WHERE id = $1
	`
	return r.scanWallet(r.pool.QueryRow(ctx, query, walletID))
}

// ExistsByUserID checks if a wallet exists for the given user ID
func (r *PostgresWalletRepository) ExistsByUserID(ctx context.Context, userID uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM wallets WHERE user_id = $1)`
	var exists bool
	err := r.pool.QueryRow(ctx, query, userID).Scan(&exists)
	return exists, err
}

// UpdateBalanceOptimistic updates wallet balance with optimistic locking
func (r *PostgresWalletRepository) UpdateBalanceOptimistic(ctx context.Context, walletID uuid.UUID, newBalance decimal.Decimal, newVersion, expectedVersion int) error {
	query := `
		UPDATE wallets
		SET balance = $1, version = $2, updated_at = NOW()
		WHERE id = $3 AND version = $4
	`
	tag, err := r.pool.Exec(ctx, query, newBalance, newVersion, walletID, expectedVersion)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrVersionConflict
	}
	return nil
}

// CreateTransaction creates a new wallet transaction
func (r *PostgresWalletRepository) CreateTransaction(ctx context.Context, tx *domain.WalletTransaction) error {
	query := `
		INSERT INTO wallet_transactions (
			id, wallet_id, user_id, type, status, amount, balance_before, balance_after,
			currency, reference_id, reference_type, description, payment_tx_id,
			idempotency_key, metadata, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`
	_, err := r.pool.Exec(ctx, query,
		tx.ID,
		tx.WalletID,
		tx.UserID,
		tx.Type,
		tx.Status,
		tx.Amount,
		tx.BalanceBefore,
		tx.BalanceAfter,
		tx.Currency,
		tx.ReferenceID,
		tx.ReferenceType,
		tx.Description,
		tx.PaymentTxID,
		tx.IdempotencyKey,
		tx.Metadata,
		tx.CreatedAt,
		tx.UpdatedAt,
	)
	if err != nil && isDuplicateKeyError(err) {
		return domain.ErrDuplicateTx
	}
	return err
}

// GetTransactionByID retrieves a transaction by ID
func (r *PostgresWalletRepository) GetTransactionByID(ctx context.Context, txID uuid.UUID) (*domain.WalletTransaction, error) {
	query := `
		SELECT id, wallet_id, user_id, type, status, amount, balance_before, balance_after,
			currency, reference_id, reference_type, description, payment_tx_id,
			idempotency_key, metadata, created_at, updated_at
		FROM wallet_transactions
		WHERE id = $1
	`
	return r.scanTransaction(r.pool.QueryRow(ctx, query, txID))
}

// GetTransactionByIdempotencyKey retrieves a transaction by idempotency key
func (r *PostgresWalletRepository) GetTransactionByIdempotencyKey(ctx context.Context, key string) (*domain.WalletTransaction, error) {
	query := `
		SELECT id, wallet_id, user_id, type, status, amount, balance_before, balance_after,
			currency, reference_id, reference_type, description, payment_tx_id,
			idempotency_key, metadata, created_at, updated_at
		FROM wallet_transactions
		WHERE idempotency_key = $1
	`
	return r.scanTransaction(r.pool.QueryRow(ctx, query, key))
}

// GetTransactionByPaymentTxID retrieves a transaction by payment transaction ID
func (r *PostgresWalletRepository) GetTransactionByPaymentTxID(ctx context.Context, paymentTxID uuid.UUID) (*domain.WalletTransaction, error) {
	query := `
		SELECT id, wallet_id, user_id, type, status, amount, balance_before, balance_after,
			currency, reference_id, reference_type, description, payment_tx_id,
			idempotency_key, metadata, created_at, updated_at
		FROM wallet_transactions
		WHERE payment_tx_id = $1
	`
	return r.scanTransaction(r.pool.QueryRow(ctx, query, paymentTxID))
}

// UpdateTransactionStatus updates the status of a transaction
func (r *PostgresWalletRepository) UpdateTransactionStatus(ctx context.Context, txID uuid.UUID, status domain.TxStatus) error {
	query := `
		UPDATE wallet_transactions
		SET status = $1, updated_at = NOW()
		WHERE id = $2
	`
	tag, err := r.pool.Exec(ctx, query, status, txID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrTransactionNotFound
	}
	return nil
}

// ListTransactions lists transactions for a wallet with pagination
func (r *PostgresWalletRepository) ListTransactions(ctx context.Context, walletID uuid.UUID, limit, offset int) ([]*domain.WalletTransaction, int64, error) {
	// Get total count
	countQuery := `SELECT COUNT(*) FROM wallet_transactions WHERE wallet_id = $1`
	var total int64
	if err := r.pool.QueryRow(ctx, countQuery, walletID).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Get transactions
	query := `
		SELECT id, wallet_id, user_id, type, status, amount, balance_before, balance_after,
			currency, reference_id, reference_type, description, payment_tx_id,
			idempotency_key, metadata, created_at, updated_at
		FROM wallet_transactions
		WHERE wallet_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.pool.Query(ctx, query, walletID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	transactions, err := r.scanTransactions(rows)
	if err != nil {
		return nil, 0, err
	}

	return transactions, total, nil
}

// CreateOutboxEvent creates a new outbox event
func (r *PostgresWalletRepository) CreateOutboxEvent(ctx context.Context, e *domain.OutboxEvent) error {
	query := `
		INSERT INTO outbox_events (id, aggregate_id, event_type, payload, status, retry_count, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.pool.Exec(ctx, query,
		e.ID,
		e.AggregateID,
		e.EventType,
		e.Payload,
		e.Status,
		e.RetryCount,
		e.CreatedAt,
	)
	return err
}

// GetPendingOutboxEvents retrieves pending outbox events
func (r *PostgresWalletRepository) GetPendingOutboxEvents(ctx context.Context, limit int) ([]*domain.OutboxEvent, error) {
	query := `
		SELECT id, aggregate_id, event_type, payload, status, retry_count, created_at, published_at
		FROM outbox_events
		WHERE status = 'PENDING'
		ORDER BY created_at ASC
		LIMIT $1
	`
	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*domain.OutboxEvent
	for rows.Next() {
		e := &domain.OutboxEvent{}
		err := rows.Scan(
			&e.ID,
			&e.AggregateID,
			&e.EventType,
			&e.Payload,
			&e.Status,
			&e.RetryCount,
			&e.CreatedAt,
			&e.PublishedAt,
		)
		if err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// MarkOutboxPublished marks an outbox event as published
func (r *PostgresWalletRepository) MarkOutboxPublished(ctx context.Context, eventID uuid.UUID) error {
	now := time.Now()
	query := `
		UPDATE outbox_events
		SET status = 'PUBLISHED', published_at = $1
		WHERE id = $2
	`
	_, err := r.pool.Exec(ctx, query, now, eventID)
	return err
}

// MarkOutboxFailed marks an outbox event as failed
func (r *PostgresWalletRepository) MarkOutboxFailed(ctx context.Context, eventID uuid.UUID) error {
	query := `
		UPDATE outbox_events
		SET status = 'PENDING', retry_count = retry_count + 1
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, query, eventID)
	return err
}

// RunInTransaction executes fn within a single PostgreSQL transaction
func (r *PostgresWalletRepository) RunInTransaction(ctx context.Context, fn func(tx port.TxContext) error) error {
	pgTx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer pgTx.Rollback(ctx) // no-op if committed

	txCtx := &pgTxContext{tx: pgTx}
	if err := fn(txCtx); err != nil {
		return err
	}
	return pgTx.Commit(ctx)
}

// pgTxContext implements port.TxContext
type pgTxContext struct {
	tx pgx.Tx
}

// CreateTransaction creates a transaction within the pgx transaction
func (t *pgTxContext) CreateTransaction(ctx context.Context, tx *domain.WalletTransaction) error {
	query := `
		INSERT INTO wallet_transactions (
			id, wallet_id, user_id, type, status, amount, balance_before, balance_after,
			currency, reference_id, reference_type, description, payment_tx_id,
			idempotency_key, metadata, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`
	_, err := t.tx.Exec(ctx, query,
		tx.ID,
		tx.WalletID,
		tx.UserID,
		tx.Type,
		tx.Status,
		tx.Amount,
		tx.BalanceBefore,
		tx.BalanceAfter,
		tx.Currency,
		tx.ReferenceID,
		tx.ReferenceType,
		tx.Description,
		tx.PaymentTxID,
		tx.IdempotencyKey,
		tx.Metadata,
		tx.CreatedAt,
		tx.UpdatedAt,
	)
	if err != nil && isDuplicateKeyError(err) {
		return domain.ErrDuplicateTx
	}
	return err
}

// UpdateBalanceOptimistic updates wallet balance with optimistic locking
func (t *pgTxContext) UpdateBalanceOptimistic(ctx context.Context, walletID uuid.UUID, newBalance decimal.Decimal, newVersion, expectedVersion int) error {
	query := `
		UPDATE wallets
		SET balance = $1, version = $2, updated_at = NOW()
		WHERE id = $3 AND version = $4
	`
	tag, err := t.tx.Exec(ctx, query, newBalance, newVersion, walletID, expectedVersion)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrVersionConflict
	}
	return nil
}

// CreateOutboxEvent creates an outbox event within the pgx transaction
func (t *pgTxContext) CreateOutboxEvent(ctx context.Context, e *domain.OutboxEvent) error {
	query := `
		INSERT INTO outbox_events (id, aggregate_id, event_type, payload, status, retry_count, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := t.tx.Exec(ctx, query,
		e.ID,
		e.AggregateID,
		e.EventType,
		e.Payload,
		e.Status,
		e.RetryCount,
		e.CreatedAt,
	)
	return err
}

// UpdateTransactionStatus updates transaction status within the pgx transaction
func (t *pgTxContext) UpdateTransactionStatus(ctx context.Context, txID uuid.UUID, status domain.TxStatus) error {
	query := `
		UPDATE wallet_transactions
		SET status = $1, updated_at = NOW()
		WHERE id = $2
	`
	tag, err := t.tx.Exec(ctx, query, status, txID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrTransactionNotFound
	}
	return nil
}

// Helper functions

func (r *PostgresWalletRepository) scanWallet(row pgx.Row) (*domain.Wallet, error) {
	w := &domain.Wallet{}
	err := row.Scan(
		&w.ID,
		&w.UserID,
		&w.Balance,
		&w.Currency,
		&w.Status,
		&w.Version,
		&w.CreatedAt,
		&w.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrWalletNotFound
	}
	if err != nil {
		return nil, err
	}
	return w, nil
}

func (r *PostgresWalletRepository) scanTransaction(row pgx.Row) (*domain.WalletTransaction, error) {
	tx := &domain.WalletTransaction{}
	err := row.Scan(
		&tx.ID,
		&tx.WalletID,
		&tx.UserID,
		&tx.Type,
		&tx.Status,
		&tx.Amount,
		&tx.BalanceBefore,
		&tx.BalanceAfter,
		&tx.Currency,
		&tx.ReferenceID,
		&tx.ReferenceType,
		&tx.Description,
		&tx.PaymentTxID,
		&tx.IdempotencyKey,
		&tx.Metadata,
		&tx.CreatedAt,
		&tx.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrTransactionNotFound
	}
	if err != nil {
		return nil, err
	}
	return tx, nil
}

func (r *PostgresWalletRepository) scanTransactions(rows pgx.Rows) ([]*domain.WalletTransaction, error) {
	var transactions []*domain.WalletTransaction
	for rows.Next() {
		tx := &domain.WalletTransaction{}
		err := rows.Scan(
			&tx.ID,
			&tx.WalletID,
			&tx.UserID,
			&tx.Type,
			&tx.Status,
			&tx.Amount,
			&tx.BalanceBefore,
			&tx.BalanceAfter,
			&tx.Currency,
			&tx.ReferenceID,
			&tx.ReferenceType,
			&tx.Description,
			&tx.PaymentTxID,
			&tx.IdempotencyKey,
			&tx.Metadata,
			&tx.CreatedAt,
			&tx.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, tx)
	}
	return transactions, rows.Err()
}

func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "duplicate key") ||
		strings.Contains(err.Error(), "unique constraint") ||
		strings.Contains(err.Error(), "23505")
}
