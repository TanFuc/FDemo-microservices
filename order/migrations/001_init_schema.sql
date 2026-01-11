-- ============================================================================
-- TAFU-ORDER Order Service Database Migration
-- Version: 001
-- Date: 2026-01-07
-- Database: PostgreSQL 15+
-- ORM: GORM (Go)
-- ============================================================================

-- Enable extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ===================
-- Enum: order_status
-- ===================
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'order_status') THEN
        CREATE TYPE order_status AS ENUM ('PENDING', 'PAID', 'SHIPPED', 'COMPLETED', 'CANCELLED');
    END IF;
END$$;

-- ===================
-- Table: orders (Aggregate Root)
-- ===================
CREATE TABLE IF NOT EXISTS orders (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    
    -- Money fields with strict decimal precision
    total_amount DECIMAL(19, 4) NOT NULL DEFAULT 0 CHECK (total_amount >= 0),
    shipping_fee DECIMAL(19, 4) NOT NULL DEFAULT 0 CHECK (shipping_fee >= 0),
    discount_amount DECIMAL(19, 4) NOT NULL DEFAULT 0 CHECK (discount_amount >= 0),
    final_amount DECIMAL(19, 4) NOT NULL DEFAULT 0 CHECK (final_amount >= 0),
    
    status order_status NOT NULL DEFAULT 'PENDING',
    payment_method VARCHAR(50) NOT NULL,
    
    -- Snapshot Data - address at order time (immutable)
    shipping_address JSONB NOT NULL DEFAULT '{}',
    
    -- Voucher tracking
    voucher_code VARCHAR(50),
    
    -- Notes
    customer_note TEXT,
    internal_note TEXT,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders(user_id);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);
CREATE INDEX IF NOT EXISTS idx_orders_created_at ON orders(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_orders_payment_method ON orders(payment_method);
CREATE INDEX IF NOT EXISTS idx_orders_voucher_code ON orders(voucher_code);

-- Partial index for pending orders (used by payment timeout job)
CREATE INDEX IF NOT EXISTS idx_orders_pending_created 
    ON orders(created_at) 
    WHERE status = 'PENDING';

-- ===================
-- Table: order_items (Product Snapshot)
-- ===================
CREATE TABLE IF NOT EXISTS order_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id UUID NOT NULL,
    
    -- Product Snapshot (immutable - never lookup catalog)
    product_id VARCHAR(100) NOT NULL,
    sku_id VARCHAR(100) NOT NULL,
    product_name VARCHAR(255) NOT NULL,
    sku_code VARCHAR(100) NOT NULL,
    thumbnail VARCHAR(500),
    
    -- Variant attributes at purchase time
    attributes JSONB DEFAULT '{}',
    
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    unit_price DECIMAL(19, 4) NOT NULL CHECK (unit_price >= 0),
    sub_total DECIMAL(19, 4) NOT NULL CHECK (sub_total >= 0),
    
    -- Inventory reservation tracking
    reservation_id VARCHAR(100),
    
    CONSTRAINT fk_order_items_order FOREIGN KEY (order_id) 
        REFERENCES orders(id) ON DELETE CASCADE
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_order_items_order_id ON order_items(order_id);
CREATE INDEX IF NOT EXISTS idx_order_items_product_id ON order_items(product_id);
CREATE INDEX IF NOT EXISTS idx_order_items_sku_id ON order_items(sku_id);

-- ===================
-- Table: order_status_history
-- ===================
CREATE TABLE IF NOT EXISTS order_status_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id UUID NOT NULL,
    from_status order_status,
    to_status order_status NOT NULL,
    changed_by VARCHAR(100),  -- user_id or 'system'
    reason TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT fk_order_status_history_order FOREIGN KEY (order_id) 
        REFERENCES orders(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_order_status_history_order_id ON order_status_history(order_id);
CREATE INDEX IF NOT EXISTS idx_order_status_history_created_at ON order_status_history(created_at);

-- ===================
-- Table: order_events (Outbox Pattern)
-- ===================
CREATE TABLE IF NOT EXISTS order_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    aggregate_id UUID NOT NULL,  -- order_id
    event_type VARCHAR(100) NOT NULL,  -- order.created, order.paid, etc.
    payload JSONB NOT NULL,
    status VARCHAR(20) DEFAULT 'PENDING',  -- PENDING, PUBLISHED, FAILED
    retry_count INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    published_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_order_events_status ON order_events(status);
CREATE INDEX IF NOT EXISTS idx_order_events_created_at ON order_events(created_at);
CREATE INDEX IF NOT EXISTS idx_order_events_pending 
    ON order_events(created_at) 
    WHERE status = 'PENDING';

-- ===================
-- Triggers
-- ===================
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

-- Trigger to log status changes
CREATE OR REPLACE FUNCTION log_order_status_change()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.status IS DISTINCT FROM NEW.status THEN
        INSERT INTO order_status_history (order_id, from_status, to_status, changed_by)
        VALUES (NEW.id, OLD.status, NEW.status, 'system');
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_log_order_status
    AFTER UPDATE ON orders
    FOR EACH ROW
    EXECUTE FUNCTION log_order_status_change();

-- ===================
-- Comments
-- ===================
COMMENT ON TABLE orders IS 'Order aggregate root - central transaction entity';
COMMENT ON COLUMN orders.shipping_address IS 'JSONB snapshot - immutable after creation';
COMMENT ON COLUMN orders.final_amount IS 'total_amount + shipping_fee - discount_amount';

COMMENT ON TABLE order_items IS 'Product snapshot at purchase time';
COMMENT ON COLUMN order_items.product_name IS 'Snapshot - never lookup from catalog';
COMMENT ON COLUMN order_items.unit_price IS 'Price at purchase time - immutable';

COMMENT ON TABLE order_events IS 'Outbox pattern for reliable event publishing';
