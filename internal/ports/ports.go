// Package ports содержит доменные типы, интерфейсы и sentinel-ошибки,
// общие для пакетов Dev A и Dev B (contracts.md, разделы 2 и 3).
//
// Реализации живут в internal/store и internal/cryptoclient, потребители
// зависят только от этого пакета.
package ports

import (
	"context"
	"errors"
	"time"
)

// Role — роль пользователя из сервиса учётных записей.
type Role string

// Роли пользователей.
const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

// Method — код метода шифрования.
type Method string

// Методы шифрования.
const (
	MethodRSA        Method = "rsa"
	MethodKuznyechik Method = "kuznyechik"
)

// Статусы ключа RSA пользователя.
const (
	KeyStatusGenerating = "generating"
	KeyStatusReady      = "ready"
	KeyStatusFailed     = "failed"
)

// Sentinel-ошибки. Возвращаются реализациями, проверяются через errors.Is.
var (
	ErrNotFound          = errors.New("not found")
	ErrConflict          = errors.New("conflict")
	ErrCryptoBadInput    = errors.New("crypto: bad input")
	ErrCryptoUnavailable = errors.New("crypto: unavailable")
	ErrUnauthorized      = errors.New("unauthorized")
)

// MethodRow — строка таблицы methods. Имя Method уже занято строковым типом.
type MethodRow struct {
	Code    string
	Title   string
	KeySize int
	Enabled bool
}

// UserKey — ключ RSA пользователя. Закрытый ключ хранится только запечатанным.
type UserKey struct {
	UserID        int64
	Bits          int
	PublicKeyPEM  string
	PrivateSealed []byte // формат sealed v1, contracts.md раздел 6
	KeygenMs      int64
	Status        string
	UpdatedAt     time.Time
}

// Message — зашифрованное сообщение с метриками.
type Message struct {
	ID           int64
	OwnerID      int64
	OwnerLogin   string
	Method       string
	KeySize      int
	Plaintext    string
	Ciphertext   []byte
	KuzKeySealed []byte // формат sealed v1
	KuzIV        []byte // IV шифрования сообщения, к sealed отношения не имеет
	KeygenMs     int64
	EncryptMs    int64
	DecryptMs    int64
	CreatedAt    time.Time
}

// Principal — аутентифицированный пользователь запроса.
type Principal struct {
	UserID int64
	Login  string
	Role   Role
}

// Store — хранилище сервиса.
type Store interface {
	Methods(ctx context.Context) ([]MethodRow, error)
	// SetMethodEnabled возвращает ErrNotFound, если code отсутствует.
	SetMethodEnabled(ctx context.Context, code string, on bool) error
	// UserKey возвращает ErrNotFound, если записи нет.
	UserKey(ctx context.Context, userID int64) (*UserKey, error)
	SaveUserKey(ctx context.Context, k *UserKey) error
	CreateMessage(ctx context.Context, m *Message) (int64, error)
	GetMessage(ctx context.Context, id int64) (*Message, error)
	// ListMessages: ownerID == nil — все сообщения (admin), иначе только свои.
	// Сортировка: created_at DESC, id DESC. limit 1..100, offset >= 0.
	ListMessages(ctx context.Context, ownerID *int64, limit, offset int) ([]*Message, error)
	SetDecryptMs(ctx context.Context, id, ms int64) error
	// LogOp: userID == nil — анонимный субъект (NULL в БД), не 0.
	LogOp(ctx context.Context, userID *int64, op, ref string, ok bool) error
}
