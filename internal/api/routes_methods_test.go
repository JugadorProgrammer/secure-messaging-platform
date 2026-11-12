package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/httpx"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/methods"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/ports"
)

// patchMethod отправляет PATCH /api/methods/{code} через собранный обработчик.
func patchMethod(t *testing.T, h http.Handler, code, token, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPatch, "/api/methods/"+code, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
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
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/methods", http.NoBody))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}

	var got []methods.MethodDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body %q: %v", rec.Body.String(), err)
	}

	// Список отсортирован по коду: kuznyechik раньше rsa.
	if len(got) != 2 || got[0].Code != ports.MethodKuznyechik || got[1].Code != ports.MethodRSA {
		t.Errorf("body = %+v, want [kuznyechik rsa]", got)
	}
}

func TestMethodsPATCHRequiresAdmin(t *testing.T) {
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
			rec := patchMethod(t, h, "rsa", tc.token, `{"enabled":false}`)
			assertJSONError(t, rec, tc.status, tc.code)
		})
	}
}

func TestMethodsPATCHByAdmin(t *testing.T) {
	backend := stubAccounts(t)
	defer backend.Close()

	st := newMemStore()
	h := New(testConfig(t, func(env map[string]string) { env["ACCOUNTS_URL"] = backend.URL }), st, discardLogger())

	rec := patchMethod(t, h, "rsa", adminToken, `{"enabled":false}`)

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
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/methods", http.NoBody))

	var got []methods.MethodDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body %q: %v", rec.Body.String(), err)
	}

	for _, m := range got {
		if m.Code == ports.MethodRSA && m.Enabled {
			t.Error("rsa остался включённым после PATCH")
		}
	}
}

func TestMethodsPATCHUnknownCode(t *testing.T) {
	backend := stubAccounts(t)
	defer backend.Close()

	h := New(testConfig(t, func(env map[string]string) { env["ACCOUNTS_URL"] = backend.URL }), newMemStore(), discardLogger())

	rec := patchMethod(t, h, "gost", adminToken, `{"enabled":false}`)

	assertJSONError(t, rec, http.StatusNotFound, httpx.CodeNotFound)
}

func TestMethodsPATCHBodyLimit(t *testing.T) {
	backend := stubAccounts(t)
	defer backend.Close()

	h := New(testConfig(t, func(env map[string]string) { env["ACCOUNTS_URL"] = backend.URL }), newMemStore(), discardLogger())

	body := `{"enabled":false,"pad":"` + strings.Repeat("x", int(methods.MaxPatchBodyBytes)+1) + `"}`

	rec := patchMethod(t, h, "rsa", adminToken, body)

	assertJSONError(t, rec, http.StatusRequestEntityTooLarge, httpx.CodePayloadTooLarge)
}

func TestMethodRoutesAreNotRegisteredWithoutStore(t *testing.T) {
	backend := stubAccounts(t)
	defer backend.Close()

	h := New(testConfig(t, func(env map[string]string) { env["ACCOUNTS_URL"] = backend.URL }), nil, discardLogger())

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/methods", http.NoBody))

	assertJSONError(t, rec, http.StatusNotFound, httpx.CodeNotFound)
}

// Без ACCOUNTS_URL проверять роль нечем, поэтому публичный список остаётся,
// а PATCH не регистрируется.
func TestMethodsPATCHIsNotRegisteredWithoutAccounts(t *testing.T) {
	st := newMemStore()
	h := New(stubConfig(nil), st, discardLogger())

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/methods", http.NoBody))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 для публичного списка", rec.Code)
	}

	rec = patchMethod(t, h, "rsa", adminToken, `{"enabled":false}`)
	assertJSONError(t, rec, http.StatusNotFound, httpx.CodeNotFound)

	if len(st.ops) != 0 {
		t.Errorf("журнал = %v, want пустой", st.ops)
	}
}
