package config

import (
	"os"
	"strconv"
	"time"

	"microservices/pkg/cache"
	"microservices/pkg/customfields"
)

// SharedPackagesConfig holds configuration for shared packages (cache and customfields)
type SharedPackagesConfig struct {
	Cache        CachePackageConfig        `json:"cache"`
	CustomFields CustomFieldsPackageConfig `json:"custom_fields"`
}

// CachePackageConfig holds cache package configuration
type CachePackageConfig struct {
	Enabled       bool   `json:"enabled"`
	Type          string `json:"type"` // redis, memory, multilayer
	ServiceName   string `json:"service_name"`
	RedisAddr     string `json:"redis_addr"`
	RedisPassword string `json:"redis_password"`
	RedisDB       int    `json:"redis_db"`
	PoolSize      int    `json:"pool_size"`
	MinIdleConns  int    `json:"min_idle_conns"`
}

// CustomFieldsPackageConfig holds custom fields package configuration
type CustomFieldsPackageConfig struct {
	Enabled       bool   `json:"enabled"`
	ServiceName   string `json:"service_name"`
	DefaultTenant string `json:"default_tenant"`

	// Database
	DBHost     string `json:"db_host"`
	DBPort     int    `json:"db_port"`
	DBUser     string `json:"db_user"`
	DBPassword string `json:"db_password"`
	DBName     string `json:"db_name"`

	// Cache
	CacheEnabled  bool   `json:"cache_enabled"`
	RedisAddr     string `json:"redis_addr"`
	RedisPassword string `json:"redis_password"`
	RedisDB       int    `json:"redis_db"`
}

// LoadSharedPackagesConfig loads shared packages configuration from environment
// Uses main config's Redis settings as defaults
func LoadSharedPackagesConfig(mainCfg *Config) *SharedPackagesConfig {
	// Use main config Redis settings as defaults
	redisAddr := "localhost:6379"
	redisPassword := ""
	redisDB := 0
	if mainCfg != nil {
		redisAddr = mainCfg.Redis.Addr
		redisPassword = mainCfg.Redis.Password
		redisDB = mainCfg.Redis.DB
	}

	return &SharedPackagesConfig{
		Cache: CachePackageConfig{
			Enabled:       getEnvBool("CACHE_ENABLED", true),
			Type:          getEnvString("CACHE_TYPE", "redis"),
			ServiceName:   getEnvString("CACHE_SERVICE_NAME", "search-service"),
			RedisAddr:     getEnvString("REDIS_ADDR", redisAddr),
			RedisPassword: getEnvString("REDIS_PASSWORD", redisPassword),
			RedisDB:       getEnvInt("REDIS_DB", redisDB),
			PoolSize:      getEnvInt("REDIS_POOL_SIZE", 10),
			MinIdleConns:  getEnvInt("REDIS_MIN_IDLE_CONNS", 2),
		},
		CustomFields: CustomFieldsPackageConfig{
			Enabled:       getEnvBool("CUSTOMFIELDS_ENABLED", false),
			ServiceName:   getEnvString("CUSTOMFIELDS_SERVICE_NAME", "search-service"),
			DefaultTenant: getEnvString("CUSTOMFIELDS_DEFAULT_TENANT", "default"),
			DBHost:        getEnvString("CUSTOMFIELDS_DB_HOST", "localhost"),
			DBPort:        getEnvInt("CUSTOMFIELDS_DB_PORT", 5432),
			DBUser:        getEnvString("CUSTOMFIELDS_DB_USER", "postgres"),
			DBPassword:    getEnvString("CUSTOMFIELDS_DB_PASSWORD", "postgres"),
			DBName:        getEnvString("CUSTOMFIELDS_DB_NAME", "customfields"),
			CacheEnabled:  getEnvBool("CUSTOMFIELDS_CACHE_ENABLED", true),
			RedisAddr:     getEnvString("CUSTOMFIELDS_REDIS_ADDR", redisAddr),
			RedisPassword: getEnvString("CUSTOMFIELDS_REDIS_PASSWORD", redisPassword),
			RedisDB:       getEnvInt("CUSTOMFIELDS_REDIS_DB", 2),
		},
	}
}

// ToCacheConfig converts to pkg/cache configuration
func (c *CachePackageConfig) ToCacheConfig() *cache.Config {
	cacheType := cache.CacheTypeRedis
	switch c.Type {
	case "memory":
		cacheType = cache.CacheTypeMemory
	case "multilayer":
		cacheType = cache.CacheTypeMultilayer
	}

	cfg := &cache.Config{
		Type:        cacheType,
		ServiceName: c.ServiceName,
		Metrics:     true,
		Redis: &cache.RedisConfig{
			Addr:         c.RedisAddr,
			Password:     c.RedisPassword,
			DB:           c.RedisDB,
			PoolSize:     c.PoolSize,
			MinIdleConns: c.MinIdleConns,
			DialTimeout:  5 * time.Second,
			ReadTimeout:  3 * time.Second,
			WriteTimeout: 3 * time.Second,
			MaxRetries:   3,
		},
	}

	if c.Type == "memory" || c.Type == "multilayer" {
		cfg.Memory = &cache.MemoryConfig{
			MaxSize:         50 * 1024 * 1024, // 50MB
			DefaultTTL:      5 * time.Minute,
			CleanupInterval: time.Minute,
			EvictionPolicy:  "lru",
			MaxItems:        10000,
		}
	}

	return cfg
}

// ToCustomFieldsConfig converts to pkg/customfields configuration
func (c *CustomFieldsPackageConfig) ToCustomFieldsConfig() *customfields.Config {
	return &customfields.Config{
		ServiceName:     c.ServiceName,
		DefaultTenantID: c.DefaultTenant,
		Database: &customfields.DatabaseConfig{
			Driver:          "postgres",
			Host:            c.DBHost,
			Port:            c.DBPort,
			User:            c.DBUser,
			Password:        c.DBPassword,
			Database:        c.DBName,
			SSLMode:         "disable",
			MaxIdleConns:    10,
			MaxOpenConns:    100,
			ConnMaxLifetime: time.Hour,
		},
		Cache: &customfields.CacheConfig{
			Enabled: c.CacheEnabled,
			Type:    "redis",
			TTL:     5 * time.Minute,
			Redis: &customfields.RedisConfig{
				Addr:         c.RedisAddr,
				Password:     c.RedisPassword,
				DB:           c.RedisDB,
				PoolSize:     10,
				MinIdleConns: 5,
				DialTimeout:  5 * time.Second,
				ReadTimeout:  3 * time.Second,
				WriteTimeout: 3 * time.Second,
				MaxRetries:   3,
				KeyPrefix:    "cf:" + c.ServiceName + ":",
			},
		},
		BatchSize:        100,
		QueryTimeout:     30 * time.Second,
		EnableAuditLog:   true,
		EnableEncryption: false,
		MetricsEnabled:   true,
	}
}

// Helper functions for environment variable parsing
func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}
