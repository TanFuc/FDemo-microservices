package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	NATS     NATSConfig
	Stripe   StripeConfig
	MoMo     MoMoConfig
	Webhook  WebhookConfig
	Job      JobConfig
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Host string
	Port int
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	URL string
}

// NATSConfig holds NATS configuration
type NATSConfig struct {
	URL string
}

// StripeConfig holds Stripe configuration
type StripeConfig struct {
	SecretKey     string
	WebhookSecret string
}

// MoMoConfig holds MoMo configuration
type MoMoConfig struct {
	PartnerCode string
	AccessKey   string
	SecretKey   string
	Endpoint    string
}

// WebhookConfig holds webhook configuration
type WebhookConfig struct {
	BaseURL string
}

// JobConfig holds background job configuration
type JobConfig struct {
	ReconciliationSchedule string
	ReconciliationTimeout  time.Duration
	ReconciliationOlderThan time.Duration
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Host: getEnv("SERVER_HOST", "0.0.0.0"),
			Port: getEnvInt("SERVER_PORT", 8080),
		},
		Database: DatabaseConfig{
			URL: getEnv("DATABASE_URL", "postgres://payment:payment_secret@localhost:5432/payment_db?sslmode=disable"),
		},
		NATS: NATSConfig{
			URL: getEnv("NATS_URL", "nats://localhost:4222"),
		},
		Stripe: StripeConfig{
			SecretKey:     getEnv("STRIPE_SECRET_KEY", ""),
			WebhookSecret: getEnv("STRIPE_WEBHOOK_SECRET", ""),
		},
		MoMo: MoMoConfig{
			PartnerCode: getEnv("MOMO_PARTNER_CODE", ""),
			AccessKey:   getEnv("MOMO_ACCESS_KEY", ""),
			SecretKey:   getEnv("MOMO_SECRET_KEY", ""),
			Endpoint:    getEnv("MOMO_ENDPOINT", "https://test-payment.momo.vn/v2/gateway/api"),
		},
		Webhook: WebhookConfig{
			BaseURL: getEnv("WEBHOOK_BASE_URL", "http://localhost:8080/api/v1/webhooks"),
		},
		Job: JobConfig{
			ReconciliationSchedule: getEnv("RECONCILIATION_SCHEDULE", "0 */10 * * * *"), // Every 10 minutes (with seconds)
			ReconciliationTimeout:  getEnvDuration("RECONCILIATION_TIMEOUT", 5*time.Minute),
			ReconciliationOlderThan: getEnvDuration("RECONCILIATION_OLDER_THAN", 10*time.Minute),
		},
	}

	return cfg, nil
}

// Address returns the server address
func (c *ServerConfig) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt gets an environment variable as int with a default value
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvDuration gets an environment variable as duration with a default value
func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
