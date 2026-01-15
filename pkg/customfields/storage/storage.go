package storage

import (
	"context"

	"microservices/pkg/customfields"
)

// Storage is the interface for custom field storage operations
type Storage interface {
	// Field Definition Operations
	CreateDefinition(ctx context.Context, def *customfields.FieldDefinition) error
	UpdateDefinition(ctx context.Context, id string, def *customfields.FieldDefinition) error
	DeleteDefinition(ctx context.Context, id string) error
	GetDefinition(ctx context.Context, id string) (*customfields.FieldDefinition, error)
	GetDefinitionByName(ctx context.Context, entityType, name string) (*customfields.FieldDefinition, error)
	ListDefinitions(ctx context.Context, filter *customfields.FieldFilter) ([]*customfields.FieldDefinition, error)
	CountDefinitions(ctx context.Context, filter *customfields.FieldFilter) (int64, error)

	// Field Value Operations
	SetValue(ctx context.Context, value *customfields.FieldValue) error
	SetValues(ctx context.Context, entityType, entityID string, values []*customfields.FieldValue) error
	GetValue(ctx context.Context, entityType, entityID, fieldID string) (*customfields.FieldValue, error)
	GetAllValues(ctx context.Context, entityType, entityID string) ([]*customfields.FieldValue, error)
	DeleteValue(ctx context.Context, entityType, entityID, fieldID string) error
	DeleteAllValues(ctx context.Context, entityType, entityID string) error
	CountValues(ctx context.Context, entityType string) (int64, error)

	// Bulk Operations
	BulkSetValues(ctx context.Context, operations []*customfields.BulkFieldOperation) error
	BulkGetValues(ctx context.Context, entityType string, entityIDs []string) (map[string][]*customfields.FieldValue, error)

	// Search Operations
	Search(ctx context.Context, query *customfields.FieldQuery) ([]string, error) // Returns entity IDs
	SearchWithValues(ctx context.Context, query *customfields.FieldQuery) ([]*customfields.EntityFieldValues, error)

	// Audit Log Operations
	CreateAuditLog(ctx context.Context, log *customfields.FieldAuditLog) error
	GetAuditLogs(ctx context.Context, entityType, entityID string, limit int) ([]*customfields.FieldAuditLog, error)

	// Health & Maintenance
	Ping(ctx context.Context) error
	Migrate(ctx context.Context) error
	Close() error
}

// Cache is the interface for caching field definitions and values
type Cache interface {
	// Definition Cache
	GetDefinition(ctx context.Context, id string) (*customfields.FieldDefinition, error)
	SetDefinition(ctx context.Context, def *customfields.FieldDefinition) error
	DeleteDefinition(ctx context.Context, id string) error
	GetDefinitionsByEntity(ctx context.Context, entityType string) ([]*customfields.FieldDefinition, error)
	SetDefinitionsByEntity(ctx context.Context, entityType string, defs []*customfields.FieldDefinition) error
	DeleteDefinitionsByEntity(ctx context.Context, entityType string) error

	// Value Cache
	GetValues(ctx context.Context, entityType, entityID string) (map[string]interface{}, error)
	SetValues(ctx context.Context, entityType, entityID string, values map[string]interface{}) error
	DeleteValues(ctx context.Context, entityType, entityID string) error

	// Stats
	GetStats(ctx context.Context) (*CacheStats, error)

	// Health
	Ping(ctx context.Context) error
	Clear(ctx context.Context) error
	Close() error
}

// CacheStats represents cache statistics
type CacheStats struct {
	Hits        int64   `json:"hits"`
	Misses      int64   `json:"misses"`
	HitRate     float64 `json:"hit_rate"`
	Size        int64   `json:"size"`
	MemoryUsage int64   `json:"memory_usage"`
}
