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
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMP
);

CREATE INDEX idx_inbox_pending ON inbox (status) WHERE status = 'PENDING';
CREATE INDEX idx_inbox_dlq ON inbox (status) WHERE status = 'DLQ';

CREATE TABLE IF NOT EXISTS outbox (
    id UUID,
    event_type VARCHAR(100) NOT NULL,
    payload BYTEA NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}', -- W3C Trace Context + Baggage
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING', -- PENDING, PROCESSING, COMPLETED, FAILED, DLQ
    retry_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMP,
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- O particionamento em produção deve ser gerenciado por extensões como pg_partman
-- Para fins de execução local/demonstração, instanciamos a partição default.
CREATE TABLE outbox_default PARTITION OF outbox DEFAULT;

CREATE INDEX idx_outbox_pending ON outbox (status) WHERE status = 'PENDING';

CREATE TABLE IF NOT EXISTS transactions (
    id VARCHAR(255) PRIMARY KEY,
    customer_id VARCHAR(255) NOT NULL,
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
