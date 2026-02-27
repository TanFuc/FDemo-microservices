-- Wallet Service Database Schema
-- Migration: 001_init.sql

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- =====================================================
-- ENUM Types
-- =====================================================
CREATE TYPE wallet_status    AS ENUM ('ACTIVE', 'SUSPENDED', 'FROZEN');
CREATE TYPE wallet_tx_type   AS ENUM ('TOP_UP', 'PAYMENT', 'REFUND', 'ADJUSTMENT');
CREATE TYPE wallet_tx_status AS ENUM ('PENDING', 'COMPLETED', 'FAILED', 'REVERSED');

-- =====================================================
-- wallets — one row per user (enforced by UNIQUE)
-- =====================================================
CREATE TABLE wallets (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID NOT NULL UNIQUE,
    balance     DECIMAL(19,4) NOT NULL DEFAULT 0 CHECK (balance >= 0),
    currency    VARCHAR(3) NOT NULL DEFAULT 'VND',
    status      wallet_status NOT NULL DEFAULT 'ACTIVE',
    version     INT NOT NULL DEFAULT 1,           -- optimistic locking
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_wallets_user_id ON wallets(user_id);

-- =====================================================
-- wallet_transactions — immutable ledger
-- =====================================================
CREATE TABLE wallet_transactions (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    wallet_id       UUID NOT NULL REFERENCES wallets(id),
    user_id         UUID NOT NULL,
    type            wallet_tx_type NOT NULL,
    status          wallet_tx_status NOT NULL DEFAULT 'PENDING',
    amount          DECIMAL(19,4) NOT NULL CHECK (amount > 0),
    balance_before  DECIMAL(19,4) NOT NULL,
    balance_after   DECIMAL(19,4) NOT NULL,
    currency        VARCHAR(3) NOT NULL DEFAULT 'VND',
    reference_id    VARCHAR(255),
    reference_type  VARCHAR(50),
    description     TEXT,
    payment_tx_id   UUID,                   -- links to payment service transaction
    idempotency_key VARCHAR(512),
    metadata        JSONB DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_wallet_tx_wallet_id    ON wallet_transactions(wallet_id);
CREATE INDEX idx_wallet_tx_user_id      ON wallet_transactions(user_id);
CREATE INDEX idx_wallet_tx_status       ON wallet_transactions(status);
CREATE INDEX idx_wallet_tx_reference    ON wallet_transactions(reference_id, reference_type);
CREATE INDEX idx_wallet_tx_payment_tx   ON wallet_transactions(payment_tx_id) WHERE payment_tx_id IS NOT NULL;
CREATE INDEX idx_wallet_tx_created_at   ON wallet_transactions(created_at DESC);
CREATE UNIQUE INDEX idx_wallet_tx_idempotency ON wallet_transactions(idempotency_key)
    WHERE idempotency_key IS NOT NULL;

-- =====================================================
-- outbox_events — Transactional Outbox Pattern
-- Guarantees at-least-once NATS delivery even when NATS is temporarily down.
-- =====================================================
CREATE TABLE outbox_events (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    aggregate_id VARCHAR(255) NOT NULL,
    event_type   VARCHAR(100) NOT NULL,
    payload      JSONB NOT NULL,
    status       VARCHAR(20) NOT NULL DEFAULT 'PENDING',   -- PENDING | PUBLISHED | FAILED
    retry_count  INT NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ
);

CREATE INDEX idx_outbox_pending   ON outbox_events(created_at) WHERE status = 'PENDING';
CREATE INDEX idx_outbox_aggregate ON outbox_events(aggregate_id);

-- =====================================================
-- Auto-update updated_at trigger
-- =====================================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN NEW.updated_at = NOW(); RETURN NEW; END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_wallets_updated_at
    BEFORE UPDATE ON wallets FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trg_wallet_tx_updated_at
    BEFORE UPDATE ON wallet_transactions FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- =====================================================
-- Comments
-- =====================================================
COMMENT ON TABLE wallets IS
    'Internal user wallets — one wallet per user (enforced by UNIQUE on user_id)';
COMMENT ON TABLE wallet_transactions IS
    'Immutable double-entry ledger — never UPDATE amount or type fields';
COMMENT ON TABLE outbox_events IS
    'Transactional outbox — guarantees at-least-once NATS delivery';
COMMENT ON COLUMN wallets.version IS
    'Optimistic locking version — must match on UPDATE or reject with ErrVersionConflict';
COMMENT ON COLUMN wallet_transactions.idempotency_key IS
    'Idempotency key format: {type}:{userID}:{referenceID}';
