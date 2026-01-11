package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	App     AppConfig
	MongoDB MongoDBConfig
	NATS    NATSConfig
}

type AppConfig struct {
	Name              string
	Env               string
	Port              string
	APIPrefix         string
	CORSOrigins       []string
	Debug             bool
	InternalServiceKey string
}

type MongoDBConfig struct {
	URI            string
	Database       string
	ConnectTimeout time.Duration
	MaxPoolSize    uint64
	MinPoolSize    uint64
}

type NATSConfig struct {
	URL string
}

func Load() (*Config, error) {
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	viper.AddConfigPath("..")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Set defaults
	setDefaults()

	// Read config file (optional)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	config := &Config{
		App: AppConfig{
			Name:              viper.GetString("APP_NAME"),
			Env:               viper.GetString("NODE_ENV"),
			Port:              viper.GetString("PORT"),
			APIPrefix:         viper.GetString("API_PREFIX"),
			CORSOrigins:       strings.Split(viper.GetString("CORS_ORIGIN"), ","),
			Debug:             viper.GetBool("DEBUG"),
			InternalServiceKey: viper.GetString("INTERNAL_SERVICE_KEY"),
		},
		MongoDB: MongoDBConfig{
			URI:            viper.GetString("MONGODB_URI"),
			Database:       viper.GetString("MONGODB_DATABASE"),
			ConnectTimeout: viper.GetDuration("MONGODB_CONNECT_TIMEOUT"),
			MaxPoolSize:    viper.GetUint64("MONGODB_MAX_POOL_SIZE"),
			MinPoolSize:    viper.GetUint64("MONGODB_MIN_POOL_SIZE"),
		},
		NATS: NATSConfig{
			URL: viper.GetString("NATS_URL"),
		},
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

func setDefaults() {
	viper.SetDefault("APP_NAME", "tafu-profile")
	viper.SetDefault("NODE_ENV", "development")
	viper.SetDefault("PORT", "3002")
	viper.SetDefault("API_PREFIX", "api/v1")
	viper.SetDefault("CORS_ORIGIN", "*")
	viper.SetDefault("DEBUG", false)

	viper.SetDefault("MONGODB_URI", "mongodb://localhost:27017")
	viper.SetDefault("MONGODB_DATABASE", "tafu_profile")
	viper.SetDefault("MONGODB_CONNECT_TIMEOUT", "10s")
	viper.SetDefault("MONGODB_MAX_POOL_SIZE", 100)
	viper.SetDefault("MONGODB_MIN_POOL_SIZE", 10)

	viper.SetDefault("NATS_URL", "nats://localhost:4222")
}

func (c *Config) Validate() error {
	if c.MongoDB.URI == "" {
		return fmt.Errorf("MONGODB_URI is required")
	}
	return nil
}

func (c *Config) IsDevelopment() bool {
	return c.App.Env == "development"
}

func (c *Config) IsProduction() bool {
	return c.App.Env == "production"
}
