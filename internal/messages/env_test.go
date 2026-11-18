package messages

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha512"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/auth"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/httpx"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/keys"
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
	otherUserID  = 8
	otherLogin   = "petrov"
	adminUserID  = 1
)

var (
	testMasterKey = strings.Repeat("11", 32)
	testKuzKey    = strings.Repeat("ab", 32)
	testKuzIV     = strings.Repeat("cd", 16)
	testCreatedAt = time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	testNow       = time.Date(2026, 11, 14, 9, 30, 0, 0, time.UTC)
)

// fakeStore — хранилище в памяти. Методы ports.Store, которые обработчик не
// вызывает, остаются nil-интерфейсом: случайный вызов упадёт заметно.
type fakeStore struct {
	ports.Store

	mu sync.Mutex

	methods    []ports.MethodRow
	methodsErr error
	createErr  error
	getErr     error
	listErr    error
	keyErr     error
	created    []*ports.Message
	userKeys   map[int64]*ports.UserKey
	decryptMs  map[int64]int64
	listOwners []*int64
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

func (s *fakeStore) GetMessage(_ context.Context, id int64) (*ports.Message, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}

	for _, m := range s.created {
		if m.ID == id {
			return m, nil
		}
	}

	return nil, ports.ErrNotFound
}

// ListMessages отдаёт сообщения от новых к старым, как настоящее хранилище.
func (s *fakeStore) ListMessages(_ context.Context, ownerID *int64, limit, offset int) ([]*ports.Message, error) {
	s.listOwners = append(s.listOwners, ownerID)

	if s.listErr != nil {
		return nil, s.listErr
	}

	var all []*ports.Message

	for _, m := range slices.Backward(s.created) {
		if ownerID == nil || m.OwnerID == *ownerID {
			all = append(all, m)
		}
	}

	if offset >= len(all) {
		return nil, nil
	}

	return all[offset:min(offset+limit, len(all))], nil
}

func (s *fakeStore) MessageByCiphertext(_ context.Context, method string, ct []byte) (*ports.Message, error) {
	for _, m := range slices.Backward(s.created) {
		if m.Method == method && bytes.Equal(m.Ciphertext, ct) {
			return m, nil
		}
	}

	return nil, ports.ErrNotFound
}

func (s *fakeStore) SetDecryptMs(_ context.Context, id, ms int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.decryptMs == nil {
		s.decryptMs = make(map[int64]int64)
	}

	s.decryptMs[id] = ms

	return nil
}

func (s *fakeStore) UserKey(_ context.Context, userID int64) (*ports.UserKey, error) {
	if s.keyErr != nil {
		return nil, s.keyErr
	}

	k, ok := s.userKeys[userID]
	if !ok {
		return nil, ports.ErrNotFound
	}

	return k, nil
}

// fakeCrypto — Крипто-модуль: Кузнечик — XOR, чтобы sealed работал
// по-настоящему, RSA — пометка префиксом, Стрибог — SHA-512, подпись —
// префикс к хешу. Ошибки задаются по операциям.
type fakeCrypto struct {
	ports.CryptoClient

	kuzKeysErr    error
	kuzEncryptErr error
	kuzDecryptErr error
	rsaEncryptErr error
	hashErr       error
	signErr       error
	kuzKeysCalls  int
	signed        [][]byte
}

func (f *fakeCrypto) KuzKeys(context.Context) (keyHex, ivHex string, err error) {
	f.kuzKeysCalls++

	return testKuzKey, testKuzIV, f.kuzKeysErr
}

func (f *fakeCrypto) KuzEncrypt(_ context.Context, keyHex, ivHex string, data []byte) (ct []byte, ms int64, err error) {
	if f.kuzEncryptErr != nil {
		return nil, 0, f.kuzEncryptErr
	}

	return xor(keyHex+ivHex, data), 4, nil
}

