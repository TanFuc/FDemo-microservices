package config

import (
	"os"
	"strconv"
	"time"

	"microservices/pkg/authorization"
	"microservices/pkg/cache"
	"microservices/pkg/customfields"
)

// SharedPackagesConfig holds configuration for all shared packages
type SharedPackagesConfig struct {
	Cache         CachePackageConfig         `json:"cache"`
	Authorization AuthorizationPackageConfig `json:"authorization"`
	CustomFields  CustomFieldsPackageConfig  `json:"custom_fields"`
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

// AuthorizationPackageConfig holds authorization package configuration
type AuthorizationPackageConfig struct {
	Enabled       bool   `json:"enabled"`
	ModelType     string `json:"model_type"` // rbac, abac, hybrid
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
func LoadSharedPackagesConfig() *SharedPackagesConfig {
	redisAddr := getEnvString("REDIS_ADDR", "localhost:6379")
	redisPassword := getEnvString("REDIS_PASSWORD", "")

	return &SharedPackagesConfig{
		Cache: CachePackageConfig{
			Enabled:       getEnvBool("CACHE_ENABLED", true),
			Type:          getEnvString("CACHE_TYPE", "redis"),
			ServiceName:   getEnvString("CACHE_SERVICE_NAME", "review-service"),
			RedisAddr:     redisAddr,
			RedisPassword: redisPassword,
			RedisDB:       getEnvInt("REDIS_DB", 0),
			PoolSize:      getEnvInt("REDIS_POOL_SIZE", 10),
			MinIdleConns:  getEnvInt("REDIS_MIN_IDLE_CONNS", 2),
		},
		Authorization: AuthorizationPackageConfig{
			Enabled:       getEnvBool("AUTHZ_ENABLED", false),
			ModelType:     getEnvString("AUTHZ_MODEL_TYPE", "rbac"),
			ServiceName:   getEnvString("AUTHZ_SERVICE_NAME", "review-service"),
			DefaultTenant: getEnvString("AUTHZ_DEFAULT_TENANT", "default"),
			DBHost:        getEnvString("AUTHZ_DB_HOST", "localhost"),
			DBPort:        getEnvInt("AUTHZ_DB_PORT", 5432),
			DBUser:        getEnvString("AUTHZ_DB_USER", "postgres"),
			DBPassword:    getEnvString("AUTHZ_DB_PASSWORD", "postgres"),
			DBName:        getEnvString("AUTHZ_DB_NAME", "authorization"),
			CacheEnabled:  getEnvBool("AUTHZ_CACHE_ENABLED", true),
			RedisAddr:     getEnvString("AUTHZ_REDIS_ADDR", redisAddr),
			RedisPassword: getEnvString("AUTHZ_REDIS_PASSWORD", redisPassword),
			RedisDB:       getEnvInt("AUTHZ_REDIS_DB", 1),
		},
		CustomFields: CustomFieldsPackageConfig{
			Enabled:       getEnvBool("CUSTOMFIELDS_ENABLED", false),
			ServiceName:   getEnvString("CUSTOMFIELDS_SERVICE_NAME", "review-service"),
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

// ToAuthorizationConfig converts to pkg/authorization configuration
func (c *AuthorizationPackageConfig) ToAuthorizationConfig() *authorization.Config {
	modelType := authorization.ModelTypeRBAC
	switch c.ModelType {
	case "abac":
		modelType = authorization.ModelTypeABAC
	case "hybrid":
		modelType = authorization.ModelTypeHybrid
	}

	return &authorization.Config{
		ModelType:       modelType,
		ServiceName:     c.ServiceName,
		DefaultTenantID: c.DefaultTenant,
		DefaultDomain:   "default",
		Database: &authorization.DatabaseConfig{
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
		Cache: &authorization.CacheConfig{
			Enabled: c.CacheEnabled,
			Type:    "redis",
			TTL:     5 * time.Minute,
			Redis: &authorization.RedisConfig{
				Addr:         c.RedisAddr,
				Password:     c.RedisPassword,
				DB:           c.RedisDB,
				PoolSize:     10,
				MinIdleConns: 5,
				DialTimeout:  5 * time.Second,
				ReadTimeout:  3 * time.Second,
				WriteTimeout: 3 * time.Second,
				MaxRetries:   3,
				KeyPrefix:    "authz:" + c.ServiceName + ":",
			},
		},
		AutoLoadPolicy: true,
		AutoSavePolicy: true,
		BatchSize:      100,
		QueryTimeout:   30 * time.Second,
		MetricsEnabled: true,
	}
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
