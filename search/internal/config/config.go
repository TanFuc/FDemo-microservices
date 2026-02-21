package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	App           AppConfig           `mapstructure:"app"`
	Elasticsearch ElasticsearchConfig `mapstructure:"elasticsearch"`
	Redis         RedisConfig         `mapstructure:"redis"`
	NATS          NATSConfig          `mapstructure:"nats"`
}

// AppConfig holds application configuration
type AppConfig struct {
	Name         string `mapstructure:"name"`
	Env          string `mapstructure:"env"`
	Port         string `mapstructure:"port"`
	Host         string `mapstructure:"host"`
	AuthGRPCAddr string `mapstructure:"auth_grpc_addr"`
}

// ElasticsearchConfig holds Elasticsearch configuration
type ElasticsearchConfig struct {
	Addresses []string `mapstructure:"addresses"`
	Username  string   `mapstructure:"username"`
	Password  string   `mapstructure:"password"`
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// NATSConfig holds NATS configuration
type NATSConfig struct {
	URL string `mapstructure:"url"`
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
	v.SetDefault("app.name", "search-service")
	v.SetDefault("app.env", "development")
	v.SetDefault("app.port", "8086")
	v.SetDefault("app.host", "0.0.0.0")
	v.SetDefault("app.auth_grpc_addr", "localhost:50051")

	// Elasticsearch defaults
	v.SetDefault("elasticsearch.addresses", []string{"http://localhost:9200"})
	v.SetDefault("elasticsearch.username", "")
	v.SetDefault("elasticsearch.password", "")

	// Redis defaults
	v.SetDefault("redis.addr", "localhost:6379")
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)

	// NATS defaults
	v.SetDefault("nats.url", "nats://localhost:4222")
}

func bindEnvVars(v *viper.Viper) {
	// App
	v.BindEnv("app.name", "APP_NAME")
	v.BindEnv("app.env", "APP_ENV")
	v.BindEnv("app.port", "APP_PORT", "API_PORT")
	v.BindEnv("app.host", "APP_HOST")
	v.BindEnv("app.auth_grpc_addr", "AUTH_GRPC_ADDR")

	// Elasticsearch
	v.BindEnv("elasticsearch.addresses", "ELASTIC_ADDRESSES")
	v.BindEnv("elasticsearch.username", "ELASTIC_USERNAME")
	v.BindEnv("elasticsearch.password", "ELASTIC_PASSWORD")

	// Redis
	v.BindEnv("redis.addr", "REDIS_ADDR")
	v.BindEnv("redis.password", "REDIS_PASSWORD")
	v.BindEnv("redis.db", "REDIS_DB")

	// NATS
	v.BindEnv("nats.url", "NATS_URL")
}
