-- ============================================================================
-- TAFU MICROSERVICES - COMPLETE DATABASE SCHEMA
-- Version: 1.0.0
-- Date: 2026-01-07
-- Description: Consolidated database schema for all TAFU microservices
-- ============================================================================

-- ============================================================================
-- DATABASE 1: identity_db (tafu-auth service)
-- Engine: PostgreSQL 15+
-- ORM: TypeORM (NestJS)
-- ============================================================================

\c identity_db;

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ===================
-- Table: users
-- ===================
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(255) NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_is_active ON users(is_active);

COMMENT ON TABLE users IS 'Core user accounts for authentication';
COMMENT ON COLUMN users.password_hash IS 'Bcrypt hashed password, never store plain text';

-- ===================
-- Table: roles
-- ===================
CREATE TABLE IF NOT EXISTS roles (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    description VARCHAR(500),
    is_system BOOLEAN DEFAULT FALSE
);

CREATE INDEX idx_roles_name ON roles(name);

COMMENT ON TABLE roles IS 'Role definitions for RBAC (CUSTOMER, SELLER, ADMIN)';

-- Insert default roles
INSERT INTO roles (name, description, is_system) VALUES 
    ('CUSTOMER', 'Default role for customers', TRUE),
    ('SELLER', 'Role for merchants/shop owners', TRUE),
    ('ADMIN', 'System administrator', TRUE)
ON CONFLICT (name) DO NOTHING;

-- ===================
-- Table: permissions
-- ===================
CREATE TABLE IF NOT EXISTS permissions (
    id SERIAL PRIMARY KEY,
    resource VARCHAR(100) NOT NULL,
    action VARCHAR(100) NOT NULL,
    slug VARCHAR(201) NOT NULL UNIQUE,
    description VARCHAR(500)
);

CREATE INDEX idx_permissions_resource ON permissions(resource);
CREATE INDEX idx_permissions_action ON permissions(action);
CREATE INDEX idx_permissions_slug ON permissions(slug);

COMMENT ON TABLE permissions IS 'Granular permissions (resource:action format)';

-- Insert default permissions
INSERT INTO permissions (resource, action, slug, description) VALUES 
    ('product', 'create', 'product:create', 'Create new products'),
    ('product', 'read', 'product:read', 'View products'),
    ('product', 'update', 'product:update', 'Update products'),
    ('product', 'delete', 'product:delete', 'Delete products'),
    ('order', 'create', 'order:create', 'Create orders'),
    ('order', 'read_own', 'order:read_own', 'View own orders'),
    ('order', 'read_all', 'order:read_all', 'View all orders (admin)'),
    ('order', 'cancel', 'order:cancel', 'Cancel orders'),
    ('user', 'read', 'user:read', 'View user profiles'),
    ('user', 'update', 'user:update', 'Update user profiles'),
    ('shop', 'manage', 'shop:manage', 'Manage shop settings')
ON CONFLICT (slug) DO NOTHING;

-- ===================
-- Table: role_permissions (junction)
-- ===================
CREATE TABLE IF NOT EXISTS role_permissions (
    id SERIAL PRIMARY KEY,
    role_id INTEGER NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id INTEGER NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    UNIQUE(role_id, permission_id)
);

CREATE INDEX idx_role_permissions_role_id ON role_permissions(role_id);
CREATE INDEX idx_role_permissions_permission_id ON role_permissions(permission_id);

-- Assign permissions to roles
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p 
WHERE r.name = 'CUSTOMER' AND p.slug IN ('product:read', 'order:create', 'order:read_own', 'order:cancel', 'user:read', 'user:update')
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p 
WHERE r.name = 'SELLER' AND p.slug IN ('product:create', 'product:read', 'product:update', 'product:delete', 'order:read_all', 'shop:manage')
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p 
WHERE r.name = 'ADMIN'
ON CONFLICT DO NOTHING;

-- ===================
-- Table: user_roles (junction)
-- ===================
CREATE TABLE IF NOT EXISTS user_roles (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id INTEGER NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    assigned_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, role_id)
);

CREATE INDEX idx_user_roles_user_id ON user_roles(user_id);
CREATE INDEX idx_user_roles_role_id ON user_roles(role_id);

-- ===================
-- Table: refresh_tokens
-- ===================
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL,
    device_info VARCHAR(500),
    ip_address VARCHAR(45),
    expires_at TIMESTAMPTZ NOT NULL,
    is_revoked BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);

COMMENT ON TABLE refresh_tokens IS 'JWT refresh tokens with device tracking';

