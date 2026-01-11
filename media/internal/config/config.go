package config

import (
	"os"
	"time"
)

type Config struct {
	Server  ServerConfig
	MinIO   MinIOConfig
	NATS    NATSConfig
	Media   MediaConfig
}

type ServerConfig struct {
	Port string
}

type MinIOConfig struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	UseSSL          bool
	BucketName      string
}

type NATSConfig struct {
	URL        string
	StreamName string
	Subject    string
}

type MediaConfig struct {
	UploadURLExpiry  time.Duration
	ThumbnailSize    int
	MediumSize       int
	AllowedMimeTypes []string
	TempDir          string
}

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port: getEnv("SERVER_PORT", "8080"),
		},
		MinIO: MinIOConfig{
			Endpoint:        getEnv("MINIO_ENDPOINT", "localhost:9000"),
			AccessKeyID:     getEnv("MINIO_ACCESS_KEY", "minioadmin"),
			SecretAccessKey: getEnv("MINIO_SECRET_KEY", "minioadmin"),
			UseSSL:          getEnv("MINIO_USE_SSL", "false") == "true",
			BucketName:      getEnv("MINIO_BUCKET", "ecommerce-media"),
		},
		NATS: NATSConfig{
			URL:        getEnv("NATS_URL", "nats://localhost:4222"),
			StreamName: getEnv("NATS_STREAM", "MEDIA"),
			Subject:    getEnv("NATS_SUBJECT", "media.uploaded"),
		},
		Media: MediaConfig{
			UploadURLExpiry: 15 * time.Minute,
			ThumbnailSize:   200,
			MediumSize:      800,
			AllowedMimeTypes: []string{
				"image/jpeg",
				"image/png",
				"image/gif",
				"image/webp",
				"video/mp4",
				"video/webm",
			},
			TempDir: getEnv("TEMP_DIR", os.TempDir()),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
