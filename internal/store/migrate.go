package store

import (
	"context"
	"fmt"
	"io/fs"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/JugadorProgrammer/secure-messaging-platform/migrations"
)

// migrationLockKey — ключ pg_advisory_lock для миграций. Значение произвольное,
// важно только, чтобы все реплики сервиса использовали одно и то же.
const migrationLockKey int64 = 0x736d702d636f6f72 // "smp-coor"

const createSchemaMigrations = `
CREATE TABLE IF NOT EXISTS schema_migrations (
    version    TEXT        PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
)`

// seedMethods заполняет справочник методов. enabled при конфликте не трогаем:
// администратор мог выключить метод, и рестарт не должен его включать.
const seedMethods = `
INSERT INTO methods (code, title, key_size, enabled)
VALUES ('rsa', 'RSA', $1, TRUE),
       ('kuznyechik', 'Кузнечик', 256, TRUE)
ON CONFLICT (code) DO UPDATE SET key_size = EXCLUDED.key_size`

// Migrate применяет миграции из migrations.FS, которые ещё не записаны
// в schema_migrations. Каждая миграция выполняется в своей транзакции:
// при ошибке схема остаётся в последнем согласованном состоянии.
//
// Реплики, стартующие одновременно, сериализуются через pg_advisory_lock.
// Список применённых версий читается уже под блокировкой, поэтому вторая
// реплика видит работу первой и ничего не применяет повторно.
//
// rsaBits — размер ключа RSA из конфигурации, попадает в справочник методов.
func (s *Store) Migrate(ctx context.Context, rsaBits int) error {
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("store: acquire connection for migrations: %w", err)
	}
	defer conn.Release()

	// Блокировка сессионная: она живёт на этом соединении, поэтому все шаги
	// ниже выполняются через conn, а не через пул.
	if _, err := conn.Exec(ctx, `SELECT pg_advisory_lock($1)`, migrationLockKey); err != nil {
		return fmt.Errorf("store: lock migrations: %w", err)
	}
	defer func() {
		// Снимаем блокировку и при отменённом ctx, иначе она останется
		// на соединении, которое вернётся в пул.
		_, _ = conn.Exec(context.WithoutCancel(ctx), `SELECT pg_advisory_unlock($1)`, migrationLockKey)
	}()

	if _, err := conn.Exec(ctx, createSchemaMigrations); err != nil {
		return fmt.Errorf("store: create schema_migrations: %w", err)
	}

	files, err := fs.Glob(migrations.FS, "*.sql")
	if err != nil {
		return fmt.Errorf("store: list migrations: %w", err)
	}

	applied, err := appliedVersions(ctx, conn)
	if err != nil {
		return err
	}

	for _, name := range files {
		if applied[name] {
			continue
		}

		// Не логируем здесь: ошибку пишет в лог run, иначе одно событие даст две записи.
		if err := applyMigration(ctx, conn, name, rsaBits); err != nil {
			return fmt.Errorf("store: apply migration %s: %w", name, err)
		}

		s.log.Info("migration applied", "version", name)
	}

	return nil
}

func appliedVersions(ctx context.Context, conn *pgxpool.Conn) (map[string]bool, error) {
	rows, err := conn.Query(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("store: read schema_migrations: %w", err)
	}

	versions, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, fmt.Errorf("store: read schema_migrations: %w", err)
	}

	applied := make(map[string]bool, len(versions))
	for _, v := range versions {
		applied[v] = true
	}

	return applied, nil
}

func applyMigration(ctx context.Context, conn *pgxpool.Conn, name string, rsaBits int) error {
	body, err := fs.ReadFile(migrations.FS, name)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := tx.Exec(ctx, string(body)); err != nil {
		return fmt.Errorf("exec body: %w", err)
	}

	// сид методов в той же транзакции, что и схема
	if name == "001_init.sql" {
		if _, err := tx.Exec(ctx, seedMethods, rsaBits); err != nil {
			return fmt.Errorf("seed methods: %w", err)
		}
	}

	if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, name); err != nil {
		return fmt.Errorf("record version: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	return nil
}