-- ============================================================================
-- DATABASE 2: order_db (tafu-order service)
-- Engine: PostgreSQL 15+
-- ORM: GORM (Go)
-- ============================================================================

\c order_db;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Enum for order status
CREATE TYPE order_status AS ENUM ('PENDING', 'PAID', 'SHIPPED', 'COMPLETED', 'CANCELLED');

-- ===================
-- Table: orders (Aggregate Root)
-- ===================
CREATE TABLE IF NOT EXISTS orders (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    
    -- Money fields (strict decimal precision)
    total_amount DECIMAL(19, 4) NOT NULL DEFAULT 0,
    shipping_fee DECIMAL(19, 4) NOT NULL DEFAULT 0,
    discount_amount DECIMAL(19, 4) NOT NULL DEFAULT 0,
    final_amount DECIMAL(19, 4) NOT NULL DEFAULT 0,
    
    status order_status NOT NULL DEFAULT 'PENDING',
    payment_method VARCHAR(50) NOT NULL,
    
    -- Snapshot Data - immutable address at order time
    shipping_address JSONB NOT NULL,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_orders_user_id ON orders(user_id);
CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_orders_created_at ON orders(created_at DESC);
CREATE INDEX idx_orders_payment_method ON orders(payment_method);

COMMENT ON TABLE orders IS 'Order aggregate root with snapshotted data';
COMMENT ON COLUMN orders.shipping_address IS 'JSONB snapshot of address at order time - never reference profile';

-- ===================
-- Table: order_items (Product Snapshot)
-- ===================
CREATE TABLE IF NOT EXISTS order_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    
    -- Product Snapshot (immutable)
    product_id VARCHAR(100) NOT NULL,
    sku_id VARCHAR(100) NOT NULL,
    product_name VARCHAR(255) NOT NULL,
    sku_code VARCHAR(100) NOT NULL,
    thumbnail VARCHAR(500),
    
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    unit_price DECIMAL(19, 4) NOT NULL CHECK (unit_price >= 0),
    sub_total DECIMAL(19, 4) NOT NULL,
    
    reservation_id VARCHAR(100)
);

CREATE INDEX idx_order_items_order_id ON order_items(order_id);
CREATE INDEX idx_order_items_product_id ON order_items(product_id);
CREATE INDEX idx_order_items_sku_id ON order_items(sku_id);

COMMENT ON TABLE order_items IS 'Snapshot of products at purchase time - price/name immutable';

-- Trigger for updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_orders_updated_at
    BEFORE UPDATE ON orders
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- DATABASE 3: inventory_db (tafu-inventory service)
-- Engine: PostgreSQL 15+
-- ORM: GORM (Go)
-- ============================================================================

\c inventory_db;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Enum for reservation status
CREATE TYPE reservation_status AS ENUM ('PENDING', 'CONFIRMED', 'CANCELLED');

-- ===================
-- Table: inventory_items (The Ledger)
-- ===================
CREATE TABLE IF NOT EXISTS inventory_items (
    sku_id VARCHAR(100) PRIMARY KEY,
    total_stock INTEGER NOT NULL DEFAULT 0 CHECK (total_stock >= 0),
    reserved_stock INTEGER NOT NULL DEFAULT 0 CHECK (reserved_stock >= 0),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT chk_stock CHECK (total_stock >= reserved_stock)
);

CREATE INDEX idx_inventory_items_updated_at ON inventory_items(updated_at);

COMMENT ON TABLE inventory_items IS 'Inventory ledger - total vs reserved stock';
COMMENT ON COLUMN inventory_items.reserved_stock IS 'Stock currently held by pending orders';

-- ===================
-- Table: stock_reservations (Transaction Log)
-- ===================
CREATE TABLE IF NOT EXISTS stock_reservations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id VARCHAR(100) NOT NULL,
    sku_id VARCHAR(100) NOT NULL REFERENCES inventory_items(sku_id),
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    status reservation_status NOT NULL DEFAULT 'PENDING',
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_stock_reservations_order_id ON stock_reservations(order_id);
CREATE INDEX idx_stock_reservations_sku_id ON stock_reservations(sku_id);
CREATE INDEX idx_stock_reservations_status ON stock_reservations(status);
CREATE INDEX idx_stock_reservations_expires_at ON stock_reservations(expires_at);

-- Composite index for cleanup job
CREATE INDEX idx_stock_reservations_pending_expires 
    ON stock_reservations(status, expires_at) 
    WHERE status = 'PENDING';

COMMENT ON TABLE stock_reservations IS 'Two-phase reservation log for stock management';

