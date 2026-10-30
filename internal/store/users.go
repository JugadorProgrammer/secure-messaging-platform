package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/ports"
)

// UserKey возвращает ключ RSA пользователя. Если записи нет — ports.ErrNotFound.
func (s *Store) UserKey(ctx context.Context, userID int64) (*ports.UserKey, error) {
	const q = `
SELECT user_id, bits, public_key_pem, private_sealed, keygen_ms, status, updated_at
FROM user_keys
WHERE user_id = $1`

	var k ports.UserKey

	err := s.pool.QueryRow(ctx, q, userID).Scan(
		&k.UserID, &k.Bits, &k.PublicKeyPEM, &k.PrivateSealed, &k.KeygenMs, &k.Status, &k.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ports.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("store: get user key %d: %w", userID, err)
	}

	return &k, nil
}

// SaveUserKey создаёт или полностью перезаписывает ключ пользователя.
// updated_at выставляется базой, значение из k игнорируется.
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

	_, err := s.pool.Exec(ctx, q, k.UserID, k.Bits, k.PublicKeyPEM, k.PrivateSealed, k.KeygenMs, k.Status)
	if err != nil {
		return fmt.Errorf("store: save user key %d: %w", k.UserID, err)
	}

	return nil
}
