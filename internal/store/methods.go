package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/ports"
)

// возвращает все методы шифрования, отсортированные по коду
func (s *Store) Methods(ctx context.Context) ([]ports.MethodRow, error) {
	rows, err := s.pool.Query(ctx, `SELECT code, title, key_size, enabled FROM methods ORDER BY code`)
	if err != nil {
		return nil, fmt.Errorf("store: query methods: %w", err)
	}

	methods, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (ports.MethodRow, error) {
		var m ports.MethodRow
		err := row.Scan(&m.Code, &m.Title, &m.KeySize, &m.Enabled)

		return m, err
	})
	if err != nil {
		return nil, fmt.Errorf("store: scan methods: %w", err)
	}

	return methods, nil
}

// SetMethodEnabled включает или выключает метод. Неизвестный код — ports.ErrNotFound.
func (s *Store) SetMethodEnabled(ctx context.Context, code string, on bool) error {
	tag, err := s.pool.Exec(ctx, `UPDATE methods SET enabled = $2 WHERE code = $1`, code, on)
	if err != nil {
		return fmt.Errorf("store: update method %q: %w", code, err)
	}

	if tag.RowsAffected() == 0 {
		return ports.ErrNotFound
	}

	return nil
}
