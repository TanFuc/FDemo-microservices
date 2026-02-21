package postgres

import (
	"fmt"

	"microservices/inventory/internal/config"
	"microservices/inventory/internal/model"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(cfg *config.PostgresConfig, debug bool) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)

	logLevel := logger.Silent
	if debug {
		logLevel = logger.Info
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	return db, nil
}

func AutoMigrate(db *gorm.DB) error {
	// Migrate models
	if err := db.AutoMigrate(&model.InventoryItem{}, &model.StockReservation{}); err != nil {
		return err
	}

	// Add check constraint for total_stock >= reserved_stock
	if err := db.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_constraint WHERE conname = 'chk_stock_balance'
			) THEN
				ALTER TABLE inventory_items
				ADD CONSTRAINT chk_stock_balance
				CHECK (total_stock >= reserved_stock);
			END IF;
		END $$;
	`).Error; err != nil {
		return err
	}

	return nil
}
