package customfields

import (
	"fmt"
	"log/slog"
	"time"
)

// Config represents the custom fields configuration
type Config struct {
	// Database configuration
	Database *DatabaseConfig `json:"database"`

	// Cache configuration
	Cache *CacheConfig `json:"cache"`

	// Service configuration
	ServiceName string `json:"service_name"`

	// Multi-tenancy
	DefaultTenantID string `json:"default_tenant_id"`

	// Performance settings
	BatchSize    int           `json:"batch_size"`
	QueryTimeout time.Duration `json:"query_timeout"`

	// Features
	EnableAuditLog   bool `json:"enable_audit_log"`
	EnableEncryption bool `json:"enable_encryption"`

	// Metrics
	MetricsEnabled bool `json:"metrics_enabled"`
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	Driver          string        `json:"driver"` // postgres
	DSN             string        `json:"dsn"`
	Host            string        `json:"host"`
	Port            int           `json:"port"`
	User            string        `json:"user"`
	Password        string        `json:"password"`
	Database        string        `json:"database"`
	SSLMode         string        `json:"ssl_mode"`
	MaxIdleConns    int           `json:"max_idle_conns"`
	MaxOpenConns    int           `json:"max_open_conns"`
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `json:"conn_max_idle_time"`
}

// GetDSN returns the DSN string
func (c *DatabaseConfig) GetDSN() string {
	if c.DSN != "" {
		return c.DSN
	}
	sslMode := c.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.Database, sslMode)
}

// CacheConfig represents cache configuration
type CacheConfig struct {
	Enabled         bool          `json:"enabled"`
	Type            string        `json:"type"` // redis, memory
	TTL             time.Duration `json:"ttl"`
	CleanupInterval time.Duration `json:"cleanup_interval"`
	MaxSize         int           `json:"max_size"`

	// Redis specific
	Redis *RedisConfig `json:"redis"`
}

// RedisConfig represents Redis configuration
type RedisConfig struct {
	Addr         string        `json:"addr"`
	Password     string        `json:"password"`
	DB           int           `json:"db"`
	PoolSize     int           `json:"pool_size"`
	MinIdleConns int           `json:"min_idle_conns"`
	DialTimeout  time.Duration `json:"dial_timeout"`
	ReadTimeout  time.Duration `json:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout"`
	MaxRetries   int           `json:"max_retries"`
	KeyPrefix    string        `json:"key_prefix"`
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Database: &DatabaseConfig{
			Driver:          "postgres",
			Host:            "localhost",
			Port:            5432,
			SSLMode:         "disable",
			MaxIdleConns:    10,
			MaxOpenConns:    100,
			ConnMaxLifetime: time.Hour,
			ConnMaxIdleTime: 10 * time.Minute,
		},
		Cache: &CacheConfig{
			Enabled:         true,
			Type:            "redis",
			TTL:             5 * time.Minute,
			CleanupInterval: time.Minute,
			MaxSize:         10000,
			Redis: &RedisConfig{
				Addr:         "localhost:6379",
				DB:           0,
				PoolSize:     10,
				MinIdleConns: 5,
				DialTimeout:  5 * time.Second,
				ReadTimeout:  3 * time.Second,
				WriteTimeout: 3 * time.Second,
				MaxRetries:   3,
				KeyPrefix:    "cf:",
			},
		},
		ServiceName:      "customfields",
		DefaultTenantID:  "default",
		BatchSize:        100,
		QueryTimeout:     30 * time.Second,
		EnableAuditLog:   true,
		EnableEncryption: false,
		MetricsEnabled:   true,
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Database == nil {
		return fmt.Errorf("%w: database configuration is required", ErrInvalidConfig)
	}

	if c.Database.DSN == "" && (c.Database.Host == "" || c.Database.Database == "") {
		return fmt.Errorf("%w: database DSN or host/database is required", ErrInvalidConfig)
	}

	if c.Cache != nil && c.Cache.Enabled {
		if c.Cache.Type == "redis" && c.Cache.Redis == nil {
			return fmt.Errorf("%w: redis configuration is required when cache type is redis", ErrInvalidConfig)
		}
	}

	return nil
}

// WithDefaults applies default values to missing configuration fields
func (c *Config) WithDefaults() *Config {
	defaults := DefaultConfig()

	if c.Database == nil {
		c.Database = defaults.Database
	} else {
		if c.Database.MaxIdleConns == 0 {
			c.Database.MaxIdleConns = defaults.Database.MaxIdleConns
		}
		if c.Database.MaxOpenConns == 0 {
			c.Database.MaxOpenConns = defaults.Database.MaxOpenConns
		}
		if c.Database.ConnMaxLifetime == 0 {
			c.Database.ConnMaxLifetime = defaults.Database.ConnMaxLifetime
		}
	}

	if c.Cache == nil {
		c.Cache = defaults.Cache
	} else if c.Cache.Enabled && c.Cache.Redis == nil && c.Cache.Type == "redis" {
		c.Cache.Redis = defaults.Cache.Redis
	}

	if c.ServiceName == "" {
		c.ServiceName = defaults.ServiceName
	}

	if c.DefaultTenantID == "" {
		c.DefaultTenantID = defaults.DefaultTenantID
	}

	if c.BatchSize == 0 {
		c.BatchSize = defaults.BatchSize
	}

	if c.QueryTimeout == 0 {
		c.QueryTimeout = defaults.QueryTimeout
	}

	return c
}

// Option represents a functional option for configuring the field manager
type Option func(*options)

type options struct {
	logger         *slog.Logger
	metricsEnabled bool
	cacheEnabled   bool
	auditEnabled   bool
}

// WithLogger sets the logger
func WithLogger(logger *slog.Logger) Option {
	return func(o *options) {
		o.logger = logger
	}
}

// WithMetrics enables or disables metrics collection
func WithMetrics(enabled bool) Option {
	return func(o *options) {
		o.metricsEnabled = enabled
	}
}

// WithCache enables or disables caching
func WithCache(enabled bool) Option {
	return func(o *options) {
		o.cacheEnabled = enabled
	}
}

// WithAuditLog enables or disables audit logging
func WithAuditLog(enabled bool) Option {
	return func(o *options) {
		o.auditEnabled = enabled
	}
}

// defaultOptions returns the default options
func defaultOptions() *options {
	return &options{
		logger:         slog.Default(),
		metricsEnabled: true,
		cacheEnabled:   true,
		auditEnabled:   true,
	}
}

// applyOptions applies functional options
func applyOptions(opts ...Option) *options {
	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}
	return o
}