func (f *fakeCrypto) KuzDecrypt(_ context.Context, keyHex, ivHex string, ct []byte) (data []byte, ms int64, err error) {
	if f.kuzDecryptErr != nil {
		return nil, 0, f.kuzDecryptErr
	}

	return xor(keyHex+ivHex, ct), 3, nil
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

func (f *fakeCrypto) RSADecrypt(_ context.Context, privPEM string, ct []byte) (data []byte, ms int64, err error) {
	plain, ok := bytes.CutPrefix(ct, []byte("rsa:"))
	if privPEM != testPrivPEM || !ok {
		return nil, 0, ports.ErrCryptoBadInput
	}

	return plain, 11, nil
}

func (f *fakeCrypto) Streebog512(_ context.Context, data []byte) ([]byte, error) {
	if f.hashErr != nil {
		return nil, f.hashErr
	}

	sum := sha512.Sum512(data)

	return sum[:], nil
}

func (f *fakeCrypto) SignHash(_ context.Context, privPEM string, hash []byte) ([]byte, error) {
	if f.signErr != nil {
		return nil, f.signErr
	}

	if privPEM != testPrivPEM {
		return nil, ports.ErrCryptoBadInput
	}

	f.signed = append(f.signed, hash)

	return append([]byte("sig:"), hash...), nil
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

// newEnv собирает окружение: оба метода включены, ключ RSA пользователя и
// системный ключ готовы и запечатаны тем же sealed, что и в работе.
func newEnv(t *testing.T) *env {
	t.Helper()

	e := &env{
		store: &fakeStore{methods: []ports.MethodRow{
			{Code: "kuznyechik", Title: "Кузнечик", KeySize: 256, Enabled: true},
			{Code: "rsa", Title: "RSA", KeySize: 2048, Enabled: true},
		}},
		crypto: &fakeCrypto{},
	}

	userKey := &ports.UserKey{
		UserID: testUserID, Bits: 2048, PublicKeyPEM: testPubPEM, PrivateSealed: e.seal(t, testPrivPEM),
		KeygenMs: 1500, Status: ports.KeyStatusReady,
	}
	systemKey := &ports.UserKey{
		UserID: keys.SystemUserID, Bits: 1024, PublicKeyPEM: systemPublicPEM(t), PrivateSealed: e.seal(t, testPrivPEM),
		Status: ports.KeyStatusReady,
	}

	e.store.userKeys = map[int64]*ports.UserKey{testUserID: userKey, keys.SystemUserID: systemKey}
	e.keys = &fakeKeys{key: userKey}
	e.crypto.kuzKeysCalls = 0

	return e
}

func (e *env) seal(t *testing.T, secret string) []byte {
	t.Helper()

	blob, err := sealed.Seal(t.Context(), e.crypto, testMasterKey, []byte(secret))
	if err != nil {
		t.Fatalf("seal test secret: %v", err)
	}

	return blob
}

// systemPublicPEM — настоящий открытый ключ: отпечаток считается по DER,
// и метка вместо ключа не разберётся.
func systemPublicPEM(t *testing.T) string {
	t.Helper()

	priv, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}

	return string(pem.EncodeToMemory(&pem.Block{Type: "RSA PUBLIC KEY", Bytes: x509.MarshalPKCS1PublicKey(&priv.PublicKey)}))
}

func (e *env) newHandler() *Handler {
	log := slog.New(slog.NewTextHandler(&e.log, &slog.HandlerOptions{Level: slog.LevelDebug}))
	audit := func(_ context.Context, userID *int64, op, ref string, ok bool) {
		e.mu.Lock()
		defer e.mu.Unlock()

		e.audit = append(e.audit, auditRecord{userID: *userID, op: op, ref: ref, ok: ok})
	}

	h := NewHandler(e.store, e.crypto, e.keys, audit, Config{MasterKeyHex: testMasterKey, MaxTextBytes: testMaxBytes}, log)
	h.now = func() time.Time { return testNow }

	return h
}

// mux собирает маршруты так же, как пакет api, но без проверки сессии:
// пользователь кладётся в контекст запроса напрямую.
func (e *env) mux() http.Handler {
	h := e.newHandler()
	mux := http.NewServeMux()

	mux.Handle("POST /api/encrypt", httpx.BodyLimit(h.EncryptBodyLimit())(http.HandlerFunc(h.Encrypt)))
	mux.Handle("POST /api/decrypt", httpx.BodyLimit(h.DecryptBodyLimit())(http.HandlerFunc(h.Decrypt)))
	mux.Handle("POST /api/exportPdfWithSignature", httpx.BodyLimit(h.ExportBodyLimit())(http.HandlerFunc(h.ExportPDF)))
	mux.HandleFunc("GET /api/getAllMessages", h.List)

	return mux
}

func userPrincipal() auth.Principal {
	return auth.Principal{UserID: testUserID, Login: testLogin, Role: auth.RoleUser}
}

func otherPrincipal() auth.Principal {
	return auth.Principal{UserID: otherUserID, Login: otherLogin, Role: auth.RoleUser}
}

func adminPrincipal() auth.Principal {
	return auth.Principal{UserID: adminUserID, Login: "admin", Role: auth.RoleAdmin}
}

// do отправляет запрос от имени p. contentType пустой — application/json.
func (e *env) do(t *testing.T, p auth.Principal, method, path, contentType, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequestWithContext(auth.WithPrincipal(t.Context(), p), method, path, strings.NewReader(body))

	if contentType == "" {
		contentType = "application/json"
	}

	req.Header.Set("Content-Type", contentType)

	rec := httptest.NewRecorder()
	e.mux().ServeHTTP(rec, req)

	return rec
}

func (e *env) post(t *testing.T, path string, p auth.Principal, body any) *httptest.ResponseRecorder {
	t.Helper()

	return e.do(t, p, http.MethodPost, path, "", mustJSON(t, body))
}

// encrypt шифрует текст от имени пользователя и возвращает сообщение.
func (e *env) encrypt(t *testing.T, p auth.Principal, text, method string) MessageDTO {
	t.Helper()

	rec := e.post(t, "/api/encrypt", p, EncryptRequest{Plaintext: text, Method: method})
	if rec.Code != http.StatusCreated {
		t.Fatalf("encrypt status = %d, want 201 (body %q)", rec.Code, rec.Body.String())
	}

	return decodeBody[MessageDTO](t, rec)
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()

	if s, ok := v.(string); ok {
		return s
	}

	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	return string(b)
}

func decodeBody[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()

	var got T
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

	body := decodeBody[httpx.ErrorResponse](t, rec)
	if body.Error.Code != code || body.Error.Message == "" {
		t.Errorf("error = %+v, want code %q with message", body.Error, code)
	}
}

func (e *env) auditLog() []auditRecord {
	e.mu.Lock()
	defer e.mu.Unlock()

	return slices.Clone(e.audit)
}

func (e *env) resetAudit() {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.audit = nil
}

func mustHex(t *testing.T, s string) []byte {
	t.Helper()

	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("decode hex %q: %v", s, err)
	}

	return b
}
