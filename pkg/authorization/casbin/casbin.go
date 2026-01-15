package casbin

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"microservices/pkg/authorization"
	"microservices/pkg/authorization/cache"
)

//go:embed models/rbac_model.conf
var rbacModel string

//go:embed models/abac_model.conf
var abacModel string

//go:embed models/hybrid_model.conf
var hybridModel string

func init() {
	// Register the factory function
	authorization.RegisterCasbinAuthorizer(NewCasbinAuthorizer)
}

// CasbinAuthorizer implements the Authorizer interface using Casbin
type CasbinAuthorizer struct {
	enforcer *casbin.SyncedEnforcer
	adapter  *Adapter
	cache    cache.Cache
	config   *authorization.Config
	logger   *slog.Logger
	db       *gorm.DB

	// Statistics
	enforceCount      int64
	totalEnforceTime  int64
	cacheHits         int64
	cacheMisses       int64

	// Mutex for stats
	mu sync.RWMutex
}

// NewCasbinAuthorizer creates a new CasbinAuthorizer
func NewCasbinAuthorizer(cfg *authorization.Config, opts ...authorization.Option) (authorization.Authorizer, error) {
	options := authorization.ApplyOptions(opts...)

	// Create database adapter
	adapter, err := NewAdapter(cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to create adapter: %w", err)
	}

	// Migrate additional tables
	if err := adapter.MigrateAuthorizationTables(); err != nil {
		return nil, fmt.Errorf("failed to migrate tables: %w", err)
	}

	// Get the model configuration
	var modelText string
	switch cfg.ModelType {
	case authorization.ModelTypeRBAC:
		modelText = rbacModel
	case authorization.ModelTypeABAC:
		modelText = abacModel
	case authorization.ModelTypeHybrid:
		modelText = hybridModel
	default:
		modelText = rbacModel
	}

	// Load model from text
	m, err := model.NewModelFromString(modelText)
	if err != nil {
		return nil, fmt.Errorf("failed to create model: %w", err)
	}

	// Create enforcer
	enforcer, err := casbin.NewSyncedEnforcer(m)
	if err != nil {
		return nil, fmt.Errorf("failed to create enforcer: %w", err)
	}

	// Set auto-save
	enforcer.EnableAutoSave(cfg.AutoSavePolicy)

	ca := &CasbinAuthorizer{
		enforcer: enforcer,
		adapter:  adapter,
		config:   cfg,
		logger:   options.Logger,
		db:       adapter.DB(),
	}

	// Initialize cache if enabled
	if cfg.Cache != nil && cfg.Cache.Enabled {
		redisCache, err := cache.NewRedisCache(&cache.RedisCacheConfig{
			Addr:         cfg.Cache.Redis.Addr,
			Password:     cfg.Cache.Redis.Password,
			DB:           cfg.Cache.Redis.DB,
			PoolSize:     cfg.Cache.Redis.PoolSize,
			MinIdleConns: cfg.Cache.Redis.MinIdleConns,
			DialTimeout:  cfg.Cache.Redis.DialTimeout,
			ReadTimeout:  cfg.Cache.Redis.ReadTimeout,
			WriteTimeout: cfg.Cache.Redis.WriteTimeout,
			MaxRetries:   cfg.Cache.Redis.MaxRetries,
			KeyPrefix:    cfg.Cache.Redis.KeyPrefix,
			DefaultTTL:   cfg.Cache.TTL,
		})
		if err != nil {
			ca.logger.Warn("failed to initialize cache, continuing without cache", "error", err)
		} else {
			ca.cache = redisCache
		}
	}

	// Load existing policies
	if cfg.AutoLoadPolicy {
		policies, err := adapter.LoadPolicy()
		if err != nil {
			return nil, fmt.Errorf("failed to load policies: %w", err)
		}

		for _, policy := range policies {
			if len(policy) > 0 {
				ptype := policy[0]
				rule := policy[1:]
				if ptype == "p" {
					_, _ = enforcer.AddPolicy(rule)
				} else if ptype == "g" {
					_, _ = enforcer.AddGroupingPolicy(rule)
				}
			}
		}
	}

	return ca, nil
}

