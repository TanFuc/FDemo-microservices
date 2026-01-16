package config

import (
	"os"
	"strconv"
	"time"

	"microservices/pkg/authorization"
	"microservices/pkg/cache"
)

// SharedPackagesConfig holds configuration for shared packages (cache and authorization)
type SharedPackagesConfig struct {
	Cache         CachePackageConfig         `json:"cache"`
	Authorization AuthorizationPackageConfig `json:"authorization"`
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

// LoadSharedPackagesConfig loads shared packages configuration from environment
func LoadSharedPackagesConfig() *SharedPackagesConfig {
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	redisPassword := getEnv("REDIS_PASSWORD", "")

	return &SharedPackagesConfig{
		Cache: CachePackageConfig{
			Enabled:       getEnvBool("CACHE_ENABLED", true),
			Type:          getEnv("CACHE_TYPE", "redis"),
			ServiceName:   getEnv("CACHE_SERVICE_NAME", "payment-service"),
			RedisAddr:     redisAddr,
			RedisPassword: redisPassword,
			RedisDB:       getEnvInt("REDIS_DB", 0),
			PoolSize:      getEnvInt("REDIS_POOL_SIZE", 10),
			MinIdleConns:  getEnvInt("REDIS_MIN_IDLE_CONNS", 2),
		},
		Authorization: AuthorizationPackageConfig{
			Enabled:       getEnvBool("AUTHZ_ENABLED", false),
			ModelType:     getEnv("AUTHZ_MODEL_TYPE", "rbac"),
			ServiceName:   getEnv("AUTHZ_SERVICE_NAME", "payment-service"),
			DefaultTenant: getEnv("AUTHZ_DEFAULT_TENANT", "default"),
			DBHost:        getEnv("AUTHZ_DB_HOST", "localhost"),
			DBPort:        getEnvInt("AUTHZ_DB_PORT", 5432),
			DBUser:        getEnv("AUTHZ_DB_USER", "postgres"),
			DBPassword:    getEnv("AUTHZ_DB_PASSWORD", "postgres"),
			DBName:        getEnv("AUTHZ_DB_NAME", "authorization"),
			CacheEnabled:  getEnvBool("AUTHZ_CACHE_ENABLED", true),
			RedisAddr:     getEnv("AUTHZ_REDIS_ADDR", redisAddr),
			RedisPassword: getEnv("AUTHZ_REDIS_PASSWORD", redisPassword),
			RedisDB:       getEnvInt("AUTHZ_REDIS_DB", 1),
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

// getEnvBool gets a boolean environment variable with a default value
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}
