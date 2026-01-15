package casbin

import (
	"context"
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"microservices/pkg/authorization"
)

// CasbinRule represents a policy rule stored in the database
type CasbinRule struct {
	ID        uint      `gorm:"primaryKey;autoIncrement"`
	Ptype     string    `gorm:"size:100;not null;index:idx_ptype"`
	V0        string    `gorm:"size:200;index:idx_v0_v1_v2"`
	V1        string    `gorm:"size:200;index:idx_v0_v1_v2"`
	V2        string    `gorm:"size:200;index:idx_v0_v1_v2"`
	V3        string    `gorm:"size:200"`
	V4        string    `gorm:"size:200"`
	V5        string    `gorm:"size:200"`
	V6        string    `gorm:"size:200"`
	V7        string    `gorm:"size:200"`
	TenantID  string    `gorm:"size:100;index:idx_tenant"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

// TableName returns the table name for CasbinRule
func (CasbinRule) TableName() string {
	return "casbin_rules"
}

// Adapter represents the GORM adapter for Casbin
type Adapter struct {
	db          *gorm.DB
	tableName   string
	tablePrefix string
	tenantID    string
	isFiltered  bool
}

// AdapterOption represents an option for the adapter
type AdapterOption func(*Adapter)

// WithTablePrefix sets the table prefix
func WithTablePrefix(prefix string) AdapterOption {
	return func(a *Adapter) {
		a.tablePrefix = prefix
	}
}

// WithTenantID sets the tenant ID for multi-tenancy
func WithTenantID(tenantID string) AdapterOption {
	return func(a *Adapter) {
		a.tenantID = tenantID
	}
}

// NewAdapter creates a new GORM adapter for Casbin
func NewAdapter(cfg *authorization.DatabaseConfig, opts ...AdapterOption) (*Adapter, error) {
	dsn := cfg.GetDSN()

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	}

	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to connect to database: %v", authorization.ErrConnectionFailed, err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to get sql.DB: %v", authorization.ErrConnectionFailed, err)
	}

	// Configure connection pool
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	adapter := &Adapter{
		db:          db,
		tableName:   "casbin_rules",
		tablePrefix: cfg.TablePrefix,
		tenantID:    "",
		isFiltered:  false,
	}

	for _, opt := range opts {
		opt(adapter)
	}

	// Auto-migrate the casbin_rules table
	if err := adapter.migrate(); err != nil {
		return nil, err
	}

	return adapter, nil
}

// NewAdapterWithDB creates a new adapter with an existing GORM DB connection
func NewAdapterWithDB(db *gorm.DB, opts ...AdapterOption) (*Adapter, error) {
	adapter := &Adapter{
		db:         db,
		tableName:  "casbin_rules",
		isFiltered: false,
	}

	for _, opt := range opts {
		opt(adapter)
	}

	// Auto-migrate the casbin_rules table
	if err := adapter.migrate(); err != nil {
		return nil, err
	}

	return adapter, nil
}

// migrate creates the casbin_rules table
func (a *Adapter) migrate() error {
	return a.db.AutoMigrate(&CasbinRule{})
}

// DB returns the underlying database connection
func (a *Adapter) DB() *gorm.DB {
	return a.db
}

// Close closes the database connection
func (a *Adapter) Close() error {
	sqlDB, err := a.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Ping checks the database connection
func (a *Adapter) Ping(ctx context.Context) error {
	sqlDB, err := a.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

// getQuery returns a query with tenant filter if set
func (a *Adapter) getQuery() *gorm.DB {
	query := a.db.Model(&CasbinRule{})
	if a.tenantID != "" {
		query = query.Where("tenant_id = ?", a.tenantID)
	}
	return query
}

// LoadPolicy loads all policy rules from the database
func (a *Adapter) LoadPolicy() ([][]string, error) {
	var rules []CasbinRule
	if err := a.getQuery().Find(&rules).Error; err != nil {
		return nil, fmt.Errorf("%w: failed to load policies: %v", authorization.ErrDatabaseOperation, err)
	}

	var policies [][]string
	for _, rule := range rules {
		policies = append(policies, a.ruleToPolicy(&rule))
	}
	return policies, nil
}

// LoadFilteredPolicy loads policy rules that match the filter
func (a *Adapter) LoadFilteredPolicy(filter interface{}) ([][]string, error) {
	filterMap, ok := filter.(map[string]string)
	if !ok {
		return nil, fmt.Errorf("%w: invalid filter type", authorization.ErrInvalidInput)
	}

	query := a.getQuery()
	for key, value := range filterMap {
		query = query.Where(key+" = ?", value)
	}

	var rules []CasbinRule
	if err := query.Find(&rules).Error; err != nil {
		return nil, fmt.Errorf("%w: failed to load filtered policies: %v", authorization.ErrDatabaseOperation, err)
	}

	var policies [][]string
	for _, rule := range rules {
		policies = append(policies, a.ruleToPolicy(&rule))
	}

	a.isFiltered = true
	return policies, nil
}

// SavePolicy saves all policy rules to the database
func (a *Adapter) SavePolicy(policies [][]string) error {
	return a.db.Transaction(func(tx *gorm.DB) error {
		// Delete existing policies
		if a.tenantID != "" {
			if err := tx.Where("tenant_id = ?", a.tenantID).Delete(&CasbinRule{}).Error; err != nil {
				return fmt.Errorf("%w: failed to clear policies: %v", authorization.ErrDatabaseOperation, err)
			}
		} else {
			if err := tx.Where("1 = 1").Delete(&CasbinRule{}).Error; err != nil {
				return fmt.Errorf("%w: failed to clear policies: %v", authorization.ErrDatabaseOperation, err)
			}
		}

		// Insert new policies
		for _, policy := range policies {
			rule := a.policyToRule(policy)
			if err := tx.Create(&rule).Error; err != nil {
				return fmt.Errorf("%w: failed to save policy: %v", authorization.ErrDatabaseOperation, err)
			}
		}
		return nil
	})
}

// AddPolicy adds a policy rule to the database
func (a *Adapter) AddPolicy(sec string, ptype string, rule []string) error {
	policy := append([]string{ptype}, rule...)
	casbinRule := a.policyToRule(policy)

	if err := a.db.Create(&casbinRule).Error; err != nil {
		return fmt.Errorf("%w: failed to add policy: %v", authorization.ErrDatabaseOperation, err)
	}
	return nil
}

// AddPolicies adds multiple policy rules to the database
func (a *Adapter) AddPolicies(sec string, ptype string, rules [][]string) error {
	return a.db.Transaction(func(tx *gorm.DB) error {
		for _, rule := range rules {
			policy := append([]string{ptype}, rule...)
			casbinRule := a.policyToRule(policy)
			if err := tx.Create(&casbinRule).Error; err != nil {
				return fmt.Errorf("%w: failed to add policies: %v", authorization.ErrDatabaseOperation, err)
			}
		}
		return nil
	})
}

// RemovePolicy removes a policy rule from the database
func (a *Adapter) RemovePolicy(sec string, ptype string, rule []string) error {
	query := a.getQuery().Where("ptype = ?", ptype)

	for i, v := range rule {
		query = query.Where(fmt.Sprintf("v%d = ?", i), v)
	}

	if err := query.Delete(&CasbinRule{}).Error; err != nil {
		return fmt.Errorf("%w: failed to remove policy: %v", authorization.ErrDatabaseOperation, err)
	}
	return nil
}

// RemovePolicies removes multiple policy rules from the database
func (a *Adapter) RemovePolicies(sec string, ptype string, rules [][]string) error {
	return a.db.Transaction(func(tx *gorm.DB) error {
		for _, rule := range rules {
			query := tx.Model(&CasbinRule{}).Where("ptype = ?", ptype)
			if a.tenantID != "" {
				query = query.Where("tenant_id = ?", a.tenantID)
			}

			for i, v := range rule {
				query = query.Where(fmt.Sprintf("v%d = ?", i), v)
			}

			if err := query.Delete(&CasbinRule{}).Error; err != nil {
				return fmt.Errorf("%w: failed to remove policies: %v", authorization.ErrDatabaseOperation, err)
			}
		}
		return nil
	})
}

// RemoveFilteredPolicy removes policy rules that match the filter
func (a *Adapter) RemoveFilteredPolicy(sec string, ptype string, fieldIndex int, fieldValues ...string) error {
	query := a.getQuery().Where("ptype = ?", ptype)

	for i, v := range fieldValues {
		if v != "" {
			query = query.Where(fmt.Sprintf("v%d = ?", fieldIndex+i), v)
		}
	}

	if err := query.Delete(&CasbinRule{}).Error; err != nil {
		return fmt.Errorf("%w: failed to remove filtered policy: %v", authorization.ErrDatabaseOperation, err)
	}
	return nil
}

// UpdatePolicy updates a policy rule in the database
func (a *Adapter) UpdatePolicy(sec string, ptype string, oldRule, newRule []string) error {
	return a.db.Transaction(func(tx *gorm.DB) error {
		// Remove old policy
		if err := a.RemovePolicy(sec, ptype, oldRule); err != nil {
			return err
		}

		// Add new policy
		return a.AddPolicy(sec, ptype, newRule)
	})
}

// UpdatePolicies updates multiple policy rules in the database
func (a *Adapter) UpdatePolicies(sec string, ptype string, oldRules, newRules [][]string) error {
	return a.db.Transaction(func(tx *gorm.DB) error {
		for i := range oldRules {
			if err := a.UpdatePolicy(sec, ptype, oldRules[i], newRules[i]); err != nil {
				return err
			}
		}
		return nil
	})
}

// IsFiltered returns true if the adapter is filtered
func (a *Adapter) IsFiltered() bool {
	return a.isFiltered
}

// ruleToPolicy converts a CasbinRule to a policy array
func (a *Adapter) ruleToPolicy(rule *CasbinRule) []string {
	policy := []string{rule.Ptype}

	if rule.V0 != "" {
		policy = append(policy, rule.V0)
	}
	if rule.V1 != "" {
		policy = append(policy, rule.V1)
	}
	if rule.V2 != "" {
		policy = append(policy, rule.V2)
	}
	if rule.V3 != "" {
		policy = append(policy, rule.V3)
	}
	if rule.V4 != "" {
		policy = append(policy, rule.V4)
	}
	if rule.V5 != "" {
		policy = append(policy, rule.V5)
	}
	if rule.V6 != "" {
		policy = append(policy, rule.V6)
	}
	if rule.V7 != "" {
		policy = append(policy, rule.V7)
	}

	return policy
}

// policyToRule converts a policy array to a CasbinRule
func (a *Adapter) policyToRule(policy []string) *CasbinRule {
	rule := &CasbinRule{
		TenantID: a.tenantID,
	}

	if len(policy) > 0 {
		rule.Ptype = policy[0]
	}
	if len(policy) > 1 {
		rule.V0 = policy[1]
	}
	if len(policy) > 2 {
		rule.V1 = policy[2]
	}
	if len(policy) > 3 {
		rule.V2 = policy[3]
	}
	if len(policy) > 4 {
		rule.V3 = policy[4]
	}
	if len(policy) > 5 {
		rule.V4 = policy[5]
	}
	if len(policy) > 6 {
		rule.V5 = policy[6]
	}
	if len(policy) > 7 {
		rule.V6 = policy[7]
	}
	if len(policy) > 8 {
		rule.V7 = policy[8]
	}

	return rule
}

// MigrateAuthorizationTables creates additional authorization tables
func (a *Adapter) MigrateAuthorizationTables() error {
	return a.db.AutoMigrate(
		&authorization.Role{},
		&authorization.Permission{},
		&authorization.Policy{},
		&authorization.UserRole{},
	)
}
