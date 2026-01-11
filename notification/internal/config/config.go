package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	NATS     NATSConfig
	RabbitMQ RabbitMQConfig
	MongoDB  MongoDBConfig
	SMTP     SMTPConfig
}

type NATSConfig struct {
	URL string
}

type RabbitMQConfig struct {
	URL string
}

type MongoDBConfig struct {
	URI      string
	Database string
}

type SMTPConfig struct {
	Host string
	Port int
	User string
	Pass string
	From string
}

func Load() (*Config, error) {
	// Load .env file if exists
	_ = godotenv.Load()

	smtpPort, _ := strconv.Atoi(getEnv("SMTP_PORT", "587"))

	return &Config{
		NATS: NATSConfig{
			URL: getEnv("NATS_URL", "nats://localhost:4222"),
		},
		RabbitMQ: RabbitMQConfig{
			URL: getEnv("AMQP_URL", "amqp://guest:guest@localhost:5672/"),
		},
		MongoDB: MongoDBConfig{
			URI:      getEnv("MONGODB_URI", "mongodb://localhost:27017"),
			Database: getEnv("MONGODB_DATABASE", "notification_service"),
		},
		SMTP: SMTPConfig{
			Host: getEnv("SMTP_HOST", "smtp.gmail.com"),
			Port: smtpPort,
			User: getEnv("SMTP_USER", ""),
			Pass: getEnv("SMTP_PASS", ""),
			From: getEnv("SMTP_FROM", ""),
		},
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