// Enforce checks if the subject has permission to perform the action on the object
func (ca *CasbinAuthorizer) Enforce(ctx context.Context, request *authorization.AuthRequest) (*authorization.AuthResult, error) {
	start := time.Now()
	defer func() {
		atomic.AddInt64(&ca.enforceCount, 1)
		atomic.AddInt64(&ca.totalEnforceTime, int64(time.Since(start)))
	}()

	if err := request.Validate(); err != nil {
		return nil, err
	}

	domain := request.Domain
	if domain == "" {
		domain = ca.config.DefaultDomain
	}

	// Check cache first
	if ca.cache != nil {
		cached, err := ca.cache.Get(ctx, ca.getCacheKey(request))
		if err == nil && cached != nil {
			atomic.AddInt64(&ca.cacheHits, 1)
			return &authorization.AuthResult{
				Allowed:   cached.Allowed,
				Reason:    cached.Reason,
				PolicyID:  cached.PolicyID,
				Duration:  time.Since(start),
				FromCache: true,
			}, nil
		}
		atomic.AddInt64(&ca.cacheMisses, 1)
	}

	// Perform enforcement
	var allowed bool
	var err error

	switch ca.config.ModelType {
	case authorization.ModelTypeABAC:
		allowed, err = ca.enforcer.Enforce(request.Subject, request.Object, request.Action, request.Attributes)
	case authorization.ModelTypeHybrid:
		allowed, err = ca.enforcer.Enforce(request.Subject, request.Object, request.Action, domain, request.Attributes)
	default: // RBAC
		allowed, err = ca.enforcer.Enforce(request.Subject, request.Object, request.Action, domain)
	}

	if err != nil {
		return nil, authorization.NewAuthError("enforce", request.Subject, request.Object, request.Action, err)
	}

	result := &authorization.AuthResult{
		Allowed:   allowed,
		Duration:  time.Since(start),
		FromCache: false,
	}

	// Cache the result
	if ca.cache != nil {
		_ = ca.cache.Set(ctx, ca.getCacheKey(request), &cache.CachedDecision{
			Allowed: allowed,
		}, ca.config.Cache.TTL)
	}

	return result, nil
}

// EnforceBatch checks multiple permissions at once
func (ca *CasbinAuthorizer) EnforceBatch(ctx context.Context, requests []*authorization.AuthRequest) ([]*authorization.AuthResult, error) {
	results := make([]*authorization.AuthResult, len(requests))

	for i, req := range requests {
		result, err := ca.Enforce(ctx, req)
		if err != nil {
			results[i] = &authorization.AuthResult{
				Allowed: false,
				Reason:  err.Error(),
			}
		} else {
			results[i] = result
		}
	}

	return results, nil
}

// AddPolicy adds a policy
func (ca *CasbinAuthorizer) AddPolicy(ctx context.Context, policy *authorization.Policy) error {
	if policy.ID == "" {
		policy.ID = uuid.New().String()
	}

	// Save to database
	if err := ca.db.Create(policy).Error; err != nil {
		return fmt.Errorf("%w: failed to save policy: %v", authorization.ErrDatabaseOperation, err)
	}

	// Add to enforcer
	effect := "allow"
	if policy.Effect == authorization.EffectDeny {
		effect = "deny"
	}

	domain := policy.Domain
	if domain == "" {
		domain = ca.config.DefaultDomain
	}

	_, err := ca.enforcer.AddPolicy(policy.Subject, policy.Object, policy.Action, effect, domain)
	if err != nil {
		return fmt.Errorf("%w: failed to add policy to enforcer: %v", authorization.ErrPolicyEvaluation, err)
	}

	// Invalidate cache
	if ca.cache != nil {
		_ = ca.cache.Clear(ctx)
	}

	return nil
}

