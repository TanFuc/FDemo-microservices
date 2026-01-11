-- Migration: Create shipping_orders table
-- Version: 001
-- Description: Initial schema for logistics service

-- Enable UUID extension if not already enabled
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Create shipping_orders table
CREATE TABLE IF NOT EXISTS shipping_orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    internal_order_id UUID NOT NULL,
    provider VARCHAR(50) NOT NULL,
    tracking_code VARCHAR(100),
    carrier_status VARCHAR(100),
    system_status VARCHAR(50) NOT NULL DEFAULT 'PENDING',
    shipping_fee DECIMAL(10,2),
    cod_amount DECIMAL(10,2) DEFAULT 0,
    label_url TEXT,
    metadata JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create indexes for common queries
CREATE INDEX IF NOT EXISTS idx_shipping_orders_internal_order_id ON shipping_orders(internal_order_id);
CREATE INDEX IF NOT EXISTS idx_shipping_orders_tracking_code ON shipping_orders(tracking_code);
CREATE INDEX IF NOT EXISTS idx_shipping_orders_system_status ON shipping_orders(system_status);
CREATE INDEX IF NOT EXISTS idx_shipping_orders_provider ON shipping_orders(provider);
CREATE INDEX IF NOT EXISTS idx_shipping_orders_created_at ON shipping_orders(created_at);

-- Create function to auto-update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger for auto-updating updated_at
DROP TRIGGER IF EXISTS update_shipping_orders_updated_at ON shipping_orders;
CREATE TRIGGER update_shipping_orders_updated_at
    BEFORE UPDATE ON shipping_orders
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Add comments for documentation
COMMENT ON TABLE shipping_orders IS 'Stores shipping order records from various logistics providers';
COMMENT ON COLUMN shipping_orders.internal_order_id IS 'Reference to the order in tafu-order service';
COMMENT ON COLUMN shipping_orders.provider IS 'Shipping provider: GHN, GHTK, MOCK';
COMMENT ON COLUMN shipping_orders.tracking_code IS 'Waybill code from the provider';
COMMENT ON COLUMN shipping_orders.carrier_status IS 'Raw status string from provider';
COMMENT ON COLUMN shipping_orders.system_status IS 'Normalized status: PENDING, PICKING, SHIPPING, DELIVERED, RETURNED, CANCELLED';
COMMENT ON COLUMN shipping_orders.metadata IS 'Raw JSON response from provider for debugging';
