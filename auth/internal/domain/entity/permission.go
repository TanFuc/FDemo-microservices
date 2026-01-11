package entity

import "fmt"

type Permission struct {
	ID              int64            `gorm:"primaryKey;autoIncrement" json:"id"`
	Resource        string           `gorm:"type:varchar(50);not null;uniqueIndex:idx_permissions_resource_action" json:"resource"`
	Action          string           `gorm:"type:varchar(50);not null;uniqueIndex:idx_permissions_resource_action" json:"action"`
	Slug            string           `gorm:"type:varchar(101);uniqueIndex;not null" json:"slug"`
	DisplayName     string           `gorm:"type:varchar(100);not null" json:"displayName"`
	Description     *string          `gorm:"type:text" json:"description,omitempty"`
	Category        *string          `gorm:"type:varchar(50);index" json:"category,omitempty"`
	IsDangerous     bool             `gorm:"default:false" json:"isDangerous"`
	RolePermissions []RolePermission `gorm:"foreignKey:PermissionID" json:"-"`
}

func (Permission) TableName() string {
	return "permissions"
}

func NewPermission(resource, action, displayName string, category *string, isDangerous bool) Permission {
	return Permission{
		Resource:    resource,
		Action:      action,
		Slug:        fmt.Sprintf("%s:%s", resource, action),
		DisplayName: displayName,
		Category:    category,
		IsDangerous: isDangerous,
	}
}

var categoryPtr = func(s string) *string { return &s }

var DefaultPermissions = []Permission{
	// User Management
	NewPermission("user", "create", "Create Users", categoryPtr("User Management"), false),
	NewPermission("user", "read", "View Users", categoryPtr("User Management"), false),
	NewPermission("user", "update", "Update Users", categoryPtr("User Management"), false),
	NewPermission("user", "delete", "Delete Users", categoryPtr("User Management"), true),
	NewPermission("user", "ban", "Ban Users", categoryPtr("User Management"), true),
	NewPermission("user", "impersonate", "Impersonate Users", categoryPtr("User Management"), true),
	NewPermission("user", "export", "Export Users", categoryPtr("User Management"), false),

	// Product Management
	NewPermission("product", "create", "Create Products", categoryPtr("Product Management"), false),
	NewPermission("product", "read", "View Products", categoryPtr("Product Management"), false),
	NewPermission("product", "update", "Update Products", categoryPtr("Product Management"), false),
	NewPermission("product", "update_own", "Update Own Products", categoryPtr("Product Management"), false),
	NewPermission("product", "delete", "Delete Products", categoryPtr("Product Management"), true),
	NewPermission("product", "approve", "Approve Products", categoryPtr("Product Management"), false),
	NewPermission("product", "feature", "Feature Products", categoryPtr("Product Management"), false),
	NewPermission("product", "bulk_edit", "Bulk Edit Products", categoryPtr("Product Management"), false),

	// Order Management
	NewPermission("order", "create", "Create Orders", categoryPtr("Order Management"), false),
	NewPermission("order", "read_own", "View Own Orders", categoryPtr("Order Management"), false),
	NewPermission("order", "read_all", "View All Orders", categoryPtr("Order Management"), false),
	NewPermission("order", "update", "Update Orders", categoryPtr("Order Management"), false),
	NewPermission("order", "cancel", "Cancel Orders", categoryPtr("Order Management"), false),
	NewPermission("order", "refund", "Refund Orders", categoryPtr("Order Management"), true),
	NewPermission("order", "export", "Export Orders", categoryPtr("Order Management"), false),

	// Shop Management
	NewPermission("shop", "create", "Create Shops", categoryPtr("Shop Management"), false),
	NewPermission("shop", "read", "View Shops", categoryPtr("Shop Management"), false),
	NewPermission("shop", "update", "Update Shops", categoryPtr("Shop Management"), false),
	NewPermission("shop", "delete", "Delete Shops", categoryPtr("Shop Management"), true),
	NewPermission("shop", "verify", "Verify Shops", categoryPtr("Shop Management"), false),
	NewPermission("shop", "suspend", "Suspend Shops", categoryPtr("Shop Management"), true),
	NewPermission("shop", "manage", "Manage Shops", categoryPtr("Shop Management"), false),

	// Campaign Management
	NewPermission("campaign", "create", "Create Campaigns", categoryPtr("Campaign Management"), false),
	NewPermission("campaign", "read", "View Campaigns", categoryPtr("Campaign Management"), false),
	NewPermission("campaign", "update", "Update Campaigns", categoryPtr("Campaign Management"), false),
	NewPermission("campaign", "delete", "Delete Campaigns", categoryPtr("Campaign Management"), true),
	NewPermission("campaign", "approve", "Approve Campaigns", categoryPtr("Campaign Management"), false),

	// Voucher Management
	NewPermission("voucher", "create", "Create Vouchers", categoryPtr("Campaign Management"), false),
	NewPermission("voucher", "read", "View Vouchers", categoryPtr("Campaign Management"), false),
	NewPermission("voucher", "update", "Update Vouchers", categoryPtr("Campaign Management"), false),
	NewPermission("voucher", "delete", "Delete Vouchers", categoryPtr("Campaign Management"), true),

	// Content Management
	NewPermission("banner", "create", "Create Banners", categoryPtr("Content Management"), false),
	NewPermission("banner", "read", "View Banners", categoryPtr("Content Management"), false),
	NewPermission("banner", "update", "Update Banners", categoryPtr("Content Management"), false),
	NewPermission("banner", "delete", "Delete Banners", categoryPtr("Content Management"), false),
	NewPermission("category", "manage", "Manage Categories", categoryPtr("Content Management"), false),
	NewPermission("notification", "send", "Send Notifications", categoryPtr("Content Management"), false),

	// Reports & Analytics
	NewPermission("report", "view", "View Reports", categoryPtr("Reports & Analytics"), false),
	NewPermission("report", "export", "Export Reports", categoryPtr("Reports & Analytics"), false),
	NewPermission("dashboard", "view", "View Dashboard", categoryPtr("Reports & Analytics"), false),
	NewPermission("analytics", "view", "View Analytics", categoryPtr("Reports & Analytics"), false),

	// System Settings
	NewPermission("settings", "read", "View Settings", categoryPtr("System Settings"), false),
	NewPermission("settings", "update", "Update Settings", categoryPtr("System Settings"), true),
	NewPermission("role", "manage", "Manage Roles", categoryPtr("System Settings"), true),
	NewPermission("permission", "manage", "Manage Permissions", categoryPtr("System Settings"), true),
	NewPermission("audit", "view", "View Audit Logs", categoryPtr("System Settings"), false),

	// Financial
	NewPermission("payment", "view", "View Payments", categoryPtr("Financial"), false),
	NewPermission("payment", "process", "Process Payments", categoryPtr("Financial"), false),
	NewPermission("payout", "approve", "Approve Payouts", categoryPtr("Financial"), true),
	NewPermission("finance", "report", "View Finance Reports", categoryPtr("Financial"), false),

	// Inventory & Logistics
	NewPermission("inventory", "read", "View Inventory", categoryPtr("Inventory & Logistics"), false),
	NewPermission("inventory", "update", "Update Inventory", categoryPtr("Inventory & Logistics"), false),
	NewPermission("warehouse", "manage", "Manage Warehouse", categoryPtr("Inventory & Logistics"), false),
	NewPermission("stock", "adjust", "Adjust Stock", categoryPtr("Inventory & Logistics"), false),
	NewPermission("shipping", "manage", "Manage Shipping", categoryPtr("Inventory & Logistics"), false),
	NewPermission("carrier", "manage", "Manage Carriers", categoryPtr("Inventory & Logistics"), false),

	// Review Management
	NewPermission("review", "read", "View Reviews", categoryPtr("Review Management"), false),
	NewPermission("review", "moderate", "Moderate Reviews", categoryPtr("Review Management"), false),
	NewPermission("review", "delete", "Delete Reviews", categoryPtr("Review Management"), true),

	// Media Management
	NewPermission("media", "upload", "Upload Media", categoryPtr("Media Management"), false),
	NewPermission("media", "delete", "Delete Media", categoryPtr("Media Management"), false),
	NewPermission("media", "manage", "Manage Media", categoryPtr("Media Management"), false),
}

