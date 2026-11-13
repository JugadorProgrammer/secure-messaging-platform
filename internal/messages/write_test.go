package messages

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/auth"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/httpx"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/ports"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/sealed"
)

// Фикстуры заведомо недействительны: это метки для поиска утечек в логе.
const (
	testText     = "Секретное сообщение"
	testPubPEM   = "-----BEGIN RSA PUBLIC KEY-----\nUFVCLUZJWFRVUkU=\n-----END RSA PUBLIC KEY-----\n"
	testPrivPEM  = "-----BEGIN RSA PRIVATE KEY-----\nUFJJVi1GSVhUVVJF\n-----END RSA PRIVATE KEY-----\n"
	testMaxBytes = 64 << 10
	testUserID   = 7
	testLogin    = "ivanov"
)

var (
	testMasterKey = strings.Repeat("11", 32)
	testKuzKey    = strings.Repeat("ab", 32)
	testKuzIV     = strings.Repeat("cd", 16)
	testCreatedAt = time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
)

// fakeStore — хранилище: методы шифрования и запись сообщений. Остальные
// методы ports.Store не вызываются, и встроенный nil-интерфейс сделает
// случайный вызов заметным.
type fakeStore struct {
	ports.Store

	methods    []ports.MethodRow
	methodsErr error
	createErr  error
	created    []*ports.Message
}

func (s *fakeStore) Methods(context.Context) ([]ports.MethodRow, error) {
	return s.methods, s.methodsErr
}

func (s *fakeStore) CreateMessage(_ context.Context, m *ports.Message) (int64, error) {
	if s.createErr != nil {
		return 0, s.createErr
	}

	s.created = append(s.created, m)
	m.ID, m.CreatedAt = int64(len(s.created)), testCreatedAt

	return m.ID, nil
}

// fakeCrypto — Крипто-модуль: Кузнечик — XOR, чтобы sealed работал по-настоящему,
// RSA — пометка префиксом. Ошибки задаются по операциям.
type fakeCrypto struct {
	ports.CryptoClient

	kuzKeysErr    error
	kuzEncryptErr error
	rsaEncryptErr error
	kuzKeysCalls  int
}

func (f *fakeCrypto) KuzKeys(context.Context) (keyHex, ivHex string, err error) {
	f.kuzKeysCalls++

	// Первый вызов — ключ сообщения, следующие — IV для sealed.
	return testKuzKey, testKuzIV, f.kuzKeysErr
}

func (f *fakeCrypto) KuzEncrypt(_ context.Context, keyHex, ivHex string, data []byte) (ct []byte, ms int64, err error) {
	if f.kuzEncryptErr != nil {
		return nil, 0, f.kuzEncryptErr
	}

	return xor(keyHex+ivHex, data), 4, nil
}

func (f *fakeCrypto) KuzDecrypt(_ context.Context, keyHex, ivHex string, ct []byte) (data []byte, ms int64, err error) {
	return xor(keyHex+ivHex, ct), 1, nil
}

func (f *fakeCrypto) RSAEncrypt(_ context.Context, pubPEM string, data []byte) (ct []byte, ms int64, err error) {
	if f.rsaEncryptErr != nil {
		return nil, 0, f.rsaEncryptErr
	}

	if pubPEM != testPubPEM {
		return nil, 0, ports.ErrCryptoBadInput
	}

	return append([]byte("rsa:"), data...), 9, nil
}

func xor(pad string, data []byte) []byte {
	out := make([]byte, len(data))
	for i, b := range data {
		out[i] = b ^ pad[i%len(pad)]
	}

	return out
}

type fakeKeys struct {
	key   *ports.UserKey
	err   error
	calls []int64
}

func (k *fakeKeys) EnsureUserKey(_ context.Context, userID int64) (*ports.UserKey, error) {
	k.calls = append(k.calls, userID)

	return k.key, k.err
}

type auditRecord struct {
	userID int64
	op     string
	ref    string
	ok     bool
}

type env struct {
	store  *fakeStore
	crypto *fakeCrypto
	keys   *fakeKeys
	audit  []auditRecord
	log    bytes.Buffer
	mu     sync.Mutex
}

