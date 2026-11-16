BEGIN;

CREATE TABLE IF NOT EXISTS recovery_codes (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_hash BYTEA NOT NULL CHECK (octet_length(code_hash) = 32),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    used_at TIMESTAMPTZ,
    PRIMARY KEY (user_id, code_hash)
);

COMMIT;
