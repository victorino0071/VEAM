-- Schema para Inbox e Outbox (PostgreSQL)

CREATE TABLE IF NOT EXISTS inbox (
    id UUID PRIMARY KEY,
    external_id VARCHAR(255) UNIQUE NOT NULL, -- Idempotência absoluta
    event_type VARCHAR(100) NOT NULL,
    payload BYTEA NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}', -- W3C Trace Context + Baggage
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING', -- PENDING, PROCESSING, COMPLETED, FAILED
    retry_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMP
);

CREATE INDEX idx_inbox_pending ON inbox (status) WHERE status = 'PENDING';

CREATE TABLE IF NOT EXISTS outbox (
    id UUID PRIMARY KEY,
    event_type VARCHAR(100) NOT NULL,
    payload BYTEA NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}', -- W3C Trace Context + Baggage
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING', -- PENDING, PROCESSING, COMPLETED, FAILED
    retry_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMP
);

CREATE INDEX idx_outbox_pending ON outbox (status) WHERE status = 'PENDING';
