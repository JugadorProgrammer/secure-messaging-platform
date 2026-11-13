package keys

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/ports"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/sealed"
)

// Фикстуры заведомо недействительны: это метки, по которым тест ищет утечки.
const (
	testBits     = 2048
	testPubPEM   = "-----BEGIN RSA PUBLIC KEY-----\nUFVCLUZJWFRVUkU=\n-----END RSA PUBLIC KEY-----\n"
	testPrivPEM  = "-----BEGIN RSA PRIVATE KEY-----\nUFJJVi1GSVhUVVJF\n-----END RSA PRIVATE KEY-----\n"
	testKeygenMs = 1500
)

var testMasterKey = strings.Repeat("11", 32)

// memStore — хранилище в памяти с той же атомарностью, что у store.Store:
// StartUserKey и RestartUserKey выигрывает только один из конкурентов.
type memStore struct {
	mu      sync.Mutex
	keys    map[int64]ports.UserKey
	now     func() time.Time
	saveErr error
}

func newMemStore(now func() time.Time) *memStore {
	return &memStore{keys: make(map[int64]ports.UserKey), now: now}
}

func (s *memStore) UserKey(_ context.Context, userID int64) (*ports.UserKey, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	k, ok := s.keys[userID]
	if !ok {
		return nil, ports.ErrNotFound
	}

	return &k, nil
}

func (s *memStore) SaveUserKey(_ context.Context, k *ports.UserKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.saveErr != nil && k.Status == ports.KeyStatusReady {
		return s.saveErr
	}

	saved := *k
	saved.UpdatedAt = s.now()
	s.keys[k.UserID] = saved

	return nil
}

func (s *memStore) StartUserKey(_ context.Context, userID int64, bits int) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.keys[userID]; ok {
		return false, nil
	}

	s.keys[userID] = ports.UserKey{UserID: userID, Bits: bits, Status: ports.KeyStatusGenerating, UpdatedAt: s.now()}

	return true, nil
}

func (s *memStore) RestartUserKey(_ context.Context, userID int64, bits int, staleBefore time.Time) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	k, ok := s.keys[userID]
	stale := k.Status == ports.KeyStatusGenerating && k.UpdatedAt.Before(staleBefore)

	if !ok || (k.Status != ports.KeyStatusFailed && !stale) {
		return false, nil
	}

	s.keys[userID] = ports.UserKey{UserID: userID, Bits: bits, Status: ports.KeyStatusGenerating, UpdatedAt: s.now()}

	return true, nil
}

func (s *memStore) put(k ports.UserKey) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.keys[k.UserID] = k
}

// fakeCrypto — Крипто-модуль: RSAKeys считает вызовы и может ждать сигнала
// release, Кузнечик — XOR, чтобы sealed.Unseal восстановил исходный PEM.
type fakeCrypto struct {
	ports.CryptoClient

	rsaCalls   atomic.Int32
	release    chan struct{}
	rsaErr     error
	encryptErr error
}

func (f *fakeCrypto) RSAKeys(ctx context.Context, _ int) (pub, priv string, ms int64, err error) {
	f.rsaCalls.Add(1)

	if f.release != nil {
		select {
		case <-f.release:
		case <-ctx.Done():
			return "", "", 0, ctx.Err()
		}
	}

	if f.rsaErr != nil {
		return "", "", 0, f.rsaErr
	}

	return testPubPEM, testPrivPEM, testKeygenMs, nil
}

func (f *fakeCrypto) KuzKeys(context.Context) (keyHex, ivHex string, err error) {
	return strings.Repeat("ff", 32), strings.Repeat("ab", 16), nil
}

func (f *fakeCrypto) KuzEncrypt(_ context.Context, keyHex, ivHex string, data []byte) (ct []byte, ms int64, err error) {
	if f.encryptErr != nil {
		return nil, 0, f.encryptErr
	}

	return xor(keyHex+ivHex, data), 1, nil
}

func (f *fakeCrypto) KuzDecrypt(_ context.Context, keyHex, ivHex string, ct []byte) (data []byte, ms int64, err error) {
	return xor(keyHex+ivHex, ct), 1, nil
}

func xor(pad string, data []byte) []byte {
	out := make([]byte, len(data))
	for i, b := range data {
		out[i] = b ^ pad[i%len(pad)]
	}

	return out
}

type fixture struct {
	mgr    *Manager
	store  *memStore
	crypto *fakeCrypto
	log    *bytes.Buffer
	now    time.Time
}

func newFixture(t *testing.T, crypto *fakeCrypto) *fixture {
	t.Helper()

	f := &fixture{crypto: crypto, log: &bytes.Buffer{}, now: time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)}
	clock := func() time.Time { return f.now }
	f.store = newMemStore(clock)

	log := slog.New(slog.NewTextHandler(&lockedWriter{w: f.log}, &slog.HandlerOptions{Level: slog.LevelDebug}))
	f.mgr = New(f.store, crypto, Config{MasterKeyHex: testMasterKey, Bits: testBits, KeygenTimeout: time.Hour}, log)
	f.mgr.now = clock

	t.Cleanup(f.mgr.Close)

	return f
}

