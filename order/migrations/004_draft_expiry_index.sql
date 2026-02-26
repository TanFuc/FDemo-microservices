-- ============================================================================
-- Order Service — Migration 004
-- Date: 2026-02-26
-- Description: Add partial index to support efficient draft order cleanup
-- ============================================================================

-- Partial index: only indexes rows that the cleanup worker will query.
-- Significantly reduces index size and speeds up the 3AM deletion query.
CREATE INDEX IF NOT EXISTS idx_orders_draft_created_at
    ON orders(created_at)
    WHERE status = 'DRAFT' AND deleted_at IS NULL;

COMMENT ON INDEX idx_orders_draft_created_at IS
    'Supports the nightly 3AM cleanup worker for expired DRAFT orders.';
