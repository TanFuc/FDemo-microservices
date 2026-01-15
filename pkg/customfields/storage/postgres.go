package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"microservices/pkg/customfields"
)

func init() {
	// Register the PostgresFieldManager factory
	customfields.RegisterFieldManager(NewPostgresFieldManager)
}

// PostgresStorage implements the Storage interface using PostgreSQL
type PostgresStorage struct {
	db       *gorm.DB
	logger   *slog.Logger
	tenantID string
}

// PostgresConfig represents PostgreSQL configuration
type PostgresConfig struct {
	DSN             string
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	TenantID        string
	Logger          *slog.Logger
}

// NewPostgresStorage creates a new PostgreSQL storage
func NewPostgresStorage(cfg *PostgresConfig) (*PostgresStorage, error) {
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	}

	db, err := gorm.Open(postgres.Open(cfg.DSN), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to connect to database: %v", customfields.ErrConnectionFailed, err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get sql.DB: %v", customfields.ErrConnectionFailed, err)
	}

	// Configure connection pool
	if cfg.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}
	if cfg.ConnMaxIdleTime > 0 {
		sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)
	}

	log := cfg.Logger
	if log == nil {
		log = slog.Default()
	}

	storage := &PostgresStorage{
		db:       db,
		logger:   log,
		tenantID: cfg.TenantID,
	}

	return storage, nil
}

// NewPostgresStorageWithDB creates a new PostgreSQL storage with an existing DB connection
func NewPostgresStorageWithDB(db *gorm.DB, tenantID string, logger *slog.Logger) *PostgresStorage {
	if logger == nil {
		logger = slog.Default()
	}
	return &PostgresStorage{
		db:       db,
		logger:   logger,
		tenantID: tenantID,
	}
}

// Migrate creates the required database tables
func (s *PostgresStorage) Migrate(ctx context.Context) error {
	return s.db.AutoMigrate(
		&customfields.FieldDefinition{},
		&customfields.FieldValue{},
		&customfields.FieldAuditLog{},
	)
}

// Ping checks the database connection
func (s *PostgresStorage) Ping(ctx context.Context) error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

// Close closes the database connection
func (s *PostgresStorage) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// CreateDefinition creates a new field definition
func (s *PostgresStorage) CreateDefinition(ctx context.Context, def *customfields.FieldDefinition) error {
	if def.ID == "" {
		def.ID = uuid.New().String()
	}
	if def.TenantID == "" {
		def.TenantID = s.tenantID
	}

	if err := def.BeforeSave(); err != nil {
		return fmt.Errorf("%w: failed to serialize field definition: %v", customfields.ErrDatabaseOperation, err)
	}

	if err := s.db.WithContext(ctx).Create(def).Error; err != nil {
		return fmt.Errorf("%w: failed to create field definition: %v", customfields.ErrDatabaseOperation, err)
	}

	return nil
}

// UpdateDefinition updates an existing field definition
func (s *PostgresStorage) UpdateDefinition(ctx context.Context, id string, def *customfields.FieldDefinition) error {
	def.ID = id
	def.Version++

	if err := def.BeforeSave(); err != nil {
		return fmt.Errorf("%w: failed to serialize field definition: %v", customfields.ErrDatabaseOperation, err)
	}

	result := s.db.WithContext(ctx).Save(def)
	if result.Error != nil {
		return fmt.Errorf("%w: failed to update field definition: %v", customfields.ErrDatabaseOperation, result.Error)
	}
	if result.RowsAffected == 0 {
		return customfields.ErrFieldNotFound
	}

	return nil
}

// DeleteDefinition deletes a field definition
func (s *PostgresStorage) DeleteDefinition(ctx context.Context, id string) error {
	result := s.db.WithContext(ctx).Delete(&customfields.FieldDefinition{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("%w: failed to delete field definition: %v", customfields.ErrDatabaseOperation, result.Error)
	}
	if result.RowsAffected == 0 {
		return customfields.ErrFieldNotFound
	}

	// Also delete associated values
	s.db.WithContext(ctx).Delete(&customfields.FieldValue{}, "field_id = ?", id)

	return nil
}

// GetDefinition retrieves a field definition by ID
func (s *PostgresStorage) GetDefinition(ctx context.Context, id string) (*customfields.FieldDefinition, error) {
	var def customfields.FieldDefinition
	result := s.db.WithContext(ctx).First(&def, "id = ?", id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, customfields.ErrFieldNotFound
		}
		return nil, fmt.Errorf("%w: failed to get field definition: %v", customfields.ErrDatabaseOperation, result.Error)
	}

	if err := def.AfterFind(); err != nil {
		return nil, fmt.Errorf("%w: failed to deserialize field definition: %v", customfields.ErrDatabaseOperation, err)
	}

	return &def, nil
}

