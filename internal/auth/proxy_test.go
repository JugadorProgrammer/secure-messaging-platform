package auth

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/httpx"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()

	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse url %q: %v", raw, err)
	}

	return u
}

// echoHandler возвращает обработчик, который запоминает запрос и отвечает 201.
func echoHandler(t *testing.T, seen *capturedRequest) http.Handler {
	t.Helper()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*seen = captureRequest(r)

		w.Header().Set("Content-Type", contentTypeJSON)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
}

type capturedRequest struct {
	Method        string
	Path          string
	RawQuery      string
	Host          string
	Header        http.Header
	Body          string
	XForwardedFor []string
}

func captureRequest(r *http.Request) capturedRequest {
	body, _ := io.ReadAll(r.Body)

	return capturedRequest{
		Method:        r.Method,
		Path:          r.URL.Path,
		RawQuery:      r.URL.RawQuery,
		Host:          r.Host,
		Header:        r.Header.Clone(),
		Body:          string(body),
		XForwardedFor: r.Header.Values("X-Forwarded-For"),
	}
}

func TestProxyForwardsEveryRoute(t *testing.T) {
	var seen capturedRequest
	backend := httptest.NewServer(echoHandler(t, &seen))
	defer backend.Close()

	proxy := http.StripPrefix("/api", NewAccountsProxy(mustParseURL(t, backend.URL), discardLogger()))

	routes := []struct {
		method string
		path   string
		want   string
	}{
		{http.MethodPost, "/api/auth/signup", "/auth/signup"},
		{http.MethodPost, "/api/auth/login", "/auth/login"},
		{http.MethodPost, "/api/auth/login/email-code", "/auth/login/email-code"},
		{http.MethodPost, "/api/auth/login/recovery-code", "/auth/login/recovery-code"},
		{http.MethodPost, "/api/auth/password/restore", "/auth/password/restore"},
		{http.MethodPost, "/api/auth/password/confirm", "/auth/password/confirm"},
		{http.MethodPost, "/api/auth/logout", "/auth/logout"},
		{http.MethodGet, "/api/users", "/users"},
		{http.MethodPost, "/api/users", "/users"},
	}

	for _, rt := range routes {
		t.Run(rt.method+" "+rt.path, func(t *testing.T) {
			req := httptest.NewRequest(rt.method, rt.path, strings.NewReader(`{"a":1}`))
			rec := httptest.NewRecorder()

			proxy.ServeHTTP(rec, req)

			if rec.Code != http.StatusCreated {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
			}
			if seen.Path != rt.want {
				t.Errorf("forwarded path = %q, want %q", seen.Path, rt.want)
			}
			if seen.Method != rt.method {
				t.Errorf("forwarded method = %q, want %q", seen.Method, rt.method)
			}
			if seen.Body != `{"a":1}` {
				t.Errorf("forwarded body = %q", seen.Body)
			}
		})
	}
}

func TestProxyRewritesHostAndKeepsQuery(t *testing.T) {
	var seen capturedRequest
	backend := httptest.NewServer(echoHandler(t, &seen))
	defer backend.Close()

	proxy := http.StripPrefix("/api", NewAccountsProxy(mustParseURL(t, backend.URL), discardLogger()))

	req := httptest.NewRequest(http.MethodGet, "/api/users?limit=10", http.NoBody)
	req.Host = "coord.example"
	rec := httptest.NewRecorder()

	proxy.ServeHTTP(rec, req)

	target := mustParseURL(t, backend.URL)
	if seen.Host != target.Host {
		t.Errorf("Host = %q, want %q", seen.Host, target.Host)
	}
	if seen.Path != "/users" {
		t.Errorf("path = %q, want /users", seen.Path)
	}
	if seen.Header.Get("X-Forwarded-Host") != "coord.example" {
		t.Errorf("X-Forwarded-Host = %q", seen.Header.Get("X-Forwarded-Host"))
	}
	if seen.RawQuery != "limit=10" {
		t.Errorf("query = %q, want limit=10", seen.RawQuery)
	}
}