// lockedWriter защищает буфер лога: в него пишут фоновые горутины.
type lockedWriter struct {
	mu sync.Mutex
	w  *bytes.Buffer
}

func (l *lockedWriter) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.w.Write(p)
}

func (f *fixture) ensure(t *testing.T, userID int64) *ports.UserKey {
	t.Helper()

	k, err := f.mgr.EnsureUserKey(t.Context(), userID)
	if err != nil {
		t.Fatalf("EnsureUserKey(%d) error = %v", userID, err)
	}

	return k
}

// wait дожидается окончания фоновых генераций, не закрывая менеджер.
func (f *fixture) wait() {
	f.mgr.wg.Wait()
}

func TestEnsureUserKeyStates(t *testing.T) {
	tests := []struct {
		name         string
		existing     *ports.UserKey
		wantStatus   string
		wantRSACalls int32
	}{
		{
			name:         "нет ключа — создаётся и генерируется",
			wantStatus:   ports.KeyStatusReady,
			wantRSACalls: 1,
		},
		{
			name: "ready — повторной генерации нет",
			existing: &ports.UserKey{
				UserID: 7, Bits: testBits, PublicKeyPEM: testPubPEM, PrivateSealed: []byte{1}, Status: ports.KeyStatusReady,
			},
			wantStatus:   ports.KeyStatusReady,
			wantRSACalls: 0,
		},
		{
			name:         "generating — вторая генерация не запускается",
			existing:     &ports.UserKey{UserID: 7, Bits: testBits, Status: ports.KeyStatusGenerating},
			wantStatus:   ports.KeyStatusGenerating,
			wantRSACalls: 0,
		},
		{
			name:         "failed — повторная попытка",
			existing:     &ports.UserKey{UserID: 7, Bits: testBits, Status: ports.KeyStatusFailed},
			wantStatus:   ports.KeyStatusReady,
			wantRSACalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t, &fakeCrypto{})
			if tt.existing != nil {
				existing := *tt.existing
				existing.UpdatedAt = f.now
				f.store.put(existing)
			}

			f.ensure(t, 7)
			f.wait()

			got, err := f.store.UserKey(t.Context(), 7)
			if err != nil {
				t.Fatalf("UserKey() error = %v", err)
			}

			if got.Status != tt.wantStatus {
				t.Errorf("status = %q, want %q", got.Status, tt.wantStatus)
			}

			if n := f.crypto.rsaCalls.Load(); n != tt.wantRSACalls {
				t.Errorf("RSAKeys called %d times, want %d", n, tt.wantRSACalls)
			}
		})
	}
}

func TestEnsureUserKeyReturnsGeneratingWithoutWaiting(t *testing.T) {
	f := newFixture(t, &fakeCrypto{release: make(chan struct{})})

	k := f.ensure(t, 7)
	if k.Status != ports.KeyStatusGenerating {
		t.Errorf("status = %q, want generating: EnsureUserKey must not wait for keygen", k.Status)
	}

	// Повторный вызов, пока генерация идёт, вторую не запускает.
	f.ensure(t, 7)
	close(f.crypto.release)
	f.wait()

	if n := f.crypto.rsaCalls.Load(); n != 1 {
		t.Errorf("RSAKeys called %d times, want 1", n)
	}
}

func TestEnsureUserKeyConcurrent(t *testing.T) {
	f := newFixture(t, &fakeCrypto{release: make(chan struct{})})

	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := f.mgr.EnsureUserKey(t.Context(), 7); err != nil {
				t.Errorf("EnsureUserKey() error = %v", err)
			}
		}()
	}
	wg.Wait()

	close(f.crypto.release)
	f.wait()

	if n := f.crypto.rsaCalls.Load(); n != 1 {
		t.Errorf("RSAKeys called %d times for 10 concurrent calls, want 1", n)
	}
}

func TestGenerationResult(t *testing.T) {
	f := newFixture(t, &fakeCrypto{})

	f.ensure(t, 7)
	f.wait()

	k, err := f.store.UserKey(t.Context(), 7)
	if err != nil {
		t.Fatalf("UserKey() error = %v", err)
	}

	if k.Status != ports.KeyStatusReady || k.PublicKeyPEM != testPubPEM || k.Bits != testBits || k.KeygenMs != testKeygenMs {
		t.Errorf("key = %+v, want ready with public PEM, %d bits and keygen_ms %d", k, testBits, testKeygenMs)
	}

	if len(k.PrivateSealed) == 0 || bytes.Contains(k.PrivateSealed, []byte("PRIVATE KEY")) {
		t.Fatalf("private key is not sealed: %q", k.PrivateSealed)
	}

	plain, err := sealed.Unseal(t.Context(), f.crypto, testMasterKey, k.PrivateSealed)
	if err != nil {
		t.Fatalf("Unseal() error = %v", err)
	}

	if string(plain) != testPrivPEM {
		t.Errorf("Unseal() = %q, want original private PEM", plain)
	}
}

