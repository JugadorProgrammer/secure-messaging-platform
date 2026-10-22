// Package store реализует ports.Store поверх PostgreSQL.
//
// Пакет не импортирует бизнес-пакеты (contracts.md, раздел 1, правило 2):
// он знает только про ports и драйвер базы.
package store

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/ports"
)

// Проверка на этапе компиляции: Store реализует контракт целиком.
var _ ports.Store = (*Store)(nil)

// Store — хранилище на пуле соединений pgx.
type Store struct {
	pool *pgxpool.Pool
	log  *slog.Logger
}

// NewStore открывает пул соединений и проверяет, что база доступна.
func NewStore(ctx context.Context, dsn string, log *slog.Logger) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		// Текст ошибки разбора может содержать DSN целиком, поэтому не оборачиваем его.
		return nil, fmt.Errorf("store: parse dsn: invalid connection string")
	}

	cfg.MaxConns = 10
	cfg.MinConns = 2
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = 30 * time.Minute
	cfg.ConnConfig.ConnectTimeout = 5 * time.Second

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("store: create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()

		return nil, fmt.Errorf("store: ping: %w", err)
	}

	return &Store{pool: pool, log: log}, nil
}

// Close закрывает пул. После Close методы Store использовать нельзя.
func (s *Store) Close() {
	s.pool.Close()
}
