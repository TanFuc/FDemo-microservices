package config

import (
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	App      AppConfig
	Database DatabaseConfig
}

type AppConfig struct {
	Name     string
	Env      string
	Port     string
	GRPCPort string
	Debug    bool
}

type DatabaseConfig struct {
	Host         string
	Port         string
	User         string
	Password     string
	Name         string
	SSLMode      string
	MaxIdleConns int
	MaxOpenConns int
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
			Name:     viper.GetString("app.name"),
			Env:      viper.GetString("app.env"),
			Port:     viper.GetString("app.port"),
			GRPCPort: viper.GetString("app.grpc_port"),
			Debug:    viper.GetBool("app.debug"),
		},
		Database: DatabaseConfig{
			Host:         viper.GetString("database.host"),
			Port:         viper.GetString("database.port"),
			User:         viper.GetString("database.user"),
			Password:     viper.GetString("database.password"),
			Name:         viper.GetString("database.name"),
			SSLMode:      viper.GetString("database.ssl_mode"),
			MaxIdleConns: viper.GetInt("database.max_idle_conns"),
			MaxOpenConns: viper.GetInt("database.max_open_conns"),
		},
	}

	return cfg, nil
}

func setDefaults() {
	// App defaults
	viper.SetDefault("app.name", "template-service")
	viper.SetDefault("app.env", "development")
	viper.SetDefault("app.port", "8080")
	viper.SetDefault("app.grpc_port", "50051")
	viper.SetDefault("app.debug", true)

	// Database defaults
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", "5432")
	viper.SetDefault("database.user", "postgres")
	viper.SetDefault("database.password", "postgres")
	viper.SetDefault("database.name", "template_service")
	viper.SetDefault("database.ssl_mode", "disable")
	viper.SetDefault("database.max_idle_conns", 10)
	viper.SetDefault("database.max_open_conns", 100)
}