// GetDefinitionByName retrieves a field definition by entity type and name
func (s *PostgresStorage) GetDefinitionByName(ctx context.Context, entityType, name string) (*customfields.FieldDefinition, error) {
	var def customfields.FieldDefinition
	query := s.db.WithContext(ctx).Where("entity_type = ? AND name = ?", entityType, name)
	if s.tenantID != "" {
		query = query.Where("tenant_id = ?", s.tenantID)
	}

	result := query.First(&def)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, customfields.ErrFieldNotFound
		}
		return nil, fmt.Errorf("%w: failed to get field definition: %v", customfields.ErrDatabaseOperation, result.Error)
	}

	if err := def.AfterFind(); err != nil {
		return nil, fmt.Errorf("%w: failed to deserialize field definition: %v", customfields.ErrDatabaseOperation, err)
	}

	return &def, nil
}

// ListDefinitions lists field definitions matching the filter
func (s *PostgresStorage) ListDefinitions(ctx context.Context, filter *customfields.FieldFilter) ([]*customfields.FieldDefinition, error) {
	query := s.db.WithContext(ctx).Model(&customfields.FieldDefinition{})

	if s.tenantID != "" {
		query = query.Where("tenant_id = ?", s.tenantID)
	}

	if filter != nil {
		if filter.EntityType != nil {
			query = query.Where("entity_type = ?", *filter.EntityType)
		}
		if filter.FieldType != nil {
			query = query.Where("field_type = ?", *filter.FieldType)
		}
		if filter.Name != nil {
			query = query.Where("name LIKE ?", "%"+*filter.Name+"%")
		}
		if filter.IsRequired != nil {
			query = query.Where("is_required = ?", *filter.IsRequired)
		}
		if filter.IsSearchable != nil {
			query = query.Where("is_searchable = ?", *filter.IsSearchable)
		}
		if filter.DisplayGroup != nil {
			query = query.Where("display_group = ?", *filter.DisplayGroup)
		}
		if filter.TenantID != nil {
			query = query.Where("tenant_id = ?", *filter.TenantID)
		}
		if filter.Limit > 0 {
			query = query.Limit(filter.Limit)
		}
		if filter.Offset > 0 {
			query = query.Offset(filter.Offset)
		}
	}

	query = query.Order("display_order ASC, created_at ASC")

	var defs []*customfields.FieldDefinition
	if err := query.Find(&defs).Error; err != nil {
		return nil, fmt.Errorf("%w: failed to list field definitions: %v", customfields.ErrDatabaseOperation, err)
	}

	for _, def := range defs {
		if err := def.AfterFind(); err != nil {
			s.logger.Warn("failed to deserialize field definition", "id", def.ID, "error", err)
		}
	}

	return defs, nil
}

// CountDefinitions counts field definitions matching the filter
func (s *PostgresStorage) CountDefinitions(ctx context.Context, filter *customfields.FieldFilter) (int64, error) {
	query := s.db.WithContext(ctx).Model(&customfields.FieldDefinition{})

	if s.tenantID != "" {
		query = query.Where("tenant_id = ?", s.tenantID)
	}

	if filter != nil {
		if filter.EntityType != nil {
			query = query.Where("entity_type = ?", *filter.EntityType)
		}
		if filter.FieldType != nil {
			query = query.Where("field_type = ?", *filter.FieldType)
		}
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return 0, fmt.Errorf("%w: failed to count field definitions: %v", customfields.ErrDatabaseOperation, err)
	}

	return count, nil
}

// SetValue sets a single field value
func (s *PostgresStorage) SetValue(ctx context.Context, value *customfields.FieldValue) error {
	if value.TenantID == "" {
		value.TenantID = s.tenantID
	}
	value.UpdatedAt = time.Now()

	// Upsert - update if exists, create if not
	result := s.db.WithContext(ctx).Where(
		"entity_type = ? AND entity_id = ? AND field_id = ? AND tenant_id = ?",
		value.EntityType, value.EntityID, value.FieldID, value.TenantID,
	).Assign(value).FirstOrCreate(value)

	if result.Error != nil {
		return fmt.Errorf("%w: failed to set field value: %v", customfields.ErrDatabaseOperation, result.Error)
	}

	return nil
}

// SetValues sets multiple field values for an entity
func (s *PostgresStorage) SetValues(ctx context.Context, entityType, entityID string, values []*customfields.FieldValue) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, value := range values {
			value.EntityType = entityType
			value.EntityID = entityID
			if value.TenantID == "" {
				value.TenantID = s.tenantID
			}
			value.UpdatedAt = time.Now()

			result := tx.Where(
				"entity_type = ? AND entity_id = ? AND field_id = ? AND tenant_id = ?",
				value.EntityType, value.EntityID, value.FieldID, value.TenantID,
			).Assign(value).FirstOrCreate(value)

			if result.Error != nil {
				return fmt.Errorf("%w: failed to set field value: %v", customfields.ErrDatabaseOperation, result.Error)
			}
		}
		return nil
	})
}

