package sealed

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/ports"
)

// Фикстуры заведомо недействительны: это не боевые ключи.
var (
	testMasterKey  = strings.Repeat("11", masterKeyLen)
	otherMasterKey = strings.Repeat("22", masterKeyLen)
	testIV         = strings.Repeat("ab", ivLen)
)

// fakeCipher заменяет Крипто-модуль: «шифрует» XOR с ключом и IV. Этого
// достаточно, чтобы проверить формат контейнера и то, какие ключ и IV уходят
// в модуль; стойкость шифра здесь не проверяется.
type fakeCipher struct {
	ivHex      string
	keysErr    error
	encryptErr error
	decryptErr error
	emptyCT    bool

	decryptCalls int
}

func (f *fakeCipher) KuzKeys(context.Context) (keyHex, ivHex string, err error) {
	return strings.Repeat("ff", 32), f.ivHex, f.keysErr
}

func (f *fakeCipher) KuzEncrypt(_ context.Context, keyHex, ivHex string, data []byte) (ct []byte, ms int64, err error) {
	if f.encryptErr != nil {
		return nil, 0, f.encryptErr
	}
	if f.emptyCT {
		return nil, 1, nil
	}

	return xor(keyHex, ivHex, data), 1, nil
}

func (f *fakeCipher) KuzDecrypt(_ context.Context, keyHex, ivHex string, ct []byte) (data []byte, ms int64, err error) {
	f.decryptCalls++
	if f.decryptErr != nil {
		return nil, 0, f.decryptErr
	}

	return xor(keyHex, ivHex, ct), 1, nil
}

func xor(keyHex, ivHex string, data []byte) []byte {
	pad := []byte(keyHex + ivHex)
	out := make([]byte, len(data))

	for i, b := range data {
		out[i] = b ^ pad[i%len(pad)]
	}

	return out
}

func TestSealUnsealRoundTrip(t *testing.T) {
	plain := []byte("-----BEGIN RSA PRIVATE KEY-----\nTk9ULUEtS0VZ\n-----END RSA PRIVATE KEY-----\n")
	c := &fakeCipher{ivHex: testIV}

	blob, err := Seal(t.Context(), c, testMasterKey, plain)
	if err != nil {
		t.Fatalf("Seal() error = %v", err)
	}

	if blob[0] != version1 {
		t.Errorf("version = 0x%02x, want 0x01", blob[0])
	}

	if got := hex.EncodeToString(blob[1 : 1+ivLen]); got != testIV {
		t.Errorf("iv in container = %s, want %s", got, testIV)
	}

	if len(blob) != 1+ivLen+len(plain)+macLen {
		t.Errorf("len(blob) = %d, want %d", len(blob), 1+ivLen+len(plain)+macLen)
	}

	if bytes.Contains(blob, plain) {
		t.Error("container contains plaintext")
	}

	got, err := Unseal(t.Context(), c, testMasterKey, blob)
	if err != nil {
		t.Fatalf("Unseal() error = %v", err)
	}

	if !bytes.Equal(got, plain) {
		t.Errorf("Unseal() = %q, want %q", got, plain)
	}
}

func TestSealErrors(t *testing.T) {
	cryptoDown := errors.Join(ports.ErrCryptoUnavailable, errors.New("connection refused"))

	tests := []struct {
		name      string
		masterKey string
		cipher    *fakeCipher
		want      error
	}{
		{"мастер-ключ не hex", "zz", &fakeCipher{ivHex: testIV}, ErrMasterKey},
		{"мастер-ключ 16 байт", strings.Repeat("11", 16), &fakeCipher{ivHex: testIV}, ErrMasterKey},
		{"модуль не выдал iv", testMasterKey, &fakeCipher{keysErr: cryptoDown}, ports.ErrCryptoUnavailable},
		{"iv не той длины", testMasterKey, &fakeCipher{ivHex: "abab"}, ports.ErrCryptoUnavailable},
		{"ошибка шифрования", testMasterKey, &fakeCipher{ivHex: testIV, encryptErr: cryptoDown}, ports.ErrCryptoUnavailable},
		{"пустой шифртекст", testMasterKey, &fakeCipher{ivHex: testIV, emptyCT: true}, ports.ErrCryptoUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Seal(t.Context(), tt.cipher, tt.masterKey, []byte("secret")); !errors.Is(err, tt.want) {
				t.Errorf("Seal() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestUnsealErrors(t *testing.T) {
	blob, err := Seal(t.Context(), &fakeCipher{ivHex: testIV}, testMasterKey, []byte("secret"))
	if err != nil {
		t.Fatalf("Seal() error = %v", err)
	}

	tamper := func(i int) []byte {
		b := bytes.Clone(blob)
		b[i] ^= 0x01

		return b
	}

	tests := []struct {
		name      string
		masterKey string
		blob      []byte
		want      error
	}{
		{"короткий контейнер", testMasterKey, blob[:minLen-1], ErrFormat},
		{"неизвестная версия", testMasterKey, append([]byte{0x02}, blob[1:]...), ErrFormat},
		{"изменён iv", testMasterKey, tamper(1), ErrIntegrity},
		{"изменён шифртекст", testMasterKey, tamper(1 + ivLen), ErrIntegrity},
		{"изменён mac", testMasterKey, tamper(len(blob) - 1), ErrIntegrity},
		{"другой мастер-ключ", otherMasterKey, blob, ErrIntegrity},
		{"мастер-ключ не hex", "zz", blob, ErrMasterKey},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &fakeCipher{ivHex: testIV}

			if _, err := Unseal(t.Context(), c, tt.masterKey, tt.blob); !errors.Is(err, tt.want) {
				t.Errorf("Unseal() error = %v, want %v", err, tt.want)
			}

			if c.decryptCalls != 0 {
				t.Errorf("crypto module called %d times for a rejected container", c.decryptCalls)
			}
		})
	}
}

func TestUnsealCryptoError(t *testing.T) {
	blob, err := Seal(t.Context(), &fakeCipher{ivHex: testIV}, testMasterKey, []byte("secret"))
	if err != nil {
		t.Fatalf("Seal() error = %v", err)
	}

	c := &fakeCipher{ivHex: testIV, decryptErr: ports.ErrCryptoUnavailable}

	if _, err := Unseal(t.Context(), c, testMasterKey, blob); !errors.Is(err, ports.ErrCryptoUnavailable) {
		t.Errorf("Unseal() error = %v, want ErrCryptoUnavailable", err)
	}
}
