-- Migration: Add wallet top-up support columns
-- This migration adds columns to track wallet top-up payments

ALTER TABLE payment_transactions
    ADD COLUMN IF NOT EXISTS is_wallet_topup BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS wallet_tx_id    VARCHAR(255);

CREATE INDEX IF NOT EXISTS idx_payment_tx_wallet_topup
    ON payment_transactions(is_wallet_topup) WHERE is_wallet_topup = TRUE;

COMMENT ON COLUMN payment_transactions.is_wallet_topup IS
    'Indicates if this payment is a wallet top-up transaction';
COMMENT ON COLUMN payment_transactions.wallet_tx_id IS
    'The wallet service transaction ID for wallet top-up payments';