// GetValue retrieves a single field value
func (s *PostgresStorage) GetValue(ctx context.Context, entityType, entityID, fieldID string) (*customfields.FieldValue, error) {
	var value customfields.FieldValue
	query := s.db.WithContext(ctx).Where(
		"entity_type = ? AND entity_id = ? AND field_id = ?",
		entityType, entityID, fieldID,
	)
	if s.tenantID != "" {
		query = query.Where("tenant_id = ?", s.tenantID)
	}

	result := query.First(&value)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, customfields.ErrFieldNotFound
		}
		return nil, fmt.Errorf("%w: failed to get field value: %v", customfields.ErrDatabaseOperation, result.Error)
	}

	return &value, nil
}

// GetAllValues retrieves all field values for an entity
func (s *PostgresStorage) GetAllValues(ctx context.Context, entityType, entityID string) ([]*customfields.FieldValue, error) {
	var values []*customfields.FieldValue
	query := s.db.WithContext(ctx).Where("entity_type = ? AND entity_id = ?", entityType, entityID)
	if s.tenantID != "" {
		query = query.Where("tenant_id = ?", s.tenantID)
	}

	if err := query.Find(&values).Error; err != nil {
		return nil, fmt.Errorf("%w: failed to get field values: %v", customfields.ErrDatabaseOperation, err)
	}

	return values, nil
}

// DeleteValue deletes a single field value
func (s *PostgresStorage) DeleteValue(ctx context.Context, entityType, entityID, fieldID string) error {
	query := s.db.WithContext(ctx).Where(
		"entity_type = ? AND entity_id = ? AND field_id = ?",
		entityType, entityID, fieldID,
	)
	if s.tenantID != "" {
		query = query.Where("tenant_id = ?", s.tenantID)
	}

	result := query.Delete(&customfields.FieldValue{})
	if result.Error != nil {
		return fmt.Errorf("%w: failed to delete field value: %v", customfields.ErrDatabaseOperation, result.Error)
	}

	return nil
}

// DeleteAllValues deletes all field values for an entity
func (s *PostgresStorage) DeleteAllValues(ctx context.Context, entityType, entityID string) error {
	query := s.db.WithContext(ctx).Where("entity_type = ? AND entity_id = ?", entityType, entityID)
	if s.tenantID != "" {
		query = query.Where("tenant_id = ?", s.tenantID)
	}

	result := query.Delete(&customfields.FieldValue{})
	if result.Error != nil {
		return fmt.Errorf("%w: failed to delete field values: %v", customfields.ErrDatabaseOperation, result.Error)
	}

	return nil
}

// CountValues counts field values for an entity type
func (s *PostgresStorage) CountValues(ctx context.Context, entityType string) (int64, error) {
	var count int64
	query := s.db.WithContext(ctx).Model(&customfields.FieldValue{}).Where("entity_type = ?", entityType)
	if s.tenantID != "" {
		query = query.Where("tenant_id = ?", s.tenantID)
	}

	if err := query.Count(&count).Error; err != nil {
		return 0, fmt.Errorf("%w: failed to count field values: %v", customfields.ErrDatabaseOperation, err)
	}

	return count, nil
}

// BulkSetValues sets field values for multiple entities
func (s *PostgresStorage) BulkSetValues(ctx context.Context, operations []*customfields.BulkFieldOperation) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, op := range operations {
			for fieldID, value := range op.Values {
				fv := &customfields.FieldValue{
					EntityType: op.EntityType,
					EntityID:   op.EntityID,
					FieldID:    fieldID,
					TenantID:   s.tenantID,
					UpdatedAt:  time.Now(),
				}

				// Get the field definition to determine data type
				var def customfields.FieldDefinition
				if err := tx.First(&def, "id = ?", fieldID).Error; err == nil {
					if err := def.AfterFind(); err == nil {
						if err := fv.SetValue(value, def.DataType); err != nil {
							return fmt.Errorf("%w: failed to set value for field %s: %v",
								customfields.ErrInvalidFieldValue, fieldID, err)
						}
					}
				}

				result := tx.Where(
					"entity_type = ? AND entity_id = ? AND field_id = ? AND tenant_id = ?",
					fv.EntityType, fv.EntityID, fv.FieldID, fv.TenantID,
				).Assign(fv).FirstOrCreate(fv)

				if result.Error != nil {
					return fmt.Errorf("%w: failed to set field value: %v", customfields.ErrDatabaseOperation, result.Error)
				}
			}
		}
		return nil
	})
}

// BulkGetValues retrieves field values for multiple entities
func (s *PostgresStorage) BulkGetValues(ctx context.Context, entityType string, entityIDs []string) (map[string][]*customfields.FieldValue, error) {
	var values []*customfields.FieldValue
	query := s.db.WithContext(ctx).Where("entity_type = ? AND entity_id IN ?", entityType, entityIDs)
	if s.tenantID != "" {
		query = query.Where("tenant_id = ?", s.tenantID)
	}

	if err := query.Find(&values).Error; err != nil {
		return nil, fmt.Errorf("%w: failed to get field values: %v", customfields.ErrDatabaseOperation, err)
	}

	result := make(map[string][]*customfields.FieldValue)
	for _, v := range values {
		result[v.EntityID] = append(result[v.EntityID], v)
	}

	return result, nil
}

