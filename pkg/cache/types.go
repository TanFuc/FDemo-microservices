// Package cache provides a unified caching interface with multiple backend support.
package cache

import (
	"errors"
	"fmt"
	"log/slog"
	"time"
)

// =============================================================================
// Cache Types
// =============================================================================

// CacheType represents the type of cache backend.
type CacheType string

const (
	// CacheTypeRedis uses Redis as the cache backend.
	CacheTypeRedis CacheType = "redis"

	// CacheTypeMemory uses in-memory cache.
	CacheTypeMemory CacheType = "memory"

	// CacheTypeMultilayer uses a multi-layer cache (L1 memory + L2 Redis).
	CacheTypeMultilayer CacheType = "multilayer"
)

// =============================================================================
// Configuration Types
// =============================================================================

// Config is the main configuration for the cache package.
type Config struct {
	// Type specifies the cache backend type.
	Type CacheType `json:"type" yaml:"type"`

	// ServiceName is used for key prefixing to avoid collisions.
	ServiceName string `json:"serviceName" yaml:"serviceName"`

	// KeySeparator is the separator used in cache keys (default: ":").
	KeySeparator string `json:"keySeparator" yaml:"keySeparator"`

	// Redis contains Redis-specific configuration.
	Redis *RedisConfig `json:"redis,omitempty" yaml:"redis,omitempty"`

	// Memory contains in-memory cache configuration.
	Memory *MemoryConfig `json:"memory,omitempty" yaml:"memory,omitempty"`

	// Metrics enables metrics collection.
	Metrics bool `json:"metrics" yaml:"metrics"`

	// Compress enables compression for large values.
	Compress bool `json:"compress" yaml:"compress"`

	// Serializer specifies the serialization format ("json" or "msgpack").
	Serializer string `json:"serializer" yaml:"serializer"`
}

// RedisConfig contains Redis-specific configuration.
type RedisConfig struct {
	// Addr is the Redis server address (e.g., "localhost:6379").
	Addr string `json:"addr" yaml:"addr"`

	// Password is the Redis password (leave empty for no auth).
	Password string `json:"password" yaml:"password"`

	// DB is the Redis database number.
	DB int `json:"db" yaml:"db"`

	// PoolSize is the maximum number of connections in the pool.
	PoolSize int `json:"poolSize" yaml:"poolSize"`

	// MinIdleConns is the minimum number of idle connections.
	MinIdleConns int `json:"minIdleConns" yaml:"minIdleConns"`

	// DialTimeout is the timeout for establishing a connection.
	DialTimeout time.Duration `json:"dialTimeout" yaml:"dialTimeout"`

	// ReadTimeout is the timeout for read operations.
	ReadTimeout time.Duration `json:"readTimeout" yaml:"readTimeout"`

	// WriteTimeout is the timeout for write operations.
	WriteTimeout time.Duration `json:"writeTimeout" yaml:"writeTimeout"`

	// MaxRetries is the maximum number of retries for failed operations.
	MaxRetries int `json:"maxRetries" yaml:"maxRetries"`

	// Cluster enables Redis Cluster mode.
	Cluster bool `json:"cluster" yaml:"cluster"`

	// ClusterAddrs contains the Redis Cluster node addresses.
	ClusterAddrs []string `json:"clusterAddrs" yaml:"clusterAddrs"`

	// TLSEnabled enables TLS for Redis connections.
	TLSEnabled bool `json:"tlsEnabled" yaml:"tlsEnabled"`

	// KeyPrefix is an optional prefix for all keys.
	KeyPrefix string `json:"keyPrefix" yaml:"keyPrefix"`
}

// MemoryConfig contains in-memory cache configuration.
type MemoryConfig struct {
	// MaxSize is the maximum memory size in bytes.
	MaxSize int64 `json:"maxSize" yaml:"maxSize"`

	// DefaultTTL is the default TTL for cached items.
	DefaultTTL time.Duration `json:"defaultTTL" yaml:"defaultTTL"`

	// CleanupInterval is the interval for cleaning expired items.
	CleanupInterval time.Duration `json:"cleanupInterval" yaml:"cleanupInterval"`

	// EvictionPolicy is the eviction policy ("lru", "lfu", or "fifo").
	EvictionPolicy string `json:"evictionPolicy" yaml:"evictionPolicy"`

	// MaxItems is the maximum number of items (0 = unlimited).
	MaxItems int `json:"maxItems" yaml:"maxItems"`
}

