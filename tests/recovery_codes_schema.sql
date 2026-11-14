BEGIN;

DO $$
DECLARE
    uid UUID;
    another_uid UUID;
    code_bytes BYTEA := decode(repeat('ab', 32), 'hex');
    updated_count INTEGER;
BEGIN
    INSERT INTO users (login, email, password_hash)
    VALUES ('recovery_test_' || gen_random_uuid(), 'recovery_test_' || gen_random_uuid() || '@example.test', 'test-hash')
    RETURNING id INTO uid;

    INSERT INTO users (login, email, password_hash)
    VALUES ('recovery_test_' || gen_random_uuid(), 'recovery_test_' || gen_random_uuid() || '@example.test', 'test-hash')
    RETURNING id INTO another_uid;

    INSERT INTO recovery_codes (user_id, code_hash) VALUES (uid, code_bytes);

    IF NOT EXISTS (SELECT 1 FROM recovery_codes WHERE user_id = uid AND used_at IS NULL) THEN
        RAISE EXCEPTION 'Unused recovery code not stored';
    END IF;

    BEGIN
        INSERT INTO recovery_codes (user_id, code_hash) VALUES (uid, code_bytes);
        RAISE EXCEPTION 'Duplicate code accepted';
    EXCEPTION WHEN unique_violation THEN NULL;
    END;

    BEGIN
        INSERT INTO recovery_codes (user_id, code_hash) VALUES (uid, decode('ff', 'hex'));
        RAISE EXCEPTION 'Invalid hash length accepted';
    EXCEPTION WHEN check_violation THEN NULL;
    END;

    UPDATE recovery_codes SET used_at = now()
    WHERE user_id = another_uid AND code_hash = code_bytes AND used_at IS NULL;
    GET DIAGNOSTICS updated_count = ROW_COUNT;
    IF updated_count <> 0 THEN
        RAISE EXCEPTION 'Another user could consume recovery code';
    END IF;

    UPDATE recovery_codes SET used_at = now()
    WHERE user_id = uid AND code_hash = code_bytes AND used_at IS NULL;
    GET DIAGNOSTICS updated_count = ROW_COUNT;
    IF updated_count <> 1 THEN
        RAISE EXCEPTION 'Recovery code not consumed';
    END IF;

    UPDATE recovery_codes SET used_at = now()
    WHERE user_id = uid AND code_hash = code_bytes AND used_at IS NULL;
    GET DIAGNOSTICS updated_count = ROW_COUNT;
    IF updated_count <> 0 THEN
        RAISE EXCEPTION 'Recovery code used twice';
    END IF;

    DELETE FROM users WHERE id = uid;
    IF EXISTS (SELECT 1 FROM recovery_codes WHERE user_id = uid) THEN
        RAISE EXCEPTION 'Recovery codes not deleted with user';
    END IF;
END $$;

ROLLBACK;