// Search searches for entities by field values
func (s *PostgresStorage) Search(ctx context.Context, query *customfields.FieldQuery) ([]string, error) {
	results, err := s.SearchWithValues(ctx, query)
	if err != nil {
		return nil, err
	}

	entityIDs := make([]string, len(results))
	for i, r := range results {
		entityIDs[i] = r.EntityID
	}
	return entityIDs, nil
}

// SearchWithValues searches for entities and returns their field values
func (s *PostgresStorage) SearchWithValues(ctx context.Context, query *customfields.FieldQuery) ([]*customfields.EntityFieldValues, error) {
	// Build the query - this is a simplified implementation
	// For production, you'd want a more sophisticated query builder
	subQuery := s.db.WithContext(ctx).Model(&customfields.FieldValue{}).
		Select("DISTINCT entity_id").
		Where("entity_type = ?", query.EntityType)

	if s.tenantID != "" {
		subQuery = subQuery.Where("tenant_id = ?", s.tenantID)
	}

	for _, cond := range query.Conditions {
		subQuery = s.applyCondition(subQuery, cond)
	}

	// Get matching entity IDs
	var entityIDs []string
	if err := subQuery.Pluck("entity_id", &entityIDs).Error; err != nil {
		return nil, fmt.Errorf("%w: search failed: %v", customfields.ErrDatabaseOperation, err)
	}

	if len(entityIDs) == 0 {
		return []*customfields.EntityFieldValues{}, nil
	}

	// Apply pagination
	if query.Limit > 0 {
		if query.Offset >= len(entityIDs) {
			return []*customfields.EntityFieldValues{}, nil
		}
		end := query.Offset + query.Limit
		if end > len(entityIDs) {
			end = len(entityIDs)
		}
		entityIDs = entityIDs[query.Offset:end]
	}

	// Get field values for matching entities
	valuesMap, err := s.BulkGetValues(ctx, query.EntityType, entityIDs)
	if err != nil {
		return nil, err
	}

	// Build result
	results := make([]*customfields.EntityFieldValues, 0, len(entityIDs))
	for _, entityID := range entityIDs {
		values := valuesMap[entityID]
		fields := make(map[string]*customfields.FieldValue)
		for _, v := range values {
			fields[v.FieldID] = v
		}
		results = append(results, &customfields.EntityFieldValues{
			EntityType: query.EntityType,
			EntityID:   entityID,
			Fields:     fields,
			TenantID:   s.tenantID,
		})
	}

	return results, nil
}

// applyCondition applies a query condition to the query builder
func (s *PostgresStorage) applyCondition(query *gorm.DB, cond *customfields.QueryCondition) *gorm.DB {
	switch cond.Operator {
	case customfields.ConditionEquals:
		return query.Where("field_id = ? AND (value_string = ? OR value_int = ? OR value_float = ?)",
			cond.FieldID, cond.Value, cond.Value, cond.Value)
	case customfields.ConditionNotEquals:
		return query.Where("field_id = ? AND (value_string != ? AND value_int != ? AND value_float != ?)",
			cond.FieldID, cond.Value, cond.Value, cond.Value)
	case customfields.ConditionContains:
		return query.Where("field_id = ? AND value_string LIKE ?", cond.FieldID, "%"+fmt.Sprintf("%v", cond.Value)+"%")
	case customfields.ConditionGreaterThan:
		return query.Where("field_id = ? AND (value_int > ? OR value_float > ?)",
			cond.FieldID, cond.Value, cond.Value)
	case customfields.ConditionLessThan:
		return query.Where("field_id = ? AND (value_int < ? OR value_float < ?)",
			cond.FieldID, cond.Value, cond.Value)
	default:
		return query
	}
}

// CreateAuditLog creates an audit log entry
func (s *PostgresStorage) CreateAuditLog(ctx context.Context, log *customfields.FieldAuditLog) error {
	if log.TenantID == "" {
		log.TenantID = s.tenantID
	}
	log.ChangedAt = time.Now()

	if err := s.db.WithContext(ctx).Create(log).Error; err != nil {
		return fmt.Errorf("%w: failed to create audit log: %v", customfields.ErrDatabaseOperation, err)
	}
	return nil
}

// GetAuditLogs retrieves audit logs for an entity
func (s *PostgresStorage) GetAuditLogs(ctx context.Context, entityType, entityID string, limit int) ([]*customfields.FieldAuditLog, error) {
	var logs []*customfields.FieldAuditLog
	query := s.db.WithContext(ctx).
		Where("entity_type = ? AND entity_id = ?", entityType, entityID).
		Order("changed_at DESC")

	if s.tenantID != "" {
		query = query.Where("tenant_id = ?", s.tenantID)
	}
	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&logs).Error; err != nil {
		return nil, fmt.Errorf("%w: failed to get audit logs: %v", customfields.ErrDatabaseOperation, err)
	}

	return logs, nil
}

