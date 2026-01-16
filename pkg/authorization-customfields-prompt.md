# Authorization & Custom Fields Package Implementation Prompt

## Tổng quan

Implement 2 package chính trong `pkg/` directory cho microservices architecture:

1. **authorization** - Casbin-based RBAC/ABAC authorization system
2. **customfields** - Flexible custom fields management với permission control

## Package 1: Authorization (Casbin-based)

### Cấu trúc thư mục

```text
pkg/authorization/
├── README.md
├── ARCHITECTURE.md
├── go.mod
├── authorization.go          # Main interface & factory
├── types.go                  # Types, constants, errors
├── models.go                 # Domain models
├── middleware.go             # HTTP middleware integration
├── authorization_test.go
├── casbin/
│   ├── casbin.go            # Casbin adapter implementation
│   ├── enforcer.go          # Custom enforcer với caching
│   ├── adapter.go           # Database adapter (PostgreSQL, Redis)
│   ├── models/
│   │   ├── rbac_model.conf  # RBAC model
│   │   ├── abac_model.conf  # ABAC model
│   │   └── hybrid_model.conf # Hybrid RBAC+ABAC
│   ├── policies/
│   │   └── default_policies.csv
│   └── watcher.go           # Distributed policy watcher
├── cache/
│   ├── permission_cache.go  # Permission decision cache
│   └── policy_cache.go      # Policy cache layer
└── examples/
    ├── basic_rbac.go
    ├── advanced_abac.go
    └── integration_example.go
```

### Core Requirements

#### 1. Main Interface (authorization.go)

```go
package authorization

import "context"

// Authorizer là main interface cho authorization system
type Authorizer interface {
    // Enforce kiểm tra permission
    Enforce(ctx context.Context, request *AuthRequest) (bool, error)

    // EnforceBatch kiểm tra multiple permissions cùng lúc
    EnforceBatch(ctx context.Context, requests []*AuthRequest) ([]bool, error)

    // Policy Management
    AddPolicy(ctx context.Context, policy *Policy) error
    RemovePolicy(ctx context.Context, policy *Policy) error
    UpdatePolicy(ctx context.Context, old, new *Policy) error
    GetPolicies(ctx context.Context, filter *PolicyFilter) ([]*Policy, error)

    // Role Management
    AddRole(ctx context.Context, role *Role) error
    RemoveRole(ctx context.Context, roleID string) error
    AssignRole(ctx context.Context, userID, roleID string) error
    RevokeRole(ctx context.Context, userID, roleID string) error
    GetUserRoles(ctx context.Context, userID string) ([]*Role, error)

    // Permission Management
    AddPermission(ctx context.Context, perm *Permission) error
    RemovePermission(ctx context.Context, permID string) error
    GetRolePermissions(ctx context.Context, roleID string) ([]*Permission, error)

    // Dynamic Resource Authorization
    CheckResourceAccess(ctx context.Context, userID, resource, action string, attributes map[string]interface{}) (bool, error)

    // Hierarchy & Inheritance
    AddRoleInheritance(ctx context.Context, child, parent string) error
    RemoveRoleInheritance(ctx context.Context, child, parent string) error

    // Cache Management
    InvalidateCache(ctx context.Context, keys ...string) error
    RefreshPolicies(ctx context.Context) error

    // Health & Stats
    HealthCheck(ctx context.Context) error
    GetStats(ctx context.Context) (*AuthStats, error)
}

// Factory function
func NewAuthorizer(config *Config) (Authorizer, error)
```

#### 2. Types & Models (types.go, models.go)