-- Trigger for updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_stock_reservations_updated_at
    BEFORE UPDATE ON stock_reservations
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- DATABASE 4: payment_db (tafu-payment service)
-- Engine: PostgreSQL 15+
-- Note: Already has migration 001_init.sql - this is reference
-- ============================================================================

-- Schema already exists in tafu-payment/migrations/001_init.sql

-- ============================================================================
-- DATABASE 5: logistics_db (tafu-logistic service)
-- Engine: PostgreSQL 15+
-- Note: Already has migrations - this is reference
-- ============================================================================

-- Schema already exists in:
-- - tafu-logistic/migrations/001_create_shipping_orders.sql
-- - tafu-logistic/migrations/002_create_webhook_logs.sql

-- ============================================================================
-- DATABASE 6: campaign_db (tafu-campaign service)
-- Engine: PostgreSQL 15+
-- Note: Already has migration 001_init.sql - this is reference
-- ============================================================================

-- Schema already exists in tafu-campaign/migrations/001_init.sql

-- ============================================================================
-- DATABASE 7: analytics_db (tafu-analytic service)
-- Engine: ClickHouse
-- ============================================================================

-- ClickHouse DDL must not run through the PostgreSQL entrypoint.
-- The canonical schema is mounted separately from:
-- analytic/migrations/001_init_schema.sql

-- ============================================================================
-- MONGODB COLLECTIONS (tafu-profile, tafu-catalog, tafu-cart, 
--                       tafu-review, tafu-notification)
-- Engine: MongoDB 6+
-- ============================================================================

-- Collection: profiles (tafu-profile)
-- Index: { userId: 1 } unique
-- Index: { "shopConfig.shopName": 1 } unique, sparse
/*
{
  "_id": ObjectId,
  "userId": "uuid-string",
  "displayName": "string",
  "email": "string",
  "avatarUrl": "string",
  "bio": "string",
  "shopConfig": {
    "shopName": "string",
    "description": "string",
    "logoUrl": "string",
    "pickupAddressId": ObjectId
  },
  "createdAt": ISODate,
  "updatedAt": ISODate
}
*/

-- Collection: addresses (tafu-profile)
-- Index: { userId: 1 }
/*
{
  "_id": ObjectId,
  "userId": "uuid-string",
  "contactName": "string",
  "phone": "string",
  "provinceCode": "string",
  "districtCode": "string",
  "wardCode": "string",
  "streetLine": "string",
  "fullAddress": "string",
  "isDefault": boolean,
  "type": "HOME" | "OFFICE",
  "createdAt": ISODate,
  "updatedAt": ISODate
}
*/

-- Collection: categories (tafu-catalog)
-- Index: { slug: 1 } unique
-- Index: { parentId: 1 }
/*
{
  "_id": ObjectId,
  "name": "string",
  "slug": "string",
  "imageUrl": "string",
  "parentId": ObjectId | null,
  "attributeDefinitions": [
    {
      "key": "string",
      "type": "text" | "number" | "select",
      "isRequired": boolean,
      "options": ["string"]
    }
  ]
}
*/

-- Collection: brands (tafu-catalog)
-- Index: { slug: 1 } unique
/*
{
  "_id": ObjectId,
  "name": "string",
  "slug": "string",
  "logoUrl": "string",
  "status": "ACTIVE" | "INACTIVE"
}
*/

-- Collection: products (tafu-catalog)
-- Index: { slug: 1 } unique
-- Index: { categoryId: 1 }
-- Index: { brandId: 1 }
-- Index: { name: "text" }
-- Index: { status: 1, createdAt: -1 }
/*
{
  "_id": ObjectId,
  "name": "string",
  "slug": "string",
  "categoryId": ObjectId,
  "brandId": ObjectId,
  "thumbnail": "string",
  "images": ["string"],
  "videoUrl": "string",
  "description": "string",
  "specs": { ... },
  "variations": [
    {
      "sku": "string",
      "price": number,
      "stock": number,
      "imageUrl": "string",
      "attributes": { "color": "Red", "size": "M" },
      "tierIndex": [0, 1],
      "metadata": { "preOrder": true }
    }
  ],
  "metadata": { "isFlashSale": true, "campaignId": "string", "tags": ["hot"] },
  "status": "DRAFT" | "PUBLISHED",
  "createdAt": ISODate,
  "updatedAt": ISODate
}
*/