// PostgresFieldManager implements the FieldManager interface
type PostgresFieldManager struct {
	storage   *PostgresStorage
	validator *customfields.Validator
	cache     Cache
	config    *customfields.Config
	logger    *slog.Logger

	// Stats
	cacheHits   int64
	cacheMisses int64
}

// NewPostgresFieldManager creates a new PostgresFieldManager
func NewPostgresFieldManager(cfg *customfields.Config, opts ...customfields.Option) (customfields.FieldManager, error) {
	options := applyOptions(opts...)

	storage, err := NewPostgresStorage(&PostgresConfig{
		DSN:             cfg.Database.GetDSN(),
		MaxIdleConns:    cfg.Database.MaxIdleConns,
		MaxOpenConns:    cfg.Database.MaxOpenConns,
		ConnMaxLifetime: cfg.Database.ConnMaxLifetime,
		ConnMaxIdleTime: cfg.Database.ConnMaxIdleTime,
		TenantID:        cfg.DefaultTenantID,
		Logger:          options.logger,
	})
	if err != nil {
		return nil, err
	}

	// Run migrations
	if err := storage.Migrate(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	fm := &PostgresFieldManager{
		storage:   storage,
		validator: customfields.NewValidator(),
		config:    cfg,
		logger:    options.logger,
	}

	// Initialize cache if enabled
	if cfg.Cache != nil && cfg.Cache.Enabled {
		redisCache, err := NewRedisCache(&RedisCacheConfig{
			Addr:       cfg.Cache.Redis.Addr,
			Password:   cfg.Cache.Redis.Password,
			DB:         cfg.Cache.Redis.DB,
			KeyPrefix:  cfg.Cache.Redis.KeyPrefix,
			DefaultTTL: cfg.Cache.TTL,
		})
		if err != nil {
			fm.logger.Warn("failed to initialize cache, continuing without cache", "error", err)
		} else {
			fm.cache = redisCache
		}
	}

	return fm, nil
}

// applyOptions applies functional options and returns the options struct
func applyOptions(opts ...customfields.Option) *options {
	o := &options{
		logger: slog.Default(),
	}
	// Note: The functional options from customfields package use a different internal type
	// For now, we just use defaults. In a full implementation, you would pass the logger
	// through the config or use a shared options pattern.
	return o
}

type options struct {
	logger *slog.Logger
}

// CreateFieldDefinition creates a new field definition
func (fm *PostgresFieldManager) CreateFieldDefinition(ctx context.Context, def *customfields.FieldDefinition) error {
	if err := fm.storage.CreateDefinition(ctx, def); err != nil {
		return err
	}

	// Invalidate cache
	if fm.cache != nil {
		fm.cache.DeleteDefinitionsByEntity(ctx, def.EntityType)
	}

	return nil
}

// UpdateFieldDefinition updates an existing field definition
func (fm *PostgresFieldManager) UpdateFieldDefinition(ctx context.Context, id string, def *customfields.FieldDefinition) error {
	if err := fm.storage.UpdateDefinition(ctx, id, def); err != nil {
		return err
	}

	// Invalidate cache
	if fm.cache != nil {
		fm.cache.DeleteDefinition(ctx, id)
		fm.cache.DeleteDefinitionsByEntity(ctx, def.EntityType)
	}

	return nil
}

// DeleteFieldDefinition deletes a field definition
func (fm *PostgresFieldManager) DeleteFieldDefinition(ctx context.Context, id string) error {
	// Get definition first for cache invalidation
	def, err := fm.storage.GetDefinition(ctx, id)
	if err != nil {
		return err
	}

	if err := fm.storage.DeleteDefinition(ctx, id); err != nil {
		return err
	}

	// Invalidate cache
	if fm.cache != nil {
		fm.cache.DeleteDefinition(ctx, id)
		fm.cache.DeleteDefinitionsByEntity(ctx, def.EntityType)
	}

	return nil
}

// GetFieldDefinition retrieves a field definition by ID
func (fm *PostgresFieldManager) GetFieldDefinition(ctx context.Context, id string) (*customfields.FieldDefinition, error) {
	// Check cache
	if fm.cache != nil {
		if cached, err := fm.cache.GetDefinition(ctx, id); err == nil && cached != nil {
			atomic.AddInt64(&fm.cacheHits, 1)
			return cached, nil
		}
		atomic.AddInt64(&fm.cacheMisses, 1)
	}

	def, err := fm.storage.GetDefinition(ctx, id)
	if err != nil {
		return nil, err
	}

	// Update cache
	if fm.cache != nil {
		fm.cache.SetDefinition(ctx, def)
	}

	return def, nil
}

// GetFieldDefinitionByName retrieves a field definition by entity type and name
func (fm *PostgresFieldManager) GetFieldDefinitionByName(ctx context.Context, entityType, name string) (*customfields.FieldDefinition, error) {
	return fm.storage.GetDefinitionByName(ctx, entityType, name)
}

// ListFieldDefinitions lists field definitions matching the filter
func (fm *PostgresFieldManager) ListFieldDefinitions(ctx context.Context, filter *customfields.FieldFilter) ([]*customfields.FieldDefinition, error) {
	return fm.storage.ListDefinitions(ctx, filter)
}

// SetFieldValue sets field values for an entity
func (fm *PostgresFieldManager) SetFieldValue(ctx context.Context, entityType, entityID string, values map[string]interface{}) error {
	// Get definitions for validation
	defs, err := fm.storage.ListDefinitions(ctx, &customfields.FieldFilter{EntityType: &entityType})
	if err != nil {
		return err
	}

	// Create a map for quick lookup
	defMap := make(map[string]*customfields.FieldDefinition)
	for _, def := range defs {
		defMap[def.ID] = def
		defMap[def.Name] = def // Also allow lookup by name
	}

	// Validate values
	for fieldKey, value := range values {
		def, exists := defMap[fieldKey]
		if !exists {
			continue // Skip unknown fields
		}
		if err := fm.validator.ValidateField(ctx, def, value); err != nil {
			return err
		}
	}

	// Convert to FieldValue objects
	fieldValues := make([]*customfields.FieldValue, 0, len(values))
	for fieldKey, value := range values {
		def, exists := defMap[fieldKey]
		if !exists {
			continue
		}

		fv := &customfields.FieldValue{
			EntityType: entityType,
			EntityID:   entityID,
			FieldID:    def.ID,
		}
		if err := fv.SetValue(value, def.DataType); err != nil {
			return customfields.NewFieldError(def.ID, def.Name, "failed to set value", err)
		}
		fieldValues = append(fieldValues, fv)
	}

	// Save values
	if err := fm.storage.SetValues(ctx, entityType, entityID, fieldValues); err != nil {
		return err
	}

	// Invalidate cache
	if fm.cache != nil {
		fm.cache.DeleteValues(ctx, entityType, entityID)
	}

	return nil
}

// GetFieldValue retrieves a single field value
func (fm *PostgresFieldManager) GetFieldValue(ctx context.Context, entityType, entityID, fieldID string) (interface{}, error) {
	def, err := fm.GetFieldDefinition(ctx, fieldID)
	if err != nil {
		return nil, err
	}

	fv, err := fm.storage.GetValue(ctx, entityType, entityID, fieldID)
	if err != nil {
		return nil, err
	}

	return fv.GetValue(def.DataType), nil
}

// GetAllFieldValues retrieves all field values for an entity
func (fm *PostgresFieldManager) GetAllFieldValues(ctx context.Context, entityType, entityID string) (map[string]interface{}, error) {
	// Check cache
	if fm.cache != nil {
		if cached, err := fm.cache.GetValues(ctx, entityType, entityID); err == nil && cached != nil {
			atomic.AddInt64(&fm.cacheHits, 1)
			return cached, nil
		}
		atomic.AddInt64(&fm.cacheMisses, 1)
	}

	values, err := fm.storage.GetAllValues(ctx, entityType, entityID)
	if err != nil {
		return nil, err
	}

	// Get definitions
	defs, err := fm.storage.ListDefinitions(ctx, &customfields.FieldFilter{EntityType: &entityType})
	if err != nil {
		return nil, err
	}
	defMap := make(map[string]*customfields.FieldDefinition)
	for _, def := range defs {
		defMap[def.ID] = def
	}

	// Convert to map
	result := make(map[string]interface{})
	for _, fv := range values {
		if def, exists := defMap[fv.FieldID]; exists {
			result[fv.FieldID] = fv.GetValue(def.DataType)
		}
	}

	// Update cache
	if fm.cache != nil {
		fm.cache.SetValues(ctx, entityType, entityID, result)
	}

	return result, nil
}

// DeleteFieldValue deletes a single field value
func (fm *PostgresFieldManager) DeleteFieldValue(ctx context.Context, entityType, entityID, fieldID string) error {
	if err := fm.storage.DeleteValue(ctx, entityType, entityID, fieldID); err != nil {
		return err
	}

	// Invalidate cache
	if fm.cache != nil {
		fm.cache.DeleteValues(ctx, entityType, entityID)
	}

	return nil
}

// DeleteAllFieldValues deletes all field values for an entity
func (fm *PostgresFieldManager) DeleteAllFieldValues(ctx context.Context, entityType, entityID string) error {
	if err := fm.storage.DeleteAllValues(ctx, entityType, entityID); err != nil {
		return err
	}

	// Invalidate cache
	if fm.cache != nil {
		fm.cache.DeleteValues(ctx, entityType, entityID)
	}

	return nil
}

// BulkSetFieldValues sets field values for multiple entities
func (fm *PostgresFieldManager) BulkSetFieldValues(ctx context.Context, operations []*customfields.BulkFieldOperation) error {
	return fm.storage.BulkSetValues(ctx, operations)
}

// BulkGetFieldValues retrieves field values for multiple entities
func (fm *PostgresFieldManager) BulkGetFieldValues(ctx context.Context, entityType string, entityIDs []string) (map[string]map[string]interface{}, error) {
	valuesMap, err := fm.storage.BulkGetValues(ctx, entityType, entityIDs)
	if err != nil {
		return nil, err
	}

	// Get definitions
	defs, err := fm.storage.ListDefinitions(ctx, &customfields.FieldFilter{EntityType: &entityType})
	if err != nil {
		return nil, err
	}
	defMap := make(map[string]*customfields.FieldDefinition)
	for _, def := range defs {
		defMap[def.ID] = def
	}

	// Convert to result format
	result := make(map[string]map[string]interface{})
	for entityID, values := range valuesMap {
		result[entityID] = make(map[string]interface{})
		for _, fv := range values {
			if def, exists := defMap[fv.FieldID]; exists {
				result[entityID][fv.FieldID] = fv.GetValue(def.DataType)
			}
		}
	}

	return result, nil
}

// ValidateFieldValue validates a single field value
func (fm *PostgresFieldManager) ValidateFieldValue(ctx context.Context, defID string, value interface{}) error {
	def, err := fm.GetFieldDefinition(ctx, defID)
	if err != nil {
		return err
	}
	return fm.validator.ValidateField(ctx, def, value)
}

// ValidateFieldValues validates all field values for an entity type
func (fm *PostgresFieldManager) ValidateFieldValues(ctx context.Context, entityType string, values map[string]interface{}) error {
	defs, err := fm.storage.ListDefinitions(ctx, &customfields.FieldFilter{EntityType: &entityType})
	if err != nil {
		return err
	}
	return fm.validator.ValidateEntity(ctx, defs, values)
}

// SearchByFieldValue searches for entities by field values
func (fm *PostgresFieldManager) SearchByFieldValue(ctx context.Context, query *customfields.FieldQuery) ([]*customfields.EntityFieldValues, error) {
	return fm.storage.SearchWithValues(ctx, query)
}

// GetEntitySchema returns the schema for an entity type
func (fm *PostgresFieldManager) GetEntitySchema(ctx context.Context, entityType string) (*customfields.EntitySchema, error) {
	defs, err := fm.storage.ListDefinitions(ctx, &customfields.FieldFilter{EntityType: &entityType})
	if err != nil {
		return nil, err
	}

	return &customfields.EntitySchema{
		EntityType: entityType,
		Fields:     defs,
		Version:    1,
		UpdatedAt:  time.Now(),
	}, nil
}

// GenerateJSONSchema generates a JSON Schema for an entity type
func (fm *PostgresFieldManager) GenerateJSONSchema(ctx context.Context, entityType string) (map[string]interface{}, error) {
	defs, err := fm.storage.ListDefinitions(ctx, &customfields.FieldFilter{EntityType: &entityType})
	if err != nil {
		return nil, err
	}

	properties := make(map[string]interface{})
	required := []string{}

	for _, def := range defs {
		prop := fm.fieldToJSONSchema(def)
		properties[def.Name] = prop

		if def.IsRequired {
			required = append(required, def.Name)
		}
	}

	schema := map[string]interface{}{
		"$schema":    "http://json-schema.org/draft-07/schema#",
		"type":       "object",
		"title":      entityType,
		"properties": properties,
	}

	if len(required) > 0 {
		schema["required"] = required
	}

	return schema, nil
}

// fieldToJSONSchema converts a field definition to JSON Schema
func (fm *PostgresFieldManager) fieldToJSONSchema(def *customfields.FieldDefinition) map[string]interface{} {
	schema := map[string]interface{}{
		"title":       def.Label,
		"description": def.Description,
	}

	switch def.FieldType {
	case customfields.FieldTypeText, customfields.FieldTypeTextarea, customfields.FieldTypeEmail,
		customfields.FieldTypeURL, customfields.FieldTypePhone, customfields.FieldTypeColor,
		customfields.FieldTypeRichText:
		schema["type"] = "string"
	case customfields.FieldTypeNumber:
		schema["type"] = "integer"
	case customfields.FieldTypeDecimal, customfields.FieldTypeCurrency, customfields.FieldTypePercentage:
		schema["type"] = "number"
	case customfields.FieldTypeBoolean, customfields.FieldTypeCheckbox:
		schema["type"] = "boolean"
	case customfields.FieldTypeDate, customfields.FieldTypeDateTime:
		schema["type"] = "string"
		schema["format"] = "date-time"
	case customfields.FieldTypeSelect, customfields.FieldTypeRadio:
		schema["type"] = "string"
		if len(def.Options) > 0 {
			enum := make([]string, len(def.Options))
			for i, opt := range def.Options {
				enum[i] = opt.Value
			}
			schema["enum"] = enum
		}
	case customfields.FieldTypeMultiSelect:
		schema["type"] = "array"
		schema["items"] = map[string]interface{}{"type": "string"}
	case customfields.FieldTypeJSON, customfields.FieldTypeObject:
		schema["type"] = "object"
	case customfields.FieldTypeArray:
		schema["type"] = "array"
	case customfields.FieldTypeRating:
		schema["type"] = "integer"
		schema["minimum"] = 0
		schema["maximum"] = 5
	}

	// Add validation rules
	if def.Validation != nil {
		if def.Validation.MinLength != nil {
			schema["minLength"] = *def.Validation.MinLength
		}
		if def.Validation.MaxLength != nil {
			schema["maxLength"] = *def.Validation.MaxLength
		}
		if def.Validation.Min != nil {
			schema["minimum"] = *def.Validation.Min
		}
		if def.Validation.Max != nil {
			schema["maximum"] = *def.Validation.Max
		}
		if def.Validation.Pattern != nil {
			schema["pattern"] = *def.Validation.Pattern
		}
	}

	return schema
}

// CheckFieldPermission checks if a user has permission to access a field
func (fm *PostgresFieldManager) CheckFieldPermission(ctx context.Context, userID, entityType, fieldID, action string) (bool, error) {
	def, err := fm.GetFieldDefinition(ctx, fieldID)
	if err != nil {
		return false, err
	}

	if def.Permissions == nil {
		return true, nil // No permissions defined, allow by default
	}

	switch action {
	case "read":
		if len(def.Permissions.ReadRoles) == 0 {
			return true, nil
		}
		// Would integrate with authorization package here
		return true, nil
	case "write":
		if len(def.Permissions.WriteRoles) == 0 {
			return true, nil
		}
		return true, nil
	}

	return true, nil
}

// GetUserVisibleFields returns fields visible to a user
func (fm *PostgresFieldManager) GetUserVisibleFields(ctx context.Context, userID, entityType string) ([]*customfields.FieldDefinition, error) {
	defs, err := fm.storage.ListDefinitions(ctx, &customfields.FieldFilter{EntityType: &entityType})
	if err != nil {
		return nil, err
	}

	// Filter by permissions - simplified version
	visible := make([]*customfields.FieldDefinition, 0, len(defs))
	for _, def := range defs {
		canRead, _ := fm.CheckFieldPermission(ctx, userID, entityType, def.ID, "read")
		if canRead {
			visible = append(visible, def)
		}
	}

	return visible, nil
}

// ExportFieldDefinitions exports field definitions for an entity type
func (fm *PostgresFieldManager) ExportFieldDefinitions(ctx context.Context, entityType string) ([]byte, error) {
	defs, err := fm.storage.ListDefinitions(ctx, &customfields.FieldFilter{EntityType: &entityType})
	if err != nil {
		return nil, err
	}
	return json.Marshal(defs)
}

// ImportFieldDefinitions imports field definitions
func (fm *PostgresFieldManager) ImportFieldDefinitions(ctx context.Context, data []byte) error {
	var defs []*customfields.FieldDefinition
	if err := json.Unmarshal(data, &defs); err != nil {
		return fmt.Errorf("%w: invalid import data: %v", customfields.ErrInvalidInput, err)
	}

	for _, def := range defs {
		def.ID = "" // Clear ID to create new
		if err := fm.CreateFieldDefinition(ctx, def); err != nil {
			return err
		}
	}

	return nil
}

// InvalidateCache invalidates cache for an entity type
func (fm *PostgresFieldManager) InvalidateCache(ctx context.Context, entityType string) error {
	if fm.cache != nil {
		return fm.cache.DeleteDefinitionsByEntity(ctx, entityType)
	}
	return nil
}

// InvalidateEntityCache invalidates cache for a specific entity
func (fm *PostgresFieldManager) InvalidateEntityCache(ctx context.Context, entityType, entityID string) error {
	if fm.cache != nil {
		return fm.cache.DeleteValues(ctx, entityType, entityID)
	}
	return nil
}

// HealthCheck checks the health of the field manager
func (fm *PostgresFieldManager) HealthCheck(ctx context.Context) error {
	if err := fm.storage.Ping(ctx); err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}
	if fm.cache != nil {
		if err := fm.cache.Ping(ctx); err != nil {
			return fmt.Errorf("cache health check failed: %w", err)
		}
	}
	return nil
}

// GetStats returns statistics for the field manager
func (fm *PostgresFieldManager) GetStats(ctx context.Context) (*customfields.FieldStats, error) {
	defCount, _ := fm.storage.CountDefinitions(ctx, nil)
	valCount, _ := fm.storage.CountValues(ctx, "")

	hits := atomic.LoadInt64(&fm.cacheHits)
	misses := atomic.LoadInt64(&fm.cacheMisses)
	var hitRate float64
	if hits+misses > 0 {
		hitRate = float64(hits) / float64(hits+misses)
	}

	return &customfields.FieldStats{
		TotalDefinitions: defCount,
		TotalValues:      valCount,
		CacheHits:        hits,
		CacheMisses:      misses,
		CacheHitRate:     hitRate,
	}, nil
}

// Close closes the field manager
func (fm *PostgresFieldManager) Close() error {
	if fm.cache != nil {
		fm.cache.Close()
	}
	return fm.storage.Close()
}
