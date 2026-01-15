package customfields

import (
	"context"
)

// FieldManager is the main interface for custom fields management
type FieldManager interface {
	// Field Definition Management
	CreateFieldDefinition(ctx context.Context, def *FieldDefinition) error
	UpdateFieldDefinition(ctx context.Context, id string, def *FieldDefinition) error
	DeleteFieldDefinition(ctx context.Context, id string) error
	GetFieldDefinition(ctx context.Context, id string) (*FieldDefinition, error)
	GetFieldDefinitionByName(ctx context.Context, entityType, name string) (*FieldDefinition, error)
	ListFieldDefinitions(ctx context.Context, filter *FieldFilter) ([]*FieldDefinition, error)

	// Field Value Management
	SetFieldValue(ctx context.Context, entityType, entityID string, values map[string]interface{}) error
	GetFieldValue(ctx context.Context, entityType, entityID, fieldID string) (interface{}, error)
	GetAllFieldValues(ctx context.Context, entityType, entityID string) (map[string]interface{}, error)
	DeleteFieldValue(ctx context.Context, entityType, entityID, fieldID string) error
	DeleteAllFieldValues(ctx context.Context, entityType, entityID string) error

	// Bulk Operations
	BulkSetFieldValues(ctx context.Context, operations []*BulkFieldOperation) error
	BulkGetFieldValues(ctx context.Context, entityType string, entityIDs []string) (map[string]map[string]interface{}, error)

	// Validation
	ValidateFieldValue(ctx context.Context, defID string, value interface{}) error
	ValidateFieldValues(ctx context.Context, entityType string, values map[string]interface{}) error

	// Search & Query
	SearchByFieldValue(ctx context.Context, query *FieldQuery) ([]*EntityFieldValues, error)

	// Schema Management
	GetEntitySchema(ctx context.Context, entityType string) (*EntitySchema, error)
	GenerateJSONSchema(ctx context.Context, entityType string) (map[string]interface{}, error)

	// Permission Integration
	CheckFieldPermission(ctx context.Context, userID, entityType, fieldID, action string) (bool, error)
	GetUserVisibleFields(ctx context.Context, userID, entityType string) ([]*FieldDefinition, error)

	// Import/Export
	ExportFieldDefinitions(ctx context.Context, entityType string) ([]byte, error)
	ImportFieldDefinitions(ctx context.Context, data []byte) error

	// Cache Management
	InvalidateCache(ctx context.Context, entityType string) error
	InvalidateEntityCache(ctx context.Context, entityType, entityID string) error

	// Health & Stats
	HealthCheck(ctx context.Context) error
	GetStats(ctx context.Context) (*FieldStats, error)

	// Lifecycle
	Close() error
}

// FieldStats represents statistics for the field manager
type FieldStats struct {
	TotalDefinitions int64   `json:"total_definitions"`
	TotalValues      int64   `json:"total_values"`
	CacheHits        int64   `json:"cache_hits"`
	CacheMisses      int64   `json:"cache_misses"`
	CacheHitRate     float64 `json:"cache_hit_rate"`
	AvgQueryLatency  int64   `json:"avg_query_latency_ms"`
}

// DefinitionManager is a subset interface for field definition operations
type DefinitionManager interface {
	CreateFieldDefinition(ctx context.Context, def *FieldDefinition) error
	UpdateFieldDefinition(ctx context.Context, id string, def *FieldDefinition) error
	DeleteFieldDefinition(ctx context.Context, id string) error
	GetFieldDefinition(ctx context.Context, id string) (*FieldDefinition, error)
	ListFieldDefinitions(ctx context.Context, filter *FieldFilter) ([]*FieldDefinition, error)
}

// ValueManager is a subset interface for field value operations
type ValueManager interface {
	SetFieldValue(ctx context.Context, entityType, entityID string, values map[string]interface{}) error
	GetFieldValue(ctx context.Context, entityType, entityID, fieldID string) (interface{}, error)
	GetAllFieldValues(ctx context.Context, entityType, entityID string) (map[string]interface{}, error)
	DeleteFieldValue(ctx context.Context, entityType, entityID, fieldID string) error
}

// New creates a new FieldManager with the given configuration
func New(cfg *Config, opts ...Option) (FieldManager, error) {
	// Apply defaults
	cfg = cfg.WithDefaults()

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// Create and return the field manager implementation
	return newFieldManager(cfg, opts...)
}