func TestProxyStripsIdentityHeaders(t *testing.T) {
	var seen capturedRequest
	backend := httptest.NewServer(echoHandler(t, &seen))
	defer backend.Close()

	proxy := http.StripPrefix("/api", NewAccountsProxy(mustParseURL(t, backend.URL), discardLogger()))

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader("{}"))
	req.Header.Set("X-User-Id", "1")
	req.Header.Set("X-User-Role", string(RoleAdmin))
	req.Header.Set("X-Real-IP", "203.0.113.77")
	req.Header.Set("Forwarded", "for=203.0.113.77")
	req.Header.Set("X-Original-URL", "/auth/logout")
	req.Header.Set("X-Rewrite-URL", "/admin")
	req.Header.Set("Proxy", "evil")
	req.Header.Set("X-Forwarded-For", "203.0.113.77")
	req.Header.Set(httpx.HeaderRequestID, "req-42")
	req.Header.Set(headerAuthorization, "Bearer secret-token")

	rec := httptest.NewRecorder()
	proxy.ServeHTTP(rec, req)

	for _, name := range []string{"X-User-Id", "X-User-Role", "X-Real-IP", "Forwarded",
		"X-Original-URL", "X-Rewrite-URL", "Proxy"} {
		if got := seen.Header.Get(name); got != "" {
			t.Errorf("%s reached accounts: %q", name, got)
		}
	}

	// Поддельный адрес не попадает в цепочку: SetXForwarded вызван после
	// очистки, поэтому в заголовке только адрес клиента.
	if got := strings.Join(seen.XForwardedFor, ","); got != "192.0.2.1" {
		t.Errorf("X-Forwarded-For = %q, want 192.0.2.1", got)
	}
	if got := seen.Header.Get(httpx.HeaderRequestID); got != "req-42" {
		t.Errorf("X-Request-ID = %q, want req-42", got)
	}
	if got := seen.Header.Get(headerAuthorization); got != "Bearer secret-token" {
		t.Errorf("Authorization = %q, want it forwarded", got)
	}
}

func TestProxyWithoutRequestID(t *testing.T) {
	var seen capturedRequest
	backend := httptest.NewServer(echoHandler(t, &seen))
	defer backend.Close()

	proxy := http.StripPrefix("/api", NewAccountsProxy(mustParseURL(t, backend.URL), discardLogger()))

	req := httptest.NewRequest(http.MethodGet, "/api/users", http.NoBody)
	rec := httptest.NewRecorder()

	proxy.ServeHTTP(rec, req)

	if got := seen.Header.Get(httpx.HeaderRequestID); got != "" {
		t.Errorf("X-Request-ID = %q, want empty", got)
	}
}

func TestProxyAccountsUnavailable(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	target := mustParseURL(t, backend.URL)
	backend.Close()

	proxy := http.StripPrefix("/api", NewAccountsProxy(target, discardLogger()))

	rec := httptest.NewRecorder()
	proxy.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader("{}")))

	assertErrorResponse(t, rec, http.StatusServiceUnavailable, httpx.CodeAccountsUnavailable)
}

func TestProxyKeepsBackendStatusAndBody(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentTypeJSON)
		w.WriteHeader(http.StatusUnauthorized)

		_, _ = w.Write([]byte(`{"error":{"code":"unauthorized","message":"Неверный пароль"}}`))
	}))
	defer backend.Close()

	proxy := http.StripPrefix("/api", NewAccountsProxy(mustParseURL(t, backend.URL), discardLogger()))

	rec := httptest.NewRecorder()
	proxy.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader("{}")))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	assertErrorResponse(t, rec, http.StatusUnauthorized, httpx.CodeUnauthorized)
}

func TestProxyBodyLimit(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	handler := httpx.BodyLimit(4)(http.StripPrefix("/api",
		NewAccountsProxy(mustParseURL(t, backend.URL), discardLogger())))

	// ContentLength -1 заставляет прокси читать тело потоково, и лимит
	// обнаруживается уже в Transport — ErrorHandler обязан ответить 413.
	req := httptest.NewRequest(http.MethodPost, "/api/auth/signup", strings.NewReader("0123456789"))
	req.ContentLength = -1

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertErrorResponse(t, rec, http.StatusRequestEntityTooLarge, httpx.CodePayloadTooLarge)
}

func TestProxyDoesNotLogBodiesOrToken(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"token":"issued-secret-token","password":"backend-secret"}`))
	}))
	defer backend.Close()

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	proxy := http.StripPrefix("/api", NewAccountsProxy(mustParseURL(t, backend.URL), logger))

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", strings.NewReader(`{"login":"ivanov","password":"client-secret"}`))
	req.Header.Set(headerAuthorization, "Bearer request-secret-token")

	rec := httptest.NewRecorder()
	proxy.ServeHTTP(rec, req)

	logs := buf.String()
	for _, secret := range []string{"client-secret", "backend-secret", "issued-secret-token", "request-secret-token", "ivanov"} {
		if strings.Contains(logs, secret) {
			t.Errorf("log contains %q: %s", secret, logs)
		}
	}
}

// assertErrorResponse проверяет JSON ошибки в едином формате.
func assertErrorResponse(t *testing.T, rec *httptest.ResponseRecorder, wantStatus int, wantCode string) {
	t.Helper()

	if rec.Code != wantStatus {
		t.Fatalf("status = %d, want %d (body %q)", rec.Code, wantStatus, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type = %q, want json", ct)
	}

	var got httpx.ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode body %q: %v", rec.Body.String(), err)
	}
	if got.Error.Code != wantCode {
		t.Errorf("code = %q, want %q", got.Error.Code, wantCode)
	}
	if got.Error.Message == "" {
		t.Error("message is empty")
	}
}
