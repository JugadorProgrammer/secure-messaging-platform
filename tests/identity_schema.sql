BEGIN;

DO $$
DECLARE
    suffix TEXT := replace(gen_random_uuid()::text, '-', '');
    uid UUID;
    user_role_id BIGINT;
    admin_role_id BIGINT;
BEGIN
    SELECT id INTO user_role_id FROM roles WHERE code = 'user';
    SELECT id INTO admin_role_id FROM roles WHERE code = 'admin';

    IF user_role_id IS NULL OR admin_role_id IS NULL THEN
        RAISE EXCEPTION 'System roles missing';
    END IF;

    INSERT INTO users (login, email, password_hash)
    VALUES ('test_' || suffix, 'test_' || suffix || '@example.test', 'hash')
    RETURNING id INTO uid;

    INSERT INTO user_roles (user_id, role_id)
    VALUES (uid, user_role_id), (uid, admin_role_id);

    IF (SELECT count(*) FROM user_roles WHERE user_id = uid) <> 2 THEN
        RAISE EXCEPTION 'Multiple roles test failed';
    END IF;

    BEGIN
        INSERT INTO user_roles VALUES (uid, user_role_id);
        RAISE EXCEPTION 'Duplicate role assignment accepted';
    EXCEPTION WHEN unique_violation THEN NULL;
    END;

    BEGIN
        INSERT INTO user_roles VALUES (uid, -1);
        RAISE EXCEPTION 'Invalid role reference accepted';
    EXCEPTION WHEN foreign_key_violation THEN NULL;
    END;

    BEGIN
        INSERT INTO users (login, email, password_hash)
        VALUES (upper('test_' || suffix), 'other_' || suffix || '@example.test', 'hash');
        RAISE EXCEPTION 'Duplicate login accepted';
    EXCEPTION WHEN unique_violation THEN NULL;
    END;

    DELETE FROM user_roles WHERE user_id = uid AND role_id = admin_role_id;

    IF EXISTS (
        SELECT 1 FROM user_roles WHERE user_id = uid AND role_id = admin_role_id
    ) THEN
        RAISE EXCEPTION 'Role revocation failed';
    END IF;

    DELETE FROM users WHERE id = uid;

    IF EXISTS (SELECT 1 FROM user_roles WHERE user_id = uid) THEN
        RAISE EXCEPTION 'Cascade deletion failed';
    END IF;
END $$;

ROLLBACK;