// =============================================================================
// Configuration Methods
// =============================================================================

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.Type == "" {
		return fmt.Errorf("%w: cache type is required", ErrInvalidConfig)
	}

	switch c.Type {
	case CacheTypeRedis:
		if c.Redis == nil {
			return fmt.Errorf("%w: redis configuration is required for redis cache type", ErrInvalidConfig)
		}
		if err := c.Redis.Validate(); err != nil {
			return err
		}
	case CacheTypeMemory:
		if c.Memory == nil {
			return fmt.Errorf("%w: memory configuration is required for memory cache type", ErrInvalidConfig)
		}
		if err := c.Memory.Validate(); err != nil {
			return err
		}
	case CacheTypeMultilayer:
		if c.Redis == nil && c.Memory == nil {
			return fmt.Errorf("%w: both redis and memory configurations are required for multilayer cache", ErrInvalidConfig)
		}
	default:
		return fmt.Errorf("%w: unsupported cache type: %s", ErrInvalidConfig, c.Type)
	}

	return nil
}

// Validate validates the Redis configuration.
func (c *RedisConfig) Validate() error {
	if c.Cluster {
		if len(c.ClusterAddrs) == 0 {
			return fmt.Errorf("%w: cluster addresses are required for cluster mode", ErrInvalidConfig)
		}
	} else {
		if c.Addr == "" {
			return fmt.Errorf("%w: redis address is required", ErrInvalidConfig)
		}
	}
	return nil
}

// Validate validates the memory cache configuration.
func (c *MemoryConfig) Validate() error {
	if c.MaxSize < 0 {
		return fmt.Errorf("%w: max size cannot be negative", ErrInvalidConfig)
	}
	if c.MaxItems < 0 {
		return fmt.Errorf("%w: max items cannot be negative", ErrInvalidConfig)
	}
	return nil
}

// GetAddr returns the Redis address with a default value.
func (c *RedisConfig) GetAddr() string {
	if c.Addr == "" {
		return "localhost:6379"
	}
	return c.Addr
}

// DefaultConfig returns a default configuration.
func DefaultConfig() Config {
	return Config{
		Type:         CacheTypeRedis,
		KeySeparator: ":",
		Serializer:   "json",
		Metrics:      false,
		Compress:     false,
		Redis:        DefaultRedisConfig(),
	}
}

// DefaultRedisConfig returns default Redis configuration.
func DefaultRedisConfig() *RedisConfig {
	return &RedisConfig{
		Addr:         "localhost:6379",
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 2,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		MaxRetries:   3,
	}
}

// DefaultMemoryConfig returns default in-memory cache configuration.
func DefaultMemoryConfig() *MemoryConfig {
	return &MemoryConfig{
		MaxSize:         100 * 1024 * 1024, // 100MB
		DefaultTTL:      5 * time.Minute,
		CleanupInterval: 1 * time.Minute,
		EvictionPolicy:  "lru",
		MaxItems:        10000,
	}
}

// WithDefaults applies default values to the configuration.
func (c *Config) WithDefaults() *Config {
	if c.KeySeparator == "" {
		c.KeySeparator = ":"
	}
	if c.Serializer == "" {
		c.Serializer = "json"
	}

	if c.Redis != nil {
		c.Redis = c.Redis.WithDefaults()
	}
	if c.Memory != nil {
		c.Memory = c.Memory.WithDefaults()
	}

	return c
}

// WithDefaults applies default values to the Redis configuration.
func (c *RedisConfig) WithDefaults() *RedisConfig {
	defaults := DefaultRedisConfig()

	if c.Addr == "" && !c.Cluster {
		c.Addr = defaults.Addr
	}
	if c.PoolSize == 0 {
		c.PoolSize = defaults.PoolSize
	}
	if c.MinIdleConns == 0 {
		c.MinIdleConns = defaults.MinIdleConns
	}
	if c.DialTimeout == 0 {
		c.DialTimeout = defaults.DialTimeout
	}
	if c.ReadTimeout == 0 {
		c.ReadTimeout = defaults.ReadTimeout
	}
	if c.WriteTimeout == 0 {
		c.WriteTimeout = defaults.WriteTimeout
	}
	if c.MaxRetries == 0 {
		c.MaxRetries = defaults.MaxRetries
	}

	return c
}

