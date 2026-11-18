package messages

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/httpx"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/methods"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/ports"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/sealed"
)

const encryptPath = "/api/encrypt"

func TestEncryptKuznyechik(t *testing.T) {
	e := newEnv(t)

	rec := e.do(t, userPrincipal(), http.MethodPost, encryptPath, "application/json; charset=utf-8",
		mustJSON(t, EncryptRequest{Plaintext: testText, Method: methods.NameKuznyechik}))
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body %q)", rec.Code, rec.Body.String())
	}

	if cc := rec.Header().Get("Cache-Control"); cc != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store: в ответе ключи", cc)
	}

	got := decodeBody[MessageDTO](t, rec)

	wantCT := hex.EncodeToString(xor(testKuzKey+testKuzIV, []byte(testText)))
	if got.ID != 1 || got.User != (UserDTO{ID: testUserID, Name: testLogin}) || got.Method != methods.NameKuznyechik ||
		got.Plaintext != testText || got.Ciphertext != wantCT || !got.Timestamp.Equal(testCreatedAt) {
		t.Errorf("message = %+v", got)
	}

	if got.Keys != (KeysDTO{Key: testKuzKey, IV: testKuzIV}) {
		t.Errorf("keys = %+v, want key and iv only", got.Keys)
	}

	if got.Metrics != (Metrics{KeygenMs: 0, EncryptMs: 4}) {
		t.Errorf("metrics = %+v, want keygen_ms 0 and encrypt_ms 4", got.Metrics)
	}

	saved := e.store.created[0]
	if saved.KeySize != kuzKeyBits || !bytes.Equal(saved.KuzIV, mustHex(t, testKuzIV)) {
		t.Errorf("saved key_size = %d, kuz_iv = %x", saved.KeySize, saved.KuzIV)
	}

	key, err := sealed.Unseal(t.Context(), e.crypto, testMasterKey, saved.KuzKeySealed)
	if err != nil || string(key) != testKuzKey {
		t.Errorf("Unseal(kuz_key_sealed) = %q, %v, want the message key", key, err)
	}

	if want := []auditRecord{{testUserID, opEncrypt, "1", true}}; !slices.Equal(e.auditLog(), want) {
		t.Errorf("audit = %+v, want %+v", e.auditLog(), want)
	}
}

func TestEncryptRSA(t *testing.T) {
	e := newEnv(t)

	got := e.encrypt(t, userPrincipal(), testText, methods.NameRSA)

	if got.Method != methods.NameRSA || got.Ciphertext != hex.EncodeToString([]byte("rsa:"+testText)) {
		t.Errorf("message = %+v", got)
	}

	if got.Keys != (KeysDTO{PublicKey: testPubPEM, PrivateKey: testPrivPEM}) {
		t.Errorf("keys = %+v, want the user's PEM pair", got.Keys)
	}

	if got.Metrics != (Metrics{KeygenMs: 1500, EncryptMs: 9}) {
		t.Errorf("metrics = %+v, want keygen_ms from the key and encrypt_ms 9", got.Metrics)
	}

	saved := e.store.created[0]
	if saved.KeySize != 2048 || saved.KuzKeySealed != nil || saved.KuzIV != nil {
		t.Errorf("saved rsa message = %+v", saved)
	}

	if len(e.keys.calls) != 1 || e.keys.calls[0] != testUserID {
		t.Errorf("EnsureUserKey calls = %v, want [%d]", e.keys.calls, testUserID)
	}
}

// Интерфейс шлёт имя метода, но код тоже принимается: им пользуются скрипты
// и старые клиенты.
func TestEncryptAcceptsMethodCode(t *testing.T) {
	e := newEnv(t)

	if got := e.encrypt(t, userPrincipal(), testText, string(ports.MethodKuznyechik)); got.Method != methods.NameKuznyechik {
		t.Errorf("method = %q, want %q", got.Method, methods.NameKuznyechik)
	}
}

