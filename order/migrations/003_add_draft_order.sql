-- ============================================================================
-- TAFU-ORDER Order Service Database Migration
-- Version: 003
-- Date: 2026-02-25
-- Description: Add DRAFT status for cart checkout flow
-- ============================================================================

-- ===================
-- Add DRAFT to order_status enum
-- ===================
-- PostgreSQL requires ALTER TYPE to add new enum values
ALTER TYPE order_status ADD VALUE IF NOT EXISTS 'DRAFT' BEFORE 'PENDING';

-- ===================
-- Add confirmed_at column to orders table
-- Tracks when a draft order is confirmed (stock reserved)
-- ===================
ALTER TABLE orders
ADD COLUMN IF NOT EXISTS confirmed_at TIMESTAMPTZ;

-- ===================
-- Add version column for optimistic locking
-- ===================
ALTER TABLE orders
ADD COLUMN IF NOT EXISTS version INTEGER NOT NULL DEFAULT 1;

-- ===================
-- Add voucher tracking fields
-- ===================
ALTER TABLE orders
ADD COLUMN IF NOT EXISTS voucher_id VARCHAR(100);

ALTER TABLE orders
ADD COLUMN IF NOT EXISTS campaign_id VARCHAR(100);

ALTER TABLE orders
ADD COLUMN IF NOT EXISTS voucher_discount DECIMAL(19, 4) NOT NULL DEFAULT 0;

ALTER TABLE orders
ADD COLUMN IF NOT EXISTS sub_total DECIMAL(19, 4) NOT NULL DEFAULT 0;

-- ===================
-- Index for draft orders by user
-- ===================
CREATE INDEX IF NOT EXISTS idx_orders_user_draft
    ON orders(user_id)
    WHERE status = 'DRAFT';

-- ===================
-- Index for voucher lookups
-- ===================
CREATE INDEX IF NOT EXISTS idx_orders_voucher_id ON orders(voucher_id);
CREATE INDEX IF NOT EXISTS idx_orders_campaign_id ON orders(campaign_id);

-- ===================
-- Comments
-- ===================
COMMENT ON COLUMN orders.confirmed_at IS 'When draft order was confirmed and stock reserved';
COMMENT ON COLUMN orders.version IS 'Optimistic locking version for concurrent updates';
COMMENT ON COLUMN orders.voucher_id IS 'Reference to campaign voucher if applied';
COMMENT ON COLUMN orders.campaign_id IS 'Reference to campaign if voucher applied';
COMMENT ON COLUMN orders.voucher_discount IS 'Discount amount from voucher';
COMMENT ON COLUMN orders.sub_total IS 'Sum of item subtotals before fees and discounts';