```go
// AuthRequest represents an authorization request
type AuthRequest struct {
    Subject    string                 // User/Service ID
    Object     string                 // Resource
    Action     string                 // Operation (read, write, delete, etc.)
    Attributes map[string]interface{} // Additional context
    TenantID   string                 // Multi-tenancy support
}

// Policy represents an authorization policy
type Policy struct {
    ID          string
    Type        PolicyType // RBAC, ABAC, Rule-based
    Subject     string
    Object      string
    Action      string
    Effect      Effect // Allow, Deny
    Conditions  []*Condition
    Priority    int
    TenantID    string
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

// Role with hierarchical support
type Role struct {
    ID          string
    Name        string
    Description string
    ParentID    *string // For role hierarchy
    Permissions []*Permission
    Metadata    map[string]interface{}
    TenantID    string
    IsActive    bool
    CreatedAt   time.Time
}

// Permission with resource-level granularity
type Permission struct {
    ID          string
    Resource    string // e.g., "order", "product"
    Action      string // e.g., "read", "write", "delete"
    Scope       Scope  // Global, Tenant, Resource, Owner
    Conditions  []*Condition
    Description string
}

// Condition for ABAC
type Condition struct {
    Field    string
    Operator ConditionOperator // Equals, NotEquals, In, GreaterThan, etc.
    Value    interface{}
}

type PolicyType string
const (
    PolicyTypeRBAC     PolicyType = "rbac"
    PolicyTypeABAC     PolicyType = "abac"
    PolicyTypeRuleBased PolicyType = "rule_based"
)

type Effect string
const (
    EffectAllow Effect = "allow"
    EffectDeny  Effect = "deny"
)

type Scope string
const (
    ScopeGlobal   Scope = "global"
    ScopeTenant   Scope = "tenant"
    ScopeResource Scope = "resource"
    ScopeOwner    Scope = "owner"
)

type ConditionOperator string
const (
    OpEquals        ConditionOperator = "eq"
    OpNotEquals     ConditionOperator = "ne"
    OpIn            ConditionOperator = "in"
    OpNotIn         ConditionOperator = "nin"
    OpGreaterThan   ConditionOperator = "gt"
    OpLessThan      ConditionOperator = "lt"
    OpContains      ConditionOperator = "contains"
    OpRegex         ConditionOperator = "regex"
)
```

#### 3. Casbin Implementation (casbin/casbin.go)

```go
package casbin

import (
    "github.com/casbin/casbin/v2"
    "github.com/casbin/casbin/v2/model"
    "github.com/casbin/casbin/v2/persist"
)

type CasbinAuthorizer struct {
    enforcer      *casbin.SyncedEnforcer
    adapter       persist.Adapter
    watcher       persist.Watcher
    cache         Cache
    config        *Config
    metrics       *Metrics
}

func NewCasbinAuthorizer(config *Config) (*CasbinAuthorizer, error) {
    // 1. Initialize model (RBAC, ABAC, or Hybrid)
    // 2. Setup adapter (PostgreSQL, Redis)
    // 3. Configure watcher for distributed systems
    // 4. Setup permission cache
    // 5. Load policies
    // 6. Initialize metrics
}

// Implement high-performance enforcement với caching
func (ca *CasbinAuthorizer) Enforce(ctx context.Context, req *AuthRequest) (bool, error) {
    // 1. Check cache first
    // 2. Build casbin request
    // 3. Evaluate with enforcer
    // 4. Cache result
    // 5. Record metrics
}
```

#### 4. Database Adapter (casbin/adapter.go)

```go
// PostgreSQL Adapter for policy storage
type PostgreSQLAdapter struct {
    db        *sql.DB
    tableName string
}

// Implement persist.Adapter interface
func (a *PostgreSQLAdapter) LoadPolicy(model model.Model) error
func (a *PostgreSQLAdapter) SavePolicy(model model.Model) error
func (a *PostgreSQLAdapter) AddPolicy(sec, ptype string, rule []string) error
func (a *PostgreSQLAdapter) RemovePolicy(sec, ptype string, rule []string) error
func (a *PostgreSQLAdapter) RemoveFilteredPolicy(sec, ptype string, fieldIndex int, fieldValues ...string) error

// Migration schema
const policyTableSchema = `
CREATE TABLE IF NOT EXISTS casbin_rule (
    id SERIAL PRIMARY KEY,
    ptype VARCHAR(100) NOT NULL,
    v0 VARCHAR(100),
    v1 VARCHAR(100),
    v2 VARCHAR(100),
    v3 VARCHAR(100),
    v4 VARCHAR(100),
    v5 VARCHAR(100),
    tenant_id VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_ptype (ptype),
    INDEX idx_tenant (tenant_id),
    INDEX idx_v0_v1_v2 (v0, v1, v2)
);
`
```

#### 5. Middleware Integration (middleware.go)