// RemovePolicy removes a policy
func (ca *CasbinAuthorizer) RemovePolicy(ctx context.Context, policy *authorization.Policy) error {
	// Remove from database
	if err := ca.db.Delete(&authorization.Policy{}, "id = ?", policy.ID).Error; err != nil {
		return fmt.Errorf("%w: failed to delete policy: %v", authorization.ErrDatabaseOperation, err)
	}

	effect := "allow"
	if policy.Effect == authorization.EffectDeny {
		effect = "deny"
	}

	domain := policy.Domain
	if domain == "" {
		domain = ca.config.DefaultDomain
	}

	// Remove from enforcer
	_, err := ca.enforcer.RemovePolicy(policy.Subject, policy.Object, policy.Action, effect, domain)
	if err != nil {
		return fmt.Errorf("%w: failed to remove policy from enforcer: %v", authorization.ErrPolicyEvaluation, err)
	}

	// Invalidate cache
	if ca.cache != nil {
		_ = ca.cache.Clear(ctx)
	}

	return nil
}

// UpdatePolicy updates a policy
func (ca *CasbinAuthorizer) UpdatePolicy(ctx context.Context, oldPolicy, newPolicy *authorization.Policy) error {
	if err := ca.RemovePolicy(ctx, oldPolicy); err != nil {
		return err
	}
	return ca.AddPolicy(ctx, newPolicy)
}

// GetPolicy retrieves a policy by ID
func (ca *CasbinAuthorizer) GetPolicy(ctx context.Context, id string) (*authorization.Policy, error) {
	var policy authorization.Policy
	if err := ca.db.First(&policy, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, authorization.ErrNotFound
		}
		return nil, fmt.Errorf("%w: failed to get policy: %v", authorization.ErrDatabaseOperation, err)
	}
	return &policy, nil
}

// GetPolicies retrieves policies matching the filter
func (ca *CasbinAuthorizer) GetPolicies(ctx context.Context, filter *authorization.PolicyFilter) ([]*authorization.Policy, error) {
	query := ca.db.Model(&authorization.Policy{})

	if filter != nil {
		if filter.Type != nil {
			query = query.Where("type = ?", *filter.Type)
		}
		if filter.Subject != nil {
			query = query.Where("subject = ?", *filter.Subject)
		}
		if filter.Object != nil {
			query = query.Where("object = ?", *filter.Object)
		}
		if filter.Action != nil {
			query = query.Where("action = ?", *filter.Action)
		}
		if filter.Effect != nil {
			query = query.Where("effect = ?", *filter.Effect)
		}
		if filter.Domain != nil {
			query = query.Where("domain = ?", *filter.Domain)
		}
		if filter.TenantID != nil {
			query = query.Where("tenant_id = ?", *filter.TenantID)
		}
		if filter.IsActive != nil {
			query = query.Where("is_active = ?", *filter.IsActive)
		}
		if filter.Limit > 0 {
			query = query.Limit(filter.Limit)
		}
		if filter.Offset > 0 {
			query = query.Offset(filter.Offset)
		}
	}

	var policies []*authorization.Policy
	if err := query.Find(&policies).Error; err != nil {
		return nil, fmt.Errorf("%w: failed to get policies: %v", authorization.ErrDatabaseOperation, err)
	}

	return policies, nil
}

// AddRole adds a role
func (ca *CasbinAuthorizer) AddRole(ctx context.Context, role *authorization.Role) error {
	if role.ID == "" {
		role.ID = uuid.New().String()
	}

	// Marshal metadata
	if role.Metadata != nil {
		data, err := json.Marshal(role.Metadata)
		if err != nil {
			return fmt.Errorf("%w: failed to marshal metadata: %v", authorization.ErrInvalidInput, err)
		}
		role.MetadataJSON = string(data)
	}

	if err := ca.db.Create(role).Error; err != nil {
		return fmt.Errorf("%w: failed to create role: %v", authorization.ErrDatabaseOperation, err)
	}

	return nil
}

// UpdateRole updates a role
func (ca *CasbinAuthorizer) UpdateRole(ctx context.Context, role *authorization.Role) error {
	// Marshal metadata
	if role.Metadata != nil {
		data, err := json.Marshal(role.Metadata)
		if err != nil {
			return fmt.Errorf("%w: failed to marshal metadata: %v", authorization.ErrInvalidInput, err)
		}
		role.MetadataJSON = string(data)
	}

	if err := ca.db.Save(role).Error; err != nil {
		return fmt.Errorf("%w: failed to update role: %v", authorization.ErrDatabaseOperation, err)
	}

	// Invalidate cache
	if ca.cache != nil {
		_ = ca.cache.Clear(ctx)
	}

	return nil
}

