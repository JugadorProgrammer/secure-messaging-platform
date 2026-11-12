package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/httpx"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/opslog"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/ports"
)

// opsLogConfig поднимает сервис с адресом Сервиса учётных записей из заглушки.
func opsLogConfig(t *testing.T, accountsURL string) Config {
	t.Helper()

	return testConfig(t, func(env map[string]string) { env["ACCOUNTS_URL"] = accountsURL })
}

func getAs(t *testing.T, handler http.Handler, target, token string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, target, http.NoBody)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	return rec
}

func TestOpsLogIsAdminOnly(t *testing.T) {
	backend := stubAccounts(t)
	defer backend.Close()

	st := &memStore{entries: []ports.OpLogEntry{
		{ID: 3, UserID: int64Ptr(7), Op: opslog.OpAccessDenied, Ref: "GET /api/users", OK: false},
		{ID: 2, UserID: nil, Op: opslog.OpSignatureVerify, Ref: "sha256:0123456789abcdef", OK: true},
	}}
	handler := New(opsLogConfig(t, backend.URL), st, discardLogger())

	t.Run("без токена", func(t *testing.T) {
		rec := getAs(t, handler, "/api/ops-log", "")

		assertJSONError(t, rec, http.StatusUnauthorized, httpx.CodeUnauthorized)
		if len(st.logged) != 0 {
			t.Errorf("logged = %v, want none: отказ без аутентификации не пишется", st.logged)
		}
	})

	t.Run("роль user", func(t *testing.T) {
		rec := getAs(t, handler, "/api/ops-log", userToken)

		assertJSONError(t, rec, http.StatusForbidden, httpx.CodeForbidden)

		if len(st.logged) != 1 {
			t.Fatalf("logged = %d, want 1", len(st.logged))
		}

		got := st.logged[0]
		if got.op != opslog.OpAccessDenied || got.ref != "GET /api/ops-log" || got.ok {
			t.Errorf("record = %q %q ok=%v, want %q %q ok=false",
				got.op, got.ref, got.ok, opslog.OpAccessDenied, "GET /api/ops-log")
		}
		if got.userID == nil || *got.userID != 7 {
			t.Errorf("userID = %v, want 7", got.userID)
		}
	})

	t.Run("роль admin", func(t *testing.T) {
		st.logged = nil

		rec := getAs(t, handler, "/api/ops-log", adminToken)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
		}
		if len(st.logged) != 0 {
			t.Errorf("logged = %v, want none: успешный доступ не событие отказа", st.logged)
		}

		var entries []opslog.EntryDTO
		if err := json.Unmarshal(rec.Body.Bytes(), &entries); err != nil {
			t.Fatalf("decode body %q: %v", rec.Body.String(), err)
		}
		if len(entries) != 2 {
			t.Fatalf("len(entries) = %d, want 2", len(entries))
		}
		if entries[0].ID != 3 || entries[0].Op != opslog.OpAccessDenied {
			t.Errorf("entry[0] = %+v, want the denied access record", entries[0])
		}
		if entries[1].UserID != nil {
			t.Errorf("entry[1].UserID = %d, want nil for the anonymous record", *entries[1].UserID)
		}
	})
}

func TestOpsLogPageIsPassedToStore(t *testing.T) {
	backend := stubAccounts(t)
	defer backend.Close()

	st := &memStore{}
	handler := New(opsLogConfig(t, backend.URL), st, discardLogger())

	rec := getAs(t, handler, "/api/ops-log?limit=5&offset=10", adminToken)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	if st.listLimit != 5 || st.listOffset != 10 {
		t.Errorf("store got limit=%d offset=%d, want 5/10", st.listLimit, st.listOffset)
	}
	if body := rec.Body.String(); body != "[]" {
		t.Errorf("body = %q, want an empty array", body)
	}

	rec = getAs(t, handler, "/api/ops-log?limit=101", adminToken)
	assertJSONError(t, rec, http.StatusBadRequest, httpx.CodeValidation)
}

// TestDeniedRequestsAreJournaled проверяет, что уже существующие точки отказа
// пишут access.denied: роль пользователя не подошла на административном
// маршруте.
func TestDeniedRequestsAreJournaled(t *testing.T) {
	backend := stubAccounts(t)
	defer backend.Close()

	st := &memStore{}
	handler := New(opsLogConfig(t, backend.URL), st, discardLogger())

	cases := []struct {
		target string
		token  string
		want   string
	}{
		{"/api/users", userToken, "GET /api/users"},
		{"/api/ops-log", userToken, "GET /api/ops-log"},
	}

	for _, tc := range cases {
		t.Run(tc.target, func(t *testing.T) {
			st.logged = nil

			rec := getAs(t, handler, tc.target, tc.token)
			assertJSONError(t, rec, http.StatusForbidden, httpx.CodeForbidden)

			if len(st.logged) != 1 {
				t.Fatalf("logged = %d, want 1", len(st.logged))
			}
			if got := st.logged[0]; got.op != opslog.OpAccessDenied || got.ref != tc.want || got.ok {
				t.Errorf("record = %q %q ok=%v, want %q %q ok=false",
					got.op, got.ref, got.ok, opslog.OpAccessDenied, tc.want)
			}
		})
	}
}

// TestOpsLogRouteNeedsStore фиксирует, что без хранилища маршрут не открывается:
// пустой ответ выглядел бы как «журнал пуст», хотя читать нечего.
func TestOpsLogRouteNeedsStore(t *testing.T) {
	backend := stubAccounts(t)
	defer backend.Close()

	handler := New(opsLogConfig(t, backend.URL), nil, discardLogger())

	rec := getAs(t, handler, "/api/ops-log", adminToken)

	assertJSONError(t, rec, http.StatusNotFound, httpx.CodeNotFound)
}

// TestJournalFailureDoesNotChangeAnswer проверяет, что ошибка записи журнала
// не влияет на ответ клиента: отказ в доступе остаётся отказом.
func TestJournalFailureDoesNotChangeAnswer(t *testing.T) {
	backend := stubAccounts(t)
	defer backend.Close()

	handler := New(opsLogConfig(t, backend.URL), &memStore{logErr: errWriteFailed}, discardLogger())

	rec := getAs(t, handler, "/api/ops-log", userToken)

	assertJSONError(t, rec, http.StatusForbidden, httpx.CodeForbidden)
}

func TestOpsLogRejectsWrongMethod(t *testing.T) {
	backend := stubAccounts(t)
	defer backend.Close()

	handler := New(opsLogConfig(t, backend.URL), &memStore{}, discardLogger())

	req := httptest.NewRequest(http.MethodPost, "/api/ops-log", strings.NewReader("{}"))
	req.Header.Set("Authorization", "Bearer "+adminToken)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertJSONError(t, rec, http.StatusMethodNotAllowed, httpx.CodeMethodNotAllowed)
}

// errWriteFailed — имитация сбоя записи в ops_log.
var errWriteFailed = errors.New("ops_log insert failed")
