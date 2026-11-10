package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/httpx"
)

// Токены заглушки introspect: по ним тест проверяет роль пользователя.
const (
	adminToken = "admin-token"
	userToken  = "user-token"
)

// stubAccounts имитирует Сервис учётных записей: проверяет токен по пути
// /internal/introspect и на остальных путях отвечает своим путем в теле.
func stubAccounts(t *testing.T) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/internal/introspect" {
			introspectStub(w, r)

			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"path": r.URL.Path})
	}))
}

func introspectStub(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token string `json:"token"`
	}

	body, _ := io.ReadAll(r.Body)
	_ = json.Unmarshal(body, &req)

	session := sessionStub{Active: false}

	switch req.Token {
	case adminToken:
		session = sessionStub{Active: true, Stage: "full", UserID: 1, Login: "admin", Role: "admin"}
	case userToken:
		session = sessionStub{Active: true, Stage: "full", UserID: 7, Login: "ivanov", Role: "user"}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(session); err != nil {
		// Заголовок уже отправлен, сообщить об ошибке некуда.
		_ = err
	}
}

// sessionStub — ответ /internal/introspect заглушки Сервиса учётных записей.
type sessionStub struct {
	Active bool   `json:"active"`
	Stage  string `json:"stage"`
	UserID int64  `json:"user_id"`
	Login  string `json:"login"`
	Role   string `json:"role"`
}

// proxyRoutes — маршруты, которые проксируются, и путь, который должен
// дойти до Сервиса учётных записей.
var proxyRoutes = []struct {
	method string
	path   string
	token  string
	want   string
}{
	{http.MethodPost, "/api/auth/signup", "", "/auth/signup"},
	{http.MethodPost, "/api/auth/login", "", "/auth/login"},
	{http.MethodPost, "/api/auth/login/email-code", "", "/auth/login/email-code"},
	{http.MethodPost, "/api/auth/login/recovery-code", "", "/auth/login/recovery-code"},
	{http.MethodPost, "/api/auth/password/restore", "", "/auth/password/restore"},
	{http.MethodPost, "/api/auth/password/confirm", "", "/auth/password/confirm"},
	{http.MethodPost, "/api/auth/logout", userToken, "/auth/logout"},
	{http.MethodGet, "/api/users", adminToken, "/users"},
	{http.MethodPost, "/api/users", adminToken, "/users"},
}

func TestAccountsRoutesAreProxied(t *testing.T) {
	backend := stubAccounts(t)
	defer backend.Close()

	handler := New(testConfig(t, func(env map[string]string) { env["ACCOUNTS_URL"] = backend.URL }), discardLogger())

	for _, rt := range proxyRoutes {
		t.Run(rt.method+" "+rt.path, func(t *testing.T) {
			req := httptest.NewRequest(rt.method, rt.path, strings.NewReader(`{"a":1}`))
			if rt.token != "" {
				req.Header.Set("Authorization", "Bearer "+rt.token)
			}

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
			}

			var got map[string]string
			if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
				t.Fatalf("decode body %q: %v", rec.Body.String(), err)
			}
			if got["path"] != rt.want {
				t.Errorf("accounts saw path %q, want %q", got["path"], rt.want)
			}
		})
	}
}

func TestUsersRequiresAdminRole(t *testing.T) {
	backend := stubAccounts(t)
	defer backend.Close()

	handler := New(testConfig(t, func(env map[string]string) { env["ACCOUNTS_URL"] = backend.URL }), discardLogger())

	cases := []struct {
		name  string
		token string
		want  int
		code  string
	}{
		{"no token", "", http.StatusUnauthorized, httpx.CodeUnauthorized},
		{"unknown token", "bogus", http.StatusUnauthorized, httpx.CodeUnauthorized},
		{"user role", userToken, http.StatusForbidden, httpx.CodeForbidden},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/users", http.NoBody)
			if tc.token != "" {
				req.Header.Set("Authorization", "Bearer "+tc.token)
			}

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			assertJSONError(t, rec, tc.want, tc.code)
		})
	}
}

func TestLogoutRequiresToken(t *testing.T) {
	backend := stubAccounts(t)
	defer backend.Close()

	handler := New(testConfig(t, func(env map[string]string) { env["ACCOUNTS_URL"] = backend.URL }), discardLogger())

	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", strings.NewReader("{}"))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assertJSONError(t, rec, http.StatusUnauthorized, httpx.CodeUnauthorized)
}

func TestAuthRoutesRejectWrongMethod(t *testing.T) {
	backend := stubAccounts(t)
	defer backend.Close()

	handler := New(testConfig(t, func(env map[string]string) { env["ACCOUNTS_URL"] = backend.URL }), discardLogger())

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/auth/login", http.NoBody))

	assertJSONError(t, rec, http.StatusMethodNotAllowed, httpx.CodeMethodNotAllowed)
}

func TestAuthBodyLimitIsApplied(t *testing.T) {
	backend := stubAccounts(t)
	defer backend.Close()

	handler := New(testConfig(t, func(env map[string]string) {
		env["ACCOUNTS_URL"] = backend.URL
		env["MAX_BODY_AUTH"] = "16"
	}), discardLogger())

	req := httptest.NewRequest(http.MethodPost, "/api/auth/signup", strings.NewReader(strings.Repeat("a", 64)))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assertJSONError(t, rec, http.StatusRequestEntityTooLarge, httpx.CodePayloadTooLarge)
}

func TestAccountsDownReturnsJSON(t *testing.T) {
	backend := stubAccounts(t)
	target := backend.URL
	backend.Close()

	handler := New(testConfig(t, func(env map[string]string) { env["ACCOUNTS_URL"] = target }), discardLogger())

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader("{}")))

	assertJSONError(t, rec, http.StatusServiceUnavailable, httpx.CodeAccountsUnavailable)
}

func TestAccountsDownDuringAuthReturns503(t *testing.T) {
	backend := stubAccounts(t)
	target := backend.URL
	backend.Close()

	handler := New(testConfig(t, func(env map[string]string) { env["ACCOUNTS_URL"] = target }), discardLogger())

	req := httptest.NewRequest(http.MethodGet, "/api/users", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+adminToken)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertJSONError(t, rec, http.StatusServiceUnavailable, httpx.CodeAccountsUnavailable)
}

func assertJSONError(t *testing.T, rec *httptest.ResponseRecorder, wantStatus int, wantCode string) {
	t.Helper()

	if rec.Code != wantStatus {
		t.Fatalf("status = %d, want %d (body %q)", rec.Code, wantStatus, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type = %q, want json", ct)
	}

	var got httpx.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body %q: %v", rec.Body.String(), err)
	}

	if got.Error.Code != wantCode {
		t.Errorf("code = %q, want %q (message %q)", got.Error.Code, wantCode, got.Error.Message)
	}
}
