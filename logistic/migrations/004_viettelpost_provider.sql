-- Migration: Add ViettelPost provider support
-- Version: 004
-- Description: Add support for ViettelPost shipping provider

-- Update table comment to include ViettelPost
COMMENT ON COLUMN shipping_orders.provider IS 'Shipping provider: GHN, GHTK, VIETTELPOST, MOCK';

-- Add index on tracking_code if not exists (for cancel operations)
CREATE INDEX IF NOT EXISTS idx_shipping_orders_tracking_code ON shipping_orders(tracking_code);

-- Add cancelled_at column for tracking cancellation time
ALTER TABLE shipping_orders
    ADD COLUMN IF NOT EXISTS cancelled_at TIMESTAMPTZ;

-- Add cancellation_reason column
ALTER TABLE shipping_orders
    ADD COLUMN IF NOT EXISTS cancellation_reason TEXT;

-- Create index for cancelled shipments
CREATE INDEX IF NOT EXISTS idx_shipping_orders_cancelled_at
    ON shipping_orders(cancelled_at)
    WHERE cancelled_at IS NOT NULL;

-- Add comments for new columns
COMMENT ON COLUMN shipping_orders.cancelled_at IS 'Timestamp when the shipment was cancelled';
COMMENT ON COLUMN shipping_orders.cancellation_reason IS 'Reason for shipment cancellation';
