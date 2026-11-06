-- Начальная схема координационного сервиса (contracts.md, раздел 13).
-- Колонок для IV и MAC запечатанных ключей нет: они внутри контейнера sealed v1.

CREATE TABLE IF NOT EXISTS methods (
    code     TEXT    PRIMARY KEY CHECK (code IN ('rsa', 'kuznyechik')),
    title    TEXT    NOT NULL,
    key_size INT     NOT NULL CHECK (key_size > 0),
    enabled  BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE IF NOT EXISTS user_keys (
    user_id        BIGINT      PRIMARY KEY CHECK (user_id >= 0),
    bits           INT         NOT NULL,
    public_key_pem TEXT,
    private_sealed BYTEA,
    keygen_ms      BIGINT      NOT NULL DEFAULT 0,
    status         TEXT        NOT NULL CHECK (status IN ('generating', 'ready', 'failed')),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS messages (
    id             BIGSERIAL   PRIMARY KEY,
    owner_id       BIGINT      NOT NULL,
    owner_login    TEXT        NOT NULL,
    method         TEXT        NOT NULL REFERENCES methods (code),
    key_size       INT         NOT NULL,
    plaintext      TEXT        NOT NULL,
    ciphertext     BYTEA       NOT NULL,
    kuz_key_sealed BYTEA,
    kuz_iv         BYTEA,
    keygen_ms      BIGINT      NOT NULL DEFAULT 0,
    encrypt_ms     BIGINT      NOT NULL DEFAULT 0,
    decrypt_ms     BIGINT      NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS messages_owner_created_idx
    ON messages (owner_id, created_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS ops_log (
    id      BIGSERIAL   PRIMARY KEY,
    at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    user_id BIGINT,
    op      TEXT        NOT NULL,
    ref     TEXT,
    ok      BOOLEAN     NOT NULL
);

CREATE INDEX IF NOT EXISTS ops_log_at_idx ON ops_log (at DESC);
