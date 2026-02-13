// Cache Keys
export const CACHE_KEYS = {
  USER_PERMISSIONS: (userId: string) => `identity:user:${userId}:permissions`,
  TOKEN_BLACKLIST: (jti: string) => `identity:blacklist:${jti}`,
  ACTIVE_SESSION: (userId: string, deviceId: string) => `identity:session:${userId}:${deviceId}`,
  RATE_LIMIT: (key: string) => `identity:rate-limit:${key}`,
} as const;

// Cache TTLs (in seconds)
export const CACHE_TTL = {
  PERMISSIONS: 3600, // 1 hour
  SESSION: 604800, // 7 days
  RATE_LIMIT: 60, // 1 minute
} as const;

// Token Configuration
export const TOKEN_CONFIG = {
  ACCESS_TOKEN_EXPIRY: '15m',
  REFRESH_TOKEN_EXPIRY: '7d',
  REFRESH_TOKEN_EXPIRY_DAYS: 7,
  BCRYPT_SALT_ROUNDS: 12,
} as const;

// Default Roles
export const DEFAULT_ROLES = {
  SUPER_ADMIN: 'SUPER_ADMIN',
  ADMIN: 'ADMIN',
  SUPPORT: 'SUPPORT',
  FINANCE: 'FINANCE',
  WAREHOUSE: 'WAREHOUSE',
  SELLER: 'SELLER',
  CUSTOMER: 'CUSTOMER',
} as const;

// Role Definitions with metadata
export const ROLE_DEFINITIONS = [
  { name: 'SUPER_ADMIN', displayName: 'Super Administrator', isSystem: true, priority: 100 },
  { name: 'ADMIN', displayName: 'Administrator', isSystem: true, priority: 80 },
  { name: 'SUPPORT', displayName: 'Customer Support', isSystem: true, priority: 60 },
  { name: 'FINANCE', displayName: 'Finance Team', isSystem: true, priority: 60 },
  { name: 'WAREHOUSE', displayName: 'Warehouse Staff', isSystem: true, priority: 40 },
  { name: 'SELLER', displayName: 'Seller/Merchant', isSystem: true, priority: 30 },
  { name: 'CUSTOMER', displayName: 'Customer', isSystem: true, isDefault: true, priority: 10 },
] as const;

// Permission Categories
export const PERMISSION_CATEGORIES = {
  USER_MANAGEMENT: 'User Management',
  PRODUCT_MANAGEMENT: 'Product Management',
  ORDER_MANAGEMENT: 'Order Management',
  SHOP_MANAGEMENT: 'Shop Management',
  CAMPAIGN_MANAGEMENT: 'Campaign Management',
  CONTENT_MANAGEMENT: 'Content Management',
  REPORT_ANALYTICS: 'Reports & Analytics',
  SYSTEM_SETTINGS: 'System Settings',
  FINANCIAL: 'Financial Operations',
  INVENTORY: 'Inventory Management',
  LOGISTICS: 'Logistics',
} as const;

