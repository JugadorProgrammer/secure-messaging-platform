package messages

import (
	"encoding/hex"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/httpx"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/methods"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/ports"
)

const decryptPath = "/api/decrypt"

func TestDecryptOwnMessage(t *testing.T) {
	tests := []struct {
		name   string
		method string
		req    func(m MessageDTO) DecryptRequest
		wantMs int64
	}{
		{
			name:   "RSA",
			method: methods.NameRSA,
			req: func(m MessageDTO) DecryptRequest {
				return DecryptRequest{Method: m.Method, Ciphertext: m.Ciphertext, PrivateKey: m.Keys.PrivateKey}
			},
			wantMs: 11,
		},
		{
			// Интерфейс не передаёт IV: он берётся из сохранённого сообщения.
			name:   "Кузнечик без IV",
			method: methods.NameKuznyechik,
			req: func(m MessageDTO) DecryptRequest {
				return DecryptRequest{Method: m.Method, Ciphertext: m.Ciphertext, Key: m.Keys.Key}
			},
			wantMs: 3,
		},
		{
			name:   "Кузнечик с IV в верхнем регистре",
			method: methods.NameKuznyechik,
			req: func(m MessageDTO) DecryptRequest {
				return DecryptRequest{
					Method: m.Method, Ciphertext: strings.ToUpper(m.Ciphertext),
					Key: strings.ToUpper(m.Keys.Key), IV: strings.ToUpper(m.Keys.IV),
				}
			},
			wantMs: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			m := e.encrypt(t, userPrincipal(), testText, tt.method)
			e.resetAudit()

			rec := e.post(t, decryptPath, userPrincipal(), tt.req(m))
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
			}

			if got := decodeBody[DecryptResponse](t, rec); got.Plaintext != testText {
				t.Errorf("plaintext = %q, want %q", got.Plaintext, testText)
			}

			if got := e.store.decryptMs[m.ID]; got != tt.wantMs {
				t.Errorf("decrypt_ms = %d, want %d", got, tt.wantMs)
			}

			if want := []auditRecord{{testUserID, opDecrypt, "1", true}}; !slices.Equal(e.auditLog(), want) {
				t.Errorf("audit = %+v, want %+v", e.auditLog(), want)
			}
		})
	}
}

// Чужой шифртекст расшифровывается, если известен ключ, но метрики чужого
// сообщения не меняются.
func TestDecryptForeignMessageKeepsMetrics(t *testing.T) {
	e := newEnv(t)
	m := e.encrypt(t, userPrincipal(), testText, methods.NameKuznyechik)

	rec := e.post(t, decryptPath, otherPrincipal(), DecryptRequest{Method: m.Method, Ciphertext: m.Ciphertext, Key: m.Keys.Key})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}

	if len(e.store.decryptMs) != 0 {
		t.Errorf("decrypt_ms updated for foreign message: %v", e.store.decryptMs)
	}
}

// Шифртекст не из базы: Кузнечик с IV работает, RSA тоже, ref в журнале пуст.
func TestDecryptExternalCiphertext(t *testing.T) {
	e := newEnv(t)

	ct := xor(testKuzKey+testKuzIV, []byte(testText))
	rec := e.post(t, decryptPath, userPrincipal(), DecryptRequest{
		Method: methods.NameKuznyechik, Ciphertext: hex.EncodeToString(ct), Key: testKuzKey, IV: testKuzIV,
	})

	if got := decodeBody[DecryptResponse](t, rec); rec.Code != http.StatusOK || got.Plaintext != testText {
		t.Fatalf("status = %d, plaintext = %q", rec.Code, got.Plaintext)
	}

	if want := []auditRecord{{testUserID, opDecrypt, "", true}}; !slices.Equal(e.auditLog(), want) {
		t.Errorf("audit = %+v, want %+v", e.auditLog(), want)
	}
}

func TestDecryptErrors(t *testing.T) {
	rsaCT := hex.EncodeToString([]byte("rsa:" + testText))

	tests := []struct {
		name       string
		setup      func(e *env)
		body       string
		wantStatus int
		wantCode   string
		audited    bool
	}{
		{
			name:       "тело не json",
			body:       `{"method":`,
			wantStatus: http.StatusBadRequest,
			wantCode:   httpx.CodeValidation,
		},
		{
			name:       "неизвестный метод",
			body:       `{"method":"AES","ciphertext":"00"}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   httpx.CodeValidation,
		},
		{
			name:       "шифртекст не hex",
			body:       `{"method":"RSA-32768","ciphertext":"xyz"}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   httpx.CodeBadCiphertext,
		},
		{
			name:       "пустой шифртекст",
			body:       `{"method":"RSA-32768","ciphertext":"  "}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   httpx.CodeBadCiphertext,
		},
		{
			name:       "закрытый ключ не PEM",
			body:       `{"method":"RSA-32768","ciphertext":"` + rsaCT + `","privateKey":"abc"}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   httpx.CodeBadKey,
		},
		{
			name:       "ключ Кузнечика короче 32 байт",
			body:       `{"method":"Кузнечик","ciphertext":"00","key":"abcd"}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   httpx.CodeBadKey,
		},
		{
			name:       "IV не 16 байт",
			body:       `{"method":"Кузнечик","ciphertext":"00","key":"` + testKuzKey + `","iv":"00"}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   httpx.CodeBadKey,
		},
		{
			name:       "IV не найден",
			body:       `{"method":"Кузнечик","ciphertext":"00","key":"` + testKuzKey + `"}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   httpx.CodeBadCiphertext,
		},
		{
			name: "чужой закрытый ключ",
			body: `{"method":"RSA-32768","ciphertext":"` + rsaCT + `","privateKey":` +
				`"-----BEGIN RSA PRIVATE KEY-----\nT1RIRVI=\n-----END RSA PRIVATE KEY-----\n"}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   httpx.CodeBadKey,
			audited:    true,
		},
		{
			name:  "крипто-модуль недоступен",
			setup: func(e *env) { e.crypto.kuzDecryptErr = ports.ErrCryptoUnavailable },
			body: `{"method":"Кузнечик","ciphertext":"00","key":"` + testKuzKey +
				`","iv":"` + testKuzIV + `"}`,
			wantStatus: http.StatusServiceUnavailable,
			wantCode:   httpx.CodeCryptoUnavailable,
			audited:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			if tt.setup != nil {
				tt.setup(e)
			}

			rec := e.do(t, userPrincipal(), http.MethodPost, decryptPath, "", tt.body)
			assertError(t, rec, tt.wantStatus, tt.wantCode)

			var want []auditRecord
			if tt.audited {
				want = []auditRecord{{testUserID, opDecrypt, "", false}}
			}

			if got := e.auditLog(); !slices.Equal(got, want) {
				t.Errorf("audit = %+v, want %+v", got, want)
			}
		})
	}
}

func TestDecryptDoesNotLogSecrets(t *testing.T) {
	e := newEnv(t)
	m := e.encrypt(t, userPrincipal(), testText, methods.NameRSA)

	e.post(t, decryptPath, userPrincipal(), DecryptRequest{Method: m.Method, Ciphertext: m.Ciphertext, PrivateKey: m.Keys.PrivateKey})
	e.post(t, decryptPath, userPrincipal(), DecryptRequest{Method: m.Method, Ciphertext: m.Ciphertext, PrivateKey: "bad"})

	for _, secret := range []string{testText, "UFJJVi1GSVhUVVJF", "PRIVATE KEY"} {
		if strings.Contains(e.log.String(), secret) {
			t.Errorf("log contains secret %q", secret)
		}
	}
}
