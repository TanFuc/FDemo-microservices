package config

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	App     AppConfig
	MongoDB MongoDBConfig
	Redis   RedisConfig
	NATS    NATSConfig
	Auth    AuthConfig
}

type AppConfig struct {
	Name         string
	Env          string
	Port         string
	GRPCPort     string
	Debug        bool
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type MongoDBConfig struct {
	URI      string
	Database string
	Timeout  time.Duration
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type NATSConfig struct {
	URL        string
	StreamName string
}

type AuthConfig struct {
	GRPCAddr string
	Timeout  time.Duration
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Set defaults
	setDefaults()

	// Try to read config file (optional)
	_ = viper.ReadInConfig()

	cfg := &Config{
		App: AppConfig{
			Name:         viper.GetString("app.name"),
			Env:          viper.GetString("app.env"),
			Port:         viper.GetString("app.port"),
			GRPCPort:     viper.GetString("app.grpc_port"),
			Debug:        viper.GetBool("app.debug"),
			ReadTimeout:  viper.GetDuration("app.read_timeout"),
			WriteTimeout: viper.GetDuration("app.write_timeout"),
		},
		MongoDB: MongoDBConfig{
			URI:      viper.GetString("mongodb.uri"),
			Database: viper.GetString("mongodb.database"),
			Timeout:  viper.GetDuration("mongodb.timeout"),
		},
		Redis: RedisConfig{
			Addr:     viper.GetString("redis.addr"),
			Password: viper.GetString("redis.password"),
			DB:       viper.GetInt("redis.db"),
		},
		NATS: NATSConfig{
			URL:        viper.GetString("nats.url"),
			StreamName: viper.GetString("nats.stream_name"),
		},
		Auth: AuthConfig{
			GRPCAddr: viper.GetString("auth.grpc_addr"),
			Timeout:  viper.GetDuration("auth.timeout"),
		},
	}

	return cfg, nil
}

func setDefaults() {
	// App defaults
	viper.SetDefault("app.name", "catalog-service")
	viper.SetDefault("app.env", "development")
	viper.SetDefault("app.port", "8082")
	viper.SetDefault("app.grpc_port", "50052")
	viper.SetDefault("app.debug", true)
	viper.SetDefault("app.read_timeout", "10s")
	viper.SetDefault("app.write_timeout", "10s")

	// MongoDB defaults
	viper.SetDefault("mongodb.uri", "mongodb://localhost:27017")
	viper.SetDefault("mongodb.database", "catalog")
	viper.SetDefault("mongodb.timeout", "10s")

	// Redis defaults
	viper.SetDefault("redis.addr", "localhost:6379")
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)

	// NATS defaults
	viper.SetDefault("nats.url", "nats://localhost:4222")
	viper.SetDefault("nats.stream_name", "CATALOG")

	// Auth defaults
	viper.SetDefault("auth.grpc_addr", "localhost:50051")
	viper.SetDefault("auth.timeout", "3s")
}
