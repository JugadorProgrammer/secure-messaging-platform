BEGIN;

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS failed_login_attempts INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS last_failed_login_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS login_locked_until TIMESTAMPTZ;

CREATE TABLE IF NOT EXISTS password_reset_challenges (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE CHECK (token_hash ~ '^[a-f0-9]{64}$'),
    code_hash TEXT CHECK (code_hash ~ '^[a-f0-9]{64}$'),
    attempts_remaining INTEGER NOT NULL CHECK (attempts_remaining >= 0),
    resend_after TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL
);

COMMIT;
