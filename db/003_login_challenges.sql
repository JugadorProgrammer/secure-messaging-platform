BEGIN;

CREATE TABLE IF NOT EXISTS login_challenges (
    token_hash TEXT PRIMARY KEY CHECK (token_hash ~ '^[a-f0-9]{64}$'),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email_code_hash TEXT NOT NULL CHECK (email_code_hash ~ '^[a-f0-9]{64}$'),
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS login_challenges_expiry_idx ON login_challenges(expires_at);

COMMIT;