// newEnv собирает окружение: оба метода включены, ключ RSA пользователя готов
// и запечатан тем же sealed, что и в работе.
func newEnv(t *testing.T) *env {
	t.Helper()

	e := &env{
		store: &fakeStore{methods: []ports.MethodRow{
			{Code: "kuznyechik", Title: "Кузнечик", KeySize: 256, Enabled: true},
			{Code: "rsa", Title: "RSA", KeySize: 2048, Enabled: true},
		}},
		crypto: &fakeCrypto{},
	}

	privSealed, err := sealed.Seal(t.Context(), e.crypto, testMasterKey, []byte(testPrivPEM))
	if err != nil {
		t.Fatalf("seal test key: %v", err)
	}

	e.crypto.kuzKeysCalls = 0
	e.keys = &fakeKeys{key: &ports.UserKey{
		UserID: testUserID, Bits: 2048, PublicKeyPEM: testPubPEM, PrivateSealed: privSealed,
		KeygenMs: 1500, Status: ports.KeyStatusReady,
	}}

	return e
}

func (e *env) handler() http.Handler {
	log := slog.New(slog.NewTextHandler(&e.log, &slog.HandlerOptions{Level: slog.LevelDebug}))
	audit := func(_ context.Context, userID *int64, op, ref string, ok bool) {
		e.mu.Lock()
		defer e.mu.Unlock()

		e.audit = append(e.audit, auditRecord{userID: *userID, op: op, ref: ref, ok: ok})
	}

	h := NewWriteHandler(e.store, e.crypto, e.keys, audit,
		WriteConfig{MasterKeyHex: testMasterKey, MaxTextBytes: testMaxBytes}, log)

	return httpx.BodyLimit(h.BodyLimit())(http.HandlerFunc(h.Create))
}

func (e *env) post(t *testing.T, contentType, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequestWithContext(
		auth.WithPrincipal(t.Context(), auth.Principal{UserID: testUserID, Login: testLogin, Role: auth.RoleUser}),
		http.MethodPost, "/api/messages", strings.NewReader(body))
	req.Header.Set("Content-Type", contentType)

	rec := httptest.NewRecorder()
	e.handler().ServeHTTP(rec, req)

	return rec
}

func encryptBody(t *testing.T, text string, method ports.Method) string {
	t.Helper()

	b, err := json.Marshal(EncryptRequest{Text: text, Method: method})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	return string(b)
}

func decodeMessage(t *testing.T, rec *httptest.ResponseRecorder) MessageDTO {
	t.Helper()

	var got MessageDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body %q: %v", rec.Body.String(), err)
	}

	return got
}

func assertError(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()

	if rec.Code != status {
		t.Fatalf("status = %d, want %d (body %q)", rec.Code, status, rec.Body.String())
	}

	var body httpx.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error body %q: %v", rec.Body.String(), err)
	}

	if body.Error.Code != code || body.Error.Message == "" {
		t.Errorf("error = %+v, want code %q with message", body.Error, code)
	}
}

func TestCreateKuznyechik(t *testing.T) {
	e := newEnv(t)

	rec := e.post(t, "application/json; charset=utf-8", encryptBody(t, testText, ports.MethodKuznyechik))
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body %q)", rec.Code, rec.Body.String())
	}

	if loc := rec.Header().Get("Location"); loc != "/api/messages/1" {
		t.Errorf("Location = %q, want /api/messages/1", loc)
	}

	got := decodeMessage(t, rec)

	wantCT := hex.EncodeToString(xor(testKuzKey+testKuzIV, []byte(testText)))
	if got.ID != 1 || got.Owner != testLogin || got.Method != ports.MethodKuznyechik || got.KeySize != kuzKeyBits ||
		got.Plaintext != testText || got.Ciphertext != wantCT || !got.CreatedAt.Equal(testCreatedAt) {
		t.Errorf("message = %+v", got)
	}

	if got.Keys == nil || got.Keys.KeyHex != testKuzKey || got.Keys.IVHex != testKuzIV || got.Keys.PrivateKeyPEM != "" {
		t.Errorf("keys = %+v, want key_hex and iv_hex only", got.Keys)
	}

	if got.Metrics != (Metrics{KeygenMs: 0, EncryptMs: 4}) {
		t.Errorf("metrics = %+v, want keygen_ms 0 and encrypt_ms 4", got.Metrics)
	}

	saved := e.store.created[0]
	if !bytes.Equal(saved.KuzIV, mustHex(t, testKuzIV)) {
		t.Errorf("saved kuz_iv = %x, want %s", saved.KuzIV, testKuzIV)
	}

	key, err := sealed.Unseal(t.Context(), e.crypto, testMasterKey, saved.KuzKeySealed)
	if err != nil || string(key) != testKuzKey {
		t.Errorf("Unseal(kuz_key_sealed) = %q, %v, want the message key", key, err)
	}

	if want := []auditRecord{{testUserID, opEncrypt, "1", true}}; !equalAudit(e.audit, want) {
		t.Errorf("audit = %+v, want %+v", e.audit, want)
	}
}