```go
// HTTP Middleware
func AuthorizationMiddleware(authorizer Authorizer, options ...MiddlewareOption) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // 1. Extract user from context/JWT
            // 2. Extract resource & action from request
            // 3. Build AuthRequest
            // 4. Enforce
            // 5. Handle unauthorized
        })
    }
}

// gRPC Interceptor
func UnaryServerInterceptor(authorizer Authorizer) grpc.UnaryServerInterceptor
func StreamServerInterceptor(authorizer Authorizer) grpc.StreamServerInterceptor
```

#### 6. Casbin Models (casbin/models/)

**rbac_model.conf**:

```conf
[request_definition]
r = sub, obj, act, tenant

[policy_definition]
p = sub, obj, act, eft, tenant

[role_definition]
g = _, _, tenant
g2 = _, _, tenant

[policy_effect]
e = some(where (p.eft == allow)) && !some(where (p.eft == deny))

[matchers]
m = g(r.sub, p.sub, r.tenant) && r.obj == p.obj && r.act == p.act && r.tenant == p.tenant
```

**abac_model.conf**:

```conf
[request_definition]
r = sub, obj, act, ctx

[policy_definition]
p = sub_rule, obj, act, eft

[policy_effect]
e = some(where (p.eft == allow)) && !some(where (p.eft == deny))

[matchers]
m = eval(p.sub_rule) && r.obj == p.obj && r.act == p.act
```

#### 7. Performance Features

- **Multi-level caching**: Memory + Redis cho permission decisions
- **Batch enforcement**: Reduce database roundtrips
- **Policy compilation**: Pre-compile policies cho fast evaluation
- **Distributed watcher**: Redis Pub/Sub cho policy sync across instances
- **Connection pooling**: Optimized database connections
- **Lazy loading**: Load policies on-demand

---

## Package 2: Custom Fields

### Custom Fields Directory Structure

```text
pkg/customfields/
├── README.md
├── ARCHITECTURE.md
├── go.mod
├── customfields.go          # Main interface
├── types.go                 # Field types & definitions
├── validator.go             # Field validation
├── transformer.go           # Data transformation
├── customfields_test.go
├── storage/
│   ├── postgres.go          # PostgreSQL storage
│   ├── mongodb.go           # MongoDB storage (for flexible schema)
│   └── cache.go             # Field definition cache
├── renderer/
│   ├── json.go              # JSON rendering
│   ├── form.go              # HTML form generation
│   └── schema.go            # JSON Schema generation
├── permissions/
│   ├── field_permissions.go # Field-level permission
│   └── integration.go       # Integration với authorization package
└── examples/
    ├── basic_usage.go
    └── with_permissions.go
```

### Core Requirements

#### 1. Main Interface (customfields.go)

```go
package customfields

import "context"

// FieldManager quản lý custom fields
type FieldManager interface {
    // Field Definition Management
    CreateFieldDefinition(ctx context.Context, def *FieldDefinition) error
    UpdateFieldDefinition(ctx context.Context, id string, def *FieldDefinition) error
    DeleteFieldDefinition(ctx context.Context, id string) error
    GetFieldDefinition(ctx context.Context, id string) (*FieldDefinition, error)
    ListFieldDefinitions(ctx context.Context, filter *FieldFilter) ([]*FieldDefinition, error)

    // Field Value Management
    SetFieldValue(ctx context.Context, entityType, entityID string, values map[string]interface{}) error
    GetFieldValue(ctx context.Context, entityType, entityID string, fieldID string) (interface{}, error)
    GetAllFieldValues(ctx context.Context, entityType, entityID string) (map[string]interface{}, error)
    DeleteFieldValue(ctx context.Context, entityType, entityID, fieldID string) error

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
}

func NewFieldManager(config *Config) (FieldManager, error)
```

#### 2. Field Types & Definitions (types.go)

