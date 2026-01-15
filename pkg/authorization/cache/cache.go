package cache

import (
	"context"
	"time"
)

// Cache represents the interface for permission caching
type Cache interface {
	// Get retrieves a cached permission decision
	Get(ctx context.Context, key string) (*CachedDecision, error)

	// Set caches a permission decision
	Set(ctx context.Context, key string, decision *CachedDecision, ttl time.Duration) error

	// Delete removes a cached entry
	Delete(ctx context.Context, key string) error

	// DeleteByPattern removes entries matching the pattern
	DeleteByPattern(ctx context.Context, pattern string) error

	// Clear removes all cached entries
	Clear(ctx context.Context) error

	// GetStats returns cache statistics
	GetStats(ctx context.Context) (*CacheStats, error)

	// Ping checks the cache connection
	Ping(ctx context.Context) error

	// Close closes the cache connection
	Close() error
}

// CachedDecision represents a cached authorization decision
type CachedDecision struct {
	Allowed   bool      `json:"allowed"`
	Reason    string    `json:"reason,omitempty"`
	PolicyID  string    `json:"policy_id,omitempty"`
	CachedAt  time.Time `json:"cached_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// IsExpired returns true if the cached decision has expired
func (c *CachedDecision) IsExpired() bool {
	return time.Now().After(c.ExpiresAt)
}

// CacheStats represents cache statistics
type CacheStats struct {
	Hits        int64         `json:"hits"`
	Misses      int64         `json:"misses"`
	HitRate     float64       `json:"hit_rate"`
	Size        int64         `json:"size"`
	MemoryUsage int64         `json:"memory_usage"`
	AvgLatency  time.Duration `json:"avg_latency"`
}

// KeyBuilder helps build consistent cache keys
type KeyBuilder struct {
	prefix    string
	separator string
}

// NewKeyBuilder creates a new KeyBuilder
func NewKeyBuilder(prefix string) *KeyBuilder {
	return &KeyBuilder{
		prefix:    prefix,
		separator: ":",
	}
}

// EnforceKey generates a key for an enforce decision
func (kb *KeyBuilder) EnforceKey(subject, object, action, domain string) string {
	if domain == "" {
		domain = "default"
	}
	return kb.prefix + "enforce" + kb.separator +
		domain + kb.separator +
		subject + kb.separator +
		object + kb.separator +
		action
}

// UserRolesKey generates a key for user roles
func (kb *KeyBuilder) UserRolesKey(userID, domain string) string {
	if domain == "" {
		domain = "default"
	}
	return kb.prefix + "user_roles" + kb.separator + domain + kb.separator + userID
}

// RolePermissionsKey generates a key for role permissions
func (kb *KeyBuilder) RolePermissionsKey(roleID string) string {
	return kb.prefix + "role_perms" + kb.separator + roleID
}

// UserPermissionsKey generates a key for user permissions
func (kb *KeyBuilder) UserPermissionsKey(userID, domain string) string {
	if domain == "" {
		domain = "default"
	}
	return kb.prefix + "user_perms" + kb.separator + domain + kb.separator + userID
}

// UserPattern generates a pattern to match all keys for a user
func (kb *KeyBuilder) UserPattern(userID string) string {
	return kb.prefix + "*" + kb.separator + "*" + kb.separator + userID + "*"
}

// DomainPattern generates a pattern to match all keys for a domain
func (kb *KeyBuilder) DomainPattern(domain string) string {
	return kb.prefix + "*" + kb.separator + domain + kb.separator + "*"
}

// AllPattern generates a pattern to match all cache keys
func (kb *KeyBuilder) AllPattern() string {
	return kb.prefix + "*"
}
