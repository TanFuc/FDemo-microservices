package clickhouse

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/tafu/analytics-service/internal/config"
)

func NewConnection(cfg config.ClickHouseConfig) (driver.Conn, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)},
		Auth: clickhouse.Auth{
			Database: cfg.Database,
			Username: cfg.Username,
			Password: cfg.Password,
		},
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		DialTimeout:     5 * time.Second,
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open clickhouse connection: %w", err)
	}

	if err := conn.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping clickhouse: %w", err)
	}

	return conn, nil
}

func InitSchema(ctx context.Context, conn driver.Conn) error {
	// Create database if not exists
	if err := conn.Exec(ctx, `CREATE DATABASE IF NOT EXISTS analytics`); err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}

	// Create user_events table
	query := `
		CREATE TABLE IF NOT EXISTS analytics.user_events (
			event_id UUID,
			user_id String,
			event_type String,
			metadata String,
			url String,
			ip_address String,
			user_agent String,
			created_at DateTime
		) ENGINE = MergeTree()
		PARTITION BY toYYYYMM(created_at)
		ORDER BY (event_type, created_at)
	`
	if err := conn.Exec(ctx, query); err != nil {
		return fmt.Errorf("failed to create user_events table: %w", err)
	}

	return nil
}
