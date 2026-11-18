// Package messages — сценарии работы с сообщениями: шифрование, список,
// расшифрование и PDF с подписью.
//
// Формат запросов и ответов совпадает со шлюзом фронта (HttpCoordGateway,
// FRONT-develop): названия полей, названия методов и маршруты заданы им.
package messages

import (
	"encoding/hex"
	"time"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/methods"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/ports"
)

// EncryptRequest — тело POST /api/encrypt.
type EncryptRequest struct {
	Plaintext string `json:"plaintext"`
	// Method — название метода: «RSA-32768» или «Кузнечик».
	Method string `json:"method"`
}

// DecryptRequest — тело POST /api/decrypt.
//
// Для RSA нужен PrivateKey, для Кузнечика — Key. IV Кузнечика фронт
// не передаёт: он берётся из сохранённого сообщения с тем же шифртекстом.
// Если IV передан, используется он.
type DecryptRequest struct {
	Method     string `json:"method"`
	Ciphertext string `json:"ciphertext"`
	PrivateKey string `json:"privateKey,omitempty"`
	Key        string `json:"key,omitempty"`
	IV         string `json:"iv,omitempty"`
}

// DecryptResponse — ответ POST /api/decrypt.
type DecryptResponse struct {
	Plaintext string `json:"plaintext"`
}

// ExportPDFRequest — тело POST /api/exportPdfWithSignature.
type ExportPDFRequest struct {
	ID int64 `json:"id"`
}

// MessageDTO — сообщение в ответах API.
type MessageDTO struct {
	ID        int64   `json:"id"`
	User      UserDTO `json:"user"`
	Plaintext string  `json:"plaintext"`
	// Ciphertext — шифртекст в hex, нижний регистр.
	Ciphertext string    `json:"ciphertext"`
	Timestamp  time.Time `json:"timestamp"`
	// Method — название метода: «RSA-32768» или «Кузнечик».
	Method  string  `json:"method"`
	Keys    KeysDTO `json:"keys"`
	Metrics Metrics `json:"metrics"`
}

// UserDTO — владелец сообщения.
type UserDTO struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// KeysDTO — ключи сообщения. Учебная платформа показывает пользователю, чем
// зашифрован текст: для RSA — пару PEM, для Кузнечика — ключ и IV в hex.
type KeysDTO struct {
	PublicKey  string `json:"publicKey,omitempty"`
	PrivateKey string `json:"privateKey,omitempty"`
	Key        string `json:"key,omitempty"`
	IV         string `json:"iv,omitempty"`
}

// Metrics — длительности операций по данным Крипто-модуля, в миллисекундах.
// Фронт их пока не показывает, но поле не мешает его схеме.
type Metrics struct {
	KeygenMs  int64 `json:"keygen_ms"`
	EncryptMs int64 `json:"encrypt_ms"`
	DecryptMs int64 `json:"decrypt_ms"`
}

// toDTO собирает ответ из сохранённого сообщения. Время отдаётся в UTC:
// схема фронта (z.iso.datetime) не принимает смещение часового пояса.
func toDTO(m *ports.Message, keys KeysDTO) MessageDTO {
	return MessageDTO{
		ID:         m.ID,
		User:       UserDTO{ID: m.OwnerID, Name: m.OwnerLogin},
		Plaintext:  m.Plaintext,
		Ciphertext: hex.EncodeToString(m.Ciphertext),
		Timestamp:  m.CreatedAt.UTC(),
		Method:     methods.Name(ports.Method(m.Method)),
		Keys:       keys,
		Metrics: Metrics{
			KeygenMs:  m.KeygenMs,
			EncryptMs: m.EncryptMs,
			DecryptMs: m.DecryptMs,
		},
	}
}
