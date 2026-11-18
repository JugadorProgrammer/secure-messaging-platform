package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/auth"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/httpx"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/ports"
)

// Заглушки встраивают интерфейс: тесты маршрутов до шифрования не доходят,
// и случайный вызов упадёт, а не вернёт выдуманный результат.
type cryptoStub struct{ ports.CryptoClient }

type keysStub struct{}

func (keysStub) EnsureUserKey(_ context.Context, _ int64) (*ports.UserKey, error) {
	return nil, ports.ErrNotFound
}

// plainStore — хранилище без поиска по шифртексту: маршрутам сообщений его
// недостаточно.
type plainStore struct{ ports.Store }

func devBStub() Option {
	return WithDevB(DevB{
		Crypto:          cryptoStub{},
		Keys:            keysStub{},
		MasterKeyHex:    strings.Repeat("11", 32),
		MaxMessageBytes: 64 << 10,
	})
}

// messageRoutes — маршруты сообщений и тело, которое доходит до валидации и
// отклоняется ею, не обращаясь к крипто-модулю.
var messageRoutes = []struct {
	method  string
	path    string
	invalid string
	status  int
	code    string
}{
	{http.MethodPost, "/api/encrypt", `{"plaintext":"","method":"RSA-32768"}`, http.StatusBadRequest, httpx.CodeValidation},
	{http.MethodPost, "/api/decrypt", `{"method":"Кузнечик","ciphertext":"zz"}`, http.StatusBadRequest, httpx.CodeBadCiphertext},
	{http.MethodPost, "/api/exportPdfWithSignature", `{"id":0}`, http.StatusBadRequest, httpx.CodeValidation},
	{http.MethodGet, "/api/getAllMessages", "", http.StatusOK, ""},
}

func sendMessageRoute(t *testing.T, h http.Handler, method, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	if token != "" {
		req.AddCookie(&http.Cookie{Name: auth.SessionCookie, Value: token})
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	return rec
}

func TestMessageRoutesNeedSession(t *testing.T) {
	backend := stubAccounts(t)
	defer backend.Close()

	cfg := testConfig(t, func(env map[string]string) { env["ACCOUNTS_URL"] = backend.URL })
	h := New(cfg, newMemStore(), discardLogger(), devBStub())

	for _, rt := range messageRoutes {
		t.Run(rt.path, func(t *testing.T) {
			rec := sendMessageRoute(t, h, rt.method, rt.path, "", rt.invalid)
			assertJSONError(t, rec, http.StatusUnauthorized, httpx.CodeUnauthorized)
		})
	}
}

func TestMessageRoutesWithSession(t *testing.T) {
	backend := stubAccounts(t)
	defer backend.Close()

	cfg := testConfig(t, func(env map[string]string) { env["ACCOUNTS_URL"] = backend.URL })
	h := New(cfg, newMemStore(), discardLogger(), devBStub())

	for _, rt := range messageRoutes {
		t.Run(rt.path, func(t *testing.T) {
			rec := sendMessageRoute(t, h, rt.method, rt.path, userToken, rt.invalid)

			if rt.status == http.StatusOK {
				if rec.Code != http.StatusOK {
					t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
				}

				return
			}

			assertJSONError(t, rec, rt.status, rt.code)
		})
	}
}

func TestMessageRoutesAreNotRegistered(t *testing.T) {
	backend := stubAccounts(t)
	defer backend.Close()

	cfg := testConfig(t, func(env map[string]string) { env["ACCOUNTS_URL"] = backend.URL })

	tests := []struct {
		name    string
		handler http.Handler
	}{
		{"без WithDevB", New(cfg, newMemStore(), discardLogger())},
		{"без хранилища", New(cfg, nil, discardLogger(), devBStub())},
		{"хранилище без поиска по шифртексту", New(cfg, plainStore{newMemStore()}, discardLogger(), devBStub())},
		{"без ACCOUNTS_URL", New(stubConfig(nil), newMemStore(), discardLogger(), devBStub())},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, rt := range messageRoutes {
				rec := sendMessageRoute(t, tt.handler, rt.method, rt.path, userToken, rt.invalid)
				assertJSONError(t, rec, http.StatusNotFound, httpx.CodeNotFound)
			}
		})
	}
}

func TestOldRoutesAreRemoved(t *testing.T) {
	backend := stubAccounts(t)
	defer backend.Close()

	cfg := testConfig(t, func(env map[string]string) { env["ACCOUNTS_URL"] = backend.URL })
	h := New(cfg, newMemStore(), discardLogger(), devBStub())

	for _, path := range []string{"/api/messages", "/api/methods"} {
		rec := sendMessageRoute(t, h, http.MethodGet, path, adminToken, "")
		assertJSONError(t, rec, http.StatusNotFound, httpx.CodeNotFound)
	}
}
