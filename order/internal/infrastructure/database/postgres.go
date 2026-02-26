package database

import (
	"context"
	"fmt"
	"time"

	"microservices/order/internal/config"
	"microservices/order/internal/domain"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Database wraps the GORM DB connection
type Database struct {
	DB *gorm.DB
}

// NewDatabase creates a new database connection
func NewDatabase(cfg *config.DatabaseConfig) (*Database, error) {
	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return &Database{DB: db}, nil
}

// AutoMigrate runs database migrations
func (d *Database) AutoMigrate() error {
	return d.DB.AutoMigrate(
		&domain.Order{},
		&domain.OrderItem{},
		&domain.ReturnRequest{},
	)
}

// Close closes the database connection
func (d *Database) Close() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Health checks database connectivity
func (d *Database) Health(ctx context.Context) error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

// Transaction key for context
type txKey struct{}

// BeginTx starts a new transaction and stores it in context
func (d *Database) BeginTx(ctx context.Context) (context.Context, error) {
	tx := d.DB.WithContext(ctx).Begin()
	if tx.Error != nil {
		return ctx, tx.Error
	}
	return context.WithValue(ctx, txKey{}, tx), nil
}

// CommitTx commits the transaction from context
func (d *Database) CommitTx(ctx context.Context) error {
	tx, ok := ctx.Value(txKey{}).(*gorm.DB)
	if !ok {
		return fmt.Errorf("no transaction in context")
	}
	return tx.Commit().Error
}

// RollbackTx rolls back the transaction from context
func (d *Database) RollbackTx(ctx context.Context) error {
	tx, ok := ctx.Value(txKey{}).(*gorm.DB)
	if !ok {
		return fmt.Errorf("no transaction in context")
	}
	return tx.Rollback().Error
}

// GetDB returns the DB instance, using transaction from context if available
func (d *Database) GetDB(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return tx
	}
	return d.DB.WithContext(ctx)
}
