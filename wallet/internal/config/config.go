package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all configuration for the wallet service
type Config struct {
	App     AppConfig
	DB      DatabaseConfig
	NATS    NATSConfig
	Payment PaymentServiceConfig
	Wallet  WalletLimitsConfig
}

// AppConfig holds application configuration
type AppConfig struct {
	Port         string
	Host         string
	AuthGRPCAddr string
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	URL string
}

// NATSConfig holds NATS configuration
type NATSConfig struct {
	URL string
}

// PaymentServiceConfig holds payment service configuration
type PaymentServiceConfig struct {
	BaseURL string
}

// WalletLimitsConfig holds wallet limit configuration
type WalletLimitsConfig struct {
	MinTopUpVND   int64
	MaxTopUpVND   int64
	MaxBalanceVND int64
}

// Load loads configuration from environment variables and config file
func Load() (*Config, error) {
	v := viper.New()

	// Set defaults
	v.SetDefault("app.port", "8086")
	v.SetDefault("app.host", "0.0.0.0")
	v.SetDefault("app.auth_grpc_addr", "localhost:50051")
	v.SetDefault("db.url", "postgres://wallet:wallet_secret@localhost:5432/wallet_db?sslmode=disable")
	v.SetDefault("nats.url", "nats://localhost:4222")
	v.SetDefault("payment.base_url", "http://localhost:8083")
	v.SetDefault("wallet.min_topup_vnd", 10000)
	v.SetDefault("wallet.max_topup_vnd", 50000000)
	v.SetDefault("wallet.max_balance_vnd", 200000000)

	// Read from config file if exists
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("read config file: %w", err)
		}
	}

	// Environment variables override
	v.SetEnvPrefix("")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Manual env mappings for common patterns
	bindEnvs(v)

	cfg := &Config{
		App: AppConfig{
			Port:         v.GetString("app.port"),
			Host:         v.GetString("app.host"),
			AuthGRPCAddr: v.GetString("app.auth_grpc_addr"),
		},
		DB: DatabaseConfig{
			URL: v.GetString("db.url"),
		},
		NATS: NATSConfig{
			URL: v.GetString("nats.url"),
		},
		Payment: PaymentServiceConfig{
			BaseURL: v.GetString("payment.base_url"),
		},
		Wallet: WalletLimitsConfig{
			MinTopUpVND:   v.GetInt64("wallet.min_topup_vnd"),
			MaxTopUpVND:   v.GetInt64("wallet.max_topup_vnd"),
			MaxBalanceVND: v.GetInt64("wallet.max_balance_vnd"),
		},
	}

	return cfg, nil
}

func bindEnvs(v *viper.Viper) {
	v.BindEnv("app.port", "SERVER_PORT")
	v.BindEnv("app.host", "SERVER_HOST")
	v.BindEnv("app.auth_grpc_addr", "AUTH_GRPC_ADDR")
	v.BindEnv("db.url", "DATABASE_URL")
	v.BindEnv("nats.url", "NATS_URL")
	v.BindEnv("payment.base_url", "PAYMENT_SERVICE_URL")
	v.BindEnv("wallet.min_topup_vnd", "WALLET_MIN_TOPUP")
	v.BindEnv("wallet.max_topup_vnd", "WALLET_MAX_TOPUP")
	v.BindEnv("wallet.max_balance_vnd", "WALLET_MAX_BALANCE")
}
