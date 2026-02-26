-- ============================================================================
-- TAFU-ORDER: Add return_requests table
-- Version: 002
-- ============================================================================

-- Enum for return status
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'return_status') THEN
        CREATE TYPE return_status AS ENUM (
            'PENDING', 'APPROVED', 'REJECTED',
            'AWAITING_PICKUP', 'IN_TRANSIT', 'RECEIVED',
            'INSPECTING', 'COMPLETED', 'CANCELLED'
        );
    END IF;
END$$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'return_type') THEN
        CREATE TYPE return_type AS ENUM ('REFUND', 'EXCHANGE');
    END IF;
END$$;

CREATE TABLE IF NOT EXISTS return_requests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id UUID NOT NULL,
    user_id UUID NOT NULL,
    return_number VARCHAR(50) UNIQUE NOT NULL,

    type VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    reason VARCHAR(50),

    return_items JSONB NOT NULL DEFAULT '[]',

    customer_reason TEXT,
    customer_note TEXT,
    evidence_urls TEXT,

    seller_note TEXT,
    admin_note TEXT,
    rejection_reason TEXT,

    return_shipping_method VARCHAR(50),
    return_tracking_number VARCHAR(100),
    return_carrier VARCHAR(50),
    return_shipping_fee DECIMAL(19,4) DEFAULT 0,
    return_shipping_paid_by VARCHAR(20),

    pickup_address JSONB,

    warehouse_id VARCHAR(100),
    received_at TIMESTAMPTZ,
    received_by VARCHAR(100),
    received_condition VARCHAR(50),

    inspection_note TEXT,
    inspected_at TIMESTAMPTZ,
    inspected_by VARCHAR(100),

    exchange_order_id UUID,
    exchange_product_ids TEXT,
    refund_id UUID,

    approved_by VARCHAR(100),
    approved_by_id VARCHAR(100),

    -- Stock restore tracking
    stock_restored BOOLEAN DEFAULT FALSE,
    stock_restored_at TIMESTAMPTZ,

    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    approved_at TIMESTAMPTZ,
    rejected_at TIMESTAMPTZ,
    shipped_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,

    CONSTRAINT fk_return_requests_order FOREIGN KEY (order_id)
        REFERENCES orders(id) ON DELETE CASCADE
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_return_requests_order_id ON return_requests(order_id);
CREATE INDEX IF NOT EXISTS idx_return_requests_user_id ON return_requests(user_id);
CREATE INDEX IF NOT EXISTS idx_return_requests_status ON return_requests(status);
CREATE INDEX IF NOT EXISTS idx_return_requests_return_number ON return_requests(return_number);
CREATE INDEX IF NOT EXISTS idx_return_requests_stock_restore
    ON return_requests(id)
    WHERE status = 'COMPLETED' AND stock_restored = FALSE;

-- Update trigger (reuse existing function if available)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_proc WHERE proname = 'update_updated_at_column') THEN
        CREATE FUNCTION update_updated_at_column()
        RETURNS TRIGGER AS $trigger$
        BEGIN
            NEW.updated_at = NOW();
            RETURN NEW;
        END;
        $trigger$ LANGUAGE plpgsql;
    END IF;
END$$;

CREATE TRIGGER update_return_requests_updated_at
    BEFORE UPDATE ON return_requests
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE return_requests IS 'Customer return/exchange requests';
COMMENT ON COLUMN return_requests.stock_restored IS 'Whether inventory was restored after return completion';