```go
// FieldDefinition định nghĩa một custom field
type FieldDefinition struct {
    ID              string
    EntityType      string // "order", "product", "user", etc.
    Name            string
    Label           string
    Description     string
    FieldType       FieldType
    DataType        DataType
    IsRequired      bool
    IsUnique        bool
    IsSearchable    bool
    IsSortable      bool

    // Validation
    Validation      *ValidationRules

    // Default Value
    DefaultValue    interface{}

    // Options for select/multi-select
    Options         []*FieldOption

    // Dependencies
    DependsOn       *FieldDependency

    // Display
    DisplayOrder    int
    DisplayGroup    string
    Placeholder     string
    HelpText        string

    // Permissions
    Permissions     *FieldPermissions

    // Storage optimization
    Indexed         bool
    Encrypted       bool

    // Metadata
    Metadata        map[string]interface{}
    TenantID        string
    Version         int
    CreatedBy       string
    CreatedAt       time.Time
    UpdatedAt       time.Time
}

type FieldType string
const (
    FieldTypeText         FieldType = "text"
    FieldTypeTextarea     FieldType = "textarea"
    FieldTypeNumber       FieldType = "number"
    FieldTypeDecimal      FieldType = "decimal"
    FieldTypeBoolean      FieldType = "boolean"
    FieldTypeDate         FieldType = "date"
    FieldTypeDateTime     FieldType = "datetime"
    FieldTypeTime         FieldType = "time"
    FieldTypeEmail        FieldType = "email"
    FieldTypeURL          FieldType = "url"
    FieldTypePhone        FieldType = "phone"
    FieldTypeSelect       FieldType = "select"
    FieldTypeMultiSelect  FieldType = "multi_select"
    FieldTypeRadio        FieldType = "radio"
    FieldTypeCheckbox     FieldType = "checkbox"
    FieldTypeFile         FieldType = "file"
    FieldTypeImage        FieldType = "image"
    FieldTypeJSON         FieldType = "json"
    FieldTypeArray        FieldType = "array"
    FieldTypeObject       FieldType = "object"
    FieldTypeReference    FieldType = "reference" // Foreign key
    FieldTypeFormula      FieldType = "formula"   // Computed field
    FieldTypeRichText     FieldType = "richtext"
    FieldTypeColor        FieldType = "color"
    FieldTypeCurrency     FieldType = "currency"
    FieldTypePercentage   FieldType = "percentage"
    FieldTypeRating       FieldType = "rating"
    FieldTypeLocation     FieldType = "location"  // Lat/Lng
)

type DataType string
const (
    DataTypeString   DataType = "string"
    DataTypeInt      DataType = "int"
    DataTypeFloat    DataType = "float"
    DataTypeBool     DataType = "bool"
    DataTypeDate     DataType = "date"
    DataTypeJSON     DataType = "json"
    DataTypeBinary   DataType = "binary"
)

// ValidationRules định nghĩa validation rules
type ValidationRules struct {
    MinLength    *int
    MaxLength    *int
    Min          *float64
    Max          *float64
    Pattern      *string // Regex
    Format       *string // email, url, etc.
    Enum         []interface{}
    Custom       *CustomValidation
}

type CustomValidation struct {
    FunctionName string
    Parameters   map[string]interface{}
    ErrorMessage string
}

// FieldOption for select/multi-select fields
type FieldOption struct {
    Value       string
    Label       string
    Color       string
    Icon        string
    IsDefault   bool
    DisplayOrder int
}

// FieldDependency cho conditional fields
type FieldDependency struct {
    FieldID      string
    Condition    DependencyCondition
    Value        interface{}
    ShowIf       bool // true = show if condition met, false = hide
}

type DependencyCondition string
const (
    ConditionEquals       DependencyCondition = "equals"
    ConditionNotEquals    DependencyCondition = "not_equals"
    ConditionContains     DependencyCondition = "contains"
    ConditionGreaterThan  DependencyCondition = "gt"
    ConditionLessThan     DependencyCondition = "lt"
)

// FieldPermissions cho field-level access control
type FieldPermissions struct {
    ReadRoles   []string
    WriteRoles  []string
    ViewPolicy  *PermissionPolicy
    EditPolicy  *PermissionPolicy
}

type PermissionPolicy struct {
    Type       string // "role", "attribute", "custom"
    Rules      []string
    CustomFunc *string
}

// EntityFieldValues represents field values for an entity
type EntityFieldValues struct {
    EntityType string
    EntityID   string
    Fields     map[string]*FieldValue
    TenantID   string
    UpdatedAt  time.Time
}

type FieldValue struct {
    FieldID      string
    Value        interface{}
    RawValue     []byte // For binary/encrypted data
    DisplayValue string
    UpdatedBy    string
    UpdatedAt    time.Time
}
```

#### 3. Validator Implementation (validator.go)

