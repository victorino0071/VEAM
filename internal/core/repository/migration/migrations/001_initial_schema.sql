-- Schema Version 001: Initial Setup for Inbox, Outbox and Transactions
-- This script contains the DDL for the atomic engine core.

CREATE TABLE IF NOT EXISTS payment_engine_migrations (
    version int PRIMARY KEY,
    applied_at timestamp DEFAULT now()
);

-- Core Engine Tables
CREATE TABLE IF NOT EXISTS inbox (
    id UUID PRIMARY KEY,
    external_id VARCHAR(255) UNIQUE NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    payload BYTEA NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}',
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING',
    retry_count INT NOT NULL DEFAULT 0,
    last_error TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_inbox_active_claim ON inbox (created_at) WHERE status = 'PENDING';

CREATE TABLE IF NOT EXISTS outbox (
    id UUID,
    event_type VARCHAR(100) NOT NULL,
    payload BYTEA NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}',
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING',
    retry_count INT NOT NULL DEFAULT 0,
    last_error TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMP,
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- Partition setup for outbox
DO $$ BEGIN
    IF NOT EXISTS (SELECT FROM pg_tables WHERE tablename = 'outbox_default') THEN
        CREATE TABLE outbox_default PARTITION OF outbox DEFAULT;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_outbox_active_claim ON outbox (created_at) WHERE status = 'PENDING';

-- Domain Specific Tables
CREATE TABLE IF NOT EXISTS transactions (
    id VARCHAR(255) PRIMARY KEY,
    customer_id VARCHAR(255) NOT NULL,
    provider_id VARCHAR(100) NOT NULL, -- Added for multi-gateway routing
    amount DECIMAL(10,2) NOT NULL,
    currency VARCHAR(10) NOT NULL,
    status VARCHAR(50) NOT NULL,
    description TEXT,
    due_date TIMESTAMP NOT NULL,
    payment_date TIMESTAMP,
    confirmed_date TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
