package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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

func devBStub() Option {
	return WithDevB(DevB{
		Crypto:          cryptoStub{},
		Keys:            keysStub{},
		MasterKeyHex:    strings.Repeat("11", 32),
		MaxMessageBytes: 64 << 10,
	})
}

func postMessage(t *testing.T, h http.Handler, token, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/api/messages", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	return rec
}

func TestPostMessagesRoute(t *testing.T) {
	backend := stubAccounts(t)
	defer backend.Close()

	cfg := testConfig(t, func(env map[string]string) { env["ACCOUNTS_URL"] = backend.URL })

	tests := []struct {
		name       string
		handler    http.Handler
		token      string
		wantStatus int
		wantCode   string
	}{
		{
			name:       "без токена",
			handler:    New(cfg, newMemStore(), discardLogger(), devBStub()),
			wantStatus: http.StatusUnauthorized,
			wantCode:   httpx.CodeUnauthorized,
		},
		{
			name:       "с токеном доходит до валидации",
			handler:    New(cfg, newMemStore(), discardLogger(), devBStub()),
			token:      userToken,
			wantStatus: http.StatusBadRequest,
			wantCode:   httpx.CodeValidation,
		},
		{
			name:       "без WithDevB маршрута нет",
			handler:    New(cfg, newMemStore(), discardLogger()),
			token:      userToken,
			wantStatus: http.StatusNotFound,
			wantCode:   httpx.CodeNotFound,
		},
		{
			name:       "без хранилища маршрута нет",
			handler:    New(cfg, nil, discardLogger(), devBStub()),
			token:      userToken,
			wantStatus: http.StatusNotFound,
			wantCode:   httpx.CodeNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := postMessage(t, tt.handler, tt.token, `{"text":"","method":"rsa"}`)
			assertJSONError(t, rec, tt.wantStatus, tt.wantCode)
		})
	}
}

func TestPostMessagesNeedsAccounts(t *testing.T) {
	h := New(testConfig(t, nil), newMemStore(), discardLogger(), devBStub())

	rec := postMessage(t, h, userToken, `{"text":"x","method":"rsa"}`)
	if rec.Code == http.StatusCreated || rec.Code == http.StatusBadRequest {
		t.Errorf("status = %d: route must not be served without token check", rec.Code)
	}
}
