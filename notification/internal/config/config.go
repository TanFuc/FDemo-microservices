package config

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	App       AppConfig
	NATS      NATSConfig
	RabbitMQ  RabbitMQConfig
	MongoDB   MongoDBConfig
	Redis     RedisConfig
	SMTP      SMTPConfig
	WebSocket WebSocketConfig
	Catalog   CatalogConfig
}

type AppConfig struct {
	Name         string
	Env          string
	Port         string
	GRPCPort     string
	AuthGRPCAddr string
	Debug        bool
}

type NATSConfig struct {
	URL        string
	StreamName string
	Subject    string
}

type RabbitMQConfig struct {
	URL string
}

type MongoDBConfig struct {
	URI      string
	Database string
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type CatalogConfig struct {
	URL        string
	ServiceKey string
}

type SMTPConfig struct {
	Host string
	Port int
	User string
	Pass string
	From string
}

type WebSocketConfig struct {
	Enabled         bool
	MaxConnections  int
	PingInterval    time.Duration
	PongWait        time.Duration
	WriteWait       time.Duration
	MaxMessageSize  int64
	ReadBufferSize  int
	WriteBufferSize int
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
		NATS: NATSConfig{
			URL:        viper.GetString("nats.url"),
			StreamName: viper.GetString("nats.stream_name"),
			Subject:    viper.GetString("nats.subject"),
		},
		RabbitMQ: RabbitMQConfig{
			URL: viper.GetString("rabbitmq.url"),
		},
		MongoDB: MongoDBConfig{
			URI:      viper.GetString("mongodb.uri"),
			Database: viper.GetString("mongodb.database"),
		},
		Redis: RedisConfig{
			Addr:     viper.GetString("redis.addr"),
			Password: viper.GetString("redis.password"),
			DB:       viper.GetInt("redis.db"),
		},
		Catalog: CatalogConfig{
			URL:        viper.GetString("catalog.url"),
			ServiceKey: viper.GetString("catalog.service_key"),
		},
		SMTP: SMTPConfig{
			Host: viper.GetString("smtp.host"),
			Port: viper.GetInt("smtp.port"),
			User: viper.GetString("smtp.user"),
			Pass: viper.GetString("smtp.pass"),
			From: viper.GetString("smtp.from"),
		},
		WebSocket: WebSocketConfig{
			Enabled:         viper.GetBool("websocket.enabled"),
			MaxConnections:  viper.GetInt("websocket.max_connections"),
			PingInterval:    viper.GetDuration("websocket.ping_interval"),
			PongWait:        viper.GetDuration("websocket.pong_wait"),
			WriteWait:       viper.GetDuration("websocket.write_wait"),
			MaxMessageSize:  viper.GetInt64("websocket.max_message_size"),
			ReadBufferSize:  viper.GetInt("websocket.read_buffer_size"),
			WriteBufferSize: viper.GetInt("websocket.write_buffer_size"),
		},
	}

	return cfg, nil
}

func setDefaults() {
	// App defaults
	viper.SetDefault("app.name", "notification-service")
	viper.SetDefault("app.env", "development")
	viper.SetDefault("app.port", "8083")
	viper.SetDefault("app.grpc_port", "50053")
	viper.SetDefault("app.auth_grpc_addr", "localhost:50051")
	viper.SetDefault("app.debug", true)

	// NATS defaults
	viper.SetDefault("nats.url", "nats://localhost:4222")
	viper.SetDefault("nats.stream_name", "NOTIFICATIONS")
	viper.SetDefault("nats.subject", "notification.>")

	// RabbitMQ defaults
	viper.SetDefault("rabbitmq.url", "amqp://guest:guest@localhost:5672/")

	// MongoDB defaults
	viper.SetDefault("mongodb.uri", "mongodb://localhost:27017")
	viper.SetDefault("mongodb.database", "notification_service")

	// Redis defaults
	viper.SetDefault("redis.addr", "localhost:6379")
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)

	// Catalog Service defaults
	viper.SetDefault("catalog.url", "http://localhost:8082")
	viper.SetDefault("catalog.service_key", "internal-service-key")

	// SMTP defaults
	viper.SetDefault("smtp.host", "smtp.gmail.com")
	viper.SetDefault("smtp.port", 587)
	viper.SetDefault("smtp.user", "")
	viper.SetDefault("smtp.pass", "")
	viper.SetDefault("smtp.from", "")

	// WebSocket defaults
	viper.SetDefault("websocket.enabled", true)
	viper.SetDefault("websocket.max_connections", 10000)
	viper.SetDefault("websocket.ping_interval", "30s")
	viper.SetDefault("websocket.pong_wait", "60s")
	viper.SetDefault("websocket.write_wait", "10s")
	viper.SetDefault("websocket.max_message_size", 512*1024) // 512KB
	viper.SetDefault("websocket.read_buffer_size", 1024)
	viper.SetDefault("websocket.write_buffer_size", 1024)
}