// Permissions
export const PERMISSIONS = {
  // User permissions
  USER_CREATE: 'user:create',
  USER_READ: 'user:read',
  USER_UPDATE: 'user:update',
  USER_DELETE: 'user:delete',
  USER_BAN: 'user:ban',
  USER_IMPERSONATE: 'user:impersonate',
  USER_EXPORT: 'user:export',

  // Product permissions
  PRODUCT_CREATE: 'product:create',
  PRODUCT_READ: 'product:read',
  PRODUCT_UPDATE: 'product:update',
  PRODUCT_UPDATE_OWN: 'product:update_own',
  PRODUCT_DELETE: 'product:delete',
  PRODUCT_APPROVE: 'product:approve',
  PRODUCT_FEATURE: 'product:feature',
  PRODUCT_BULK_EDIT: 'product:bulk_edit',

  // Order permissions
  ORDER_CREATE: 'order:create',
  ORDER_READ_OWN: 'order:read_own',
  ORDER_READ_ALL: 'order:read_all',
  ORDER_UPDATE: 'order:update',
  ORDER_CANCEL: 'order:cancel',
  ORDER_REFUND: 'order:refund',
  ORDER_EXPORT: 'order:export',

  // Shop permissions
  SHOP_CREATE: 'shop:create',
  SHOP_READ: 'shop:read',
  SHOP_UPDATE: 'shop:update',
  SHOP_DELETE: 'shop:delete',
  SHOP_VERIFY: 'shop:verify',
  SHOP_SUSPEND: 'shop:suspend',
  SHOP_MANAGE: 'shop:manage',

  // Campaign permissions
  CAMPAIGN_CREATE: 'campaign:create',
  CAMPAIGN_READ: 'campaign:read',
  CAMPAIGN_UPDATE: 'campaign:update',
  CAMPAIGN_DELETE: 'campaign:delete',
  CAMPAIGN_APPROVE: 'campaign:approve',
  VOUCHER_CREATE: 'voucher:create',
  VOUCHER_READ: 'voucher:read',
  VOUCHER_UPDATE: 'voucher:update',
  VOUCHER_DELETE: 'voucher:delete',

  // Content permissions
  BANNER_CREATE: 'banner:create',
  BANNER_READ: 'banner:read',
  BANNER_UPDATE: 'banner:update',
  BANNER_DELETE: 'banner:delete',
  BANNER_MANAGE: 'banner:manage',
  CATEGORY_CREATE: 'category:create',
  CATEGORY_READ: 'category:read',
  CATEGORY_UPDATE: 'category:update',
  CATEGORY_DELETE: 'category:delete',
  CATEGORY_MANAGE: 'category:manage',

  // Report permissions
  REPORT_VIEW: 'report:view',
  REPORT_EXPORT: 'report:export',
  ANALYTICS_VIEW: 'analytics:view',
  ANALYTICS_EXPORT: 'analytics:export',
  DASHBOARD_VIEW: 'dashboard:view',

  // System permissions
  SETTINGS_READ: 'settings:read',
  SETTINGS_UPDATE: 'settings:update',
  ROLE_MANAGE: 'role:manage',
  PERMISSION_MANAGE: 'permission:manage',
  AUDIT_LOG_VIEW: 'audit_log:view',

  // Financial permissions
  PAYMENT_VIEW: 'payment:view',
  PAYMENT_PROCESS: 'payment:process',
  PAYOUT_VIEW: 'payout:view',
  PAYOUT_APPROVE: 'payout:approve',
  PAYOUT_PROCESS: 'payout:process',
  FINANCE_REPORT: 'finance:report',

  // Inventory permissions
  INVENTORY_READ: 'inventory:read',
  INVENTORY_UPDATE: 'inventory:update',
  INVENTORY_TRANSFER: 'inventory:transfer',
  WAREHOUSE_MANAGE: 'warehouse:manage',
  STOCK_ADJUSTMENT: 'stock:adjustment',

  // Logistics permissions
  SHIPPING_READ: 'shipping:read',
  SHIPPING_UPDATE: 'shipping:update',
  CARRIER_MANAGE: 'carrier:manage',

  // Review permissions
  REVIEW_READ: 'review:read',
  REVIEW_MODERATE: 'review:moderate',
  REVIEW_DELETE: 'review:delete',

  // Notification permissions
  NOTIFICATION_SEND: 'notification:send',
  NOTIFICATION_TEMPLATE: 'notification:template',

  // Media permissions
  MEDIA_UPLOAD: 'media:upload',
  MEDIA_DELETE: 'media:delete',
  MEDIA_MANAGE: 'media:manage',

  // Admin wildcard
  ADMIN_ALL: '*',
} as const;

