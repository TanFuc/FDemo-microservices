package config

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"
)

// Config holds all application configuration
type Config struct {
	// Server configuration
	Port string `envconfig:"PORT" default:"8080"`

	// Database configuration
	DatabaseURL string `envconfig:"DATABASE_URL" required:"true"`

	// Redis configuration
	RedisURL string `envconfig:"REDIS_URL" required:"true"`

	// NATS configuration
	NATSURL        string `envconfig:"NATS_URL" required:"true"`
	NATSStreamName string `envconfig:"NATS_STREAM_NAME" default:"LOGISTICS"`

	// GHN Provider configuration
	GHNAPIURL string `envconfig:"GHN_API_URL" default:"https://dev-online-gateway.ghn.vn"`
	GHNToken  string `envconfig:"GHN_TOKEN"`
	GHNShopID string `envconfig:"GHN_SHOP_ID"`

	// GHTK Provider configuration
	GHTKAPIURL string `envconfig:"GHTK_API_URL" default:"https://services.giaohangtietkiem.vn"`
	GHTKToken  string `envconfig:"GHTK_TOKEN"`
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	return &cfg, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if c.RedisURL == "" {
		return fmt.Errorf("REDIS_URL is required")
	}
	if c.NATSURL == "" {
		return fmt.Errorf("NATS_URL is required")
	}
	return nil
}

// HasGHNCredentials checks if GHN credentials are configured
func (c *Config) HasGHNCredentials() bool {
	return c.GHNToken != "" && c.GHNShopID != ""
}

// HasGHTKCredentials checks if GHTK credentials are configured
func (c *Config) HasGHTKCredentials() bool {
	return c.GHTKToken != ""
}
