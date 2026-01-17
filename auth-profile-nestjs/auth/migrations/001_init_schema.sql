-- ============================================================================
-- TAFU-AUTH Identity Service Database Migration
-- Version: 001
-- Date: 2026-01-07
-- Database: PostgreSQL 15+
-- ORM: TypeORM
-- ============================================================================

-- Enable extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ===================
-- Table: users
-- ===================
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(255) NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Unique constraint
ALTER TABLE users ADD CONSTRAINT uq_users_email UNIQUE (email);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_is_active ON users(is_active);
CREATE INDEX IF NOT EXISTS idx_users_created_at ON users(created_at);

-- ===================
-- Table: roles
-- ===================
CREATE TABLE IF NOT EXISTS roles (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description VARCHAR(500),
    is_system BOOLEAN DEFAULT FALSE
);

ALTER TABLE roles ADD CONSTRAINT uq_roles_name UNIQUE (name);
CREATE INDEX IF NOT EXISTS idx_roles_name ON roles(name);

-- ===================
-- Table: permissions
-- ===================
CREATE TABLE IF NOT EXISTS permissions (
    id SERIAL PRIMARY KEY,
    resource VARCHAR(100) NOT NULL,
    action VARCHAR(100) NOT NULL,
    slug VARCHAR(201) NOT NULL,
    description VARCHAR(500)
);

ALTER TABLE permissions ADD CONSTRAINT uq_permissions_slug UNIQUE (slug);
CREATE INDEX IF NOT EXISTS idx_permissions_resource ON permissions(resource);
CREATE INDEX IF NOT EXISTS idx_permissions_action ON permissions(action);
CREATE INDEX IF NOT EXISTS idx_permissions_slug ON permissions(slug);

-- ===================
-- Table: role_permissions
-- ===================
CREATE TABLE IF NOT EXISTS role_permissions (
    id SERIAL PRIMARY KEY,
    role_id INTEGER NOT NULL,
    permission_id INTEGER NOT NULL,
    CONSTRAINT fk_role_permissions_role FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE,
    CONSTRAINT fk_role_permissions_permission FOREIGN KEY (permission_id) REFERENCES permissions(id) ON DELETE CASCADE
);

ALTER TABLE role_permissions ADD CONSTRAINT uq_role_permissions UNIQUE (role_id, permission_id);
CREATE INDEX IF NOT EXISTS idx_role_permissions_role_id ON role_permissions(role_id);
CREATE INDEX IF NOT EXISTS idx_role_permissions_permission_id ON role_permissions(permission_id);

-- ===================
-- Table: user_roles
-- ===================
CREATE TABLE IF NOT EXISTS user_roles (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL,
    role_id INTEGER NOT NULL,
    assigned_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT fk_user_roles_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_user_roles_role FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE
);

ALTER TABLE user_roles ADD CONSTRAINT uq_user_roles UNIQUE (user_id, role_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_user_id ON user_roles(user_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_role_id ON user_roles(role_id);

-- ===================
-- Table: refresh_tokens
-- ===================
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    token_hash VARCHAR(255) NOT NULL,
    device_info VARCHAR(500),
    ip_address VARCHAR(45),
    expires_at TIMESTAMPTZ NOT NULL,
    is_revoked BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT fk_refresh_tokens_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_is_revoked ON refresh_tokens(is_revoked);

-- ===================
-- Auto-update trigger
-- ===================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ===================
-- Seed data: Default roles
-- ===================
INSERT INTO roles (name, description, is_system) VALUES 
    ('CUSTOMER', 'Default role for customers', TRUE),
    ('SELLER', 'Role for merchants/shop owners', TRUE),
    ('ADMIN', 'System administrator with full access', TRUE)
ON CONFLICT (name) DO NOTHING;

-- ===================
-- Seed data: Default permissions
-- ===================
INSERT INTO permissions (resource, action, slug, description) VALUES 
    ('product', 'create', 'product:create', 'Create new products'),
    ('product', 'read', 'product:read', 'View products'),
    ('product', 'update', 'product:update', 'Update existing products'),
    ('product', 'delete', 'product:delete', 'Delete products'),
    ('order', 'create', 'order:create', 'Create new orders'),
    ('order', 'read_own', 'order:read_own', 'View own orders'),
    ('order', 'read_all', 'order:read_all', 'View all orders'),
    ('order', 'update', 'order:update', 'Update order status'),
    ('order', 'cancel', 'order:cancel', 'Cancel orders'),
    ('user', 'read', 'user:read', 'View user profiles'),
    ('user', 'update', 'user:update', 'Update user profiles'),
    ('user', 'manage', 'user:manage', 'Manage all users'),
    ('shop', 'manage', 'shop:manage', 'Manage shop settings'),
    ('inventory', 'read', 'inventory:read', 'View inventory'),
    ('inventory', 'update', 'inventory:update', 'Update inventory'),
    ('campaign', 'manage', 'campaign:manage', 'Manage campaigns'),
    ('analytics', 'read', 'analytics:read', 'View analytics')
ON CONFLICT (slug) DO NOTHING;

-- ===================
-- Seed data: Assign permissions to roles
-- ===================
-- CUSTOMER permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p 
WHERE r.name = 'CUSTOMER' AND p.slug IN (
    'product:read', 
    'order:create', 
    'order:read_own', 
    'order:cancel', 
    'user:read', 
    'user:update'
)
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- SELLER permissions (includes CUSTOMER + more)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p 
WHERE r.name = 'SELLER' AND p.slug IN (
    'product:create',
    'product:read', 
    'product:update', 
    'product:delete',
    'order:create',
    'order:read_own',
    'order:read_all',
    'order:update',
    'user:read',
    'user:update',
    'shop:manage',
    'inventory:read',
    'inventory:update'
)
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- ADMIN permissions (all)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p 
WHERE r.name = 'ADMIN'
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Add comments
COMMENT ON TABLE users IS 'Core user accounts for authentication';
COMMENT ON TABLE roles IS 'Role definitions for RBAC';
COMMENT ON TABLE permissions IS 'Granular permissions (resource:action)';
COMMENT ON TABLE role_permissions IS 'Many-to-many: roles to permissions';
COMMENT ON TABLE user_roles IS 'Many-to-many: users to roles';
COMMENT ON TABLE refresh_tokens IS 'JWT refresh tokens with device tracking';