// RemoveRole removes a role
func (ca *CasbinAuthorizer) RemoveRole(ctx context.Context, roleID string) error {
	// First, remove all user-role assignments
	if err := ca.db.Delete(&authorization.UserRole{}, "role_id = ?", roleID).Error; err != nil {
		return fmt.Errorf("%w: failed to delete user role assignments: %v", authorization.ErrDatabaseOperation, err)
	}

	// Remove the role
	if err := ca.db.Delete(&authorization.Role{}, "id = ?", roleID).Error; err != nil {
		return fmt.Errorf("%w: failed to delete role: %v", authorization.ErrDatabaseOperation, err)
	}

	// Remove from enforcer
	ca.enforcer.DeleteRole(roleID)

	// Invalidate cache
	if ca.cache != nil {
		_ = ca.cache.Clear(ctx)
	}

	return nil
}

// GetRole retrieves a role by ID
func (ca *CasbinAuthorizer) GetRole(ctx context.Context, roleID string) (*authorization.Role, error) {
	var role authorization.Role
	if err := ca.db.Preload("Permissions").First(&role, "id = ?", roleID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, authorization.ErrRoleNotFound
		}
		return nil, fmt.Errorf("%w: failed to get role: %v", authorization.ErrDatabaseOperation, err)
	}

	// Unmarshal metadata
	if role.MetadataJSON != "" {
		if err := json.Unmarshal([]byte(role.MetadataJSON), &role.Metadata); err != nil {
			ca.logger.Warn("failed to unmarshal role metadata", "error", err)
		}
	}

	return &role, nil
}

// GetRoles retrieves roles matching the filter
func (ca *CasbinAuthorizer) GetRoles(ctx context.Context, filter *authorization.RoleFilter) ([]*authorization.Role, error) {
	query := ca.db.Model(&authorization.Role{})

	if filter != nil {
		if filter.Name != nil {
			query = query.Where("name LIKE ?", "%"+*filter.Name+"%")
		}
		if filter.ParentID != nil {
			query = query.Where("parent_id = ?", *filter.ParentID)
		}
		if filter.Domain != nil {
			query = query.Where("domain = ?", *filter.Domain)
		}
		if filter.TenantID != nil {
			query = query.Where("tenant_id = ?", *filter.TenantID)
		}
		if filter.IsActive != nil {
			query = query.Where("is_active = ?", *filter.IsActive)
		}
		if filter.IsSystem != nil {
			query = query.Where("is_system = ?", *filter.IsSystem)
		}
		if filter.Limit > 0 {
			query = query.Limit(filter.Limit)
		}
		if filter.Offset > 0 {
			query = query.Offset(filter.Offset)
		}
	}

	var roles []*authorization.Role
	if err := query.Preload("Permissions").Find(&roles).Error; err != nil {
		return nil, fmt.Errorf("%w: failed to get roles: %v", authorization.ErrDatabaseOperation, err)
	}

	return roles, nil
}

// AssignRole assigns a role to a user
func (ca *CasbinAuthorizer) AssignRole(ctx context.Context, userID, roleID string, domain ...string) error {
	dom := ca.config.DefaultDomain
	if len(domain) > 0 {
		dom = domain[0]
	}

	// Check if role exists
	var count int64
	if err := ca.db.Model(&authorization.Role{}).Where("id = ?", roleID).Count(&count).Error; err != nil {
		return fmt.Errorf("%w: failed to check role: %v", authorization.ErrDatabaseOperation, err)
	}
	if count == 0 {
		return authorization.ErrRoleNotFound
	}

	// Create user-role assignment
	userRole := &authorization.UserRole{
		ID:       uuid.New().String(),
		UserID:   userID,
		RoleID:   roleID,
		Domain:   dom,
		TenantID: ca.config.DefaultTenantID,
	}

	if err := ca.db.Create(userRole).Error; err != nil {
		return fmt.Errorf("%w: failed to assign role: %v", authorization.ErrDatabaseOperation, err)
	}

	// Add to enforcer
	_, err := ca.enforcer.AddGroupingPolicy(userID, roleID, dom)
	if err != nil {
		return fmt.Errorf("%w: failed to add grouping policy: %v", authorization.ErrPolicyEvaluation, err)
	}

	// Invalidate user cache
	if ca.cache != nil {
		_ = ca.cache.DeleteByPattern(ctx, fmt.Sprintf("*%s*", userID))
	}

	return nil
}