// newFieldManager is the factory function implemented by the storage package
var newFieldManager func(cfg *Config, opts ...Option) (FieldManager, error)

// RegisterFieldManager registers the FieldManager factory
func RegisterFieldManager(factory func(cfg *Config, opts ...Option) (FieldManager, error)) {
	newFieldManager = factory
}

// MustNew creates a new FieldManager and panics on error
func MustNew(cfg *Config, opts ...Option) FieldManager {
	fm, err := New(cfg, opts...)
	if err != nil {
		panic(err)
	}
	return fm
}

// Helper functions for common operations

// GetFieldValueTyped retrieves a field value and converts it to the specified type
func GetFieldValueTyped[T any](ctx context.Context, fm FieldManager, entityType, entityID, fieldID string) (T, error) {
	var zero T
	value, err := fm.GetFieldValue(ctx, entityType, entityID, fieldID)
	if err != nil {
		return zero, err
	}
	if value == nil {
		return zero, nil
	}
	typed, ok := value.(T)
	if !ok {
		return zero, NewFieldError(fieldID, fieldID, "type conversion failed", ErrInvalidFieldValue)
	}
	return typed, nil
}

// SetFieldValueIfNotExists sets a field value only if it doesn't already exist
func SetFieldValueIfNotExists(ctx context.Context, fm FieldManager, entityType, entityID, fieldID string, value interface{}) error {
	existing, err := fm.GetFieldValue(ctx, entityType, entityID, fieldID)
	if err != nil && err != ErrFieldNotFound {
		return err
	}
	if existing != nil {
		return nil // Already exists
	}
	return fm.SetFieldValue(ctx, entityType, entityID, map[string]interface{}{
		fieldID: value,
	})
}

// CopyFieldValues copies field values from one entity to another
func CopyFieldValues(ctx context.Context, fm FieldManager, entityType, fromEntityID, toEntityID string) error {
	values, err := fm.GetAllFieldValues(ctx, entityType, fromEntityID)
	if err != nil {
		return err
	}
	if len(values) == 0 {
		return nil
	}
	return fm.SetFieldValue(ctx, entityType, toEntityID, values)
}

// MergeFieldValues merges field values from source into target, with target values taking precedence
func MergeFieldValues(ctx context.Context, fm FieldManager, entityType, sourceEntityID, targetEntityID string) error {
	sourceValues, err := fm.GetAllFieldValues(ctx, entityType, sourceEntityID)
	if err != nil {
		return err
	}
	targetValues, err := fm.GetAllFieldValues(ctx, entityType, targetEntityID)
	if err != nil {
		return err
	}

	// Merge: source values are added only if not present in target
	merged := make(map[string]interface{})
	for k, v := range sourceValues {
		merged[k] = v
	}
	for k, v := range targetValues {
		merged[k] = v // Target takes precedence
	}

	return fm.SetFieldValue(ctx, entityType, targetEntityID, merged)
}

// GetRequiredFieldIDs returns a list of required field IDs for an entity type
func GetRequiredFieldIDs(ctx context.Context, fm FieldManager, entityType string) ([]string, error) {
	isRequired := true
	defs, err := fm.ListFieldDefinitions(ctx, &FieldFilter{
		EntityType: &entityType,
		IsRequired: &isRequired,
	})
	if err != nil {
		return nil, err
	}

	ids := make([]string, len(defs))
	for i, def := range defs {
		ids[i] = def.ID
	}
	return ids, nil
}

// ValidateRequiredFields checks if all required fields have values
func ValidateRequiredFields(ctx context.Context, fm FieldManager, entityType, entityID string) error {
	requiredIDs, err := GetRequiredFieldIDs(ctx, fm, entityType)
	if err != nil {
		return err
	}

	values, err := fm.GetAllFieldValues(ctx, entityType, entityID)
	if err != nil {
		return err
	}

	var errors []*FieldError
	for _, fieldID := range requiredIDs {
		if _, exists := values[fieldID]; !exists {
			errors = append(errors, &FieldError{
				FieldID: fieldID,
				Field:   fieldID,
				Message: "required field is missing",
				Err:     ErrRequiredFieldMissing,
			})
		}
	}

	if len(errors) > 0 {
		return &ValidationError{Errors: errors}
	}
	return nil
}