func TestEncryptErrors(t *testing.T) {
	failed := []auditRecord{{testUserID, opEncrypt, "", false}}

	tests := []struct {
		name        string
		setup       func(e *env)
		contentType string
		body        string
		wantStatus  int
		wantCode    string
		wantAudit   []auditRecord
	}{
		{
			name:        "неверный content-type",
			contentType: "text/plain",
			body:        `{"plaintext":"x","method":"RSA-32768"}`,
			wantStatus:  http.StatusUnsupportedMediaType,
			wantCode:    httpx.CodeUnsupportedMediaType,
		},
		{
			name:       "тело не json",
			body:       `{"plaintext":`,
			wantStatus: http.StatusBadRequest,
			wantCode:   httpx.CodeValidation,
		},
		{
			name:       "тело больше лимита",
			body:       `{"plaintext":"` + strings.Repeat("a", testMaxBytes+bodyOverhead) + `","method":"rsa"}`,
			wantStatus: http.StatusRequestEntityTooLarge,
			wantCode:   httpx.CodePayloadTooLarge,
		},
		{
			name:       "пустой текст",
			body:       `{"plaintext":"","method":"RSA-32768"}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   httpx.CodeValidation,
		},
		{
			name:       "текст из пробелов",
			body:       `{"plaintext":" \n\t ","method":"Кузнечик"}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   httpx.CodeValidation,
		},
		{
			name:       "текст больше 64 КиБ",
			body:       `{"plaintext":"` + strings.Repeat("a", testMaxBytes+1) + `","method":"RSA-32768"}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   httpx.CodeValidation,
		},
		{
			name:       "неизвестный метод",
			body:       `{"plaintext":"x","method":"AES"}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   httpx.CodeValidation,
		},
		{
			name:       "метод выключен",
			setup:      func(e *env) { e.store.methods[0].Enabled = false },
			body:       `{"plaintext":"x","method":"Кузнечик"}`,
			wantStatus: http.StatusConflict,
			wantCode:   httpx.CodeMethodDisabled,
		},
		{
			name:       "ключ rsa генерируется",
			setup:      func(e *env) { e.keys.key.Status = ports.KeyStatusGenerating },
			body:       `{"plaintext":"x","method":"RSA-32768"}`,
			wantStatus: http.StatusConflict,
			wantCode:   httpx.CodeKeyGenerating,
		},
		{
			name:       "крипто-модуль недоступен",
			setup:      func(e *env) { e.crypto.kuzEncryptErr = ports.ErrCryptoUnavailable },
			body:       `{"plaintext":"x","method":"Кузнечик"}`,
			wantStatus: http.StatusServiceUnavailable,
			wantCode:   httpx.CodeCryptoUnavailable,
			wantAudit:  failed,
		},
		{
			name:       "крипто-модуль не выдал ключ",
			setup:      func(e *env) { e.crypto.kuzKeysErr = ports.ErrCryptoUnavailable },
			body:       `{"plaintext":"x","method":"Кузнечик"}`,
			wantStatus: http.StatusServiceUnavailable,
			wantCode:   httpx.CodeCryptoUnavailable,
			wantAudit:  failed,
		},
		{
			name:       "крипто-модуль отверг текст",
			setup:      func(e *env) { e.crypto.rsaEncryptErr = ports.ErrCryptoBadInput },
			body:       `{"plaintext":"x","method":"RSA-32768"}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   httpx.CodeValidation,
			wantAudit:  failed,
		},
		{
			name:       "повреждённый закрытый ключ",
			setup:      func(e *env) { e.keys.key.PrivateSealed[len(e.keys.key.PrivateSealed)-1] ^= 1 },
			body:       `{"plaintext":"x","method":"RSA-32768"}`,
			wantStatus: http.StatusInternalServerError,
			wantCode:   httpx.CodeInternal,
			wantAudit:  failed,
		},
		{
			name:       "ошибка менеджера ключей",
			setup:      func(e *env) { e.keys.err = errors.New("db down") },
			body:       `{"plaintext":"x","method":"RSA-32768"}`,
			wantStatus: http.StatusInternalServerError,
			wantCode:   httpx.CodeInternal,
			wantAudit:  failed,
		},
		{
			name:       "ошибка чтения методов",
			setup:      func(e *env) { e.store.methodsErr = errors.New("db down") },
			body:       `{"plaintext":"x","method":"RSA-32768"}`,
			wantStatus: http.StatusInternalServerError,
			wantCode:   httpx.CodeInternal,
			wantAudit:  failed,
		},
		{
			name:       "ошибка сохранения",
			setup:      func(e *env) { e.store.createErr = errors.New("db down") },
			body:       `{"plaintext":"x","method":"Кузнечик"}`,
			wantStatus: http.StatusInternalServerError,
			wantCode:   httpx.CodeInternal,
			wantAudit:  failed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			if tt.setup != nil {
				tt.setup(e)
			}

			rec := e.do(t, userPrincipal(), http.MethodPost, encryptPath, tt.contentType, tt.body)
			assertError(t, rec, tt.wantStatus, tt.wantCode)

			if len(e.store.created) != 0 {
				t.Errorf("message saved despite error: %+v", e.store.created)
			}

			if got := e.auditLog(); !slices.Equal(got, tt.wantAudit) {
				t.Errorf("audit = %+v, want %+v", got, tt.wantAudit)
			}
		})
	}
}

// Без пользователя в контексте обработчик собран неверно: это 500, а не 401.
func TestHandlersWithoutPrincipal(t *testing.T) {
	h := newEnv(t).newHandler()

	for name, handler := range map[string]http.HandlerFunc{
		"encrypt": h.Encrypt, "decrypt": h.Decrypt, "list": h.List, "pdf": h.ExportPDF,
	} {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", strings.NewReader(`{}`))
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			assertError(t, rec, http.StatusInternalServerError, httpx.CodeInternal)
		})
	}
}

// TestEncryptClientCanceled: клиент ушёл во время шифрования. Ответ некому
// читать, поэтому он не пишется и не превращается в 500 в логе.
func TestEncryptClientCanceled(t *testing.T) {
	e := newEnv(t)
	e.crypto.kuzEncryptErr = context.Canceled

	rec := e.post(t, encryptPath, userPrincipal(), EncryptRequest{Plaintext: testText, Method: methods.NameKuznyechik})

	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want no response for canceled request", rec.Body.String())
	}

	if strings.Contains(e.log.String(), "level=ERROR") {
		t.Errorf("canceled request logged as error: %q", e.log.String())
	}
}

func TestEncryptDoesNotLogSecrets(t *testing.T) {
	e := newEnv(t)

	e.encrypt(t, userPrincipal(), testText, methods.NameRSA)
	e.encrypt(t, userPrincipal(), testText, methods.NameKuznyechik)

	e.store.createErr = errors.New("db down")
	e.post(t, encryptPath, userPrincipal(), EncryptRequest{Plaintext: testText, Method: methods.NameRSA})

	logged := e.log.String()
	if !strings.Contains(logged, "level=ERROR") {
		t.Fatalf("expected an error record, got %q", logged)
	}

	for _, secret := range []string{testText, "UFJJVi1GSVhUVVJF", "PRIVATE KEY", testKuzKey, testMasterKey} {
		if strings.Contains(logged, secret) {
			t.Errorf("log contains secret %q", secret)
		}
	}
}

func TestBodyLimits(t *testing.T) {
	h := NewHandler(nil, nil, nil, nil, Config{MaxTextBytes: testMaxBytes}, nil)

	tests := []struct {
		name      string
		got, want int64
	}{
		{"encrypt", h.EncryptBodyLimit(), testMaxBytes + bodyOverhead},
		{"decrypt", h.DecryptBodyLimit(), 2*testMaxBytes + bodyOverhead},
		{"export", h.ExportBodyLimit(), maxExportBodyBytes},
	}

	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s limit = %d, want %d", tt.name, tt.got, tt.want)
		}
	}
}
