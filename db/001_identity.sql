BEGIN;

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    login TEXT NOT NULL CHECK (btrim(login) <> ''),
    email TEXT NOT NULL CHECK (btrim(email) <> ''),
    password_hash TEXT NOT NULL CHECK (btrim(password_hash) <> '')
);

CREATE UNIQUE INDEX IF NOT EXISTS users_login_unique ON users (lower(login));
CREATE UNIQUE INDEX IF NOT EXISTS users_email_unique ON users (lower(email));

CREATE TABLE IF NOT EXISTS roles (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code TEXT NOT NULL UNIQUE CHECK (code ~ '^[a-z][a-z0-9_]*$'),
    description TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS user_roles (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id BIGINT NOT NULL REFERENCES roles(id) ON DELETE RESTRICT,
    PRIMARY KEY (user_id, role_id)
);

CREATE INDEX IF NOT EXISTS user_roles_role_id_idx ON user_roles (role_id);

INSERT INTO roles (code, description)
VALUES ('user', 'Обычный пользователь'), ('admin', 'Администратор')
ON CONFLICT (code) DO NOTHING;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = current_schema()
          AND table_name = 'users'
          AND column_name = 'role'
    ) THEN
        IF EXISTS (
            SELECT 1 FROM users u
            WHERE u.role IS NOT NULL AND NOT EXISTS (
                SELECT 1 FROM roles r WHERE r.code = lower(u.role::text)
            )
        ) THEN
            RAISE EXCEPTION 'Unknown legacy roles found';
        END IF;

        INSERT INTO user_roles (user_id, role_id)
        SELECT u.id, r.id
        FROM users u
        JOIN roles r ON r.code = lower(u.role::text)
        ON CONFLICT DO NOTHING;

        ALTER TABLE users DROP COLUMN role;
    END IF;
END $$;

COMMIT;