// WithDefaults applies default values to the memory configuration.
func (c *MemoryConfig) WithDefaults() *MemoryConfig {
	defaults := DefaultMemoryConfig()

	if c.MaxSize == 0 {
		c.MaxSize = defaults.MaxSize
	}
	if c.DefaultTTL == 0 {
		c.DefaultTTL = defaults.DefaultTTL
	}
	if c.CleanupInterval == 0 {
		c.CleanupInterval = defaults.CleanupInterval
	}
	if c.EvictionPolicy == "" {
		c.EvictionPolicy = defaults.EvictionPolicy
	}
	if c.MaxItems == 0 {
		c.MaxItems = defaults.MaxItems
	}

	return c
}

// =============================================================================
// Errors
// =============================================================================

// Sentinel errors for cache operations.
var (
	// ErrCacheMiss is returned when a key is not found in the cache.
	ErrCacheMiss = errors.New("cache: key not found")

	// ErrInvalidValue is returned when the value type is invalid for the operation.
	ErrInvalidValue = errors.New("cache: invalid value type")

	// ErrSerialization is returned when serialization fails.
	ErrSerialization = errors.New("cache: serialization failed")

	// ErrDeserialization is returned when deserialization fails.
	ErrDeserialization = errors.New("cache: deserialization failed")

	// ErrConnection is returned when a connection error occurs.
	ErrConnection = errors.New("cache: connection failed")

	// ErrInvalidKey is returned when the cache key format is invalid.
	ErrInvalidKey = errors.New("cache: invalid key format")

	// ErrListNotFound is returned when the list is not found in cache.
	ErrListNotFound = errors.New("cache: list not found")

	// ErrItemNotFound is returned when an item is not found in the list.
	ErrItemNotFound = errors.New("cache: item not found in list")

	// ErrConcurrentWrite is returned when a concurrent write conflict is detected.
	ErrConcurrentWrite = errors.New("cache: concurrent write detected")

	// ErrClosed is returned when operating on a closed cache connection.
	ErrClosed = errors.New("cache: connection closed")

	// ErrTimeout is returned when an operation times out.
	ErrTimeout = errors.New("cache: operation timeout")

	// ErrInvalidConfig is returned when the configuration is invalid.
	ErrInvalidConfig = errors.New("cache: invalid configuration")

	// ErrNilPointer is returned when a nil pointer is passed as destination.
	ErrNilPointer = errors.New("cache: nil pointer destination")
)

// CacheError wraps an underlying error with additional context.
type CacheError struct {
	Op  string // Operation that failed (e.g., "Get", "Set")
	Key string // Cache key involved (if applicable)
	Err error  // Underlying error
}

// Error implements the error interface.
func (e *CacheError) Error() string {
	if e.Key != "" {
		return fmt.Sprintf("cache %s [key=%s]: %v", e.Op, e.Key, e.Err)
	}
	return fmt.Sprintf("cache %s: %v", e.Op, e.Err)
}

// Unwrap returns the underlying error.
func (e *CacheError) Unwrap() error {
	return e.Err
}

// WrapError wraps an error with operation context.
func WrapError(op string, key string, err error) error {
	if err == nil {
		return nil
	}
	return &CacheError{
		Op:  op,
		Key: key,
		Err: err,
	}
}

// IsNotFound checks if the error is a cache miss error.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, ErrCacheMiss) || errors.Is(err, ErrListNotFound) || errors.Is(err, ErrItemNotFound)
}

// IsConnectionError checks if the error is a connection-related error.
func IsConnectionError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, ErrConnection) || errors.Is(err, ErrClosed) || errors.Is(err, ErrTimeout)
}