func TestGenerationFailure(t *testing.T) {
	tests := []struct {
		name   string
		crypto *fakeCrypto
		store  error
	}{
		{"ошибка RSAKeys", &fakeCrypto{rsaErr: ports.ErrCryptoUnavailable}, nil},
		{"ошибка запечатывания", &fakeCrypto{encryptErr: ports.ErrCryptoUnavailable}, nil},
		{"ошибка сохранения", &fakeCrypto{}, errors.New("connection reset")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t, tt.crypto)
			f.store.saveErr = tt.store

			f.ensure(t, 7)
			f.wait()

			k, err := f.store.UserKey(t.Context(), 7)
			if err != nil {
				t.Fatalf("UserKey() error = %v", err)
			}

			if k.Status != ports.KeyStatusFailed {
				t.Errorf("status = %q, want failed", k.Status)
			}

			if k.PublicKeyPEM != "" || len(k.PrivateSealed) != 0 {
				t.Errorf("failed key keeps key material: %+v", k)
			}
		})
	}
}

func TestStaleGeneratingRestarts(t *testing.T) {
	tests := []struct {
		name         string
		age          time.Duration
		wantRSACalls int32
	}{
		{"свежий generating не трогается", 30 * time.Minute, 0},
		{"брошенный generating перезапускается", 2 * time.Hour, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t, &fakeCrypto{})
			f.store.put(ports.UserKey{
				UserID: 7, Bits: testBits, Status: ports.KeyStatusGenerating, UpdatedAt: f.now.Add(-tt.age),
			})

			f.ensure(t, 7)
			f.wait()

			if n := f.crypto.rsaCalls.Load(); n != tt.wantRSACalls {
				t.Errorf("RSAKeys called %d times, want %d", n, tt.wantRSACalls)
			}
		})
	}
}

func TestSystemKey(t *testing.T) {
	f := newFixture(t, &fakeCrypto{})

	f.ensure(t, SystemUserID)
	f.wait()

	k, err := f.store.UserKey(t.Context(), SystemUserID)
	if err != nil {
		t.Fatalf("UserKey(0) error = %v", err)
	}

	if k.UserID != SystemUserID || k.Status != ports.KeyStatusReady {
		t.Errorf("system key = %+v, want ready key of user 0", k)
	}
}

func TestCloseMarksRunningGenerationFailed(t *testing.T) {
	f := newFixture(t, &fakeCrypto{release: make(chan struct{})})

	f.ensure(t, 7)
	f.mgr.Close()

	k, err := f.store.UserKey(t.Context(), 7)
	if err != nil {
		t.Fatalf("UserKey() error = %v", err)
	}

	if k.Status != ports.KeyStatusFailed {
		t.Errorf("status after Close = %q, want failed: next start must restart keygen at once", k.Status)
	}
}

func TestRequestCancelDoesNotStopGeneration(t *testing.T) {
	f := newFixture(t, &fakeCrypto{release: make(chan struct{})})

	ctx, cancel := context.WithCancel(t.Context())
	if _, err := f.mgr.EnsureUserKey(ctx, 7); err != nil {
		t.Fatalf("EnsureUserKey() error = %v", err)
	}

	cancel()
	close(f.crypto.release)
	f.wait()

	k, err := f.store.UserKey(t.Context(), 7)
	if err != nil {
		t.Fatalf("UserKey() error = %v", err)
	}

	if k.Status != ports.KeyStatusReady {
		t.Errorf("status = %q, want ready: client cancellation must not abort keygen", k.Status)
	}
}

func TestStoreErrors(t *testing.T) {
	f := newFixture(t, &fakeCrypto{})
	broken := &brokenStore{err: errors.New("connection refused")}
	f.mgr.store = broken

	if _, err := f.mgr.EnsureUserKey(t.Context(), 7); !errors.Is(err, broken.err) {
		t.Errorf("EnsureUserKey() error = %v, want store error", err)
	}

	if n := f.crypto.rsaCalls.Load(); n != 0 {
		t.Errorf("RSAKeys called %d times after store error, want 0", n)
	}
}

type brokenStore struct {
	Store

	err error
}

func (b *brokenStore) UserKey(context.Context, int64) (*ports.UserKey, error) {
	return nil, b.err
}

func TestSecretsNotLogged(t *testing.T) {
	f := newFixture(t, &fakeCrypto{})

	f.ensure(t, 7)
	f.wait()

	failing := newFixture(t, &fakeCrypto{encryptErr: ports.ErrCryptoUnavailable})
	failing.ensure(t, 8)
	failing.wait()

	logged := f.log.String() + failing.log.String()
	if !strings.Contains(logged, "rsa key generated") || !strings.Contains(logged, "rsa key generation failed") {
		t.Fatalf("expected both log records, got %q", logged)
	}

	for _, secret := range []string{"UFJJVi1GSVhUVVJF", "UFVCLUZJWFRVUkU=", "PRIVATE KEY", testMasterKey} {
		if strings.Contains(logged, secret) {
			t.Errorf("log contains secret %q", secret)
		}
	}
}