// Permission Definitions with full metadata
export const PERMISSION_DEFINITIONS = [
  // User Management
  {
    resource: 'user',
    action: 'create',
    displayName: 'Create Users',
    category: PERMISSION_CATEGORIES.USER_MANAGEMENT,
  },
  {
    resource: 'user',
    action: 'read',
    displayName: 'View Users',
    category: PERMISSION_CATEGORIES.USER_MANAGEMENT,
  },
  {
    resource: 'user',
    action: 'update',
    displayName: 'Update Users',
    category: PERMISSION_CATEGORIES.USER_MANAGEMENT,
  },
  {
    resource: 'user',
    action: 'delete',
    displayName: 'Delete Users',
    category: PERMISSION_CATEGORIES.USER_MANAGEMENT,
    isDangerous: true,
  },
  {
    resource: 'user',
    action: 'ban',
    displayName: 'Ban Users',
    category: PERMISSION_CATEGORIES.USER_MANAGEMENT,
    isDangerous: true,
  },
  {
    resource: 'user',
    action: 'impersonate',
    displayName: 'Impersonate Users',
    category: PERMISSION_CATEGORIES.USER_MANAGEMENT,
    isDangerous: true,
  },
  {
    resource: 'user',
    action: 'export',
    displayName: 'Export User Data',
    category: PERMISSION_CATEGORIES.USER_MANAGEMENT,
  },

  // Product Management
  {
    resource: 'product',
    action: 'create',
    displayName: 'Create Products',
    category: PERMISSION_CATEGORIES.PRODUCT_MANAGEMENT,
  },
  {
    resource: 'product',
    action: 'read',
    displayName: 'View Products',
    category: PERMISSION_CATEGORIES.PRODUCT_MANAGEMENT,
  },
  {
    resource: 'product',
    action: 'update',
    displayName: 'Update Any Product',
    category: PERMISSION_CATEGORIES.PRODUCT_MANAGEMENT,
  },
  {
    resource: 'product',
    action: 'update_own',
    displayName: 'Update Own Products',
    category: PERMISSION_CATEGORIES.PRODUCT_MANAGEMENT,
  },
  {
    resource: 'product',
    action: 'delete',
    displayName: 'Delete Products',
    category: PERMISSION_CATEGORIES.PRODUCT_MANAGEMENT,
  },
  {
    resource: 'product',
    action: 'approve',
    displayName: 'Approve Products',
    category: PERMISSION_CATEGORIES.PRODUCT_MANAGEMENT,
  },
  {
    resource: 'product',
    action: 'feature',
    displayName: 'Feature Products',
    category: PERMISSION_CATEGORIES.PRODUCT_MANAGEMENT,
  },
  {
    resource: 'product',
    action: 'bulk_edit',
    displayName: 'Bulk Edit Products',
    category: PERMISSION_CATEGORIES.PRODUCT_MANAGEMENT,
  },

  // Order Management
  {
    resource: 'order',
    action: 'create',
    displayName: 'Create Orders',
    category: PERMISSION_CATEGORIES.ORDER_MANAGEMENT,
  },
  {
    resource: 'order',
    action: 'read_own',
    displayName: 'View Own Orders',
    category: PERMISSION_CATEGORIES.ORDER_MANAGEMENT,
  },
  {
    resource: 'order',
    action: 'read_all',
    displayName: 'View All Orders',
    category: PERMISSION_CATEGORIES.ORDER_MANAGEMENT,
  },
  {
    resource: 'order',
    action: 'update',
    displayName: 'Update Orders',
    category: PERMISSION_CATEGORIES.ORDER_MANAGEMENT,
  },
  {
    resource: 'order',
    action: 'cancel',
    displayName: 'Cancel Orders',
    category: PERMISSION_CATEGORIES.ORDER_MANAGEMENT,
  },
  {
    resource: 'order',
    action: 'refund',
    displayName: 'Refund Orders',
    category: PERMISSION_CATEGORIES.ORDER_MANAGEMENT,
    isDangerous: true,
  },
  {
    resource: 'order',
    action: 'export',
    displayName: 'Export Orders',
    category: PERMISSION_CATEGORIES.ORDER_MANAGEMENT,
  },

  // Shop Management
  {
    resource: 'shop',
    action: 'create',
    displayName: 'Create Shops',
    category: PERMISSION_CATEGORIES.SHOP_MANAGEMENT,
  },
  {
    resource: 'shop',
    action: 'read',
    displayName: 'View Shops',
    category: PERMISSION_CATEGORIES.SHOP_MANAGEMENT,
  },
  {
    resource: 'shop',
    action: 'update',
    displayName: 'Update Shops',
    category: PERMISSION_CATEGORIES.SHOP_MANAGEMENT,
  },
  {
    resource: 'shop',
    action: 'delete',
    displayName: 'Delete Shops',
    category: PERMISSION_CATEGORIES.SHOP_MANAGEMENT,
    isDangerous: true,
  },
  {
    resource: 'shop',
    action: 'verify',
    displayName: 'Verify Shops',
    category: PERMISSION_CATEGORIES.SHOP_MANAGEMENT,
  },
  {
    resource: 'shop',
    action: 'suspend',
    displayName: 'Suspend Shops',
    category: PERMISSION_CATEGORIES.SHOP_MANAGEMENT,
    isDangerous: true,
  },
  {
    resource: 'shop',
    action: 'manage',
    displayName: 'Manage Own Shop',
    category: PERMISSION_CATEGORIES.SHOP_MANAGEMENT,
  },

  // Campaign Management
  {
    resource: 'campaign',
    action: 'create',
    displayName: 'Create Campaigns',
    category: PERMISSION_CATEGORIES.CAMPAIGN_MANAGEMENT,
  },
  {
    resource: 'campaign',
    action: 'read',
    displayName: 'View Campaigns',
    category: PERMISSION_CATEGORIES.CAMPAIGN_MANAGEMENT,
  },
  {
    resource: 'campaign',
    action: 'update',
    displayName: 'Update Campaigns',
    category: PERMISSION_CATEGORIES.CAMPAIGN_MANAGEMENT,
  },
  {
    resource: 'campaign',
    action: 'delete',
    displayName: 'Delete Campaigns',
    category: PERMISSION_CATEGORIES.CAMPAIGN_MANAGEMENT,
  },
  {
    resource: 'campaign',
    action: 'approve',
    displayName: 'Approve Campaigns',
    category: PERMISSION_CATEGORIES.CAMPAIGN_MANAGEMENT,
  },
  {
    resource: 'voucher',
    action: 'create',
    displayName: 'Create Vouchers',
    category: PERMISSION_CATEGORIES.CAMPAIGN_MANAGEMENT,
  },
  {
    resource: 'voucher',
    action: 'read',
    displayName: 'View Vouchers',
    category: PERMISSION_CATEGORIES.CAMPAIGN_MANAGEMENT,
  },
  {
    resource: 'voucher',
    action: 'update',
    displayName: 'Update Vouchers',
    category: PERMISSION_CATEGORIES.CAMPAIGN_MANAGEMENT,
  },
  {
    resource: 'voucher',
    action: 'delete',
    displayName: 'Delete Vouchers',
    category: PERMISSION_CATEGORIES.CAMPAIGN_MANAGEMENT,
  },

  // Content Management
  {
    resource: 'banner',
    action: 'create',
    displayName: 'Create Banners',
    category: PERMISSION_CATEGORIES.CONTENT_MANAGEMENT,
  },
  {
    resource: 'banner',
    action: 'read',
    displayName: 'View Banners',
    category: PERMISSION_CATEGORIES.CONTENT_MANAGEMENT,
  },
  {
    resource: 'banner',
    action: 'update',
    displayName: 'Update Banners',
    category: PERMISSION_CATEGORIES.CONTENT_MANAGEMENT,
  },
  {
    resource: 'banner',
    action: 'delete',
    displayName: 'Delete Banners',
    category: PERMISSION_CATEGORIES.CONTENT_MANAGEMENT,
  },
  {
    resource: 'banner',
    action: 'manage',
    displayName: 'Manage Banners',
    category: PERMISSION_CATEGORIES.CONTENT_MANAGEMENT,
  },
  {
    resource: 'category',
    action: 'create',
    displayName: 'Create Categories',
    category: PERMISSION_CATEGORIES.CONTENT_MANAGEMENT,
  },
  {
    resource: 'category',
    action: 'read',
    displayName: 'View Categories',
    category: PERMISSION_CATEGORIES.CONTENT_MANAGEMENT,
  },
  {
    resource: 'category',
    action: 'update',
    displayName: 'Update Categories',
    category: PERMISSION_CATEGORIES.CONTENT_MANAGEMENT,
  },
  {
    resource: 'category',
    action: 'delete',
    displayName: 'Delete Categories',
    category: PERMISSION_CATEGORIES.CONTENT_MANAGEMENT,
  },
  {
    resource: 'category',
    action: 'manage',
    displayName: 'Manage Categories',
    category: PERMISSION_CATEGORIES.CONTENT_MANAGEMENT,
  },

  // Reports & Analytics
  {
    resource: 'report',
    action: 'view',
    displayName: 'View Reports',
    category: PERMISSION_CATEGORIES.REPORT_ANALYTICS,
  },
  {
    resource: 'report',
    action: 'export',
    displayName: 'Export Reports',
    category: PERMISSION_CATEGORIES.REPORT_ANALYTICS,
  },
  {
    resource: 'analytics',
    action: 'view',
    displayName: 'View Analytics',
    category: PERMISSION_CATEGORIES.REPORT_ANALYTICS,
  },
  {
    resource: 'analytics',
    action: 'export',
    displayName: 'Export Analytics',
    category: PERMISSION_CATEGORIES.REPORT_ANALYTICS,
  },
  {
    resource: 'dashboard',
    action: 'view',
    displayName: 'View Dashboard',
    category: PERMISSION_CATEGORIES.REPORT_ANALYTICS,
  },

  // System Settings
  {
    resource: 'settings',
    action: 'read',
    displayName: 'View Settings',
    category: PERMISSION_CATEGORIES.SYSTEM_SETTINGS,
  },
  {
    resource: 'settings',
    action: 'update',
    displayName: 'Update Settings',
    category: PERMISSION_CATEGORIES.SYSTEM_SETTINGS,
    isDangerous: true,
  },
  {
    resource: 'role',
    action: 'manage',
    displayName: 'Manage Roles',
    category: PERMISSION_CATEGORIES.SYSTEM_SETTINGS,
    isDangerous: true,
  },
  {
    resource: 'permission',
    action: 'manage',
    displayName: 'Manage Permissions',
    category: PERMISSION_CATEGORIES.SYSTEM_SETTINGS,
    isDangerous: true,
  },
  {
    resource: 'audit_log',
    action: 'view',
    displayName: 'View Audit Logs',
    category: PERMISSION_CATEGORIES.SYSTEM_SETTINGS,
  },

  // Financial Operations
  {
    resource: 'payment',
    action: 'view',
    displayName: 'View Payments',
    category: PERMISSION_CATEGORIES.FINANCIAL,
  },
  {
    resource: 'payment',
    action: 'process',
    displayName: 'Process Payments',
    category: PERMISSION_CATEGORIES.FINANCIAL,
  },
  {
    resource: 'payout',
    action: 'view',
    displayName: 'View Payouts',
    category: PERMISSION_CATEGORIES.FINANCIAL,
  },
  {
    resource: 'payout',
    action: 'approve',
    displayName: 'Approve Payouts',
    category: PERMISSION_CATEGORIES.FINANCIAL,
    isDangerous: true,
  },
  {
    resource: 'payout',
    action: 'process',
    displayName: 'Process Payouts',
    category: PERMISSION_CATEGORIES.FINANCIAL,
    isDangerous: true,
  },
  {
    resource: 'finance',
    action: 'report',
    displayName: 'Financial Reports',
    category: PERMISSION_CATEGORIES.FINANCIAL,
  },

  // Inventory Management
  {
    resource: 'inventory',
    action: 'read',
    displayName: 'View Inventory',
    category: PERMISSION_CATEGORIES.INVENTORY,
  },
  {
    resource: 'inventory',
    action: 'update',
    displayName: 'Update Inventory',
    category: PERMISSION_CATEGORIES.INVENTORY,
  },
  {
    resource: 'inventory',
    action: 'transfer',
    displayName: 'Transfer Inventory',
    category: PERMISSION_CATEGORIES.INVENTORY,
  },
  {
    resource: 'warehouse',
    action: 'manage',
    displayName: 'Manage Warehouses',
    category: PERMISSION_CATEGORIES.INVENTORY,
  },
  {
    resource: 'stock',
    action: 'adjustment',
    displayName: 'Stock Adjustments',
    category: PERMISSION_CATEGORIES.INVENTORY,
  },

  // Logistics
  {
    resource: 'shipping',
    action: 'read',
    displayName: 'View Shipments',
    category: PERMISSION_CATEGORIES.LOGISTICS,
  },
  {
    resource: 'shipping',
    action: 'update',
    displayName: 'Update Shipments',
    category: PERMISSION_CATEGORIES.LOGISTICS,
  },
  {
    resource: 'carrier',
    action: 'manage',
    displayName: 'Manage Carriers',
    category: PERMISSION_CATEGORIES.LOGISTICS,
  },

  // Review Management
  {
    resource: 'review',
    action: 'read',
    displayName: 'View Reviews',
    category: PERMISSION_CATEGORIES.PRODUCT_MANAGEMENT,
  },
  {
    resource: 'review',
    action: 'moderate',
    displayName: 'Moderate Reviews',
    category: PERMISSION_CATEGORIES.PRODUCT_MANAGEMENT,
  },
  {
    resource: 'review',
    action: 'delete',
    displayName: 'Delete Reviews',
    category: PERMISSION_CATEGORIES.PRODUCT_MANAGEMENT,
  },

  // Notification Management
  {
    resource: 'notification',
    action: 'send',
    displayName: 'Send Notifications',
    category: PERMISSION_CATEGORIES.CONTENT_MANAGEMENT,
  },
  {
    resource: 'notification',
    action: 'template',
    displayName: 'Manage Templates',
    category: PERMISSION_CATEGORIES.CONTENT_MANAGEMENT,
  },

  // Media Management
  {
    resource: 'media',
    action: 'upload',
    displayName: 'Upload Media',
    category: PERMISSION_CATEGORIES.CONTENT_MANAGEMENT,
  },
  {
    resource: 'media',
    action: 'delete',
    displayName: 'Delete Media',
    category: PERMISSION_CATEGORIES.CONTENT_MANAGEMENT,
  },
  {
    resource: 'media',
    action: 'manage',
    displayName: 'Manage All Media',
    category: PERMISSION_CATEGORIES.CONTENT_MANAGEMENT,
  },
] as const;