func TestCreateRSA(t *testing.T) {
	e := newEnv(t)

	rec := e.post(t, "application/json", encryptBody(t, testText, ports.MethodRSA))
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (body %q)", rec.Code, rec.Body.String())
	}

	got := decodeMessage(t, rec)

	if got.Method != ports.MethodRSA || got.KeySize != 2048 || got.Ciphertext != hex.EncodeToString([]byte("rsa:"+testText)) {
		t.Errorf("message = %+v", got)
	}

	if got.Keys == nil || got.Keys.PublicKeyPEM != testPubPEM || got.Keys.PrivateKeyPEM != testPrivPEM || got.Keys.KeyHex != "" {
		t.Errorf("keys = %+v, want the user's PEM pair", got.Keys)
	}

	if got.Metrics != (Metrics{KeygenMs: 1500, EncryptMs: 9}) {
		t.Errorf("metrics = %+v, want keygen_ms from the key and encrypt_ms 9", got.Metrics)
	}

	saved := e.store.created[0]
	if saved.KuzKeySealed != nil || saved.KuzIV != nil {
		t.Errorf("rsa message has kuznyechik fields: %+v", saved)
	}

	if len(e.keys.calls) != 1 || e.keys.calls[0] != testUserID {
		t.Errorf("EnsureUserKey calls = %v, want [%d]", e.keys.calls, testUserID)
	}
}

