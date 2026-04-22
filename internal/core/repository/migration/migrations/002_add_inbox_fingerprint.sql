-- Schema Version 002: Add Fingerprint for Content-Based Deduplication
-- This migration adds a fingerprint column to the inbox table to detect business duplicates
-- that may have different delivery IDs or timestamps.

ALTER TABLE inbox ADD COLUMN IF NOT EXISTS fingerprint VARCHAR(64);

-- Create a composite unique index to isolate universes per provider
CREATE UNIQUE INDEX IF NOT EXISTS idx_inbox_fingerprint ON inbox ((metadata->>'provider_id'), fingerprint);