// Role-Permission Mapping
export const ROLE_PERMISSION_MAPPING: Record<string, string[]> = {
  [DEFAULT_ROLES.SUPER_ADMIN]: [PERMISSIONS.ADMIN_ALL],
  [DEFAULT_ROLES.ADMIN]: [
    PERMISSIONS.USER_READ,
    PERMISSIONS.USER_UPDATE,
    PERMISSIONS.USER_BAN,
    PERMISSIONS.PRODUCT_READ,
    PERMISSIONS.PRODUCT_UPDATE,
    PERMISSIONS.PRODUCT_APPROVE,
    PERMISSIONS.PRODUCT_FEATURE,
    PERMISSIONS.ORDER_READ_ALL,
    PERMISSIONS.ORDER_UPDATE,
    PERMISSIONS.ORDER_CANCEL,
    PERMISSIONS.ORDER_REFUND,
    PERMISSIONS.SHOP_READ,
    PERMISSIONS.SHOP_UPDATE,
    PERMISSIONS.SHOP_VERIFY,
    PERMISSIONS.SHOP_SUSPEND,
    PERMISSIONS.CAMPAIGN_CREATE,
    PERMISSIONS.CAMPAIGN_READ,
    PERMISSIONS.CAMPAIGN_UPDATE,
    PERMISSIONS.CAMPAIGN_DELETE,
    PERMISSIONS.CAMPAIGN_APPROVE,
    PERMISSIONS.VOUCHER_CREATE,
    PERMISSIONS.VOUCHER_READ,
    PERMISSIONS.VOUCHER_UPDATE,
    PERMISSIONS.VOUCHER_DELETE,
    PERMISSIONS.BANNER_MANAGE,
    PERMISSIONS.CATEGORY_MANAGE,
    PERMISSIONS.REPORT_VIEW,
    PERMISSIONS.REPORT_EXPORT,
    PERMISSIONS.ANALYTICS_VIEW,
    PERMISSIONS.DASHBOARD_VIEW,
    PERMISSIONS.SETTINGS_READ,
    PERMISSIONS.ROLE_MANAGE,
    PERMISSIONS.AUDIT_LOG_VIEW,
    PERMISSIONS.REVIEW_READ,
    PERMISSIONS.REVIEW_MODERATE,
    PERMISSIONS.REVIEW_DELETE,
    PERMISSIONS.NOTIFICATION_SEND,
    PERMISSIONS.NOTIFICATION_TEMPLATE,
    PERMISSIONS.MEDIA_MANAGE,
  ],
  [DEFAULT_ROLES.SUPPORT]: [
    PERMISSIONS.USER_READ,
    PERMISSIONS.PRODUCT_READ,
    PERMISSIONS.ORDER_READ_ALL,
    PERMISSIONS.ORDER_UPDATE,
    PERMISSIONS.ORDER_CANCEL,
    PERMISSIONS.SHOP_READ,
    PERMISSIONS.REVIEW_READ,
    PERMISSIONS.REVIEW_MODERATE,
    PERMISSIONS.DASHBOARD_VIEW,
  ],
  [DEFAULT_ROLES.FINANCE]: [
    PERMISSIONS.ORDER_READ_ALL,
    PERMISSIONS.ORDER_REFUND,
    PERMISSIONS.ORDER_EXPORT,
    PERMISSIONS.PAYMENT_VIEW,
    PERMISSIONS.PAYMENT_PROCESS,
    PERMISSIONS.PAYOUT_VIEW,
    PERMISSIONS.PAYOUT_APPROVE,
    PERMISSIONS.PAYOUT_PROCESS,
    PERMISSIONS.FINANCE_REPORT,
    PERMISSIONS.REPORT_VIEW,
    PERMISSIONS.REPORT_EXPORT,
    PERMISSIONS.DASHBOARD_VIEW,
  ],
  [DEFAULT_ROLES.WAREHOUSE]: [
    PERMISSIONS.ORDER_READ_ALL,
    PERMISSIONS.INVENTORY_READ,
    PERMISSIONS.INVENTORY_UPDATE,
    PERMISSIONS.INVENTORY_TRANSFER,
    PERMISSIONS.WAREHOUSE_MANAGE,
    PERMISSIONS.STOCK_ADJUSTMENT,
    PERMISSIONS.SHIPPING_READ,
    PERMISSIONS.SHIPPING_UPDATE,
    PERMISSIONS.DASHBOARD_VIEW,
  ],
  [DEFAULT_ROLES.SELLER]: [
    PERMISSIONS.PRODUCT_CREATE,
    PERMISSIONS.PRODUCT_READ,
    PERMISSIONS.PRODUCT_UPDATE_OWN,
    PERMISSIONS.ORDER_READ_OWN,
    PERMISSIONS.SHOP_MANAGE,
    PERMISSIONS.INVENTORY_READ,
    PERMISSIONS.INVENTORY_UPDATE,
    PERMISSIONS.VOUCHER_CREATE,
    PERMISSIONS.VOUCHER_READ,
    PERMISSIONS.VOUCHER_UPDATE,
    PERMISSIONS.MEDIA_UPLOAD,
    PERMISSIONS.MEDIA_DELETE,
    PERMISSIONS.ANALYTICS_VIEW,
    PERMISSIONS.DASHBOARD_VIEW,
  ],
  [DEFAULT_ROLES.CUSTOMER]: [
    PERMISSIONS.PRODUCT_READ,
    PERMISSIONS.ORDER_CREATE,
    PERMISSIONS.ORDER_READ_OWN,
    PERMISSIONS.ORDER_CANCEL,
    PERMISSIONS.MEDIA_UPLOAD,
  ],
};