// RevokeRole revokes a role from a user
func (ca *CasbinAuthorizer) RevokeRole(ctx context.Context, userID, roleID string, domain ...string) error {
	dom := ca.config.DefaultDomain
	if len(domain) > 0 {
		dom = domain[0]
	}

	// Remove from database
	if err := ca.db.Delete(&authorization.UserRole{}, "user_id = ? AND role_id = ? AND domain = ?", userID, roleID, dom).Error; err != nil {
		return fmt.Errorf("%w: failed to revoke role: %v", authorization.ErrDatabaseOperation, err)
	}

	// Remove from enforcer
	_, err := ca.enforcer.RemoveGroupingPolicy(userID, roleID, dom)
	if err != nil {
		return fmt.Errorf("%w: failed to remove grouping policy: %v", authorization.ErrPolicyEvaluation, err)
	}

	// Invalidate user cache
	if ca.cache != nil {
		_ = ca.cache.DeleteByPattern(ctx, fmt.Sprintf("*%s*", userID))
	}

	return nil
}

// GetUserRoles gets all roles for a user
func (ca *CasbinAuthorizer) GetUserRoles(ctx context.Context, userID string, domain ...string) ([]*authorization.Role, error) {
	dom := ca.config.DefaultDomain
	if len(domain) > 0 {
		dom = domain[0]
	}

	// Get role IDs from user_roles table
	var userRoles []authorization.UserRole
	query := ca.db.Where("user_id = ?", userID)
	if dom != "" {
		query = query.Where("domain = ?", dom)
	}
	if err := query.Find(&userRoles).Error; err != nil {
		return nil, fmt.Errorf("%w: failed to get user roles: %v", authorization.ErrDatabaseOperation, err)
	}

	if len(userRoles) == 0 {
		return []*authorization.Role{}, nil
	}

	roleIDs := make([]string, len(userRoles))
	for i, ur := range userRoles {
		roleIDs[i] = ur.RoleID
	}

	// Get role details
	var roles []*authorization.Role
	if err := ca.db.Preload("Permissions").Where("id IN ?", roleIDs).Find(&roles).Error; err != nil {
		return nil, fmt.Errorf("%w: failed to get roles: %v", authorization.ErrDatabaseOperation, err)
	}

	return roles, nil
}

// GetRoleUsers gets all users with a specific role
func (ca *CasbinAuthorizer) GetRoleUsers(ctx context.Context, roleID string, domain ...string) ([]string, error) {
	dom := ca.config.DefaultDomain
	if len(domain) > 0 {
		dom = domain[0]
	}

	var userRoles []authorization.UserRole
	query := ca.db.Where("role_id = ?", roleID)
	if dom != "" {
		query = query.Where("domain = ?", dom)
	}
	if err := query.Find(&userRoles).Error; err != nil {
		return nil, fmt.Errorf("%w: failed to get role users: %v", authorization.ErrDatabaseOperation, err)
	}

	userIDs := make([]string, len(userRoles))
	for i, ur := range userRoles {
		userIDs[i] = ur.UserID
	}

	return userIDs, nil
}

// AddPermission adds a permission
func (ca *CasbinAuthorizer) AddPermission(ctx context.Context, perm *authorization.Permission) error {
	if perm.ID == "" {
		perm.ID = uuid.New().String()
	}

	if err := ca.db.Create(perm).Error; err != nil {
		return fmt.Errorf("%w: failed to create permission: %v", authorization.ErrDatabaseOperation, err)
	}

	return nil
}

// UpdatePermission updates a permission
func (ca *CasbinAuthorizer) UpdatePermission(ctx context.Context, perm *authorization.Permission) error {
	if err := ca.db.Save(perm).Error; err != nil {
		return fmt.Errorf("%w: failed to update permission: %v", authorization.ErrDatabaseOperation, err)
	}

	// Invalidate cache
	if ca.cache != nil {
		_ = ca.cache.Clear(ctx)
	}

	return nil
}

