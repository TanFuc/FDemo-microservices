package config

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	App   AppConfig
	MinIO MinIOConfig
	NATS  NATSConfig
	Media MediaConfig
}

type AppConfig struct {
	Name         string
	Env          string
	Port         string
	GRPCPort     string
	AuthGRPCAddr string
	Debug        bool
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
		MinIO: MinIOConfig{
			Endpoint:        viper.GetString("minio.endpoint"),
			AccessKeyID:     viper.GetString("minio.access_key_id"),
			SecretAccessKey: viper.GetString("minio.secret_access_key"),
			UseSSL:          viper.GetBool("minio.use_ssl"),
			BucketName:      viper.GetString("minio.bucket_name"),
		},
		NATS: NATSConfig{
			URL:        viper.GetString("nats.url"),
			StreamName: viper.GetString("nats.stream_name"),
			Subject:    viper.GetString("nats.subject"),
		},
		Media: MediaConfig{
			UploadURLExpiry:  viper.GetDuration("media.upload_url_expiry"),
			ThumbnailSize:    viper.GetInt("media.thumbnail_size"),
			MediumSize:       viper.GetInt("media.medium_size"),
			AllowedMimeTypes: viper.GetStringSlice("media.allowed_mime_types"),
			TempDir:          viper.GetString("media.temp_dir"),
		},
	}

	return cfg, nil
}

func setDefaults() {
	// App defaults
	viper.SetDefault("app.name", "media-service")
	viper.SetDefault("app.env", "development")
	viper.SetDefault("app.port", "8080")
	viper.SetDefault("app.grpc_port", "50052")
	viper.SetDefault("app.auth_grpc_addr", "localhost:50051")
	viper.SetDefault("app.debug", true)

	// MinIO defaults
	viper.SetDefault("minio.endpoint", "localhost:9000")
	viper.SetDefault("minio.access_key_id", "minioadmin")
	viper.SetDefault("minio.secret_access_key", "minioadmin")
	viper.SetDefault("minio.use_ssl", false)
	viper.SetDefault("minio.bucket_name", "ecommerce-media")

	// NATS defaults
	viper.SetDefault("nats.url", "nats://localhost:4222")
	viper.SetDefault("nats.stream_name", "MEDIA")
	viper.SetDefault("nats.subject", "media.uploaded")

	// Media defaults
	viper.SetDefault("media.upload_url_expiry", "15m")
	viper.SetDefault("media.thumbnail_size", 200)
	viper.SetDefault("media.medium_size", 800)
	viper.SetDefault("media.allowed_mime_types", []string{
		"image/jpeg",
		"image/png",
		"image/gif",
		"image/webp",
		"video/mp4",
		"video/webm",
	})
	viper.SetDefault("media.temp_dir", "/tmp")
}