-- Collection: carts (tafu-cart - MongoDB backup)
/*
{
  "_id": "userId",  // userId as primary key
  "items": [
    {
      "skuId": "string",
      "name": "string",
      "price": number,
      "quantity": number,
      "thumbnail": "string",
      "selected": boolean,
      "addedAt": timestamp
    }
  ],
  "updatedAt": ISODate
}
*/

-- Collection: reviews (tafu-review)
-- Index: { productId: 1, createdAt: -1 }
-- Index: { userId: 1 }
-- Index: { orderId: 1, productId: 1 } unique
/*
{
  "_id": ObjectId,
  "userId": "uuid-string",
  "userName": "string",
  "userAvatar": "url",
  "productId": "string",
  "orderId": "uuid-string",
  "rating": 1-5,
  "content": "string",
  "images": ["url"],
  "isPurchased": boolean,
  "reply": {
    "content": "string",
    "repliedAt": ISODate
  },
  "status": "VISIBLE" | "HIDDEN",
  "createdAt": ISODate,
  "updatedAt": ISODate
}
*/

-- Collection: product_ratings (tafu-review - Materialized View)
-- Index: { _id: 1 } where _id = productId
/*
{
  "_id": "productId",
  "averageRating": 4.8,
  "totalReviews": 150,
  "starCounts": { "1": 2, "2": 0, "3": 5, "4": 20, "5": 123 },
  "updatedAt": ISODate
}
*/

-- Collection: notification_logs (tafu-notification)
-- Index: { userId: 1 }
-- Index: { status: 1 }
-- Index: { createdAt: -1 }
/*
{
  "_id": ObjectId,
  "userId": "uuid-string",
  "type": "EMAIL" | "PUSH" | "SMS",
  "status": "PENDING" | "SENT" | "FAILED",
  "recipient": "email or device token",
  "template": "order_confirmation",
  "payload": { ... },
  "error": "string",
  "createdAt": ISODate
}
*/

-- ============================================================================
-- ELASTICSEARCH INDEX (tafu-search service)
-- ============================================================================

/*
PUT /products_index
{
  "settings": {
    "number_of_shards": 3,
    "number_of_replicas": 1,
    "analysis": {
      "analyzer": {
        "vietnamese_analyzer": {
          "type": "custom",
          "tokenizer": "standard",
          "filter": ["lowercase", "asciifolding"]
        }
      }
    }
  },
  "mappings": {
    "properties": {
      "id": { "type": "keyword" },
      "name": { 
        "type": "text", 
        "analyzer": "vietnamese_analyzer",
        "fields": { "keyword": { "type": "keyword" } }
      },
      "slug": { "type": "keyword" },
      "description": { "type": "text", "analyzer": "vietnamese_analyzer" },
      "categoryId": { "type": "keyword" },
      "brandId": { "type": "keyword" },
      "price": { "type": "double" },
      "thumbnail": { "type": "keyword", "index": false },
      "status": { "type": "keyword" },
      "specs": { "type": "object", "dynamic": true },
      "metadata": { "type": "object", "dynamic": true },
      "createdAt": { "type": "date" },
      "updatedAt": { "type": "date" }
    }
  }
}
*/

-- ============================================================================
-- REDIS DATA STRUCTURES
-- ============================================================================

/*
=== tafu-auth (Identity Service) ===
Key Pattern: identity:user:{userId}:permissions
Type: Set
Value: ["product:create", "order:read_own", ...]
TTL: 3600s (1 hour)

Key Pattern: identity:blacklist:{jti}
Type: String
Value: "1"
TTL: Token remaining TTL

=== tafu-cart ===
Key Pattern: cart:{userId}
Type: Hash
Fields: {skuId} -> JSON { skuId, name, price, quantity, thumbnail, selected, addedAt }
TTL: 30 days (refresh on interaction)

=== tafu-inventory ===
Key Pattern: inventory:{skuId}
Type: Hash
Fields: { total: int, reserved: int }
TTL: None (synced with DB)

=== tafu-campaign ===
Key Pattern: voucher:{code}:stock
Type: String (Integer)
TTL: Campaign end time

Key Pattern: voucher:{code}:users
Type: Set (User IDs who claimed)
TTL: Campaign end time

=== tafu-search ===
Key Pattern: search:{hash(params)}
Type: String (JSON of results)
TTL: 120s (2 minutes)

=== tafu-logistic ===
Key Pattern: fee:{provider}:{from}:{to}:{weight}
Type: String (fee amount)
TTL: 3600s (1 hour)

=== tafu-review ===
Key Pattern: rating:{productId}
Type: String (JSON { average, total, breakdown })
TTL: 1800s (30 minutes)
*/