// RemovePermission removes a permission
func (ca *CasbinAuthorizer) RemovePermission(ctx context.Context, permID string) error {
	if err := ca.db.Delete(&authorization.Permission{}, "id = ?", permID).Error; err != nil {
		return fmt.Errorf("%w: failed to delete permission: %v", authorization.ErrDatabaseOperation, err)
	}

	// Invalidate cache
	if ca.cache != nil {
		_ = ca.cache.Clear(ctx)
	}

	return nil
}

// GetPermission retrieves a permission by ID
func (ca *CasbinAuthorizer) GetPermission(ctx context.Context, permID string) (*authorization.Permission, error) {
	var perm authorization.Permission
	if err := ca.db.First(&perm, "id = ?", permID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, authorization.ErrPermissionNotFound
		}
		return nil, fmt.Errorf("%w: failed to get permission: %v", authorization.ErrDatabaseOperation, err)
	}
	return &perm, nil
}

// GetPermissions retrieves permissions matching the filter
func (ca *CasbinAuthorizer) GetPermissions(ctx context.Context, filter *authorization.PermissionFilter) ([]*authorization.Permission, error) {
	query := ca.db.Model(&authorization.Permission{})

	if filter != nil {
		if filter.Resource != nil {
			query = query.Where("resource = ?", *filter.Resource)
		}
		if filter.Action != nil {
			query = query.Where("action = ?", *filter.Action)
		}
		if filter.Scope != nil {
			query = query.Where("scope = ?", *filter.Scope)
		}
		if filter.Domain != nil {
			query = query.Where("domain = ?", *filter.Domain)
		}
		if filter.TenantID != nil {
			query = query.Where("tenant_id = ?", *filter.TenantID)
		}
		if filter.IsActive != nil {
			query = query.Where("is_active = ?", *filter.IsActive)
		}
		if filter.Limit > 0 {
			query = query.Limit(filter.Limit)
		}
		if filter.Offset > 0 {
			query = query.Offset(filter.Offset)
		}
	}

	var perms []*authorization.Permission
	if err := query.Find(&perms).Error; err != nil {
		return nil, fmt.Errorf("%w: failed to get permissions: %v", authorization.ErrDatabaseOperation, err)
	}

	return perms, nil
}

// AssignPermissionToRole assigns a permission to a role
func (ca *CasbinAuthorizer) AssignPermissionToRole(ctx context.Context, roleID, permID string) error {
	// Get permission
	perm, err := ca.GetPermission(ctx, permID)
	if err != nil {
		return err
	}

	// Get role
	role, err := ca.GetRole(ctx, roleID)
	if err != nil {
		return err
	}

	// Add permission to role
	if err := ca.db.Model(role).Association("Permissions").Append(perm); err != nil {
		return fmt.Errorf("%w: failed to assign permission: %v", authorization.ErrDatabaseOperation, err)
	}

	// Add policy to enforcer
	domain := role.Domain
	if domain == "" {
		domain = ca.config.DefaultDomain
	}
	_, err = ca.enforcer.AddPolicy(roleID, perm.Resource, perm.Action, "allow", domain)
	if err != nil {
		return fmt.Errorf("%w: failed to add policy: %v", authorization.ErrPolicyEvaluation, err)
	}

	// Invalidate cache
	if ca.cache != nil {
		_ = ca.cache.Clear(ctx)
	}

	return nil
}

