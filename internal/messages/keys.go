package messages

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/ports"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/sealed"
)

// errKeyMissing — у RSA-сообщения нет готового ключа владельца. Ключ
// создаётся до первого RSA-сообщения и после этого не перегенерируется,
// так что это нарушение инварианта, а не ошибка пользователя.
var errKeyMissing = errors.New("messages: owner rsa key is not ready")

// rsaKeys возвращает пару ключей RSA из записи ключа пользователя.
// Закрытый ключ распечатывается, буфер обнуляется сразу после копирования
// в строку ответа (styleguide.md, раздел 9). Строку обнулить нельзя: строки
// в Go неизменяемы, она живёт до сборки мусора.
func (h *Handler) rsaKeys(ctx context.Context, key *ports.UserKey) (KeysDTO, error) {
	privateBytes, err := sealed.Unseal(ctx, h.crypto, h.masterKeyHex, key.PrivateSealed)
	if err != nil {
		return KeysDTO{}, unsealErr(err)
	}
	defer clear(privateBytes)

	return KeysDTO{PublicKey: key.PublicKeyPEM, PrivateKey: string(privateBytes)}, nil
}

// kuzKeys возвращает ключ и IV Кузнечика из сохранённого сообщения.
func (h *Handler) kuzKeys(ctx context.Context, m *ports.Message) (KeysDTO, error) {
	keyBytes, err := sealed.Unseal(ctx, h.crypto, h.masterKeyHex, m.KuzKeySealed)
	if err != nil {
		return KeysDTO{}, unsealErr(err)
	}
	defer clear(keyBytes)

	return KeysDTO{Key: string(keyBytes), IV: hex.EncodeToString(m.KuzIV)}, nil
}

// keyCache раскрывает ключи сообщений списка. Пара RSA распечатывается один
// раз на владельца, а не на каждое сообщение.
type keyCache struct {
	h   *Handler
	rsa map[int64]KeysDTO
}

func newKeyCache(h *Handler) *keyCache {
	return &keyCache{h: h, rsa: make(map[int64]KeysDTO)}
}

func (c *keyCache) keysFor(ctx context.Context, m *ports.Message) (KeysDTO, error) {
	if m.Method == string(ports.MethodKuznyechik) {
		return c.h.kuzKeys(ctx, m)
	}

	if keys, ok := c.rsa[m.OwnerID]; ok {
		return keys, nil
	}

	key, err := c.h.store.UserKey(ctx, m.OwnerID)

	switch {
	case errors.Is(err, ports.ErrNotFound) || err == nil && key.Status != ports.KeyStatusReady:
		// Ключа владельца нет: одно такое сообщение не должно ломать весь
		// список. Сообщение отдаётся без ключей, расхождение пишется в лог.
		c.h.log.WarnContext(ctx, "owner rsa key unavailable", "owner_id", m.OwnerID, "err", errKeyMissing)
		c.rsa[m.OwnerID] = KeysDTO{}

		return KeysDTO{}, nil
	case err != nil:
		return KeysDTO{}, internalErr(fmt.Errorf("owner key %d: %w", m.OwnerID, err))
	}

	keys, err := c.h.rsaKeys(ctx, key)
	if err != nil {
		return KeysDTO{}, err
	}

	c.rsa[m.OwnerID] = keys

	return keys, nil
}