// Error Messages
export const ERROR_MESSAGES = {
  // Auth errors
  EMAIL_EXISTS: 'Email already exists',
  INVALID_CREDENTIALS: 'Invalid email or password',
  USER_NOT_FOUND: 'User not found',
  USER_INACTIVE: 'User account is inactive',
  USER_SUSPENDED: 'User account is suspended',
  USER_BANNED: 'User account is banned',
  ACCOUNT_LOCKED: 'Account is temporarily locked due to too many failed attempts',
  TOKEN_EXPIRED: 'Token has expired',
  TOKEN_REVOKED: 'Token has been revoked',
  TOKEN_INVALID: 'Invalid token',
  REFRESH_TOKEN_INVALID: 'Invalid refresh token',
  REFRESH_TOKEN_EXPIRED: 'Refresh token has expired',
  TWO_FACTOR_REQUIRED: 'Two-factor authentication required',
  TWO_FACTOR_INVALID: 'Invalid two-factor code',

  // Permission errors
  PERMISSION_DENIED: 'Permission denied',
  INSUFFICIENT_PERMISSIONS: 'Insufficient permissions to perform this action',

  // Validation errors
  VALIDATION_FAILED: 'Validation failed',
  EMAIL_INVALID: 'Invalid email format',
  PASSWORD_WEAK: 'Password does not meet requirements',

  // Server errors
  INTERNAL_ERROR: 'An internal error occurred',
  REGISTRATION_FAILED: 'Registration failed',
  LOGIN_FAILED: 'Login failed',
  LOGOUT_FAILED: 'Logout failed',
} as const;

