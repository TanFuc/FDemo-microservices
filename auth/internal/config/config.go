package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	App      AppConfig
	Database DatabaseConfig
	Redis    RedisConfig
	JWT      JWTConfig
	NATS     NATSConfig
	Cookie   CookieConfig
}

type AppConfig struct {
	Name        string
	Env         string
	Port        string
	APIPrefix   string
	CORSOrigins []string
	Debug       bool
}

type DatabaseConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	Name            string
	SSLMode         string
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
	LogLevel        string
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type JWTConfig struct {
	AccessSecret      string
	RefreshSecret     string
	AccessExpiry      time.Duration
	RefreshExpiry     time.Duration
	RefreshExpiryDays int
}

type NATSConfig struct {
	URL string
}

type CookieConfig struct {
	Domain   string
	Secure   bool
	SameSite string
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
			Name:        viper.GetString("APP_NAME"),
			Env:         viper.GetString("NODE_ENV"),
			Port:        viper.GetString("PORT"),
			APIPrefix:   viper.GetString("API_PREFIX"),
			CORSOrigins: strings.Split(viper.GetString("CORS_ORIGIN"), ","),
			Debug:       viper.GetBool("DEBUG"),
		},
		Database: DatabaseConfig{
			Host:            viper.GetString("DB_HOST"),
			Port:            viper.GetString("DB_PORT"),
			User:            viper.GetString("DB_USERNAME"),
			Password:        viper.GetString("DB_PASSWORD"),
			Name:            viper.GetString("DB_DATABASE"),
			SSLMode:         viper.GetString("DB_SSL_MODE"),
			MaxIdleConns:    viper.GetInt("DB_MAX_IDLE_CONNS"),
			MaxOpenConns:    viper.GetInt("DB_MAX_OPEN_CONNS"),
			ConnMaxLifetime: viper.GetDuration("DB_CONN_MAX_LIFETIME"),
			LogLevel:        viper.GetString("DB_LOG_LEVEL"),
		},
		Redis: RedisConfig{
			Host:     viper.GetString("REDIS_HOST"),
			Port:     viper.GetString("REDIS_PORT"),
			Password: viper.GetString("REDIS_PASSWORD"),
			DB:       viper.GetInt("REDIS_DB"),
		},
		JWT: JWTConfig{
			AccessSecret:      viper.GetString("JWT_ACCESS_SECRET"),
			RefreshSecret:     viper.GetString("JWT_REFRESH_SECRET"),
			AccessExpiry:      parseDuration(viper.GetString("JWT_ACCESS_EXPIRY"), 15*time.Minute),
			RefreshExpiry:     parseDuration(viper.GetString("JWT_REFRESH_EXPIRY"), 7*24*time.Hour),
			RefreshExpiryDays: 7,
		},
		NATS: NATSConfig{
			URL: viper.GetString("NATS_URL"),
		},
		Cookie: CookieConfig{
			Domain:   viper.GetString("COOKIE_DOMAIN"),
			Secure:   viper.GetBool("COOKIE_SECURE"),
			SameSite: viper.GetString("COOKIE_SAMESITE"),
		},
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

func setDefaults() {
	viper.SetDefault("APP_NAME", "tafu-auth")
	viper.SetDefault("NODE_ENV", "development")
	viper.SetDefault("PORT", "3001")
	viper.SetDefault("API_PREFIX", "api/v1")
	viper.SetDefault("CORS_ORIGIN", "*")
	viper.SetDefault("DEBUG", false)

	viper.SetDefault("DB_HOST", "localhost")
	viper.SetDefault("DB_PORT", "5432")
	viper.SetDefault("DB_SSL_MODE", "disable")
	viper.SetDefault("DB_MAX_IDLE_CONNS", 10)
	viper.SetDefault("DB_MAX_OPEN_CONNS", 100)
	viper.SetDefault("DB_CONN_MAX_LIFETIME", "1h")
	viper.SetDefault("DB_LOG_LEVEL", "warn")

	viper.SetDefault("REDIS_HOST", "localhost")
	viper.SetDefault("REDIS_PORT", "6379")
	viper.SetDefault("REDIS_DB", 0)

	viper.SetDefault("JWT_ACCESS_EXPIRY", "15m")
	viper.SetDefault("JWT_REFRESH_EXPIRY", "7d")

	viper.SetDefault("COOKIE_DOMAIN", "")
	viper.SetDefault("COOKIE_SECURE", false)
	viper.SetDefault("COOKIE_SAMESITE", "Lax")

	viper.SetDefault("NATS_URL", "nats://localhost:4222")
}

func parseDuration(s string, defaultVal time.Duration) time.Duration {
	if s == "" {
		return defaultVal
	}

	// Handle day notation (e.g., "7d")
	if strings.HasSuffix(s, "d") {
		days := strings.TrimSuffix(s, "d")
		var d int
		if _, err := fmt.Sscanf(days, "%d", &d); err == nil {
			return time.Duration(d) * 24 * time.Hour
		}
	}

	d, err := time.ParseDuration(s)
	if err != nil {
		return defaultVal
	}
	return d
}

func (c *Config) Validate() error {
	if c.JWT.AccessSecret == "" {
		return fmt.Errorf("JWT_ACCESS_SECRET is required")
	}
	if len(c.JWT.AccessSecret) < 32 {
		return fmt.Errorf("JWT_ACCESS_SECRET must be at least 32 characters")
	}
	if c.JWT.RefreshSecret == "" {
		return fmt.Errorf("JWT_REFRESH_SECRET is required")
	}
	if len(c.JWT.RefreshSecret) < 32 {
		return fmt.Errorf("JWT_REFRESH_SECRET must be at least 32 characters")
	}
	if c.Database.User == "" {
		return fmt.Errorf("DB_USERNAME is required")
	}
	if c.Database.Password == "" {
		return fmt.Errorf("DB_PASSWORD is required")
	}
	if c.Database.Name == "" {
		return fmt.Errorf("DB_DATABASE is required")
	}
	return nil
}

func (c *Config) GetDSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.Name,
		c.Database.SSLMode,
	)
}

func (c *Config) GetRedisAddr() string {
	return fmt.Sprintf("%s:%s", c.Redis.Host, c.Redis.Port)
}

func (c *Config) IsDevelopment() bool {
	return c.App.Env == "development"
}

func (c *Config) IsProduction() bool {
	return c.App.Env == "production"
}
