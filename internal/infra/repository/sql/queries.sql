-- name: SaveInboxEvent :exec
INSERT INTO inbox (id, external_id, event_type, payload, metadata)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (external_id) DO UPDATE 
SET payload = EXCLUDED.payload, metadata = EXCLUDED.metadata, updated_at = NOW()
WHERE inbox.updated_at < NOW();

-- name: ClaimInboxEvents :many
-- PHASE A: SELECT SKIP LOCKED + UPDATE status = 'PROCESSING'
UPDATE inbox
SET status = 'PROCESSING'
WHERE id IN (
    SELECT id FROM inbox
    WHERE status = 'PENDING'
    ORDER BY created_at ASC
    FOR UPDATE SKIP LOCKED
    LIMIT $1
)
RETURNING *;

-- name: FinalizeInboxEvent :exec
-- PHASE C: UPDATE status = 'COMPLETED'
UPDATE inbox
SET status = 'COMPLETED', processed_at = NOW(), updated_at = NOW()
WHERE id = $1;

-- name: MarkInboxFailed :exec
UPDATE inbox
SET status = CASE WHEN retry_count >= 4 THEN 'DLQ' ELSE 'PENDING' END,
    retry_count = retry_count + 1,
    updated_at = NOW()
WHERE id = $1;

-- name: SaveOutboxEvent :exec
INSERT INTO outbox (id, event_type, payload, metadata)
VALUES ($1, $2, $3, $4);

-- name: ClaimOutboxEvents :many
UPDATE outbox
SET status = 'PROCESSING'
WHERE id IN (
    SELECT id FROM outbox
    WHERE status = 'PENDING'
    AND created_at > NOW() - INTERVAL '48 HOURS'
    ORDER BY created_at ASC
    FOR UPDATE SKIP LOCKED
    LIMIT $1
)
RETURNING *;

-- name: FinalizeOutboxEvent :exec
UPDATE outbox
SET status = 'COMPLETED', processed_at = NOW()
WHERE id = $1;
