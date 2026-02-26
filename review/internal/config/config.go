package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	App     AppConfig     `mapstructure:"app"`
	MongoDB MongoDBConfig `mapstructure:"mongodb"`
	Redis   RedisConfig   `mapstructure:"redis"`
	GRPC    GRPCConfig    `mapstructure:"grpc"`
	NATS    NATSConfig    `mapstructure:"nats"`
}

type AppConfig struct {
	Name         string `mapstructure:"name"`
	Env          string `mapstructure:"env"`
	Port         string `mapstructure:"port"`
	Host         string `mapstructure:"host"`
	AuthGRPCAddr string `mapstructure:"auth_grpc_addr"`
}

type MongoDBConfig struct {
	URI      string `mapstructure:"uri"`
	Database string `mapstructure:"database"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type GRPCConfig struct {
	OrderServiceAddr string `mapstructure:"order_service_addr"`
}

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
	v.SetDefault("app.name", "review-service")
	v.SetDefault("app.env", "development")
	v.SetDefault("app.port", "8085")
	v.SetDefault("app.host", "0.0.0.0")
	v.SetDefault("app.auth_grpc_addr", "localhost:50051")

	// MongoDB defaults
	v.SetDefault("mongodb.uri", "mongodb://localhost:27017")
	v.SetDefault("mongodb.database", "tafu_review")

	// Redis defaults
	v.SetDefault("redis.addr", "localhost:6379")
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)

	// GRPC defaults
	v.SetDefault("grpc.order_service_addr", "localhost:50052")

	// NATS defaults
	v.SetDefault("nats.url", "nats://localhost:4222")
}

func bindEnvVars(v *viper.Viper) {
	// App
	v.BindEnv("app.name", "APP_NAME")
	v.BindEnv("app.env", "APP_ENV")
	v.BindEnv("app.port", "APP_PORT", "SERVER_PORT")
	v.BindEnv("app.host", "APP_HOST", "SERVER_HOST")
	v.BindEnv("app.auth_grpc_addr", "AUTH_GRPC_ADDR")

	// MongoDB
	v.BindEnv("mongodb.uri", "MONGODB_URI")
	v.BindEnv("mongodb.database", "MONGODB_DATABASE")

	// Redis
	v.BindEnv("redis.addr", "REDIS_ADDR")
	v.BindEnv("redis.password", "REDIS_PASSWORD")
	v.BindEnv("redis.db", "REDIS_DB")

	// GRPC
	v.BindEnv("grpc.order_service_addr", "ORDER_SERVICE_ADDR")

	// NATS
	v.BindEnv("nats.url", "NATS_URL")
}
