package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/auth"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/httpx"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/methods"
)

const (
	methodsListPath = "/api/getAllEncryptionMethods"
	methodsSetPath  = "/api/setEncryptionMethodAvailability"
)

// setMethod отправляет POST /api/setEncryptionMethodAvailability. Токен
// передаётся cookie session, как это делает интерфейс.
func setMethod(t *testing.T, h http.Handler, token, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, methodsSetPath, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	if token != "" {
		req.AddCookie(&http.Cookie{Name: auth.SessionCookie, Value: token})
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	return rec
}

func TestMethodsGETIsPublic(t *testing.T) {
	backend := stubAccounts(t)
	defer backend.Close()

	st := newMemStore()
	h := New(testConfig(t, func(env map[string]string) { env["ACCOUNTS_URL"] = backend.URL }), st, discardLogger())

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, methodsListPath, http.NoBody))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}

	var got []methods.EncryptionMethodDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body %q: %v", rec.Body.String(), err)
	}

	// Список отсортирован по коду: kuznyechik раньше rsa.
	if len(got) != 2 || got[0].Name != methods.NameKuznyechik || got[1].Name != methods.NameRSA {
		t.Errorf("body = %+v, want [Кузнечик RSA-32768]", got)
	}
}

func TestMethodsSetRequiresAdmin(t *testing.T) {
	backend := stubAccounts(t)
	defer backend.Close()

	h := New(testConfig(t, func(env map[string]string) { env["ACCOUNTS_URL"] = backend.URL }), newMemStore(), discardLogger())

	cases := []struct {
		name   string
		token  string
		status int
		code   string
	}{
		{"без токена", "", http.StatusUnauthorized, httpx.CodeUnauthorized},
		{"роль user", userToken, http.StatusForbidden, httpx.CodeForbidden},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := setMethod(t, h, tc.token, `{"name":"RSA-32768","enabled":false}`)
			assertJSONError(t, rec, tc.status, tc.code)
		})
	}
}

func TestMethodsSetByAdmin(t *testing.T) {
	backend := stubAccounts(t)
	defer backend.Close()

	st := newMemStore()
	h := New(testConfig(t, func(env map[string]string) { env["ACCOUNTS_URL"] = backend.URL }), st, discardLogger())

	rec := setMethod(t, h, adminToken, `{"name":"RSA-32768","enabled":false}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}

	if !st.setOK {
		t.Error("изменение не дошло до хранилища")
	}

	if len(st.ops) != 1 || st.ops[0] != "method.toggle rsa" {
		t.Errorf("журнал = %v, want [method.toggle rsa]", st.ops)
	}

	// Тот же метод в публичном списке теперь выключен.
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, methodsListPath, http.NoBody))

	var got []methods.EncryptionMethodDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body %q: %v", rec.Body.String(), err)
	}

	for _, m := range got {
		if m.Name == methods.NameRSA && m.Enabled {
			t.Error("RSA остался включённым после переключения")
		}
	}
}

func TestMethodsSetUnknownCode(t *testing.T) {
	backend := stubAccounts(t)
	defer backend.Close()

	h := New(testConfig(t, func(env map[string]string) { env["ACCOUNTS_URL"] = backend.URL }), newMemStore(), discardLogger())

	rec := setMethod(t, h, adminToken, `{"name":"ГОСТ","enabled":false}`)

	assertJSONError(t, rec, http.StatusNotFound, httpx.CodeNotFound)
}

func TestMethodsSetBodyLimit(t *testing.T) {
	backend := stubAccounts(t)
	defer backend.Close()

	h := New(testConfig(t, func(env map[string]string) { env["ACCOUNTS_URL"] = backend.URL }), newMemStore(), discardLogger())

	body := `{"name":"rsa","enabled":false,"pad":"` + strings.Repeat("x", int(methods.MaxSetBodyBytes)+1) + `"}`

	rec := setMethod(t, h, adminToken, body)

	assertJSONError(t, rec, http.StatusRequestEntityTooLarge, httpx.CodePayloadTooLarge)
}

func TestMethodRoutesAreNotRegisteredWithoutStore(t *testing.T) {
	backend := stubAccounts(t)
	defer backend.Close()

	h := New(testConfig(t, func(env map[string]string) { env["ACCOUNTS_URL"] = backend.URL }), nil, discardLogger())

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, methodsListPath, http.NoBody))

	assertJSONError(t, rec, http.StatusNotFound, httpx.CodeNotFound)
}

// Без ACCOUNTS_URL проверять роль нечем, поэтому публичный список остаётся,
// а переключение не регистрируется.
func TestMethodsSetIsNotRegisteredWithoutAccounts(t *testing.T) {
	st := newMemStore()
	h := New(stubConfig(nil), st, discardLogger())

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, methodsListPath, http.NoBody))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 для публичного списка", rec.Code)
	}

	rec = setMethod(t, h, adminToken, `{"name":"RSA-32768","enabled":false}`)
	assertJSONError(t, rec, http.StatusNotFound, httpx.CodeNotFound)

	if len(st.ops) != 0 {
		t.Errorf("журнал = %v, want пустой", st.ops)
	}
}

func TestRolesIsPublic(t *testing.T) {
	h := New(stubConfig(nil), nil, discardLogger())

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/roles", http.NoBody))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}

	var got []auth.RoleDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body %q: %v", rec.Body.String(), err)
	}

	if len(got) != 2 || got[0].ID != auth.RoleUserID || got[1].Name != auth.RoleAdmin {
		t.Errorf("body = %+v, want [user admin]", got)
	}
}
