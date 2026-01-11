package config

import (
	"os"
	"strconv"
)

// Config holds all configuration for the application
type Config struct {
	// Elasticsearch
	ElasticAddresses []string

	// Redis
	RedisAddr     string
	RedisPassword string
	RedisDB       int

	// NATS
	NatsURL string

	// API
	APIPort string
}

// Load loads configuration from environment variables with defaults
func Load() *Config {
	return &Config{
		ElasticAddresses: getEnvSlice("ELASTIC_ADDRESSES", []string{"http://localhost:9200"}),
		RedisAddr:        getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:    getEnv("REDIS_PASSWORD", ""),
		RedisDB:          getEnvInt("REDIS_DB", 0),
		NatsURL:          getEnv("NATS_URL", "nats://localhost:4222"),
		APIPort:          getEnv("API_PORT", "3000"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return []string{value}
	}
	return defaultValue
}