// RolePermissionMapping defines default permissions for each role
var RolePermissionMapping = map[string][]string{
	RoleSuperAdmin: {"*"}, // All permissions

	RoleAdmin: {
		"user:create", "user:read", "user:update", "user:ban", "user:export",
		"product:read", "product:approve", "product:feature", "product:bulk_edit",
		"order:read_all", "order:update", "order:cancel", "order:refund", "order:export",
		"shop:read", "shop:update", "shop:verify", "shop:suspend", "shop:manage",
		"campaign:create", "campaign:read", "campaign:update", "campaign:delete", "campaign:approve",
		"voucher:create", "voucher:read", "voucher:update", "voucher:delete",
		"banner:create", "banner:read", "banner:update", "banner:delete",
		"category:manage", "notification:send",
		"report:view", "report:export", "dashboard:view", "analytics:view",
		"settings:read", "role:manage", "audit:view",
		"review:read", "review:moderate", "review:delete",
		"media:upload", "media:delete", "media:manage",
	},

	RoleSupport: {
		"user:read",
		"product:read",
		"order:read_all", "order:update", "order:cancel",
		"shop:read",
		"review:read", "review:moderate",
		"dashboard:view",
	},

	RoleFinance: {
		"order:read_all", "order:export",
		"payment:view", "payment:process",
		"payout:approve",
		"finance:report",
		"report:view", "report:export",
		"dashboard:view",
	},

	RoleWarehouse: {
		"order:read_all", "order:update",
		"inventory:read", "inventory:update",
		"warehouse:manage",
		"stock:adjust",
		"shipping:manage",
		"dashboard:view",
	},

	RoleSeller: {
		"product:create", "product:read", "product:update_own",
		"order:read_own", "order:update",
		"shop:create", "shop:read", "shop:update",
		"inventory:read",
		"voucher:create", "voucher:read", "voucher:update",
		"media:upload", "media:delete",
		"analytics:view",
	},

	RoleCustomer: {
		"product:read",
		"order:create", "order:read_own", "order:cancel",
		"media:upload",
	},
}
