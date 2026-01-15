package authorization

import (
	"errors"
	"fmt"
	"time"
)

// PolicyType represents the type of authorization policy
type PolicyType string

const (
	PolicyTypeRBAC      PolicyType = "rbac"
	PolicyTypeABAC      PolicyType = "abac"
	PolicyTypeRuleBased PolicyType = "rule_based"
)

// Effect represents the effect of a policy (allow or deny)
type Effect string

const (
	EffectAllow Effect = "allow"
	EffectDeny  Effect = "deny"
)

// Scope represents the scope of a permission
type Scope string

const (
	ScopeGlobal   Scope = "global"
	ScopeTenant   Scope = "tenant"
	ScopeResource Scope = "resource"
	ScopeOwner    Scope = "owner"
)

// ConditionOperator represents comparison operators for ABAC conditions
type ConditionOperator string

const (
	OpEquals      ConditionOperator = "eq"
	OpNotEquals   ConditionOperator = "ne"
	OpIn          ConditionOperator = "in"
	OpNotIn       ConditionOperator = "nin"
	OpGreaterThan ConditionOperator = "gt"
	OpLessThan    ConditionOperator = "lt"
	OpContains    ConditionOperator = "contains"
	OpRegex       ConditionOperator = "regex"
)

// ModelType represents the Casbin model type
type ModelType string

const (
	ModelTypeRBAC   ModelType = "rbac"
	ModelTypeABAC   ModelType = "abac"
	ModelTypeHybrid ModelType = "hybrid"
)

// Standard errors
var (
	ErrNotFound           = errors.New("authorization: not found")
	ErrAlreadyExists      = errors.New("authorization: already exists")
	ErrInvalidInput       = errors.New("authorization: invalid input")
	ErrUnauthorized       = errors.New("authorization: unauthorized")
	ErrForbidden          = errors.New("authorization: forbidden")
	ErrPolicyEvaluation   = errors.New("authorization: policy evaluation failed")
	ErrCacheOperation     = errors.New("authorization: cache operation failed")
	ErrDatabaseOperation  = errors.New("authorization: database operation failed")
	ErrEnforcerNotReady   = errors.New("authorization: enforcer not ready")
	ErrInvalidConfig      = errors.New("authorization: invalid configuration")
	ErrConnectionFailed   = errors.New("authorization: connection failed")
	ErrInvalidModelType   = errors.New("authorization: invalid model type")
	ErrInvalidPolicyType  = errors.New("authorization: invalid policy type")
	ErrRoleNotFound       = errors.New("authorization: role not found")
	ErrPermissionNotFound = errors.New("authorization: permission not found")
	ErrUserNotFound       = errors.New("authorization: user not found")
)

// AuthError represents a detailed authorization error
type AuthError struct {
	Op      string // Operation that failed
	Subject string // Subject involved
	Object  string // Object/Resource involved
	Action  string // Action attempted
	Err     error  // Underlying error
}

func (e *AuthError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("authorization: %s failed for subject=%s, object=%s, action=%s: %v",
			e.Op, e.Subject, e.Object, e.Action, e.Err)
	}
	return fmt.Sprintf("authorization: %s failed for subject=%s, object=%s, action=%s",
		e.Op, e.Subject, e.Object, e.Action)
}

func (e *AuthError) Unwrap() error {
	return e.Err
}

// NewAuthError creates a new AuthError
func NewAuthError(op, subject, object, action string, err error) *AuthError {
	return &AuthError{
		Op:      op,
		Subject: subject,
		Object:  object,
		Action:  action,
		Err:     err,
	}
}

// Error helper functions
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound) || errors.Is(err, ErrRoleNotFound) ||
		errors.Is(err, ErrPermissionNotFound) || errors.Is(err, ErrUserNotFound)
}

func IsUnauthorized(err error) bool {
	return errors.Is(err, ErrUnauthorized)
}

func IsForbidden(err error) bool {
	return errors.Is(err, ErrForbidden)
}

func IsCacheError(err error) bool {
	return errors.Is(err, ErrCacheOperation)
}

func IsDatabaseError(err error) bool {
	return errors.Is(err, ErrDatabaseOperation)
}

