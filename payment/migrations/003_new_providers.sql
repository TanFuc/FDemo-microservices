-- Migration: Add VNPay and ZaloPay providers with refund support
-- Version: 003
-- Description: Add VNPay and ZaloPay payment providers and refund tracking columns

-- Add new payment providers to enum
-- Note: PostgreSQL requires special handling for adding values to enums
DO $$
BEGIN
    -- Add VNPAY if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM pg_enum WHERE enumlabel = 'VNPAY' AND enumtypid = 'payment_provider'::regtype) THEN
        ALTER TYPE payment_provider ADD VALUE 'VNPAY';
    END IF;
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
    -- Add ZALOPAY if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM pg_enum WHERE enumlabel = 'ZALOPAY' AND enumtypid = 'payment_provider'::regtype) THEN
        ALTER TYPE payment_provider ADD VALUE 'ZALOPAY';
    END IF;
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

-- Add refund tracking columns to payment_transactions
ALTER TABLE payment_transactions
    ADD COLUMN IF NOT EXISTS refund_provider_id VARCHAR(255);

ALTER TABLE payment_transactions
    ADD COLUMN IF NOT EXISTS refunded_amount DECIMAL(19, 4) DEFAULT 0;

ALTER TABLE payment_transactions
    ADD COLUMN IF NOT EXISTS refunded_at TIMESTAMPTZ;

ALTER TABLE payment_transactions
    ADD COLUMN IF NOT EXISTS refund_reason TEXT;

-- Create index for refunded transactions
CREATE INDEX IF NOT EXISTS idx_payment_transactions_refunded_at
    ON payment_transactions(refunded_at)
    WHERE refunded_at IS NOT NULL;

-- Add comments for new columns
COMMENT ON COLUMN payment_transactions.refund_provider_id IS 'Refund transaction ID from the payment provider';
COMMENT ON COLUMN payment_transactions.refunded_amount IS 'Amount that was refunded (supports partial refunds)';
COMMENT ON COLUMN payment_transactions.refunded_at IS 'Timestamp when the refund was processed';
COMMENT ON COLUMN payment_transactions.refund_reason IS 'Reason for the refund';
