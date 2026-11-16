BEGIN;

CREATE TABLE IF NOT EXISTS recovery_challenges (
    token_hash TEXT PRIMARY KEY CHECK (token_hash ~ '^[0-9a-f]{64}$'),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS recovery_challenges_expiry_idx ON recovery_challenges (expires_at);

COMMIT;