// Success Messages
export const SUCCESS_MESSAGES = {
  REGISTERED: 'User registered successfully',
  LOGGED_IN: 'Login successful',
  LOGGED_OUT: 'Logged out successfully',
  TOKEN_REFRESHED: 'Tokens refreshed successfully',
  PROFILE_RETRIEVED: 'Profile retrieved successfully',
  PERMISSION_GRANTED: 'Permission check passed',
  PASSWORD_RESET_SENT: 'Password reset email sent',
  PASSWORD_CHANGED: 'Password changed successfully',
  TWO_FACTOR_ENABLED: 'Two-factor authentication enabled',
  TWO_FACTOR_DISABLED: 'Two-factor authentication disabled',
} as const;

// API Response Codes
export const RESPONSE_CODES = {
  SUCCESS: 'SUCCESS',
  CREATED: 'CREATED',
  BAD_REQUEST: 'BAD_REQUEST',
  UNAUTHORIZED: 'UNAUTHORIZED',
  FORBIDDEN: 'FORBIDDEN',
  NOT_FOUND: 'NOT_FOUND',
  CONFLICT: 'CONFLICT',
  TOO_MANY_REQUESTS: 'TOO_MANY_REQUESTS',
  INTERNAL_ERROR: 'INTERNAL_ERROR',
} as const;

