-- Payment Methods and Payouts Schema
-- Migration: 002_payment_methods_payouts.sql
-- Enterprise-grade payment method storage and seller payout management

-- Payment Method Type enum
CREATE TYPE payment_method_type AS ENUM ('CARD', 'BANK_ACCOUNT', 'E_WALLET');

-- Payment Method Status enum
CREATE TYPE payment_method_status AS ENUM ('ACTIVE', 'INACTIVE', 'EXPIRED');

-- Payout Status enum
CREATE TYPE payout_status AS ENUM ('PENDING', 'APPROVED', 'PROCESSING', 'COMPLETED', 'FAILED', 'CANCELLED', 'ON_HOLD');

-- Payout Method enum
CREATE TYPE payout_method AS ENUM ('BANK_TRANSFER', 'E_WALLET');

-- =====================================================
-- Payment Methods Table (Saved cards/payment methods)
-- =====================================================
CREATE TABLE payment_methods (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    type payment_method_type NOT NULL,
    provider payment_provider NOT NULL,
    provider_token_id VARCHAR(255) NOT NULL,

    -- Card details (PCI-compliant: only store masked data)
    card_brand VARCHAR(50),
    card_last4 CHAR(4),
    card_exp_month SMALLINT CHECK (card_exp_month >= 1 AND card_exp_month <= 12),
    card_exp_year SMALLINT CHECK (card_exp_year >= 2024),
    cardholder_name VARCHAR(255),

    -- Bank account details (masked)
    bank_name VARCHAR(255),
    bank_account_last4 CHAR(4),
    account_holder VARCHAR(255),

    -- E-wallet details
    wallet_type VARCHAR(50),
    wallet_id VARCHAR(255),

    -- Preferences
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    label VARCHAR(100),

    -- Status
    status payment_method_status NOT NULL DEFAULT 'ACTIVE',
    verified_at TIMESTAMP WITH TIME ZONE,

    -- Billing address (JSONB for flexibility)
    billing_address JSONB DEFAULT '{}',

    -- Metadata
    metadata JSONB DEFAULT '{}',

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,

    -- Constraints
    CONSTRAINT check_card_details CHECK (
        type != 'CARD' OR (card_brand IS NOT NULL AND card_last4 IS NOT NULL)
    ),
    CONSTRAINT check_bank_details CHECK (
        type != 'BANK_ACCOUNT' OR (bank_name IS NOT NULL AND bank_account_last4 IS NOT NULL)
    )
);