```go
package customfields

type Validator struct {
    customValidators map[string]CustomValidatorFunc
}

type CustomValidatorFunc func(ctx context.Context, value interface{}, params map[string]interface{}) error

func NewValidator() *Validator

// Register custom validator
func (v *Validator) RegisterCustomValidator(name string, fn CustomValidatorFunc)

// Validate single field value
func (v *Validator) ValidateField(ctx context.Context, def *FieldDefinition, value interface{}) error {
    // 1. Type validation
    // 2. Required validation
    // 3. Format validation (email, url, phone, etc.)
    // 4. Range validation (min, max, length)
    // 5. Pattern validation (regex)
    // 6. Enum validation
    // 7. Custom validation
    // 8. Reference validation (check if referenced entity exists)
}

// Validate all fields for an entity
func (v *Validator) ValidateEntity(ctx context.Context, entityType string, values map[string]interface{}) error

// Validate field dependencies
func (v *Validator) ValidateDependencies(ctx context.Context, fields []*FieldDefinition, values map[string]interface{}) error
```

#### 4. Storage Layer (storage/postgres.go)

```go
package storage

// PostgreSQL schema for custom fields
const schemaSQL = `
-- Field definitions table
CREATE TABLE custom_field_definitions (
    id VARCHAR(100) PRIMARY KEY,
    entity_type VARCHAR(100) NOT NULL,
    name VARCHAR(100) NOT NULL,
    label VARCHAR(200),
    description TEXT,
    field_type VARCHAR(50) NOT NULL,
    data_type VARCHAR(50) NOT NULL,
    is_required BOOLEAN DEFAULT false,
    is_unique BOOLEAN DEFAULT false,
    is_searchable BOOLEAN DEFAULT false,
    is_sortable BOOLEAN DEFAULT false,
    validation_rules JSONB,
    default_value JSONB,
    options JSONB,
    depends_on JSONB,
    display_order INT DEFAULT 0,
    display_group VARCHAR(100),
    placeholder VARCHAR(200),
    help_text TEXT,
    permissions JSONB,
    indexed BOOLEAN DEFAULT false,
    encrypted BOOLEAN DEFAULT false,
    metadata JSONB,
    tenant_id VARCHAR(100),
    version INT DEFAULT 1,
    created_by VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(entity_type, name, tenant_id),
    INDEX idx_entity_type (entity_type),
    INDEX idx_tenant (tenant_id),
    INDEX idx_searchable (is_searchable) WHERE is_searchable = true
);

-- Field values table (EAV pattern with optimizations)
CREATE TABLE custom_field_values (
    id BIGSERIAL PRIMARY KEY,
    entity_type VARCHAR(100) NOT NULL,
    entity_id VARCHAR(100) NOT NULL,
    field_id VARCHAR(100) NOT NULL,

    -- Multiple columns for different data types (optimization)
    value_string TEXT,
    value_int BIGINT,
    value_float DOUBLE PRECISION,
    value_bool BOOLEAN,
    value_date TIMESTAMP,
    value_json JSONB,
    value_binary BYTEA,

    display_value TEXT,
    tenant_id VARCHAR(100),
    updated_by VARCHAR(100),
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (field_id) REFERENCES custom_field_definitions(id) ON DELETE CASCADE,
    UNIQUE(entity_type, entity_id, field_id, tenant_id),
    INDEX idx_entity (entity_type, entity_id),
    INDEX idx_tenant (tenant_id),
    INDEX idx_field (field_id),
    INDEX idx_value_string (value_string) WHERE value_string IS NOT NULL,
    INDEX idx_value_json (value_json) USING GIN WHERE value_json IS NOT NULL
);

-- Audit log for field changes
CREATE TABLE custom_field_audit_log (
    id BIGSERIAL PRIMARY KEY,
    entity_type VARCHAR(100),
    entity_id VARCHAR(100),
    field_id VARCHAR(100),
    action VARCHAR(50), -- create, update, delete
    old_value JSONB,
    new_value JSONB,
    changed_by VARCHAR(100),
    changed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    tenant_id VARCHAR(100),

    INDEX idx_entity_audit (entity_type, entity_id),
    INDEX idx_changed_at (changed_at)
);
`

type PostgresStorage struct {
    db    *sql.DB
    cache Cache
}

func (s *PostgresStorage) SaveFieldValue(ctx context.Context, value *FieldValue) error {
    // Use appropriate column based on data type
    // Update cache
    // Log audit
}
```

