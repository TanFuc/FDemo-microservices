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
	Name               string
	Env                string
	Port               string
	APIPrefix          string
	CORSOrigins        []string
	Debug              bool
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
	// Support both YAML config and env file
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Set defaults
	setDefaults()

	// Try to read config file (optional)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	config := &Config{
		App: AppConfig{
			Name:               viper.GetString("app.name"),
			Env:                viper.GetString("app.env"),
			Port:               viper.GetString("app.port"),
			APIPrefix:          viper.GetString("app.api_prefix"),
			CORSOrigins:        viper.GetStringSlice("app.cors_origins"),
			Debug:              viper.GetBool("app.debug"),
			InternalServiceKey: viper.GetString("app.internal_service_key"),
		},
		MongoDB: MongoDBConfig{
			URI:            viper.GetString("mongodb.uri"),
			Database:       viper.GetString("mongodb.database"),
			ConnectTimeout: viper.GetDuration("mongodb.connect_timeout"),
			MaxPoolSize:    viper.GetUint64("mongodb.max_pool_size"),
			MinPoolSize:    viper.GetUint64("mongodb.min_pool_size"),
		},
		NATS: NATSConfig{
			URL: viper.GetString("nats.url"),
		},
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

func setDefaults() {
	// App defaults
	viper.SetDefault("app.name", "profile-service")
	viper.SetDefault("app.env", "development")
	viper.SetDefault("app.port", "3002")
	viper.SetDefault("app.api_prefix", "api/v1")
	viper.SetDefault("app.cors_origins", []string{"*"})
	viper.SetDefault("app.debug", true)
	viper.SetDefault("app.internal_service_key", "")

	// MongoDB defaults
	viper.SetDefault("mongodb.uri", "mongodb://localhost:27017")
	viper.SetDefault("mongodb.database", "profile_service")
	viper.SetDefault("mongodb.connect_timeout", "10s")
	viper.SetDefault("mongodb.max_pool_size", 100)
	viper.SetDefault("mongodb.min_pool_size", 10)

	// NATS defaults
	viper.SetDefault("nats.url", "nats://localhost:4222")
}

func (c *Config) Validate() error {
	if c.MongoDB.URI == "" {
		return fmt.Errorf("mongodb.uri is required")
	}
	return nil
}

func (c *Config) IsDevelopment() bool {
	return c.App.Env == "development"
}

func (c *Config) IsProduction() bool {
	return c.App.Env == "production"
}