func TestCreateErrors(t *testing.T) {
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
			body:        `{"text":"x","method":"rsa"}`,
			wantStatus:  http.StatusUnsupportedMediaType,
			wantCode:    httpx.CodeUnsupportedMediaType,
		},
		{
			name:       "тело не json",
			body:       `{"text":`,
			wantStatus: http.StatusBadRequest,
			wantCode:   httpx.CodeValidation,
		},
		{
			name:       "тело больше лимита",
			body:       `{"text":"` + strings.Repeat("a", testMaxBytes+bodyOverhead) + `","method":"rsa"}`,
			wantStatus: http.StatusRequestEntityTooLarge,
			wantCode:   httpx.CodePayloadTooLarge,
		},
		{
			name:       "пустой текст",
			body:       `{"text":"","method":"rsa"}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   httpx.CodeValidation,
		},
		{
			name:       "текст из пробелов",
			body:       `{"text":" \n\t ","method":"kuznyechik"}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   httpx.CodeValidation,
		},
		{
			name:       "текст больше 64 КиБ",
			body:       `{"text":"` + strings.Repeat("a", testMaxBytes+1) + `","method":"rsa"}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   httpx.CodeValidation,
		},
		{
			name:       "неизвестный метод",
			body:       `{"text":"x","method":"aes"}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   httpx.CodeValidation,
		},
		{
			name: "метод выключен",
			setup: func(e *env) {
				e.store.methods[0].Enabled = false
			},
			body:       `{"text":"x","method":"kuznyechik"}`,
			wantStatus: http.StatusConflict,
			wantCode:   httpx.CodeMethodDisabled,
		},
		{
			name:       "ключ rsa генерируется",
			setup:      func(e *env) { e.keys.key.Status = ports.KeyStatusGenerating },
			body:       `{"text":"x","method":"rsa"}`,
			wantStatus: http.StatusConflict,
			wantCode:   httpx.CodeKeyGenerating,
		},
		{
			name:       "крипто-модуль недоступен",
			setup:      func(e *env) { e.crypto.kuzEncryptErr = ports.ErrCryptoUnavailable },
			body:       `{"text":"x","method":"kuznyechik"}`,
			wantStatus: http.StatusServiceUnavailable,
			wantCode:   httpx.CodeCryptoUnavailable,
			wantAudit:  []auditRecord{{testUserID, opEncrypt, "", false}},
		},
		{
			name:       "крипто-модуль не выдал ключ",
			setup:      func(e *env) { e.crypto.kuzKeysErr = ports.ErrCryptoUnavailable },
			body:       `{"text":"x","method":"kuznyechik"}`,
			wantStatus: http.StatusServiceUnavailable,
			wantCode:   httpx.CodeCryptoUnavailable,
			wantAudit:  []auditRecord{{testUserID, opEncrypt, "", false}},
		},
		{
			name:       "крипто-модуль отверг текст",
			setup:      func(e *env) { e.crypto.rsaEncryptErr = ports.ErrCryptoBadInput },
			body:       `{"text":"x","method":"rsa"}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   httpx.CodeValidation,
			wantAudit:  []auditRecord{{testUserID, opEncrypt, "", false}},
		},
		{
			name:       "повреждённый закрытый ключ",
			setup:      func(e *env) { e.keys.key.PrivateSealed[len(e.keys.key.PrivateSealed)-1] ^= 1 },
			body:       `{"text":"x","method":"rsa"}`,
			wantStatus: http.StatusInternalServerError,
			wantCode:   httpx.CodeInternal,
			wantAudit:  []auditRecord{{testUserID, opEncrypt, "", false}},
		},
		{
			name:       "ошибка менеджера ключей",
			setup:      func(e *env) { e.keys.err = errors.New("db down") },
			body:       `{"text":"x","method":"rsa"}`,
			wantStatus: http.StatusInternalServerError,
			wantCode:   httpx.CodeInternal,
			wantAudit:  []auditRecord{{testUserID, opEncrypt, "", false}},
		},
		{
			name:       "ошибка чтения методов",
			setup:      func(e *env) { e.store.methodsErr = errors.New("db down") },
			body:       `{"text":"x","method":"rsa"}`,
			wantStatus: http.StatusInternalServerError,
			wantCode:   httpx.CodeInternal,
			wantAudit:  []auditRecord{{testUserID, opEncrypt, "", false}},
		},
		{
			name:       "ошибка сохранения",
			setup:      func(e *env) { e.store.createErr = errors.New("db down") },
			body:       `{"text":"x","method":"kuznyechik"}`,
			wantStatus: http.StatusInternalServerError,
			wantCode:   httpx.CodeInternal,
			wantAudit:  []auditRecord{{testUserID, opEncrypt, "", false}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			if tt.setup != nil {
				tt.setup(e)
			}

			contentType := tt.contentType
			if contentType == "" {
				contentType = "application/json"
			}

			rec := e.post(t, contentType, tt.body)
			assertError(t, rec, tt.wantStatus, tt.wantCode)

			if len(e.store.created) != 0 {
				t.Errorf("message saved despite error: %+v", e.store.created)
			}

			if !equalAudit(e.audit, tt.wantAudit) {
				t.Errorf("audit = %+v, want %+v", e.audit, tt.wantAudit)
			}
		})
	}
}

func TestCreateWithoutPrincipal(t *testing.T) {
	e := newEnv(t)

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/messages",
		strings.NewReader(encryptBody(t, testText, ports.MethodRSA)))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	e.handler().ServeHTTP(rec, req)

	assertError(t, rec, http.StatusInternalServerError, httpx.CodeInternal)
}

// TestCreateClientCanceled: клиент ушёл во время шифрования. Ответ некому
// читать, поэтому он не пишется и не превращается в 500 в логе.
func TestCreateClientCanceled(t *testing.T) {
	e := newEnv(t)
	e.crypto.kuzEncryptErr = context.Canceled

	rec := e.post(t, "application/json", encryptBody(t, testText, ports.MethodKuznyechik))

	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want no response for canceled request", rec.Body.String())
	}

	if strings.Contains(e.log.String(), "level=ERROR") {
		t.Errorf("canceled request logged as error: %q", e.log.String())
	}
}

func TestCreateDoesNotLogSecrets(t *testing.T) {
	e := newEnv(t)

	e.post(t, "application/json", encryptBody(t, testText, ports.MethodRSA))
	e.post(t, "application/json", encryptBody(t, testText, ports.MethodKuznyechik))

	e.store.createErr = errors.New("db down")
	e.post(t, "application/json", encryptBody(t, testText, ports.MethodRSA))

	logged := e.log.String()
	if !strings.Contains(logged, "encrypt message") {
		t.Fatalf("expected an error record, got %q", logged)
	}

	for _, secret := range []string{testText, "UFJJVi1GSVhUVVJF", "PRIVATE KEY", testKuzKey, testMasterKey} {
		if strings.Contains(logged, secret) {
			t.Errorf("log contains secret %q", secret)
		}
	}
}

func TestBodyLimit(t *testing.T) {
	h := NewWriteHandler(nil, nil, nil, nil, WriteConfig{MaxTextBytes: testMaxBytes}, nil)

	if got, want := h.BodyLimit(), int64(testMaxBytes+bodyOverhead); got != want {
		t.Errorf("BodyLimit() = %d, want %d", got, want)
	}
}

func equalAudit(got, want []auditRecord) bool {
	if len(got) != len(want) {
		return false
	}

	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}

	return true
}

func mustHex(t *testing.T, s string) []byte {
	t.Helper()

	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("decode hex %q: %v", s, err)
	}

	return b
}
