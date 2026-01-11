package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server     ServerConfig
	ClickHouse ClickHouseConfig
	NATS       NATSConfig
	Worker     WorkerConfig
}

type ServerConfig struct {
	Port string
}

type ClickHouseConfig struct {
	Host     string
	Port     int
	Database string
	Username string
	Password string
}

type NATSConfig struct {
	URL        string
	StreamName string
	Subject    string
}

type WorkerConfig struct {
	BatchSize     int
	FlushInterval time.Duration
}

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port: getEnv("SERVER_PORT", "8080"),
		},
		ClickHouse: ClickHouseConfig{
			Host:     getEnv("CLICKHOUSE_HOST", "localhost"),
			Port:     getEnvAsInt("CLICKHOUSE_PORT", 9000),
			Database: getEnv("CLICKHOUSE_DATABASE", "analytics"),
			Username: getEnv("CLICKHOUSE_USERNAME", "default"),
			Password: getEnv("CLICKHOUSE_PASSWORD", ""),
		},
		NATS: NATSConfig{
			URL:        getEnv("NATS_URL", "nats://localhost:4222"),
			StreamName: getEnv("NATS_STREAM_NAME", "ANALYTICS"),
			Subject:    getEnv("NATS_SUBJECT", "analytics.events.raw"),
		},
		Worker: WorkerConfig{
			BatchSize:     getEnvAsInt("WORKER_BATCH_SIZE", 1000),
			FlushInterval: time.Duration(getEnvAsInt("WORKER_FLUSH_INTERVAL_SECONDS", 5)) * time.Second,
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
