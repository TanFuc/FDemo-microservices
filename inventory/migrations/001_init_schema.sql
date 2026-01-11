-- ============================================================================
-- TAFU-INVENTORY Inventory Service Database Migration
-- Version: 001
-- Date: 2026-01-07
-- Database: PostgreSQL 15+
-- ORM: GORM (Go)
-- ============================================================================

-- Enable extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ===================
-- Enum: reservation_status
-- ===================
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'reservation_status') THEN
        CREATE TYPE reservation_status AS ENUM ('PENDING', 'CONFIRMED', 'CANCELLED');
    END IF;
END$$;

-- ===================
-- Table: inventory_items (The Stock Ledger)
-- ===================
CREATE TABLE IF NOT EXISTS inventory_items (
    sku_id VARCHAR(100) PRIMARY KEY,
    product_id VARCHAR(100),  -- Reference to Catalog product
    
    -- Stock tracking
    total_stock INTEGER NOT NULL DEFAULT 0 CHECK (total_stock >= 0),
    reserved_stock INTEGER NOT NULL DEFAULT 0 CHECK (reserved_stock >= 0),
    
    -- Warehouse info (optional)
    warehouse_id VARCHAR(100),
    location_bin VARCHAR(50),  -- Physical bin location
    
    -- Low stock alert threshold
    low_stock_threshold INTEGER DEFAULT 10,
    
    -- Tracking
    last_restock_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Constraint: reserved cannot exceed total
    CONSTRAINT chk_stock_balance CHECK (total_stock >= reserved_stock)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_inventory_items_product_id ON inventory_items(product_id);
CREATE INDEX IF NOT EXISTS idx_inventory_items_warehouse_id ON inventory_items(warehouse_id);
CREATE INDEX IF NOT EXISTS idx_inventory_items_low_stock 
    ON inventory_items(sku_id) 
    WHERE (total_stock - reserved_stock) <= low_stock_threshold;

-- ===================
-- Table: stock_reservations (Two-Phase Reservation Log)
-- ===================
CREATE TABLE IF NOT EXISTS stock_reservations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id VARCHAR(100) NOT NULL,
    sku_id VARCHAR(100) NOT NULL,
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    status reservation_status NOT NULL DEFAULT 'PENDING',
    expires_at TIMESTAMPTZ NOT NULL,
    
    -- For tracking
    confirmed_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    cancel_reason VARCHAR(255),
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT fk_stock_reservations_sku FOREIGN KEY (sku_id) 
        REFERENCES inventory_items(sku_id) ON DELETE CASCADE
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_stock_reservations_order_id ON stock_reservations(order_id);
CREATE INDEX IF NOT EXISTS idx_stock_reservations_sku_id ON stock_reservations(sku_id);
CREATE INDEX IF NOT EXISTS idx_stock_reservations_status ON stock_reservations(status);
CREATE INDEX IF NOT EXISTS idx_stock_reservations_expires_at ON stock_reservations(expires_at);

-- Partial index for pending reservations (used by cleanup job)
CREATE INDEX IF NOT EXISTS idx_stock_reservations_pending_expires 
    ON stock_reservations(expires_at) 
    WHERE status = 'PENDING';

-- Composite index for order lookup
CREATE INDEX IF NOT EXISTS idx_stock_reservations_order_status 
    ON stock_reservations(order_id, status);

-- ===================
-- Table: stock_movements (Audit Trail)
-- ===================
CREATE TABLE IF NOT EXISTS stock_movements (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    sku_id VARCHAR(100) NOT NULL,
    movement_type VARCHAR(50) NOT NULL,  -- RESTOCK, SALE, RETURN, ADJUSTMENT, RESERVATION, RELEASE
    quantity INTEGER NOT NULL,  -- Positive = in, Negative = out
    
    -- Reference
    reference_id VARCHAR(100),  -- order_id, restock_id, etc.
    reference_type VARCHAR(50),  -- ORDER, RESTOCK, RETURN, MANUAL
    
    -- Balance after movement
    stock_before INTEGER NOT NULL,
    stock_after INTEGER NOT NULL,
    
    -- Who did it
    performed_by VARCHAR(100),
    note TEXT,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT fk_stock_movements_sku FOREIGN KEY (sku_id) 
        REFERENCES inventory_items(sku_id) ON DELETE CASCADE
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_stock_movements_sku_id ON stock_movements(sku_id);
CREATE INDEX IF NOT EXISTS idx_stock_movements_reference ON stock_movements(reference_id, reference_type);
CREATE INDEX IF NOT EXISTS idx_stock_movements_created_at ON stock_movements(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_stock_movements_type ON stock_movements(movement_type);

-- ===================
-- Table: warehouses (Optional - for multi-warehouse)
-- ===================
CREATE TABLE IF NOT EXISTS warehouses (
    id VARCHAR(100) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    address TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

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

CREATE TRIGGER update_inventory_items_updated_at
    BEFORE UPDATE ON inventory_items
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_stock_reservations_updated_at
    BEFORE UPDATE ON stock_reservations
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Trigger to log stock movements automatically
CREATE OR REPLACE FUNCTION log_stock_movement()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.total_stock != NEW.total_stock OR OLD.reserved_stock != NEW.reserved_stock THEN
        INSERT INTO stock_movements (
            sku_id, movement_type, quantity, 
            stock_before, stock_after, 
            performed_by
        )
        VALUES (
            NEW.sku_id, 
            CASE 
                WHEN NEW.total_stock > OLD.total_stock THEN 'RESTOCK'
                WHEN NEW.total_stock < OLD.total_stock AND NEW.reserved_stock < OLD.reserved_stock THEN 'SALE'
                WHEN NEW.reserved_stock > OLD.reserved_stock THEN 'RESERVATION'
                WHEN NEW.reserved_stock < OLD.reserved_stock THEN 'RELEASE'
                ELSE 'ADJUSTMENT'
            END,
            COALESCE(NEW.total_stock - OLD.total_stock, 0),
            OLD.total_stock - OLD.reserved_stock,
            NEW.total_stock - NEW.reserved_stock,
            'system'
        );
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_log_stock_movement
    AFTER UPDATE ON inventory_items
    FOR EACH ROW
    EXECUTE FUNCTION log_stock_movement();

-- ===================
-- Comments
-- ===================
COMMENT ON TABLE inventory_items IS 'Stock ledger - source of truth for available stock';
COMMENT ON COLUMN inventory_items.total_stock IS 'Physical stock in warehouse';
COMMENT ON COLUMN inventory_items.reserved_stock IS 'Stock held by pending orders';

COMMENT ON TABLE stock_reservations IS 'Two-phase reservation log for preventing overselling';
COMMENT ON COLUMN stock_reservations.expires_at IS 'Auto-release reservation if not confirmed by this time';

COMMENT ON TABLE stock_movements IS 'Audit trail for all stock changes';
