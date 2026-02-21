package config

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	App      AppConfig      `mapstructure:"app"`
	Postgres PostgresConfig `mapstructure:"postgres"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Auth     AuthConfig     `mapstructure:"auth"`
}

type AppConfig struct {
	Name           string        `mapstructure:"name"`
	Env            string        `mapstructure:"env"`
	Port           string        `mapstructure:"port"`
	GRPCPort       string        `mapstructure:"grpc_port"`
	Debug          bool          `mapstructure:"debug"`
	ReadTimeout    time.Duration `mapstructure:"read_timeout"`
	WriteTimeout   time.Duration `mapstructure:"write_timeout"`
	ReservationTTL time.Duration `mapstructure:"reservation_ttl"`
}

type PostgresConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"database"`
	SSLMode  string `mapstructure:"ssl_mode"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type AuthConfig struct {
	GRPCAddr string        `mapstructure:"grpc_addr"`
	Timeout  time.Duration `mapstructure:"timeout"`
}

func Load(path string) (*Config, error) {
	viper.SetConfigFile(path)
	viper.SetConfigType("yaml")

	viper.SetEnvPrefix("INVENTORY")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Set defaults
	setDefaults()

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func setDefaults() {
	// App defaults
	viper.SetDefault("app.name", "inventory-service")
	viper.SetDefault("app.env", "development")
	viper.SetDefault("app.port", "8083")
	viper.SetDefault("app.grpc_port", "50053")
	viper.SetDefault("app.debug", true)
	viper.SetDefault("app.read_timeout", "10s")
	viper.SetDefault("app.write_timeout", "10s")
	viper.SetDefault("app.reservation_ttl", "15m")

	// Postgres defaults
	viper.SetDefault("postgres.host", "localhost")
	viper.SetDefault("postgres.port", 5432)
	viper.SetDefault("postgres.user", "postgres")
	viper.SetDefault("postgres.password", "postgres")
	viper.SetDefault("postgres.database", "inventory")
	viper.SetDefault("postgres.ssl_mode", "disable")

	// Redis defaults
	viper.SetDefault("redis.addr", "localhost:6379")
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)

	// Auth defaults
	viper.SetDefault("auth.grpc_addr", "localhost:50051")
	viper.SetDefault("auth.timeout", "3s")
}

// GetReservationTTL returns the configured reservation TTL
func (c *Config) GetReservationTTL() time.Duration {
	if c.App.ReservationTTL > 0 {
		return c.App.ReservationTTL
	}
	return 15 * time.Minute
}
