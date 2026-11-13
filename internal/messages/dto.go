// Package messages — сценарии работы с сообщениями: шифрование и запись
// (Dev B), чтение (Dev A), расшифрование и PDF (Dev B).
//
// Файлы поделены по владельцам (contracts.md, раздел 1, правило 3): dto.go —
// контракт COORD-2 и меняется только PR с пометкой «изменение контракта».
package messages

import (
	"encoding/hex"
	"time"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/ports"
)

// EncryptRequest — тело POST /api/messages.
type EncryptRequest struct {
	Text   string       `json:"text"`
	Method ports.Method `json:"method"`
}

// MessageDTO — сообщение в ответах API.
type MessageDTO struct {
	ID        int64        `json:"id"`
	Owner     string       `json:"owner"`
	CreatedAt time.Time    `json:"created_at"`
	Method    ports.Method `json:"method"`
	KeySize   int          `json:"key_size"`
	Plaintext string       `json:"plaintext"`
	// Ciphertext — шифртекст в hex, нижний регистр.
	Ciphertext string   `json:"ciphertext"`
	Keys       *KeysDTO `json:"keys,omitempty"`
	Metrics    Metrics  `json:"metrics"`
}

// KeysDTO — ключи сообщения. Отдаются только владельцу: это учебная
// платформа, и пользователь должен видеть, чем зашифрован его текст.
type KeysDTO struct {
	PublicKeyPEM  string `json:"public_key_pem,omitempty"`
	PrivateKeyPEM string `json:"private_key_pem,omitempty"`
	KeyHex        string `json:"key_hex,omitempty"`
	IVHex         string `json:"iv_hex,omitempty"`
}

// Metrics — длительности операций по данным Крипто-модуля, в миллисекундах.
type Metrics struct {
	KeygenMs  int64 `json:"keygen_ms"`
	EncryptMs int64 `json:"encrypt_ms"`
	DecryptMs int64 `json:"decrypt_ms"`
}

// toDTO собирает ответ из сохранённого сообщения. keys == nil — ключи
// не отдаются.
func toDTO(m *ports.Message, keys *KeysDTO) MessageDTO {
	return MessageDTO{
		ID:         m.ID,
		Owner:      m.OwnerLogin,
		CreatedAt:  m.CreatedAt,
		Method:     ports.Method(m.Method),
		KeySize:    m.KeySize,
		Plaintext:  m.Plaintext,
		Ciphertext: hex.EncodeToString(m.Ciphertext),
		Keys:       keys,
		Metrics: Metrics{
			KeygenMs:  m.KeygenMs,
			EncryptMs: m.EncryptMs,
			DecryptMs: m.DecryptMs,
		},
	}
}
