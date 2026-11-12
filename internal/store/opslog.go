package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/ports"
)

// LogOp добавляет запись в журнал операций.
//
// userID == nil пишется как NULL: это анонимный субъект, например проверка
// подписи без токена. Подставлять 0 нельзя, 0 — системный ключ.
// Таймаут и context.WithoutCancel — на стороне вызывающего (contracts.md, раздел 10).
func (s *Store) LogOp(ctx context.Context, userID *int64, op, ref string, ok bool) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO ops_log (user_id, op, ref, ok) VALUES ($1, $2, $3, $4)`,
		userID, op, ref, ok,
	)
	if err != nil {
		return fmt.Errorf("store: log op %q: %w", op, err)
	}

	return nil
}

// ListOps возвращает страницу журнала операций, новые сверху.
//
// Сортировка по at не детерминирована: записи, созданные в один момент now(),
// база выдаёт в любом порядке, поэтому вторым ключом идёт id.
// COALESCE по ref нужен, чтобы NULL в колонке не ломал разбор в строку;
// user_id остаётся NULL — это анонимная операция, а не пользователь 0.
//
// limit вне 1..100 и отрицательный offset — ports.ErrInvalidArgument, как
// у ListMessages: страница нулевого размера выглядит как пустой журнал.
func (s *Store) ListOps(ctx context.Context, limit, offset int) ([]ports.OpLogEntry, error) {
	if limit < 1 || limit > maxListLimit {
		return nil, fmt.Errorf("store: list ops: limit %d out of range 1..%d: %w", limit, maxListLimit, ports.ErrInvalidArgument)
	}
	if offset < 0 {
		return nil, fmt.Errorf("store: list ops: negative offset %d: %w", offset, ports.ErrInvalidArgument)
	}

	rows, err := s.pool.Query(ctx, `SELECT id, at, user_id, op, COALESCE(ref, '') AS ref, ok FROM ops_log
ORDER BY at DESC, id DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("store: list ops: %w", err)
	}

	entries, err := pgx.CollectRows(rows, pgx.RowToStructByName[ports.OpLogEntry])
	if err != nil {
		return nil, fmt.Errorf("store: scan ops log: %w", err)
	}

	return entries, nil
}
