package authorization

import (
	"fmt"
	"log/slog"
	"time"
)

// Config represents the authorization configuration
type Config struct {
	// Model configuration
	ModelType ModelType `json:"model_type"`
	ModelPath string    `json:"model_path"` // Path to custom model file (optional)

	// Database configuration
	Database *DatabaseConfig `json:"database"`

	// Redis cache configuration
	Cache *CacheConfig `json:"cache"`

	// Watcher configuration for distributed systems
	Watcher *WatcherConfig `json:"watcher"`

	// Service configuration
	ServiceName string `json:"service_name"`

	// Multi-tenancy
	DefaultTenantID string `json:"default_tenant_id"`
	DefaultDomain   string `json:"default_domain"`

	// Performance settings
	AutoLoadPolicy   bool          `json:"auto_load_policy"`
	AutoSavePolicy   bool          `json:"auto_save_policy"`
	EnforcerPoolSize int           `json:"enforcer_pool_size"`
	BatchSize        int           `json:"batch_size"`
	QueryTimeout     time.Duration `json:"query_timeout"`

	// Metrics
	MetricsEnabled bool `json:"metrics_enabled"`
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	Driver          string        `json:"driver"` // postgres, mysql
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
	TablePrefix     string        `json:"table_prefix"`
}

// GetDSN returns the DSN string, either from DSN field or constructed from individual fields
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

	// Cluster mode
	Cluster      bool     `json:"cluster"`
	ClusterAddrs []string `json:"cluster_addrs"`

	// TLS
	TLSEnabled bool `json:"tls_enabled"`
}

// WatcherConfig represents distributed watcher configuration
type WatcherConfig struct {
	Enabled  bool   `json:"enabled"`
	Type     string `json:"type"` // redis
	Channel  string `json:"channel"`
	Redis    *RedisConfig `json:"redis"`
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		ModelType: ModelTypeRBAC,
		Database: &DatabaseConfig{
			Driver:          "postgres",
			Host:            "localhost",
			Port:            5432,
			SSLMode:         "disable",
			MaxIdleConns:    10,
			MaxOpenConns:    100,
			ConnMaxLifetime: time.Hour,
			ConnMaxIdleTime: 10 * time.Minute,
			TablePrefix:     "casbin_",
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
				KeyPrefix:    "authz:",
			},
		},
		Watcher: &WatcherConfig{
			Enabled: false,
			Type:    "redis",
			Channel: "casbin-watcher",
		},
		ServiceName:      "authorization",
		DefaultTenantID:  "default",
		DefaultDomain:    "default",
		AutoLoadPolicy:   true,
		AutoSavePolicy:   true,
		EnforcerPoolSize: 10,
		BatchSize:        100,
		QueryTimeout:     30 * time.Second,
		MetricsEnabled:   true,
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.ModelType == "" {
		return fmt.Errorf("%w: model_type is required", ErrInvalidConfig)
	}

	if c.ModelType != ModelTypeRBAC && c.ModelType != ModelTypeABAC && c.ModelType != ModelTypeHybrid {
		return fmt.Errorf("%w: invalid model_type: %s", ErrInvalidModelType, c.ModelType)
	}

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

	if c.Watcher != nil && c.Watcher.Enabled {
		if c.Watcher.Type == "redis" && c.Watcher.Redis == nil {
			// Use cache redis config if watcher redis is not specified
			if c.Cache != nil && c.Cache.Redis != nil {
				c.Watcher.Redis = c.Cache.Redis
			} else {
				return fmt.Errorf("%w: redis configuration is required for watcher", ErrInvalidConfig)
			}
		}
	}

	return nil
}

// WithDefaults applies default values to missing configuration fields
func (c *Config) WithDefaults() *Config {
	defaults := DefaultConfig()

	if c.ModelType == "" {
		c.ModelType = defaults.ModelType
	}

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

	if c.DefaultDomain == "" {
		c.DefaultDomain = defaults.DefaultDomain
	}

	if c.EnforcerPoolSize == 0 {
		c.EnforcerPoolSize = defaults.EnforcerPoolSize
	}

	if c.BatchSize == 0 {
		c.BatchSize = defaults.BatchSize
	}

	if c.QueryTimeout == 0 {
		c.QueryTimeout = defaults.QueryTimeout
	}

	return c
}

// Option represents a functional option for configuring the authorizer
type Option func(*options)

type options struct {
	Logger         *slog.Logger
	metricsEnabled bool
	cacheEnabled   bool
	watcherEnabled bool
}

// WithLogger sets the logger
func WithLogger(logger *slog.Logger) Option {
	return func(o *options) {
		o.Logger = logger
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

// WithWatcher enables or disables the distributed watcher
func WithWatcher(enabled bool) Option {
	return func(o *options) {
		o.watcherEnabled = enabled
	}
}

// defaultOptions returns the default options
func defaultOptions() *options {
	return &options{
		Logger:         slog.Default(),
		metricsEnabled: true,
		cacheEnabled:   true,
		watcherEnabled: false,
	}
}

// ApplyOptions applies functional options
func ApplyOptions(opts ...Option) *options {
	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}
	return o
}