// RevokePermissionFromRole revokes a permission from a role
func (ca *CasbinAuthorizer) RevokePermissionFromRole(ctx context.Context, roleID, permID string) error {
	// Get permission
	perm, err := ca.GetPermission(ctx, permID)
	if err != nil {
		return err
	}

	// Get role
	role, err := ca.GetRole(ctx, roleID)
	if err != nil {
		return err
	}

	// Remove permission from role
	if err := ca.db.Model(role).Association("Permissions").Delete(perm); err != nil {
		return fmt.Errorf("%w: failed to revoke permission: %v", authorization.ErrDatabaseOperation, err)
	}

	// Remove policy from enforcer
	domain := role.Domain
	if domain == "" {
		domain = ca.config.DefaultDomain
	}
	_, err = ca.enforcer.RemovePolicy(roleID, perm.Resource, perm.Action, "allow", domain)
	if err != nil {
		return fmt.Errorf("%w: failed to remove policy: %v", authorization.ErrPolicyEvaluation, err)
	}

	// Invalidate cache
	if ca.cache != nil {
		_ = ca.cache.Clear(ctx)
	}

	return nil
}

// GetRolePermissions gets all permissions for a role
func (ca *CasbinAuthorizer) GetRolePermissions(ctx context.Context, roleID string) ([]*authorization.Permission, error) {
	role, err := ca.GetRole(ctx, roleID)
	if err != nil {
		return nil, err
	}
	return role.Permissions, nil
}

// GetUserPermissions gets all permissions for a user
func (ca *CasbinAuthorizer) GetUserPermissions(ctx context.Context, userID string, domain ...string) ([]*authorization.Permission, error) {
	roles, err := ca.GetUserRoles(ctx, userID, domain...)
	if err != nil {
		return nil, err
	}

	permMap := make(map[string]*authorization.Permission)
	for _, role := range roles {
		for _, perm := range role.Permissions {
			permMap[perm.ID] = perm
		}
	}

	perms := make([]*authorization.Permission, 0, len(permMap))
	for _, perm := range permMap {
		perms = append(perms, perm)
	}

	return perms, nil
}

// CheckResourceAccess checks if a user can access a resource with attributes (ABAC)
func (ca *CasbinAuthorizer) CheckResourceAccess(ctx context.Context, userID, resource, action string, attributes map[string]interface{}) (*authorization.AuthResult, error) {
	return ca.Enforce(ctx, &authorization.AuthRequest{
		Subject:    userID,
		Object:     resource,
		Action:     action,
		Attributes: attributes,
	})
}

// AddRoleInheritance adds role inheritance
func (ca *CasbinAuthorizer) AddRoleInheritance(ctx context.Context, child, parent string, domain ...string) error {
	dom := ca.config.DefaultDomain
	if len(domain) > 0 {
		dom = domain[0]
	}

	// Update child role's parent
	if err := ca.db.Model(&authorization.Role{}).Where("id = ?", child).Update("parent_id", parent).Error; err != nil {
		return fmt.Errorf("%w: failed to update role inheritance: %v", authorization.ErrDatabaseOperation, err)
	}

	// Add to enforcer
	_, err := ca.enforcer.AddGroupingPolicy(child, parent, dom)
	if err != nil {
		return fmt.Errorf("%w: failed to add role inheritance: %v", authorization.ErrPolicyEvaluation, err)
	}

	// Invalidate cache
	if ca.cache != nil {
		_ = ca.cache.Clear(ctx)
	}

	return nil
}

// RemoveRoleInheritance removes role inheritance
func (ca *CasbinAuthorizer) RemoveRoleInheritance(ctx context.Context, child, parent string, domain ...string) error {
	dom := ca.config.DefaultDomain
	if len(domain) > 0 {
		dom = domain[0]
	}

	// Update child role's parent
	if err := ca.db.Model(&authorization.Role{}).Where("id = ?", child).Update("parent_id", nil).Error; err != nil {
		return fmt.Errorf("%w: failed to remove role inheritance: %v", authorization.ErrDatabaseOperation, err)
	}

	// Remove from enforcer
	_, err := ca.enforcer.RemoveGroupingPolicy(child, parent, dom)
	if err != nil {
		return fmt.Errorf("%w: failed to remove role inheritance: %v", authorization.ErrPolicyEvaluation, err)
	}

	// Invalidate cache
	if ca.cache != nil {
		_ = ca.cache.Clear(ctx)
	}

	return nil
}

