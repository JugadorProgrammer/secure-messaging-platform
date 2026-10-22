package api

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/config"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/httpx"
)

const testMasterKey = "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"

func testConfig(t *testing.T, mutate func(map[string]string)) *config.Config {
	t.Helper()

	env := map[string]string{
		"HTTP_ADDR":    ":8080",
		"DSN":          "postgres://coord:coord_local@localhost:5432/coord?sslmode=disable",
		"MASTER_KEY":   testMasterKey,
		"ACCOUNTS_URL": "http://localhost:8081",
		"CRYPTO_URL":   "http://localhost:8082",
	}
	if mutate != nil {
		mutate(env)
	}

	cfg, err := config.Load(func(name string) string { return env[name] })
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}

	return cfg
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestHealthRoutes(t *testing.T) {
	h := New(testConfig(t, nil), discardLogger())

	for _, path := range []string{"/health", "/api/health"} {
		t.Run(path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, http.NoBody))

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
			}
			if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
				t.Errorf("Content-Type = %q", ct)
			}

			var got healthResponse
			if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if got.Status != "ok" {
				t.Errorf("status field = %q, want ok", got.Status)
			}
		})
	}
}

func TestHealthMethodNotAllowed(t *testing.T) {
	h := New(testConfig(t, nil), discardLogger())

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/health", http.NoBody))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}

	var got httpx.ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if got.Error.Code != httpx.CodeMethodNotAllowed {
		t.Errorf("code = %q, want %q", got.Error.Code, httpx.CodeMethodNotAllowed)
	}
}

func TestUnknownRouteReturnsJSON(t *testing.T) {
	h := New(testConfig(t, nil), discardLogger())

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/unknown", http.NoBody))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	var got httpx.ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if got.Error.Code != httpx.CodeNotFound {
		t.Errorf("code = %q, want %q", got.Error.Code, httpx.CodeNotFound)
	}
}

func TestRequestIDHeaderIsSet(t *testing.T) {
	h := New(testConfig(t, nil), discardLogger())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", http.NoBody)
	req.Header.Set(httpx.HeaderRequestID, "test-request-id")
	h.ServeHTTP(rec, req)

	if got := rec.Header().Get(httpx.HeaderRequestID); got != "test-request-id" {
		t.Errorf("X-Request-ID = %q, want test-request-id", got)
	}
}

func TestCORSHeadersForAllowedOrigin(t *testing.T) {
	cfg := testConfig(t, func(env map[string]string) {
		env["CORS_ORIGINS"] = "http://localhost:5173"
	})
	h := New(cfg, discardLogger())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", http.NoBody)
	req.Header.Set("Origin", "http://localhost:5173")
	h.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Errorf("Access-Control-Allow-Origin = %q", got)
	}
}

func TestCORSDisabledByDefault(t *testing.T) {
	h := New(testConfig(t, nil), discardLogger())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", http.NoBody)
	req.Header.Set("Origin", "http://localhost:5173")
	h.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Access-Control-Allow-Origin = %q, want empty", got)
	}
}

// stubConfig реализует Config без пакета config: роутеру нужен только список
// разрешённых origin.
type stubConfig []string

func (s stubConfig) CORSOrigins() []string { return s }

func TestNewAcceptsStubConfig(t *testing.T) {
	h := New(stubConfig(nil), discardLogger())

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", http.NoBody))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}
