package authorization

import (
	"context"
)

// Authorizer is the main interface for the authorization system
type Authorizer interface {
	// Enforce checks if the subject has permission to perform the action on the object
	Enforce(ctx context.Context, request *AuthRequest) (*AuthResult, error)

	// EnforceBatch checks multiple permissions at once
	EnforceBatch(ctx context.Context, requests []*AuthRequest) ([]*AuthResult, error)

	// Policy Management
	AddPolicy(ctx context.Context, policy *Policy) error
	RemovePolicy(ctx context.Context, policy *Policy) error
	UpdatePolicy(ctx context.Context, oldPolicy, newPolicy *Policy) error
	GetPolicy(ctx context.Context, id string) (*Policy, error)
	GetPolicies(ctx context.Context, filter *PolicyFilter) ([]*Policy, error)

	// Role Management
	AddRole(ctx context.Context, role *Role) error
	UpdateRole(ctx context.Context, role *Role) error
	RemoveRole(ctx context.Context, roleID string) error
	GetRole(ctx context.Context, roleID string) (*Role, error)
	GetRoles(ctx context.Context, filter *RoleFilter) ([]*Role, error)
	AssignRole(ctx context.Context, userID, roleID string, domain ...string) error
	RevokeRole(ctx context.Context, userID, roleID string, domain ...string) error
	GetUserRoles(ctx context.Context, userID string, domain ...string) ([]*Role, error)
	GetRoleUsers(ctx context.Context, roleID string, domain ...string) ([]string, error)

	// Permission Management
	AddPermission(ctx context.Context, perm *Permission) error
	UpdatePermission(ctx context.Context, perm *Permission) error
	RemovePermission(ctx context.Context, permID string) error
	GetPermission(ctx context.Context, permID string) (*Permission, error)
	GetPermissions(ctx context.Context, filter *PermissionFilter) ([]*Permission, error)
	AssignPermissionToRole(ctx context.Context, roleID, permID string) error
	RevokePermissionFromRole(ctx context.Context, roleID, permID string) error
	GetRolePermissions(ctx context.Context, roleID string) ([]*Permission, error)
	GetUserPermissions(ctx context.Context, userID string, domain ...string) ([]*Permission, error)

	// Dynamic Resource Authorization (ABAC)
	CheckResourceAccess(ctx context.Context, userID, resource, action string, attributes map[string]interface{}) (*AuthResult, error)

	// Role Hierarchy & Inheritance
	AddRoleInheritance(ctx context.Context, child, parent string, domain ...string) error
	RemoveRoleInheritance(ctx context.Context, child, parent string, domain ...string) error
	GetRoleHierarchy(ctx context.Context, roleID string) ([]string, error)

	// Cache Management
	InvalidateCache(ctx context.Context, keys ...string) error
	InvalidateUserCache(ctx context.Context, userID string) error
	RefreshPolicies(ctx context.Context) error

	// Health & Stats
	HealthCheck(ctx context.Context) error
	GetStats(ctx context.Context) (*AuthStats, error)

	// Lifecycle
	Close() error
}

// PermissionChecker is a simplified interface for just checking permissions
type PermissionChecker interface {
	Enforce(ctx context.Context, request *AuthRequest) (*AuthResult, error)
	CheckResourceAccess(ctx context.Context, userID, resource, action string, attributes map[string]interface{}) (*AuthResult, error)
}

// RoleManager is an interface for role management operations
type RoleManager interface {
	AddRole(ctx context.Context, role *Role) error
	UpdateRole(ctx context.Context, role *Role) error
	RemoveRole(ctx context.Context, roleID string) error
	GetRole(ctx context.Context, roleID string) (*Role, error)
	GetRoles(ctx context.Context, filter *RoleFilter) ([]*Role, error)
	AssignRole(ctx context.Context, userID, roleID string, domain ...string) error
	RevokeRole(ctx context.Context, userID, roleID string, domain ...string) error
	GetUserRoles(ctx context.Context, userID string, domain ...string) ([]*Role, error)
}

// PolicyManager is an interface for policy management operations
type PolicyManager interface {
	AddPolicy(ctx context.Context, policy *Policy) error
	RemovePolicy(ctx context.Context, policy *Policy) error
	UpdatePolicy(ctx context.Context, oldPolicy, newPolicy *Policy) error
	GetPolicy(ctx context.Context, id string) (*Policy, error)
	GetPolicies(ctx context.Context, filter *PolicyFilter) ([]*Policy, error)
}

