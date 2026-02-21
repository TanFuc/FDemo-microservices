package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all application configuration
type Config struct {
	App      AppConfig      `mapstructure:"app"`
	Database DatabaseConfig `mapstructure:"database"`
	NATS     NATSConfig     `mapstructure:"nats"`
	Stripe   StripeConfig   `mapstructure:"stripe"`
	MoMo     MoMoConfig     `mapstructure:"momo"`
	Webhook  WebhookConfig  `mapstructure:"webhook"`
	Job      JobConfig      `mapstructure:"job"`
}

// AppConfig holds application configuration
type AppConfig struct {
	Name         string `mapstructure:"name"`
	Env          string `mapstructure:"env"`
	Port         string `mapstructure:"port"`
	Host         string `mapstructure:"host"`
	AuthGRPCAddr string `mapstructure:"auth_grpc_addr"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	URL string `mapstructure:"url"`
}

// NATSConfig holds NATS configuration
type NATSConfig struct {
	URL string `mapstructure:"url"`
}

// StripeConfig holds Stripe configuration
type StripeConfig struct {
	SecretKey     string `mapstructure:"secret_key"`
	WebhookSecret string `mapstructure:"webhook_secret"`
}

// MoMoConfig holds MoMo configuration
type MoMoConfig struct {
	PartnerCode string `mapstructure:"partner_code"`
	AccessKey   string `mapstructure:"access_key"`
	SecretKey   string `mapstructure:"secret_key"`
	Endpoint    string `mapstructure:"endpoint"`
}

// WebhookConfig holds webhook configuration
type WebhookConfig struct {
	BaseURL string `mapstructure:"base_url"`
}

// JobConfig holds background job configuration
type JobConfig struct {
	ReconciliationSchedule  string        `mapstructure:"reconciliation_schedule"`
	ReconciliationTimeout   time.Duration `mapstructure:"reconciliation_timeout"`
	ReconciliationOlderThan time.Duration `mapstructure:"reconciliation_older_than"`
}

// Load loads configuration from file and environment variables
func Load() (*Config, error) {
	v := viper.New()

	// Set config file
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")
	v.AddConfigPath("../")

	// Set defaults
	setDefaults(v)

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found, using defaults and env vars
	}

	// Enable environment variable override
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Bind environment variables
	bindEnvVars(v)

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	// App defaults
	v.SetDefault("app.name", "payment-service")
	v.SetDefault("app.env", "development")
	v.SetDefault("app.port", "8083")
	v.SetDefault("app.host", "0.0.0.0")
	v.SetDefault("app.auth_grpc_addr", "localhost:50051")

	// Database defaults
	v.SetDefault("database.url", "postgres://payment:payment_secret@localhost:5432/payment_db?sslmode=disable")

	// NATS defaults
	v.SetDefault("nats.url", "nats://localhost:4222")

	// Stripe defaults
	v.SetDefault("stripe.secret_key", "")
	v.SetDefault("stripe.webhook_secret", "")

	// MoMo defaults
	v.SetDefault("momo.partner_code", "")
	v.SetDefault("momo.access_key", "")
	v.SetDefault("momo.secret_key", "")
	v.SetDefault("momo.endpoint", "https://test-payment.momo.vn/v2/gateway/api")

	// Webhook defaults
	v.SetDefault("webhook.base_url", "http://localhost:8083/api/v1/webhooks")

	// Job defaults
	v.SetDefault("job.reconciliation_schedule", "0 */10 * * * *")
	v.SetDefault("job.reconciliation_timeout", "5m")
	v.SetDefault("job.reconciliation_older_than", "10m")
}

func bindEnvVars(v *viper.Viper) {
	// App
	v.BindEnv("app.name", "APP_NAME")
	v.BindEnv("app.env", "APP_ENV")
	v.BindEnv("app.port", "APP_PORT", "SERVER_PORT")
	v.BindEnv("app.host", "APP_HOST", "SERVER_HOST")
	v.BindEnv("app.auth_grpc_addr", "AUTH_GRPC_ADDR")

	// Database
	v.BindEnv("database.url", "DATABASE_URL")

	// NATS
	v.BindEnv("nats.url", "NATS_URL")

	// Stripe
	v.BindEnv("stripe.secret_key", "STRIPE_SECRET_KEY")
	v.BindEnv("stripe.webhook_secret", "STRIPE_WEBHOOK_SECRET")

	// MoMo
	v.BindEnv("momo.partner_code", "MOMO_PARTNER_CODE")
	v.BindEnv("momo.access_key", "MOMO_ACCESS_KEY")
	v.BindEnv("momo.secret_key", "MOMO_SECRET_KEY")
	v.BindEnv("momo.endpoint", "MOMO_ENDPOINT")

	// Webhook
	v.BindEnv("webhook.base_url", "WEBHOOK_BASE_URL")

	// Job
	v.BindEnv("job.reconciliation_schedule", "RECONCILIATION_SCHEDULE")
	v.BindEnv("job.reconciliation_timeout", "RECONCILIATION_TIMEOUT")
	v.BindEnv("job.reconciliation_older_than", "RECONCILIATION_OLDER_THAN")
}

// Address returns the server address
func (c *AppConfig) Address() string {
	return fmt.Sprintf("%s:%s", c.Host, c.Port)
}
