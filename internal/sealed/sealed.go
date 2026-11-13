// Package sealed запечатывает секреты под MASTER_KEY в формате sealed v1
// (contracts.md, раздел 6):
//
//	sealed = 0x01 | iv[16] | ciphertext | mac[32]
//	mac    = HMAC-SHA256(macKey, sealed[0 : len(sealed)-32])
//	macKey = HMAC-SHA256(MASTER_KEY, "smp/sealed/v1")
//
// Шифрование выполняет Крипто-модуль (Кузнечик с ключом MASTER_KEY), MAC
// считается средствами стандартной библиотеки и даёт целостность контейнера.
// Ошибки формата и MAC — внутренние: пользователь ключ не передавал, поэтому
// вызывающий отвечает 500, а не 400.
package sealed

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/ports"
)

const (
	version1 byte = 0x01

	masterKeyLen = 32
	ivLen        = 16
	macLen       = sha256.Size

	// minLen — версия, IV, хотя бы один байт шифртекста и MAC.
	minLen = 1 + ivLen + 1 + macLen

	macKeyLabel = "smp/sealed/v1"
)

// Ошибки контейнера. Проверяются через errors.Is.
var (
	ErrMasterKey = errors.New("sealed: master key must be 32 bytes in hex")
	ErrFormat    = errors.New("sealed: malformed container")
	ErrIntegrity = errors.New("sealed: integrity check failed")
)

// Cipher — операции Крипто-модуля, нужные запечатыванию. ports.CryptoClient
// реализует его целиком; узкий интерфейс объявлен здесь, у потребителя.
type Cipher interface {
	KuzKeys(ctx context.Context) (keyHex, ivHex string, err error)
	KuzEncrypt(ctx context.Context, keyHex, ivHex string, data []byte) (ct []byte, ms int64, err error)
	KuzDecrypt(ctx context.Context, keyHex, ivHex string, ct []byte) (data []byte, ms int64, err error)
}

var _ Cipher = ports.CryptoClient(nil)

// Seal шифрует plain под masterKeyHex и возвращает контейнер sealed v1.
//
// IV берётся из POST /kuznyechik/keys: так он случаен и генерируется там же,
// где вся остальная криптография. Ключ из того же ответа отбрасывается.
func Seal(ctx context.Context, c Cipher, masterKeyHex string, plain []byte) ([]byte, error) {
	masterKey, err := decodeMasterKey(masterKeyHex)
	if err != nil {
		return nil, err
	}

	_, ivHex, err := c.KuzKeys(ctx)
	if err != nil {
		return nil, fmt.Errorf("sealed: get iv: %w", err)
	}

	iv, err := hex.DecodeString(ivHex)
	if err != nil || len(iv) != ivLen {
		return nil, fmt.Errorf("sealed: crypto module returned iv of %d hex chars, want %d bytes: %w",
			len(ivHex), ivLen, ports.ErrCryptoUnavailable)
	}

	ct, _, err := c.KuzEncrypt(ctx, masterKeyHex, ivHex, plain)
	if err != nil {
		return nil, fmt.Errorf("sealed: encrypt: %w", err)
	}

	if len(ct) == 0 {
		return nil, fmt.Errorf("sealed: crypto module returned empty ciphertext: %w", ports.ErrCryptoUnavailable)
	}

	blob := make([]byte, 0, 1+ivLen+len(ct)+macLen)
	blob = append(blob, version1)
	blob = append(blob, iv...)
	blob = append(blob, ct...)
	blob = append(blob, computeMAC(masterKey, blob)...)

	return blob, nil
}

// Unseal проверяет контейнер sealed v1 и расшифровывает его под masterKeyHex.
// MAC проверяется до обращения к Крипто-модулю: подменённый контейнер
// не расшифровывается вовсе.
func Unseal(ctx context.Context, c Cipher, masterKeyHex string, blob []byte) ([]byte, error) {
	masterKey, err := decodeMasterKey(masterKeyHex)
	if err != nil {
		return nil, err
	}

	if len(blob) < minLen {
		return nil, fmt.Errorf("sealed: container is %d bytes, want at least %d: %w", len(blob), minLen, ErrFormat)
	}

	if blob[0] != version1 {
		return nil, fmt.Errorf("sealed: unsupported version 0x%02x: %w", blob[0], ErrFormat)
	}

	body, mac := blob[:len(blob)-macLen], blob[len(blob)-macLen:]
	if !hmac.Equal(mac, computeMAC(masterKey, body)) {
		return nil, ErrIntegrity
	}

	iv, ct := body[1:1+ivLen], body[1+ivLen:]

	plain, _, err := c.KuzDecrypt(ctx, masterKeyHex, hex.EncodeToString(iv), ct)
	if err != nil {
		return nil, fmt.Errorf("sealed: decrypt: %w", err)
	}

	return plain, nil
}

func decodeMasterKey(masterKeyHex string) ([]byte, error) {
	key, err := hex.DecodeString(masterKeyHex)
	if err != nil || len(key) != masterKeyLen {
		return nil, ErrMasterKey
	}

	return key, nil
}

// computeMAC считает MAC на ключе, выведенном из MASTER_KEY: тот же ключ
// шифрует данные, и использовать его ещё и для MAC напрямую нельзя.
func computeMAC(masterKey, data []byte) []byte {
	kdf := hmac.New(sha256.New, masterKey)
	kdf.Write([]byte(macKeyLabel))
	macKey := kdf.Sum(nil)

	m := hmac.New(sha256.New, macKey)
	m.Write(data)

	return m.Sum(nil)
}