// User Status
export const USER_STATUS = {
  ACTIVE: 'ACTIVE',
  INACTIVE: 'INACTIVE',
  SUSPENDED: 'SUSPENDED',
  BANNED: 'BANNED',
  PENDING: 'PENDING',
} as const;

// Login Status
export const LOGIN_STATUS = {
  SUCCESS: 'SUCCESS',
  FAILED: 'FAILED',
  BLOCKED: 'BLOCKED',
  TWO_FA_REQUIRED: '2FA_REQUIRED',
  TWO_FA_SUCCESS: '2FA_SUCCESS',
  TWO_FA_FAILED: '2FA_FAILED',
} as const;

// Auth Methods
export const AUTH_METHODS = {
  PASSWORD: 'PASSWORD',
  GOOGLE: 'GOOGLE',
  FACEBOOK: 'FACEBOOK',
  APPLE: 'APPLE',
  TWO_FA: '2FA',
} as const;

// Metadata Keys
export const METADATA_KEYS = {
  PERMISSIONS: 'permissions',
  IS_PUBLIC: 'isPublic',
  ROLES: 'roles',
} as const;

// Security Configuration
export const SECURITY_CONFIG = {
  MAX_FAILED_LOGIN_ATTEMPTS: 5,
  LOCK_DURATION_MINUTES: 30,
  PASSWORD_RESET_EXPIRY_HOURS: 24,
  REFRESH_TOKEN_ROTATION: true,
} as const;