#### 5. Permission Integration (permissions/field_permissions.go)

```go
package permissions

import (
    "github.com/your-org/pkg/authorization"
    "github.com/your-org/pkg/customfields"
)

type FieldPermissionManager struct {
    authorizer authorization.Authorizer
    fieldMgr   customfields.FieldManager
}

func NewFieldPermissionManager(authorizer authorization.Authorizer, fieldMgr customfields.FieldManager) *FieldPermissionManager

// Check if user can access a field
func (fpm *FieldPermissionManager) CanAccessField(ctx context.Context, userID, entityType, fieldID, action string) (bool, error) {
    // 1. Get field definition
    // 2. Check field-level permissions
    // 3. Check role-based permissions via authorizer
    // 4. Evaluate attribute-based conditions
    // 5. Cache result
}

// Filter fields based on user permissions
func (fpm *FieldPermissionManager) FilterFieldsByPermission(ctx context.Context, userID string, fields []*customfields.FieldDefinition, action string) ([]*customfields.FieldDefinition, error)

// Mask sensitive field values
func (fpm *FieldPermissionManager) MaskSensitiveFields(ctx context.Context, userID string, values map[string]interface{}) (map[string]interface{}, error)
```

---

## Integration Examples

### Example 1: Basic Authorization

```go
// Initialize authorizer
authConfig := &authorization.Config{
    ModelType: authorization.ModelTypeRBAC,
    Adapter: &authorization.PostgreSQLConfig{
        DSN: "postgres://...",
    },
    Cache: &authorization.CacheConfig{
        Enabled: true,
        TTL:     5 * time.Minute,
    },
}

authorizer, err := authorization.NewAuthorizer(authConfig)

// Create roles
adminRole := &authorization.Role{
    ID:   "admin",
    Name: "Administrator",
    Permissions: []*authorization.Permission{
        {Resource: "order", Action: "read", Scope: authorization.ScopeGlobal},
        {Resource: "order", Action: "write", Scope: authorization.ScopeGlobal},
    },
}
authorizer.AddRole(ctx, adminRole)

// Assign role to user
authorizer.AssignRole(ctx, "user123", "admin")

// Check permission
allowed, _ := authorizer.Enforce(ctx, &authorization.AuthRequest{
    Subject: "user123",
    Object:  "order",
    Action:  "read",
})
```

### Example 2: Custom Fields với Permissions

```go
// Initialize field manager
fieldConfig := &customfields.Config{
    Storage: &customfields.PostgreSQLConfig{
        DSN: "postgres://...",
    },
}
fieldMgr, err := customfields.NewFieldManager(fieldConfig)

// Create custom field with permissions
field := &customfields.FieldDefinition{
    EntityType: "order",
    Name:       "priority_level",
    Label:      "Priority Level",
    FieldType:  customfields.FieldTypeSelect,
    Options: []*customfields.FieldOption{
        {Value: "low", Label: "Low"},
        {Value: "medium", Label: "Medium"},
        {Value: "high", Label: "High"},
    },
    Permissions: &customfields.FieldPermissions{
        ReadRoles:  []string{"admin", "manager", "staff"},
        WriteRoles: []string{"admin", "manager"},
    },
}
fieldMgr.CreateFieldDefinition(ctx, field)

// Set field value with permission check
permMgr := permissions.NewFieldPermissionManager(authorizer, fieldMgr)
canWrite, _ := permMgr.CanAccessField(ctx, "user123", "order", field.ID, "write")
if canWrite {
    fieldMgr.SetFieldValue(ctx, "order", "order-456", map[string]interface{}{
        "priority_level": "high",
    })
}
```

### Example 3: ABAC với Custom Fields