-- Indexes for payment_methods
CREATE INDEX idx_payment_methods_user_id ON payment_methods(user_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_payment_methods_user_default ON payment_methods(user_id, is_default) WHERE deleted_at IS NULL;
CREATE INDEX idx_payment_methods_provider_token ON payment_methods(provider_token_id);
CREATE INDEX idx_payment_methods_status ON payment_methods(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_payment_methods_type ON payment_methods(type) WHERE deleted_at IS NULL;

-- Unique constraint: only one default per user per type
CREATE UNIQUE INDEX idx_payment_methods_user_default_unique
    ON payment_methods(user_id, type)
    WHERE is_default = TRUE AND deleted_at IS NULL;

-- =====================================================
-- Payout Requests Table (Seller payouts)
-- =====================================================
CREATE TABLE payout_requests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    seller_id UUID NOT NULL,
    shop_id UUID NOT NULL,

    -- Amount details
    requested_amount DECIMAL(19, 4) NOT NULL CHECK (requested_amount > 0),
    processing_fee DECIMAL(19, 4) NOT NULL DEFAULT 0,
    tax_amount DECIMAL(19, 4) NOT NULL DEFAULT 0,
    net_amount DECIMAL(19, 4) NOT NULL,
    currency currency_code NOT NULL,

    -- Payout method
    method payout_method NOT NULL,
    payment_method_id UUID REFERENCES payment_methods(id),

    -- Bank details (snapshot at request time for audit trail)
    bank_name VARCHAR(255),
    bank_account_no VARCHAR(50),
    account_holder VARCHAR(255),
    bank_branch VARCHAR(255),
    swift_code VARCHAR(11),

    -- E-wallet details
    wallet_type VARCHAR(50),
    wallet_id VARCHAR(255),

    -- Status and tracking
    status payout_status NOT NULL DEFAULT 'PENDING',
    provider_ref VARCHAR(255),
    transaction_id VARCHAR(255),

    -- Period covered
    period_start TIMESTAMP WITH TIME ZONE,
    period_end TIMESTAMP WITH TIME ZONE,

    -- Approval workflow
    requested_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    approved_by UUID,
    approved_at TIMESTAMP WITH TIME ZONE,
    processed_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,

    -- Failure handling
    failure_reason TEXT,
    retry_count SMALLINT NOT NULL DEFAULT 0,
    last_retry_at TIMESTAMP WITH TIME ZONE,

    -- Notes
    seller_note TEXT,
    admin_note TEXT,

    -- Metadata and audit
    metadata JSONB DEFAULT '{}',
    ip_address INET,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT check_net_amount CHECK (net_amount <= requested_amount),
    CONSTRAINT check_period CHECK (period_end IS NULL OR period_start IS NULL OR period_end >= period_start)
);

-- Indexes for payout_requests
CREATE INDEX idx_payout_requests_seller_id ON payout_requests(seller_id);
CREATE INDEX idx_payout_requests_shop_id ON payout_requests(shop_id);
CREATE INDEX idx_payout_requests_status ON payout_requests(status);
CREATE INDEX idx_payout_requests_created_at ON payout_requests(created_at);
CREATE INDEX idx_payout_requests_requested_at ON payout_requests(requested_at);

-- Composite index for seller payout history
CREATE INDEX idx_payout_requests_seller_status_created
    ON payout_requests(seller_id, status, created_at DESC);

-- Index for pending payouts processing
CREATE INDEX idx_payout_requests_pending_processing
    ON payout_requests(status, requested_at)
    WHERE status IN ('PENDING', 'APPROVED', 'PROCESSING');

-- Index for failed payouts that can be retried
CREATE INDEX idx_payout_requests_failed_retry
    ON payout_requests(status, retry_count, last_retry_at)
    WHERE status = 'FAILED';

-- =====================================================
-- Payout Schedules Table (Automatic payout configuration)
-- =====================================================
CREATE TABLE payout_schedules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    shop_id UUID NOT NULL UNIQUE,

    -- Schedule configuration
    frequency VARCHAR(20) NOT NULL CHECK (frequency IN ('DAILY', 'WEEKLY', 'BIWEEKLY', 'MONTHLY')),
    day_of_week SMALLINT CHECK (day_of_week >= 0 AND day_of_week <= 6),
    day_of_month SMALLINT CHECK (day_of_month >= 1 AND day_of_month <= 28),

    -- Payout settings
    min_amount DECIMAL(19, 4) NOT NULL DEFAULT 0,
    is_automatic BOOLEAN NOT NULL DEFAULT FALSE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    -- Next scheduled run
    next_run_at TIMESTAMP WITH TIME ZONE NOT NULL,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT check_weekly_day CHECK (
        frequency != 'WEEKLY' AND frequency != 'BIWEEKLY' OR day_of_week IS NOT NULL
    ),
    CONSTRAINT check_monthly_day CHECK (
        frequency != 'MONTHLY' OR day_of_month IS NOT NULL
    )
);

-- Index for active schedules due for processing
CREATE INDEX idx_payout_schedules_next_run
    ON payout_schedules(next_run_at)
    WHERE is_active = TRUE AND is_automatic = TRUE;

-- =====================================================
-- Payout Audit Log Table
-- =====================================================
CREATE TABLE payout_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    payout_id UUID NOT NULL REFERENCES payout_requests(id),
    action VARCHAR(50) NOT NULL,
    old_status payout_status,
    new_status payout_status,
    performed_by UUID,
    reason TEXT,
    metadata JSONB DEFAULT '{}',
    ip_address INET,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Index for payout audit trail
CREATE INDEX idx_payout_logs_payout_id ON payout_logs(payout_id);
CREATE INDEX idx_payout_logs_created_at ON payout_logs(created_at);

-- =====================================================
-- Triggers
-- =====================================================

-- Auto-update updated_at for payment_methods
CREATE TRIGGER update_payment_methods_updated_at
    BEFORE UPDATE ON payment_methods
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Auto-update updated_at for payout_requests
CREATE TRIGGER update_payout_requests_updated_at
    BEFORE UPDATE ON payout_requests
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Auto-update updated_at for payout_schedules
CREATE TRIGGER update_payout_schedules_updated_at
    BEFORE UPDATE ON payout_schedules
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- =====================================================
-- Comments
-- =====================================================
COMMENT ON TABLE payment_methods IS 'Stores saved payment methods for users (cards, bank accounts, e-wallets)';
COMMENT ON TABLE payout_requests IS 'Tracks seller payout requests and their processing status';
COMMENT ON TABLE payout_schedules IS 'Automatic payout schedule configuration per shop';
COMMENT ON TABLE payout_logs IS 'Audit trail for payout request status changes';