// AuthRequest represents an authorization request
type AuthRequest struct {
	Subject    string                 `json:"subject"`     // User/Service ID
	Object     string                 `json:"object"`      // Resource
	Action     string                 `json:"action"`      // Operation (read, write, delete, etc.)
	Domain     string                 `json:"domain"`      // Domain/Tenant for multi-tenancy
	TenantID   string                 `json:"tenant_id"`   // Tenant ID for multi-tenancy
	Attributes map[string]interface{} `json:"attributes"`  // Additional context for ABAC
}

// Validate validates the AuthRequest
func (r *AuthRequest) Validate() error {
	if r.Subject == "" {
		return fmt.Errorf("%w: subject is required", ErrInvalidInput)
	}
	if r.Object == "" {
		return fmt.Errorf("%w: object is required", ErrInvalidInput)
	}
	if r.Action == "" {
		return fmt.Errorf("%w: action is required", ErrInvalidInput)
	}
	return nil
}

// AuthResult represents the result of an authorization check
type AuthResult struct {
	Allowed   bool          `json:"allowed"`
	Reason    string        `json:"reason,omitempty"`
	PolicyID  string        `json:"policy_id,omitempty"`
	Duration  time.Duration `json:"duration"`
	FromCache bool          `json:"from_cache"`
}

