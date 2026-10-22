package store

import (
	"context"
	"fmt"
	"io/fs"

	"github.com/jackc/pgx/v5"

	"github.com/JugadorProgrammer/secure-messaging-platform/migrations"
)

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
// rsaBits — размер ключа RSA из конфигурации, попадает в справочник методов.
func (s *Store) Migrate(ctx context.Context, rsaBits int) error {
	if _, err := s.pool.Exec(ctx, createSchemaMigrations); err != nil {
		return fmt.Errorf("store: create schema_migrations: %w", err)
	}

	files, err := fs.Glob(migrations.FS, "*.sql")
	if err != nil {
		return fmt.Errorf("store: list migrations: %w", err)
	}

	applied, err := s.appliedVersions(ctx)
	if err != nil {
		return err
	}

	for _, name := range files {
		if applied[name] {
			continue
		}

		if err := s.applyMigration(ctx, name, rsaBits); err != nil {
			s.log.Error("migration failed", "version", name, "err", err)

			return fmt.Errorf("store: ошибка применения миграции %s: %w", name, err)
		}

		s.log.Info("migration applied", "version", name)
	}

	return nil
}

func (s *Store) appliedVersions(ctx context.Context) (map[string]bool, error) {
	rows, err := s.pool.Query(ctx, `SELECT version FROM schema_migrations`)
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

func (s *Store) applyMigration(ctx context.Context, name string, rsaBits int) error {
	body, err := fs.ReadFile(migrations.FS, name)
	if err != nil {
		return err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if _, err := tx.Exec(ctx, string(body)); err != nil {
		return err
	}

	// сид методов в той же транзакции, что и схема
	if name == "001_init.sql" {
		if _, err := tx.Exec(ctx, seedMethods, rsaBits); err != nil {
			return err
		}
	}

	if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, name); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
