package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	App      AppConfig
	Postgres PostgresConfig
	Redis    RedisConfig
	Auth     AuthConfig
}

type AppConfig struct {
	Name     string
	Env      string
	Port     string
	GRPCPort string
	Debug    bool
}

type PostgresConfig struct {
	Host         string
	Port         string
	User         string
	Password     string
	Database     string
	SSLMode      string
	MaxIdleConns int
	MaxOpenConns int
}

func (p *PostgresConfig) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		p.User, p.Password, p.Host, p.Port, p.Database, p.SSLMode)
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
	PoolSize int
}

func (r *RedisConfig) Addr() string {
	return r.Host + ":" + r.Port
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

	setDefaults()

	_ = viper.ReadInConfig()

	cfg := &Config{
		App: AppConfig{
			Name:     viper.GetString("app.name"),
			Env:      viper.GetString("app.env"),
			Port:     viper.GetString("app.port"),
			GRPCPort: viper.GetString("app.grpc_port"),
			Debug:    viper.GetBool("app.debug"),
		},
		Postgres: PostgresConfig{
			Host:         viper.GetString("postgres.host"),
			Port:         viper.GetString("postgres.port"),
			User:         viper.GetString("postgres.user"),
			Password:     viper.GetString("postgres.password"),
			Database:     viper.GetString("postgres.database"),
			SSLMode:      viper.GetString("postgres.ssl_mode"),
			MaxIdleConns: viper.GetInt("postgres.max_idle_conns"),
			MaxOpenConns: viper.GetInt("postgres.max_open_conns"),
		},
		Redis: RedisConfig{
			Host:     viper.GetString("redis.host"),
			Port:     viper.GetString("redis.port"),
			Password: viper.GetString("redis.password"),
			DB:       viper.GetInt("redis.db"),
			PoolSize: viper.GetInt("redis.pool_size"),
		},
		Auth: AuthConfig{
			GRPCAddr: viper.GetString("auth.grpc_addr"),
			Timeout:  viper.GetDuration("auth.timeout"),
		},
	}

	return cfg, nil
}

func setDefaults() {
	viper.SetDefault("app.name", "campaign-service")
	viper.SetDefault("app.env", "development")
	viper.SetDefault("app.port", "8083")
	viper.SetDefault("app.grpc_port", "50054")
	viper.SetDefault("app.debug", true)

	viper.SetDefault("postgres.host", "localhost")
	viper.SetDefault("postgres.port", "5432")
	viper.SetDefault("postgres.user", "postgres")
	viper.SetDefault("postgres.password", "postgres")
	viper.SetDefault("postgres.database", "campaign_db")
	viper.SetDefault("postgres.ssl_mode", "disable")
	viper.SetDefault("postgres.max_idle_conns", 10)
	viper.SetDefault("postgres.max_open_conns", 100)

	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", "6379")
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 1)
	viper.SetDefault("redis.pool_size", 10)

	viper.SetDefault("auth.grpc_addr", "localhost:50051")
	viper.SetDefault("auth.timeout", "3s")
}