// Policy represents an authorization policy
type Policy struct {
	ID          string       `json:"id" gorm:"primaryKey;size:100"`
	Type        PolicyType   `json:"type" gorm:"size:50;not null"`
	Subject     string       `json:"subject" gorm:"size:200;not null"`
	Object      string       `json:"object" gorm:"size:200;not null"`
	Action      string       `json:"action" gorm:"size:100;not null"`
	Effect      Effect       `json:"effect" gorm:"size:20;not null;default:allow"`
	Conditions  []*Condition `json:"conditions" gorm:"-"`
	ConditionJSON string     `json:"-" gorm:"column:conditions;type:jsonb"`
	Priority    int          `json:"priority" gorm:"default:0"`
	Domain      string       `json:"domain" gorm:"size:100;index"`
	TenantID    string       `json:"tenant_id" gorm:"size:100;index"`
	Description string       `json:"description" gorm:"size:500"`
	IsActive    bool         `json:"is_active" gorm:"default:true"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// TableName returns the table name for Policy
func (Policy) TableName() string {
	return "authorization_policies"
}

// PolicyFilter represents filters for querying policies
type PolicyFilter struct {
	Type     *PolicyType `json:"type"`
	Subject  *string     `json:"subject"`
	Object   *string     `json:"object"`
	Action   *string     `json:"action"`
	Effect   *Effect     `json:"effect"`
	Domain   *string     `json:"domain"`
	TenantID *string     `json:"tenant_id"`
	IsActive *bool       `json:"is_active"`
	Limit    int         `json:"limit"`
	Offset   int         `json:"offset"`
}

// Role represents a role with hierarchical support
type Role struct {
	ID          string                 `json:"id" gorm:"primaryKey;size:100"`
	Name        string                 `json:"name" gorm:"size:200;not null"`
	Description string                 `json:"description" gorm:"size:500"`
	ParentID    *string                `json:"parent_id" gorm:"size:100;index"`
	Permissions []*Permission          `json:"permissions" gorm:"many2many:role_permissions;"`
	Metadata    map[string]interface{} `json:"metadata" gorm:"-"`
	MetadataJSON string                `json:"-" gorm:"column:metadata;type:jsonb"`
	Domain      string                 `json:"domain" gorm:"size:100;index"`
	TenantID    string                 `json:"tenant_id" gorm:"size:100;index"`
	IsActive    bool                   `json:"is_active" gorm:"default:true"`
	IsSystem    bool                   `json:"is_system" gorm:"default:false"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// TableName returns the table name for Role
func (Role) TableName() string {
	return "authorization_roles"
}

// RoleFilter represents filters for querying roles
type RoleFilter struct {
	Name     *string `json:"name"`
	ParentID *string `json:"parent_id"`
	Domain   *string `json:"domain"`
	TenantID *string `json:"tenant_id"`
	IsActive *bool   `json:"is_active"`
	IsSystem *bool   `json:"is_system"`
	Limit    int     `json:"limit"`
	Offset   int     `json:"offset"`
}

// Permission represents a permission with resource-level granularity
type Permission struct {
	ID          string       `json:"id" gorm:"primaryKey;size:100"`
	Resource    string       `json:"resource" gorm:"size:200;not null"`
	Action      string       `json:"action" gorm:"size:100;not null"`
	Scope       Scope        `json:"scope" gorm:"size:50;default:global"`
	Conditions  []*Condition `json:"conditions" gorm:"-"`
	ConditionJSON string     `json:"-" gorm:"column:conditions;type:jsonb"`
	Description string       `json:"description" gorm:"size:500"`
	Domain      string       `json:"domain" gorm:"size:100;index"`
	TenantID    string       `json:"tenant_id" gorm:"size:100;index"`
	IsActive    bool         `json:"is_active" gorm:"default:true"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// TableName returns the table name for Permission
func (Permission) TableName() string {
	return "authorization_permissions"
}

// PermissionFilter represents filters for querying permissions
type PermissionFilter struct {
	Resource *string `json:"resource"`
	Action   *string `json:"action"`
	Scope    *Scope  `json:"scope"`
	Domain   *string `json:"domain"`
	TenantID *string `json:"tenant_id"`
	IsActive *bool   `json:"is_active"`
	Limit    int     `json:"limit"`
	Offset   int     `json:"offset"`
}

// UserRole represents the relationship between user and role
type UserRole struct {
	ID        string    `json:"id" gorm:"primaryKey;size:100"`
	UserID    string    `json:"user_id" gorm:"size:100;not null;index"`
	RoleID    string    `json:"role_id" gorm:"size:100;not null;index"`
	Domain    string    `json:"domain" gorm:"size:100;index"`
	TenantID  string    `json:"tenant_id" gorm:"size:100;index"`
	ExpiresAt *time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName returns the table name for UserRole
func (UserRole) TableName() string {
	return "authorization_user_roles"
}

// Condition represents a condition for ABAC policies
type Condition struct {
	Field    string            `json:"field"`
	Operator ConditionOperator `json:"operator"`
	Value    interface{}       `json:"value"`
}

// Evaluate evaluates the condition against the given attributes
func (c *Condition) Evaluate(attrs map[string]interface{}) bool {
	value, ok := attrs[c.Field]
	if !ok {
		return false
	}

	switch c.Operator {
	case OpEquals:
		return fmt.Sprintf("%v", value) == fmt.Sprintf("%v", c.Value)
	case OpNotEquals:
		return fmt.Sprintf("%v", value) != fmt.Sprintf("%v", c.Value)
	case OpIn:
		if arr, ok := c.Value.([]interface{}); ok {
			for _, v := range arr {
				if fmt.Sprintf("%v", value) == fmt.Sprintf("%v", v) {
					return true
				}
			}
		}
		return false
	case OpNotIn:
		if arr, ok := c.Value.([]interface{}); ok {
			for _, v := range arr {
				if fmt.Sprintf("%v", value) == fmt.Sprintf("%v", v) {
					return false
				}
			}
		}
		return true
	case OpGreaterThan:
		return compareNumeric(value, c.Value) > 0
	case OpLessThan:
		return compareNumeric(value, c.Value) < 0
	case OpContains:
		return containsString(fmt.Sprintf("%v", value), fmt.Sprintf("%v", c.Value))
	default:
		return false
	}
}

// AuthStats represents authorization statistics
type AuthStats struct {
	TotalPolicies     int64         `json:"total_policies"`
	TotalRoles        int64         `json:"total_roles"`
	TotalPermissions  int64         `json:"total_permissions"`
	TotalUserRoles    int64         `json:"total_user_roles"`
	CacheHits         int64         `json:"cache_hits"`
	CacheMisses       int64         `json:"cache_misses"`
	CacheHitRate      float64       `json:"cache_hit_rate"`
	AvgEnforceLatency time.Duration `json:"avg_enforce_latency"`
	EnforceCount      int64         `json:"enforce_count"`
	LastUpdated       time.Time     `json:"last_updated"`
}

// Helper functions
func compareNumeric(a, b interface{}) int {
	aFloat, aOk := toFloat64(a)
	bFloat, bOk := toFloat64(b)
	if !aOk || !bOk {
		return 0
	}
	if aFloat > bFloat {
		return 1
	}
	if aFloat < bFloat {
		return -1
	}
	return 0
}

func toFloat64(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case float32:
		return float64(n), true
	case float64:
		return n, true
	default:
		return 0, false
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
