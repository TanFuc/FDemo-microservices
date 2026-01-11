package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"microservices/payment/internal/domain"
	"microservices/payment/internal/port"
)

var (
	ErrNotFound = errors.New("payment transaction not found")
)

// PostgresRepository implements port.PaymentRepository using PostgreSQL
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// Ensure PostgresRepository implements port.PaymentRepository
var _ port.PaymentRepository = (*PostgresRepository)(nil)

// Create creates a new payment transaction
func (r *PostgresRepository) Create(ctx context.Context, tx *domain.PaymentTransaction) error {
	query := `
		INSERT INTO payment_transactions (id, order_id, user_id, amount, currency, provider, provider_tx_id, status, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := r.pool.Exec(ctx, query,
		tx.ID,
		tx.OrderID,
		tx.UserID,
		tx.Amount,
		tx.Currency,
		tx.Provider,
		tx.ProviderTxID,
		tx.Status,
		tx.Metadata,
		tx.CreatedAt,
		tx.UpdatedAt,
	)
	return err
}

// Update updates an existing payment transaction
func (r *PostgresRepository) Update(ctx context.Context, tx *domain.PaymentTransaction) error {
	query := `
		UPDATE payment_transactions
		SET provider_tx_id = $2, status = $3, metadata = $4, updated_at = $5
		WHERE id = $1
	`
	result, err := r.pool.Exec(ctx, query,
		tx.ID,
		tx.ProviderTxID,
		tx.Status,
		tx.Metadata,
		tx.UpdatedAt,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// GetByID retrieves a payment transaction by ID
func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.PaymentTransaction, error) {
	query := `
		SELECT id, order_id, user_id, amount, currency, provider, provider_tx_id, status, metadata, created_at, updated_at
		FROM payment_transactions
		WHERE id = $1
	`
	return r.scanTransaction(r.pool.QueryRow(ctx, query, id))
}

// GetByOrderID retrieves payment transactions by order ID
func (r *PostgresRepository) GetByOrderID(ctx context.Context, orderID uuid.UUID) ([]*domain.PaymentTransaction, error) {
	query := `
		SELECT id, order_id, user_id, amount, currency, provider, provider_tx_id, status, metadata, created_at, updated_at
		FROM payment_transactions
		WHERE order_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, query, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanTransactions(rows)
}

// GetByProviderTxID retrieves a payment transaction by provider transaction ID
func (r *PostgresRepository) GetByProviderTxID(ctx context.Context, provider domain.Provider, providerTxID string) (*domain.PaymentTransaction, error) {
	query := `
		SELECT id, order_id, user_id, amount, currency, provider, provider_tx_id, status, metadata, created_at, updated_at
		FROM payment_transactions
		WHERE provider = $1 AND provider_tx_id = $2
	`
	return r.scanTransaction(r.pool.QueryRow(ctx, query, provider, providerTxID))
}

// GetPendingTransactions retrieves pending transactions older than the given time
func (r *PostgresRepository) GetPendingTransactions(ctx context.Context, olderThan time.Time) ([]*domain.PaymentTransaction, error) {
	query := `
		SELECT id, order_id, user_id, amount, currency, provider, provider_tx_id, status, metadata, created_at, updated_at
		FROM payment_transactions
		WHERE status = 'PENDING' AND created_at < $1
		ORDER BY created_at ASC
	`
	rows, err := r.pool.Query(ctx, query, olderThan)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanTransactions(rows)
}

// CreateLog creates a payment log entry
func (r *PostgresRepository) CreateLog(ctx context.Context, log *domain.PaymentLog) error {
	query := `
		INSERT INTO payment_logs (id, transaction_id, provider, event_type, raw_payload, ip_address, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.pool.Exec(ctx, query,
		log.ID,
		log.TransactionID,
		log.Provider,
		log.EventType,
		log.RawPayload,
		log.IPAddress,
		log.CreatedAt,
	)
	return err
}

// GetLogsByTransactionID retrieves logs for a transaction
func (r *PostgresRepository) GetLogsByTransactionID(ctx context.Context, txID uuid.UUID) ([]*domain.PaymentLog, error) {
	query := `
		SELECT id, transaction_id, provider, event_type, raw_payload, ip_address, created_at
		FROM payment_logs
		WHERE transaction_id = $1
		ORDER BY created_at ASC
	`
	rows, err := r.pool.Query(ctx, query, txID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*domain.PaymentLog
	for rows.Next() {
		log := &domain.PaymentLog{}
		err := rows.Scan(
			&log.ID,
			&log.TransactionID,
			&log.Provider,
			&log.EventType,
			&log.RawPayload,
			&log.IPAddress,
			&log.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	return logs, rows.Err()
}

// scanTransaction scans a single row into a PaymentTransaction
func (r *PostgresRepository) scanTransaction(row pgx.Row) (*domain.PaymentTransaction, error) {
	tx := &domain.PaymentTransaction{}
	err := row.Scan(
		&tx.ID,
		&tx.OrderID,
		&tx.UserID,
		&tx.Amount,
		&tx.Currency,
		&tx.Provider,
		&tx.ProviderTxID,
		&tx.Status,
		&tx.Metadata,
		&tx.CreatedAt,
		&tx.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return tx, nil
}

// scanTransactions scans multiple rows into PaymentTransactions
func (r *PostgresRepository) scanTransactions(rows pgx.Rows) ([]*domain.PaymentTransaction, error) {
	var transactions []*domain.PaymentTransaction
	for rows.Next() {
		tx := &domain.PaymentTransaction{}
		err := rows.Scan(
			&tx.ID,
			&tx.OrderID,
			&tx.UserID,
			&tx.Amount,
			&tx.Currency,
			&tx.Provider,
			&tx.ProviderTxID,
			&tx.Status,
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
