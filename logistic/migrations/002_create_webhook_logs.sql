-- Migration: Create webhook_logs table
-- Version: 002
-- Description: Store webhook history for audit trail and debugging

CREATE TABLE IF NOT EXISTS webhook_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shipping_order_id UUID REFERENCES shipping_orders(id),
    provider VARCHAR(50) NOT NULL,
    tracking_code VARCHAR(100),
    carrier_status VARCHAR(100),
    system_status VARCHAR(50),
    raw_payload JSONB NOT NULL,
    http_status INT,
    error_message TEXT,
    processed_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create indexes for common queries
CREATE INDEX IF NOT EXISTS idx_webhook_logs_shipping_order_id ON webhook_logs(shipping_order_id);
CREATE INDEX IF NOT EXISTS idx_webhook_logs_tracking_code ON webhook_logs(tracking_code);
CREATE INDEX IF NOT EXISTS idx_webhook_logs_provider ON webhook_logs(provider);
CREATE INDEX IF NOT EXISTS idx_webhook_logs_created_at ON webhook_logs(created_at);

-- Add comments for documentation
COMMENT ON TABLE webhook_logs IS 'Stores all incoming webhook requests for audit trail';
COMMENT ON COLUMN webhook_logs.raw_payload IS 'Original JSON payload from provider webhook';
COMMENT ON COLUMN webhook_logs.http_status IS 'HTTP status code returned to provider';
COMMENT ON COLUMN webhook_logs.error_message IS 'Error message if webhook processing failed';
