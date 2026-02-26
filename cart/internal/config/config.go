package config

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	App      AppConfig
	Redis    RedisConfig
	Mongo    MongoConfig
	Auth     AuthConfig
	Campaign CampaignServiceConfig
	Order    OrderServiceConfig
}

type CampaignServiceConfig struct {
	HTTPURL    string
	Timeout    time.Duration
	ServiceKey string
}

type OrderServiceConfig struct {
	HTTPURL    string
	Timeout    time.Duration
	ServiceKey string
}

type AppConfig struct {
	Name     string
	Env      string
	Port     string
	GRPCPort string
	Debug    bool
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
	PoolSize int
}

type MongoConfig struct {
	URI      string
	Database string
	Timeout  int
}

type AuthConfig struct {
	GRPCAddr string
	Timeout  time.Duration
}

func (r *RedisConfig) Addr() string {
	return r.Host + ":" + r.Port
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
		Redis: RedisConfig{
			Host:     viper.GetString("redis.host"),
			Port:     viper.GetString("redis.port"),
			Password: viper.GetString("redis.password"),
			DB:       viper.GetInt("redis.db"),
			PoolSize: viper.GetInt("redis.pool_size"),
		},
		Mongo: MongoConfig{
			URI:      viper.GetString("mongo.uri"),
			Database: viper.GetString("mongo.database"),
			Timeout:  viper.GetInt("mongo.timeout"),
		},
		Auth: AuthConfig{
			GRPCAddr: viper.GetString("auth.grpc_addr"),
			Timeout:  viper.GetDuration("auth.timeout"),
		},
		Campaign: CampaignServiceConfig{
			HTTPURL:    viper.GetString("campaign_service.http_url"),
			Timeout:    viper.GetDuration("campaign_service.timeout"),
			ServiceKey: viper.GetString("campaign_service.service_key"),
		},
		Order: OrderServiceConfig{
			HTTPURL:    viper.GetString("order_service.http_url"),
			Timeout:    viper.GetDuration("order_service.timeout"),
			ServiceKey: viper.GetString("order_service.service_key"),
		},
	}

	return cfg, nil
}

func setDefaults() {
	viper.SetDefault("app.name", "cart-service")
	viper.SetDefault("app.env", "development")
	viper.SetDefault("app.port", "8082")
	viper.SetDefault("app.grpc_port", "50053")
	viper.SetDefault("app.debug", true)

	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", "6379")
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)
	viper.SetDefault("redis.pool_size", 10)

	viper.SetDefault("mongo.uri", "mongodb://localhost:27017")
	viper.SetDefault("mongo.database", "cart_db")
	viper.SetDefault("mongo.timeout", 10)

	viper.SetDefault("auth.grpc_addr", "localhost:50051")
	viper.SetDefault("auth.timeout", "3s")

	viper.SetDefault("campaign_service.http_url", "http://localhost:8083")
	viper.SetDefault("campaign_service.timeout", "5s")
	viper.SetDefault("campaign_service.service_key", "")

	viper.SetDefault("order_service.http_url", "http://localhost:8084")
	viper.SetDefault("order_service.timeout", "10s")
	viper.SetDefault("order_service.service_key", "")
}
