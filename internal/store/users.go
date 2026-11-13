package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/ports"
)

// UserKey возвращает ключ RSA пользователя. Если записи нет — ports.ErrNotFound.
//
// public_key_pem и private_sealed пусты (NULL), пока ключ генерируется или
// генерация не удалась: они читаются через указатель, а не прямо в string.
func (s *Store) UserKey(ctx context.Context, userID int64) (*ports.UserKey, error) {
	const q = `
SELECT user_id, bits, public_key_pem, private_sealed, keygen_ms, status, updated_at
FROM user_keys
WHERE user_id = $1`

	var (
		k         ports.UserKey
		publicPEM *string
	)

	err := s.pool.QueryRow(ctx, q, userID).Scan(
		&k.UserID, &k.Bits, &publicPEM, &k.PrivateSealed, &k.KeygenMs, &k.Status, &k.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ports.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("store: get user key %d: %w", userID, err)
	}

	if publicPEM != nil {
		k.PublicKeyPEM = *publicPEM
	}

	return &k, nil
}

// SaveUserKey создаёт или полностью перезаписывает ключ пользователя.
// updated_at выставляется базой, значение из k игнорируется. Пустые
// PublicKeyPEM и PrivateSealed записываются как NULL.
func (s *Store) SaveUserKey(ctx context.Context, k *ports.UserKey) error {
	const q = `
INSERT INTO user_keys (user_id, bits, public_key_pem, private_sealed, keygen_ms, status)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (user_id) DO UPDATE SET
    bits           = EXCLUDED.bits,
    public_key_pem = EXCLUDED.public_key_pem,
    private_sealed = EXCLUDED.private_sealed,
    keygen_ms      = EXCLUDED.keygen_ms,
    status         = EXCLUDED.status,
    updated_at     = now()`

	_, err := s.pool.Exec(ctx, q,
		k.UserID, k.Bits, nullIfEmpty(k.PublicKeyPEM), nullIfEmptyBytes(k.PrivateSealed), k.KeygenMs, k.Status)
	if err != nil {
		return fmt.Errorf("store: save user key %d: %w", k.UserID, err)
	}

	return nil
}

// StartUserKey создаёт запись ключа в статусе generating, если записи ещё нет.
// started сообщает, создана ли запись этим вызовом: из нескольких реплик
// и горутин, одновременно увидевших отсутствие ключа, true получает ровно одна,
// и только она запускает генерацию.
func (s *Store) StartUserKey(ctx context.Context, userID int64, bits int) (started bool, err error) {
	const q = `
INSERT INTO user_keys (user_id, bits, keygen_ms, status)
VALUES ($1, $2, 0, 'generating')
ON CONFLICT (user_id) DO NOTHING`

	tag, err := s.pool.Exec(ctx, q, userID, bits)
	if err != nil {
		return false, fmt.Errorf("store: start user key %d: %w", userID, err)
	}

	return tag.RowsAffected() == 1, nil
}

// RestartUserKey переводит ключ обратно в generating, если прошлая генерация
// не удалась (failed) или зависла: запись осталась в generating с updated_at
// раньше staleBefore, например после падения процесса посреди генерации.
// started, как и в StartUserKey, получает только один из конкурирующих вызовов.
func (s *Store) RestartUserKey(ctx context.Context, userID int64, bits int, staleBefore time.Time) (started bool, err error) {
	const q = `
UPDATE user_keys SET
    bits           = $2,
    public_key_pem = NULL,
    private_sealed = NULL,
    keygen_ms      = 0,
    status         = 'generating',
    updated_at     = now()
WHERE user_id = $1
  AND (status = 'failed' OR (status = 'generating' AND updated_at < $3))`

	tag, err := s.pool.Exec(ctx, q, userID, bits, staleBefore)
	if err != nil {
		return false, fmt.Errorf("store: restart user key %d: %w", userID, err)
	}

	return tag.RowsAffected() == 1, nil
}

func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}

	return &s
}

func nullIfEmptyBytes(b []byte) []byte {
	if len(b) == 0 {
		return nil
	}

	return b
}