// GetRoleHierarchy gets the hierarchy of a role (all parent roles)
func (ca *CasbinAuthorizer) GetRoleHierarchy(ctx context.Context, roleID string) ([]string, error) {
	var hierarchy []string
	currentID := roleID

	for {
		var role authorization.Role
		if err := ca.db.First(&role, "id = ?", currentID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				break
			}
			return nil, fmt.Errorf("%w: failed to get role: %v", authorization.ErrDatabaseOperation, err)
		}

		if role.ParentID == nil {
			break
		}

		hierarchy = append(hierarchy, *role.ParentID)
		currentID = *role.ParentID
	}

	return hierarchy, nil
}

// InvalidateCache invalidates cache entries
func (ca *CasbinAuthorizer) InvalidateCache(ctx context.Context, keys ...string) error {
	if ca.cache == nil {
		return nil
	}

	for _, key := range keys {
		if err := ca.cache.Delete(ctx, key); err != nil {
			return err
		}
	}
	return nil
}

// InvalidateUserCache invalidates all cache entries for a user
func (ca *CasbinAuthorizer) InvalidateUserCache(ctx context.Context, userID string) error {
	if ca.cache == nil {
		return nil
	}
	return ca.cache.DeleteByPattern(ctx, fmt.Sprintf("*%s*", userID))
}

// RefreshPolicies reloads policies from the database
func (ca *CasbinAuthorizer) RefreshPolicies(ctx context.Context) error {
	policies, err := ca.adapter.LoadPolicy()
	if err != nil {
		return fmt.Errorf("failed to load policies: %w", err)
	}

	// Clear existing policies
	ca.enforcer.ClearPolicy()

	// Add policies
	for _, policy := range policies {
		if len(policy) > 0 {
			ptype := policy[0]
			rule := policy[1:]
			if ptype == "p" {
				_, _ = ca.enforcer.AddPolicy(rule)
			} else if ptype == "g" {
				_, _ = ca.enforcer.AddGroupingPolicy(rule)
			}
		}
	}

	// Clear cache
	if ca.cache != nil {
		_ = ca.cache.Clear(ctx)
	}

	return nil
}

// HealthCheck checks the health of the authorization system
func (ca *CasbinAuthorizer) HealthCheck(ctx context.Context) error {
	// Check database
	if err := ca.adapter.Ping(ctx); err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}

	// Check cache
	if ca.cache != nil {
		if err := ca.cache.Ping(ctx); err != nil {
			return fmt.Errorf("cache health check failed: %w", err)
		}
	}

	return nil
}

// GetStats returns authorization statistics
func (ca *CasbinAuthorizer) GetStats(ctx context.Context) (*authorization.AuthStats, error) {
	var stats authorization.AuthStats

	// Count policies
	ca.db.Model(&authorization.Policy{}).Count(&stats.TotalPolicies)
	ca.db.Model(&authorization.Role{}).Count(&stats.TotalRoles)
	ca.db.Model(&authorization.Permission{}).Count(&stats.TotalPermissions)
	ca.db.Model(&authorization.UserRole{}).Count(&stats.TotalUserRoles)

	// Cache stats
	stats.CacheHits = atomic.LoadInt64(&ca.cacheHits)
	stats.CacheMisses = atomic.LoadInt64(&ca.cacheMisses)

	total := stats.CacheHits + stats.CacheMisses
	if total > 0 {
		stats.CacheHitRate = float64(stats.CacheHits) / float64(total)
	}

	// Enforce stats
	stats.EnforceCount = atomic.LoadInt64(&ca.enforceCount)
	totalTime := atomic.LoadInt64(&ca.totalEnforceTime)
	if stats.EnforceCount > 0 {
		stats.AvgEnforceLatency = time.Duration(totalTime / stats.EnforceCount)
	}

	stats.LastUpdated = time.Now()

	return &stats, nil
}

// Close closes the authorization system
func (ca *CasbinAuthorizer) Close() error {
	if ca.cache != nil {
		ca.cache.Close()
	}
	return ca.adapter.Close()
}

// getCacheKey generates a cache key for an auth request
func (ca *CasbinAuthorizer) getCacheKey(req *authorization.AuthRequest) string {
	domain := req.Domain
	if domain == "" {
		domain = ca.config.DefaultDomain
	}
	return fmt.Sprintf("authz:enforce:%s:%s:%s:%s", domain, req.Subject, req.Object, req.Action)
}
