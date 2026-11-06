package store

import (
	"context"
	"fmt"
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
