package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all configuration for the service
type Config struct {
	App       AppConfig       `mapstructure:"app"`
	Database  DatabaseConfig  `mapstructure:"database"`
	NATS      NATSConfig      `mapstructure:"nats"`
	Redis     RedisConfig     `mapstructure:"redis"`
	Inventory InventoryConfig `mapstructure:"inventory"`
}

// AppConfig holds application configuration
type AppConfig struct {
	Name         string `mapstructure:"name"`
	Env          string `mapstructure:"env"`
	Port         string `mapstructure:"port"`
	Host         string `mapstructure:"host"`
	AuthGRPCAddr string `mapstructure:"auth_grpc_addr"`
}

// DatabaseConfig holds PostgreSQL configuration
type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
}

// NATSConfig holds NATS JetStream configuration
type NATSConfig struct {
	URL        string `mapstructure:"url"`
	StreamName string `mapstructure:"stream_name"`
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	URL      string `mapstructure:"url"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// InventoryConfig holds Inventory gRPC service configuration
type InventoryConfig struct {
	GRPCAddress string `mapstructure:"grpc_address"`
	Timeout     int    `mapstructure:"timeout"`
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
	v.SetDefault("app.name", "order-service")
	v.SetDefault("app.env", "development")
	v.SetDefault("app.port", "8082")
	v.SetDefault("app.host", "0.0.0.0")
	v.SetDefault("app.auth_grpc_addr", "localhost:50051")

	// Database defaults
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", "5432")
	v.SetDefault("database.user", "postgres")
	v.SetDefault("database.password", "postgres")
	v.SetDefault("database.dbname", "order_service")
	v.SetDefault("database.sslmode", "disable")

	// NATS defaults
	v.SetDefault("nats.url", "nats://localhost:4222")
	v.SetDefault("nats.stream_name", "ORDERS")

	// Redis defaults
	v.SetDefault("redis.url", "localhost:6379")
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)

	// Inventory defaults
	v.SetDefault("inventory.grpc_address", "localhost:50053")
	v.SetDefault("inventory.timeout", 5)
}

func bindEnvVars(v *viper.Viper) {
	// App
	v.BindEnv("app.name", "APP_NAME")
	v.BindEnv("app.env", "APP_ENV")
	v.BindEnv("app.port", "APP_PORT", "SERVER_PORT")
	v.BindEnv("app.host", "APP_HOST", "SERVER_HOST")
	v.BindEnv("app.auth_grpc_addr", "AUTH_GRPC_ADDR")

	// Database
	v.BindEnv("database.host", "DB_HOST")
	v.BindEnv("database.port", "DB_PORT")
	v.BindEnv("database.user", "DB_USER")
	v.BindEnv("database.password", "DB_PASSWORD")
	v.BindEnv("database.dbname", "DB_NAME")
	v.BindEnv("database.sslmode", "DB_SSLMODE")

	// NATS
	v.BindEnv("nats.url", "NATS_URL")
	v.BindEnv("nats.stream_name", "NATS_STREAM_NAME")

	// Redis
	v.BindEnv("redis.url", "REDIS_URL")
	v.BindEnv("redis.password", "REDIS_PASSWORD")
	v.BindEnv("redis.db", "REDIS_DB")

	// Inventory
	v.BindEnv("inventory.grpc_address", "INVENTORY_GRPC_ADDRESS")
	v.BindEnv("inventory.timeout", "INVENTORY_TIMEOUT")
}

// DSN returns the PostgreSQL connection string
func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode)
}