// CacheManager is an interface for cache management operations
type CacheManager interface {
	InvalidateCache(ctx context.Context, keys ...string) error
	InvalidateUserCache(ctx context.Context, userID string) error
	RefreshPolicies(ctx context.Context) error
}

// New creates a new Authorizer with the given configuration
func New(cfg *Config, opts ...Option) (Authorizer, error) {
	// Apply defaults
	cfg = cfg.WithDefaults()

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// Import and create casbin authorizer
	// This is done to avoid circular imports and allow lazy loading
	return newCasbinAuthorizer(cfg, opts...)
}

// newCasbinAuthorizer creates a new CasbinAuthorizer
// This function is implemented in casbin/casbin.go and registered here
var newCasbinAuthorizer func(cfg *Config, opts ...Option) (Authorizer, error)

// RegisterCasbinAuthorizer registers the CasbinAuthorizer factory
// This is called by the casbin package during init
func RegisterCasbinAuthorizer(factory func(cfg *Config, opts ...Option) (Authorizer, error)) {
	newCasbinAuthorizer = factory
}

// MustNew creates a new Authorizer and panics on error
func MustNew(cfg *Config, opts ...Option) Authorizer {
	auth, err := New(cfg, opts...)
	if err != nil {
		panic(err)
	}
	return auth
}

// Quick helper functions for common authorization checks

// CanRead checks if the user can read the resource
func CanRead(ctx context.Context, auth Authorizer, userID, resource string) (bool, error) {
	result, err := auth.Enforce(ctx, &AuthRequest{
		Subject: userID,
		Object:  resource,
		Action:  "read",
	})
	if err != nil {
		return false, err
	}
	return result.Allowed, nil
}

// CanWrite checks if the user can write to the resource
func CanWrite(ctx context.Context, auth Authorizer, userID, resource string) (bool, error) {
	result, err := auth.Enforce(ctx, &AuthRequest{
		Subject: userID,
		Object:  resource,
		Action:  "write",
	})
	if err != nil {
		return false, err
	}
	return result.Allowed, nil
}

// CanDelete checks if the user can delete the resource
func CanDelete(ctx context.Context, auth Authorizer, userID, resource string) (bool, error) {
	result, err := auth.Enforce(ctx, &AuthRequest{
		Subject: userID,
		Object:  resource,
		Action:  "delete",
	})
	if err != nil {
		return false, err
	}
	return result.Allowed, nil
}

// CanExecute checks if the user can execute the action on the resource
func CanExecute(ctx context.Context, auth Authorizer, userID, resource, action string) (bool, error) {
	result, err := auth.Enforce(ctx, &AuthRequest{
		Subject: userID,
		Object:  resource,
		Action:  action,
	})
	if err != nil {
		return false, err
	}
	return result.Allowed, nil
}

// HasRole checks if the user has the specified role
func HasRole(ctx context.Context, auth Authorizer, userID, roleID string, domain ...string) (bool, error) {
	roles, err := auth.GetUserRoles(ctx, userID, domain...)
	if err != nil {
		return false, err
	}
	for _, role := range roles {
		if role.ID == roleID {
			return true, nil
		}
	}
	return false, nil
}

// HasAnyRole checks if the user has any of the specified roles
func HasAnyRole(ctx context.Context, auth Authorizer, userID string, roleIDs []string, domain ...string) (bool, error) {
	roles, err := auth.GetUserRoles(ctx, userID, domain...)
	if err != nil {
		return false, err
	}
	roleMap := make(map[string]bool)
	for _, role := range roles {
		roleMap[role.ID] = true
	}
	for _, roleID := range roleIDs {
		if roleMap[roleID] {
			return true, nil
		}
	}
	return false, nil
}

// HasAllRoles checks if the user has all of the specified roles
func HasAllRoles(ctx context.Context, auth Authorizer, userID string, roleIDs []string, domain ...string) (bool, error) {
	roles, err := auth.GetUserRoles(ctx, userID, domain...)
	if err != nil {
		return false, err
	}
	roleMap := make(map[string]bool)
	for _, role := range roles {
		roleMap[role.ID] = true
	}
	for _, roleID := range roleIDs {
		if !roleMap[roleID] {
			return false, nil
		}
	}
	return true, nil
}