```go
// Advanced ABAC policy: Allow if user's department matches order's department
policy := &authorization.Policy{
    Type:    authorization.PolicyTypeABAC,
    Subject: "user",
    Object:  "order",
    Action:  "update",
    Effect:  authorization.EffectAllow,
    Conditions: []*authorization.Condition{
        {
            Field:    "user.department",
            Operator: authorization.OpEquals,
            Value:    "order.custom_fields.department",
        },
        {
            Field:    "order.custom_fields.status",
            Operator: authorization.OpIn,
            Value:    []string{"pending", "processing"},
        },
    },
}
authorizer.AddPolicy(ctx, policy)

// Check with attributes
allowed, _ := authorizer.CheckResourceAccess(ctx, "user123", "order", "update", map[string]interface{}{
    "user": map[string]interface{}{
        "department": "sales",
    },
    "order": map[string]interface{}{
        "custom_fields": map[string]interface{}{
            "department": "sales",
            "status":     "pending",
        },
    },
})
```

---

## Technical Requirements

### Performance

1. **Authorization**:
   - Sub-millisecond policy evaluation với caching
   - Support 10,000+ policies without performance degradation
   - Batch enforcement: 100+ requests in single call
   - Distributed caching với Redis
   - Policy compilation và pre-loading

2. **Custom Fields**:
   - Efficient EAV storage với indexed columns
   - Field definition caching
   - Bulk operations support
   - Query optimization cho searchable fields
   - Lazy loading cho large datasets

### Scalability

- Horizontal scaling support
- Multi-tenancy isolation
- Distributed policy synchronization
- Connection pooling
- Circuit breaker pattern

### Security

- Encrypted field values
- Audit logging for all permission changes
- RBAC/ABAC hybrid models
- Field-level encryption
- SQL injection prevention

### Monitoring

- Prometheus metrics
- Permission check latency
- Cache hit/miss rates
- Policy evaluation stats
- Field access patterns

### Testing

- Unit tests: >90% coverage
- Integration tests với PostgreSQL/Redis
- Benchmark tests
- Concurrent access tests
- Load tests

---

## Implementation Checklist

### Authorization Package

- [ ] Core interfaces và types
- [ ] Casbin integration
- [ ] PostgreSQL adapter
- [ ] Redis cache layer
- [ ] Distributed watcher
- [ ] RBAC model implementation
- [ ] ABAC model implementation
- [ ] Middleware (HTTP, gRPC)
- [ ] Batch enforcement
- [ ] Role hierarchy
- [ ] Metrics và monitoring
- [ ] Comprehensive tests
- [ ] Documentation
- [ ] Examples

### Custom Fields Package

- [ ] Core interfaces và types
- [ ] Field validation engine
- [ ] PostgreSQL storage với EAV pattern
- [ ] MongoDB storage (optional)
- [ ] Field permission integration
- [ ] Query builder cho custom fields
- [ ] JSON Schema generation
- [ ] Field dependency resolution
- [ ] Bulk operations
- [ ] Import/Export functionality
- [ ] Audit logging
- [ ] Metrics và monitoring
- [ ] Comprehensive tests
- [ ] Documentation
- [ ] Examples

### Integration

- [ ] Authorization + Custom Fields integration
- [ ] Field-level permission enforcement
- [ ] ABAC policies với custom field conditions
- [ ] Unified caching strategy
- [ ] End-to-end examples
- [ ] Performance benchmarks

---

## Best Practices

1. **Caching Strategy**: Implement multi-level caching (memory + Redis)
2. **Error Handling**: Comprehensive error types với context
3. **Logging**: Structured logging với correlation IDs
4. **Versioning**: Support field definition versioning
5. **Migration**: Provide migration tools cho existing data
6. **Documentation**: Inline comments, README, architecture docs
7. **Testing**: Unit, integration, và benchmark tests
8. **Security**: Always validate input, encrypt sensitive data
9. **Performance**: Profile và optimize hot paths
10. **Observability**: Metrics, tracing, và health checks

---

## Success Criteria

- Authorization system support RBAC, ABAC, và hybrid models
- Sub-millisecond permission checks với caching
- Custom fields support 20+ field types
- Field-level permission control
- Multi-tenant isolation
- Horizontal scalability
- >90% test coverage
- Complete documentation với examples
- Production-ready error handling và logging
- Metrics và monitoring integration

---

## Delivery

Khi implementation xong, cung cấp:

1. Full source code cho cả 2 packages
2. Database migration scripts
3. README.md cho mỗi package
4. ARCHITECTURE.md explaining design decisions
5. Working examples
6. Unit và integration tests
7. Benchmark results
8. API documentation
9. Migration guide (if applicable)
10. Performance tuning guide

**Target Timeline**: Implementation should be production-ready và fully tested.
