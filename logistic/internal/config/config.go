package config

import (
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	App      AppConfig
	Database DatabaseConfig
	Redis    RedisConfig
	NATS     NATSConfig
	GHN      GHNConfig
	GHTK     GHTKConfig
}

type AppConfig struct {
	Name         string
	Env          string
	Port         string
	GRPCPort     string
	AuthGRPCAddr string
	Debug        bool
}

type DatabaseConfig struct {
	URL          string
	MaxIdleConns int
	MaxOpenConns int
}

type RedisConfig struct {
	URL string
}

type NATSConfig struct {
	URL        string
	StreamName string
}

type GHNConfig struct {
	APIURL string
	Token  string
	ShopID string
}

type GHTKConfig struct {
	APIURL string
	Token  string
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
			AuthGRPCAddr: viper.GetString("app.auth_grpc_addr"),
			Debug:        viper.GetBool("app.debug"),
		},
		Database: DatabaseConfig{
			URL:          viper.GetString("database.url"),
			MaxIdleConns: viper.GetInt("database.max_idle_conns"),
			MaxOpenConns: viper.GetInt("database.max_open_conns"),
		},
		Redis: RedisConfig{
			URL: viper.GetString("redis.url"),
		},
		NATS: NATSConfig{
			URL:        viper.GetString("nats.url"),
			StreamName: viper.GetString("nats.stream_name"),
		},
		GHN: GHNConfig{
			APIURL: viper.GetString("ghn.api_url"),
			Token:  viper.GetString("ghn.token"),
			ShopID: viper.GetString("ghn.shop_id"),
		},
		GHTK: GHTKConfig{
			APIURL: viper.GetString("ghtk.api_url"),
			Token:  viper.GetString("ghtk.token"),
		},
	}

	return cfg, nil
}

func setDefaults() {
	// App defaults
	viper.SetDefault("app.name", "logistic-service")
	viper.SetDefault("app.env", "development")
	viper.SetDefault("app.port", "8084")
	viper.SetDefault("app.grpc_port", "50054")
	viper.SetDefault("app.auth_grpc_addr", "localhost:50051")
	viper.SetDefault("app.debug", true)

	// Database defaults
	viper.SetDefault("database.url", "postgres://postgres:postgres@localhost:5432/logistic_service?sslmode=disable")
	viper.SetDefault("database.max_idle_conns", 10)
	viper.SetDefault("database.max_open_conns", 100)

	// Redis defaults
	viper.SetDefault("redis.url", "redis://localhost:6379")

	// NATS defaults
	viper.SetDefault("nats.url", "nats://localhost:4222")
	viper.SetDefault("nats.stream_name", "LOGISTICS")

	// GHN defaults
	viper.SetDefault("ghn.api_url", "https://dev-online-gateway.ghn.vn")
	viper.SetDefault("ghn.token", "")
	viper.SetDefault("ghn.shop_id", "")

	// GHTK defaults
	viper.SetDefault("ghtk.api_url", "https://services.giaohangtietkiem.vn")
	viper.SetDefault("ghtk.token", "")
}

// Validate validates the configuration
func (c *Config) Validate() error {
	return nil
}

// HasGHNCredentials checks if GHN credentials are configured
func (c *Config) HasGHNCredentials() bool {
	return c.GHN.Token != "" && c.GHN.ShopID != ""
}

// HasGHTKCredentials checks if GHTK credentials are configured
func (c *Config) HasGHTKCredentials() bool {
	return c.GHTK.Token != ""
}
