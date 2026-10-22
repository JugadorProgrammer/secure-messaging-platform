package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/ports"
)

const messageColumns = `id, owner_id, owner_login, method, key_size, plaintext, ciphertext,
kuz_key_sealed, kuz_iv, keygen_ms, encrypt_ms, decrypt_ms, created_at`

// CreateMessage сохраняет сообщение и возвращает его идентификатор.
// m.ID и m.CreatedAt игнорируются: их назначает база.
func (s *Store) CreateMessage(ctx context.Context, m *ports.Message) (int64, error) {
	const q = `
INSERT INTO messages (owner_id, owner_login, method, key_size, plaintext, ciphertext,
                      kuz_key_sealed, kuz_iv, keygen_ms, encrypt_ms, decrypt_ms)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING id`

	var id int64

	err := s.pool.QueryRow(ctx, q,
		m.OwnerID, m.OwnerLogin, m.Method, m.KeySize, m.Plaintext, m.Ciphertext,
		m.KuzKeySealed, m.KuzIV, m.KeygenMs, m.EncryptMs, m.DecryptMs,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("store: create message: %w", err)
	}

	return id, nil
}

// GetMessage возвращает сообщение по идентификатору. Если его нет — ports.ErrNotFound.
func (s *Store) GetMessage(ctx context.Context, id int64) (*ports.Message, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+messageColumns+` FROM messages WHERE id = $1`, id)
	if err != nil {
		return nil, fmt.Errorf("store: get message %d: %w", id, err)
	}

	m, err := pgx.CollectExactlyOneRow(rows, scanMessage)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ports.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("store: get message %d: %w", id, err)
	}

	return m, nil
}

// ListMessages возвращает страницу сообщений, новые сверху.
//
// ownerID == nil означает все сообщения (выборка администратора). Значение 0
// как «все» не используется: user_id 0 занят системным ключом.
// При равном created_at порядок задаёт id, иначе страницы могли бы
// пересекаться.
func (s *Store) ListMessages(ctx context.Context, ownerID *int64, limit, offset int) ([]*ports.Message, error) {
	var (
		rows pgx.Rows
		err  error
	)

	if ownerID == nil {
		rows, err = s.pool.Query(ctx, `SELECT `+messageColumns+` FROM messages
ORDER BY created_at DESC, id DESC LIMIT $1 OFFSET $2`, limit, offset)
	} else {
		rows, err = s.pool.Query(ctx, `SELECT `+messageColumns+` FROM messages
WHERE owner_id = $1 ORDER BY created_at DESC, id DESC LIMIT $2 OFFSET $3`, *ownerID, limit, offset)
	}
	if err != nil {
		return nil, fmt.Errorf("store: list messages: %w", err)
	}

	var msgList []*ports.Message

	for rows.Next() {
		m, err := scanMessage(rows)
		if err != nil {
			rows.Close()

			return nil, fmt.Errorf("store: list messages: %w", err)
		}

		msgList = append(msgList, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: list messages: %w", err)
	}

	return msgList, nil
}

// SetDecryptMs записывает время последнего расшифрования. Если сообщения
// нет — ports.ErrNotFound.
func (s *Store) SetDecryptMs(ctx context.Context, id, ms int64) error {
	tag, err := s.pool.Exec(ctx, `UPDATE messages SET decrypt_ms = $2 WHERE id = $1`, id, ms)
	if err != nil {
		return fmt.Errorf("store: set decrypt_ms %d: %w", id, err)
	}

	if tag.RowsAffected() == 0 {
		return ports.ErrNotFound
	}

	return nil
}

func scanMessage(row pgx.CollectableRow) (*ports.Message, error) {
	var m ports.Message

	err := row.Scan(
		&m.ID, &m.OwnerID, &m.OwnerLogin, &m.Method, &m.KeySize, &m.Plaintext, &m.Ciphertext,
		&m.KuzKeySealed, &m.KuzIV, &m.KeygenMs, &m.EncryptMs, &m.DecryptMs, &m.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &m, nil
}
