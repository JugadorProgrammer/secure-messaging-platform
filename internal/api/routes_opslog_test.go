package api

import (
	"context"
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

// denial — запись отказа, которую сервис должен отправить в журнал.
type denial struct {
	userID *int64
	op     string
	ref    string
	ok     bool
}

// memStore реализует ports.Store для проверки журнала: LogOp считает записи
// отказов, ListOps отдаёт заготовленные записи. Остальные методы не
// используются маршрутами этого теста и отвечают ports.ErrNotFound.
type memStore struct {
	denials []denial
	entries []ports.OpLogEntry
	limit   int
	start   int
}

func (m *memStore) LogOp(_ context.Context, userID *int64, op, ref string, ok bool) error {
	m.denials = append(m.denials, denial{userID: userID, op: op, ref: ref, ok: ok})

	return nil
}

func (m *memStore) ListOps(_ context.Context, limit, offset int) ([]ports.OpLogEntry, error) {
	m.limit = limit
	m.start = offset
	if m.entries != nil {
		return m.entries, nil
	}

	return []ports.OpLogEntry{}, nil
}

func (m *memStore) Methods(context.Context) ([]ports.MethodRow, error) {
	return nil, ports.ErrNotFound
}

func (m *memStore) SetMethodEnabled(context.Context, string, bool) error {
	return ports.ErrNotFound
}

func (m *memStore) UserKey(context.Context, int64) (*ports.UserKey, error) {
	return nil, ports.ErrNotFound
}

func (m *memStore) SaveUserKey(context.Context, *ports.UserKey) error {
	return ports.ErrNotFound
}

func (m *memStore) CreateMessage(context.Context, *ports.Message) (int64, error) {
	return 0, ports.ErrNotFound
}

func (m *memStore) GetMessage(context.Context, int64) (*ports.Message, error) {
	return nil, ports.ErrNotFound
}

func (m *memStore) ListMessages(context.Context, *int64, int, int) ([]*ports.Message, error) {
	return nil, ports.ErrNotFound
}

func (m *memStore) SetDecryptMs(context.Context, int64, int64) error {
	return ports.ErrNotFound
}

var _ ports.Store = (*memStore)(nil)

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
		if len(st.denials) != 0 {
			t.Errorf("denials = %v, want none: отказ без аутентификации не пишется", st.denials)
		}
	})

	t.Run("роль user", func(t *testing.T) {
		rec := getAs(t, handler, "/api/ops-log", userToken)

		assertJSONError(t, rec, http.StatusForbidden, httpx.CodeForbidden)

		if len(st.denials) != 1 {
			t.Fatalf("denials = %d, want 1", len(st.denials))
		}

		got := st.denials[0]
		if got.op != opslog.OpAccessDenied || got.ref != "GET /api/ops-log" || got.ok {
			t.Errorf("record = %q %q ok=%v, want %q %q ok=false",
				got.op, got.ref, got.ok, opslog.OpAccessDenied, "GET /api/ops-log")
		}
		if got.userID == nil || *got.userID != 7 {
			t.Errorf("userID = %v, want 7", got.userID)
		}
	})

	t.Run("роль admin", func(t *testing.T) {
		st.denials = nil

		rec := getAs(t, handler, "/api/ops-log", adminToken)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
		}
		if len(st.denials) != 0 {
			t.Errorf("denials = %v, want none: успешный доступ не событие отказа", st.denials)
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
	if st.limit != 5 || st.start != 10 {
		t.Errorf("store got limit=%d offset=%d, want 5/10", st.limit, st.start)
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
			st.denials = nil

			rec := getAs(t, handler, tc.target, tc.token)
			assertJSONError(t, rec, http.StatusForbidden, httpx.CodeForbidden)

			if len(st.denials) != 1 {
				t.Fatalf("denials = %d, want 1", len(st.denials))
			}
			if got := st.denials[0]; got.op != opslog.OpAccessDenied || got.ref != tc.want || got.ok {
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

	handler := New(opsLogConfig(t, backend.URL), failingStore{&memStore{}}, discardLogger())

	rec := getAs(t, handler, "/api/ops-log", userToken)

	assertJSONError(t, rec, http.StatusForbidden, httpx.CodeForbidden)
}

// failingStore — хранилище, в котором запись журнала не удаётся.
type failingStore struct{ *memStore }

func (failingStore) LogOp(context.Context, *int64, string, string, bool) error {
	return errors.New("ops_log insert failed")
}

// int64Ptr возвращает указатель на значение.
func int64Ptr(v int64) *int64 {
	return &v
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
