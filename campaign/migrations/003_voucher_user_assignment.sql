-- ============================================================================
-- CAMPAIGN: Add voucher user assignment support
-- Version: 003
-- ============================================================================

-- Add assign_type column to vouchers table
ALTER TABLE vouchers
    ADD COLUMN IF NOT EXISTS assign_type VARCHAR(20) NOT NULL DEFAULT 'ALL'
        CHECK (assign_type IN ('ALL', 'SPECIFIC'));

-- Add assigned_user_ids column (JSONB array of user UUIDs)
ALTER TABLE vouchers
    ADD COLUMN IF NOT EXISTS assigned_user_ids JSONB NOT NULL DEFAULT '[]';

-- Index for fast filtering by assign_type
CREATE INDEX IF NOT EXISTS idx_vouchers_assign_type ON vouchers(assign_type);

-- GIN index for efficient user ID lookups in the JSONB array
CREATE INDEX IF NOT EXISTS idx_vouchers_assigned_users ON vouchers USING GIN(assigned_user_ids);

-- Composite index for checking user eligibility
CREATE INDEX IF NOT EXISTS idx_vouchers_code_assign_type ON vouchers(code, assign_type);

COMMENT ON COLUMN vouchers.assign_type IS 'ALL = any user can use, SPECIFIC = only assigned users';
COMMENT ON COLUMN vouchers.assigned_user_ids IS 'Array of user UUIDs who can use this voucher (when assign_type = SPECIFIC)';