// IsSerializationError checks if the error is a serialization-related error.
func IsSerializationError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, ErrSerialization) || errors.Is(err, ErrDeserialization)
}

// IsInvalidInput checks if the error is due to invalid input.
func IsInvalidInput(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, ErrInvalidValue) || errors.Is(err, ErrInvalidKey) || errors.Is(err, ErrNilPointer)
}

// =============================================================================
// Options
// =============================================================================

// Option is a functional option for configuring cache operations.
type Option func(*cacheOptions)

// cacheOptions contains configuration options for cache operations.
type cacheOptions struct {
	serializer Serializer
	logger     *slog.Logger
	metrics    bool
	compress   bool
	prefix     string
	tags       []string
}

// defaultOptions returns the default cache options.
func defaultOptions() *cacheOptions {
	return &cacheOptions{
		serializer: DefaultSerializer(),
		logger:     slog.Default(),
		metrics:    false,
		compress:   false,
	}
}

// applyOptions applies the given options to the default options.
func applyOptions(opts ...Option) *cacheOptions {
	o := defaultOptions()
	for _, opt := range opts {
		if opt != nil {
			opt(o)
		}
	}
	return o
}

// WithSerializer sets a custom serializer.
func WithSerializer(s Serializer) Option {
	return func(o *cacheOptions) {
		if s != nil {
			o.serializer = s
		}
	}
}

// WithLogger sets a custom logger.
func WithLogger(logger *slog.Logger) Option {
	return func(o *cacheOptions) {
		if logger != nil {
			o.logger = logger
		}
	}
}

// WithMetrics enables metrics collection.
func WithMetrics(enabled bool) Option {
	return func(o *cacheOptions) {
		o.metrics = enabled
	}
}

// WithCompression enables compression for values.
func WithCompression(enabled bool) Option {
	return func(o *cacheOptions) {
		o.compress = enabled
	}
}

// WithPrefix sets a key prefix.
func WithPrefix(prefix string) Option {
	return func(o *cacheOptions) {
		o.prefix = prefix
	}
}

// WithTags sets tags for group invalidation.
func WithTags(tags ...string) Option {
	return func(o *cacheOptions) {
		o.tags = append(o.tags, tags...)
	}
}

// SetOption is a functional option for Set operations.
type SetOption func(*setOptions)

// setOptions contains options for Set operations.
type setOptions struct {
	ttl      time.Duration
	tags     []string
	compress bool
}

// defaultSetOptions returns the default Set options.
func defaultSetOptions() *setOptions {
	return &setOptions{}
}

// applySetOptions applies the given Set options.
func applySetOptions(opts ...SetOption) *setOptions {
	o := defaultSetOptions()
	for _, opt := range opts {
		if opt != nil {
			opt(o)
		}
	}
	return o
}

// WithTTL sets the TTL for a Set operation.
func WithTTL(ttl time.Duration) SetOption {
	return func(o *setOptions) {
		o.ttl = ttl
	}
}

// WithSetTags sets tags for a Set operation.
func WithSetTags(tags ...string) SetOption {
	return func(o *setOptions) {
		o.tags = append(o.tags, tags...)
	}
}

// WithSetCompression enables compression for a Set operation.
func WithSetCompression(enabled bool) SetOption {
	return func(o *setOptions) {
		o.compress = enabled
	}
}

// GetOption is a functional option for Get operations.
type GetOption func(*getOptions)

// getOptions contains options for Get operations.
type getOptions struct {
	skipL1 bool
	skipL2 bool
}

// defaultGetOptions returns the default Get options.
func defaultGetOptions() *getOptions {
	return &getOptions{}
}

// applyGetOptions applies the given Get options.
func applyGetOptions(opts ...GetOption) *getOptions {
	o := defaultGetOptions()
	for _, opt := range opts {
		if opt != nil {
			opt(o)
		}
	}
	return o
}

// SkipL1Cache skips the L1 cache in multilayer cache.
func SkipL1Cache() GetOption {
	return func(o *getOptions) {
		o.skipL1 = true
	}
}

// SkipL2Cache skips the L2 cache in multilayer cache.
func SkipL2Cache() GetOption {
	return func(o *getOptions) {
		o.skipL2 = true
	}
}
