package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/httpx"
)

// okHandler отвечает 200 и запоминает переданный Principal.
func okHandler(seen *Principal) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if p, ok := PrincipalFrom(r.Context()); ok {
			*seen = p
		}

		w.WriteHeader(http.StatusOK)
	})
}

func TestAuthnRejectsMalformedHeader(t *testing.T) {
	client, calls := stubAccounts(t, func(w http.ResponseWriter, _ *http.Request) {
		writeSession(w, http.StatusOK, fullSession())
	})

	cases := []struct {
		name   string
		header []string
	}{
		{"missing", nil},
		{"wrong scheme", []string{"Token abc"}},
		{"no scheme", []string{"abc"}},
		{"empty token", []string{"Bearer "}},
		{"two headers", []string{"Bearer a", "Bearer b"}},
		{"too long", []string{"Bearer " + strings.Repeat("a", maxTokenLen+1)}},
		{"token with space", []string{"Bearer a b"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			*calls = 0

			req := httptest.NewRequest(http.MethodGet, "/api/users", http.NoBody)
			for _, v := range tc.header {
				req.Header.Add(headerAuthorization, v)
			}

			rec := httptest.NewRecorder()
			var seen Principal

			Authn(client)(okHandler(&seen)).ServeHTTP(rec, req)

			assertErrorResponse(t, rec, http.StatusUnauthorized, httpx.CodeUnauthorized)
			if *calls != 0 {
				t.Errorf("introspect calls = %d, want 0", *calls)
			}
		})
	}
}

func TestAuthnAcceptsSchemeWithoutCase(t *testing.T) {
	client, _ := stubAccounts(t, func(w http.ResponseWriter, _ *http.Request) {
		writeSession(w, http.StatusOK, fullSession())
	})

	rec := httptest.NewRecorder()
	var seen Principal

	req := httptest.NewRequest(http.MethodGet, "/api/users", http.NoBody)
	req.Header.Set(headerAuthorization, "bearer good-token")

	Authn(client)(okHandler(&seen)).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	if want := (Principal{UserID: 7, Login: "ivanov", Role: RoleUser}); seen != want {
		t.Errorf("principal = %+v, want %+v", seen, want)
	}
}

func TestAuthnInvalidSession(t *testing.T) {
	cases := []struct {
		name string
		body introspectResponse
	}{
		{"inactive", introspectResponse{Active: false, Stage: stageFull, UserID: 7, Login: "ivanov", Role: "user"}},
		{"partial stage", introspectResponse{Active: true, Stage: "password", UserID: 7, Login: "ivanov", Role: "user"}},
		{"stage missing", introspectResponse{Active: true, UserID: 7, Login: "ivanov", Role: "user"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client, _ := stubAccounts(t, func(w http.ResponseWriter, _ *http.Request) {
				writeSession(w, http.StatusOK, tc.body)
			})

			rec, seen := serveAuthn(t, client, "any-token")

			assertErrorResponse(t, rec, http.StatusUnauthorized, httpx.CodeUnauthorized)
			if seen != (Principal{}) {
				t.Errorf("principal = %+v, want empty", seen)
			}
		})
	}
}

func TestAuthnAccountsUnavailable(t *testing.T) {
	client, _ := stubAccounts(t, func(w http.ResponseWriter, _ *http.Request) {
		writeRaw(w, http.StatusInternalServerError, "{}")
	})

	rec, _ := serveAuthn(t, client, "any-token")

	assertErrorResponse(t, rec, http.StatusServiceUnavailable, httpx.CodeAccountsUnavailable)
}

func TestAuthnDoesNotWriteAfterCancel(t *testing.T) {
	client, _ := stubAccounts(t, func(w http.ResponseWriter, _ *http.Request) {
		writeSession(w, http.StatusOK, fullSession())
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	req := httptest.NewRequest(http.MethodGet, "/api/users", http.NoBody).WithContext(ctx)
	req.Header.Set(headerAuthorization, "Bearer good-token")

	rec := httptest.NewRecorder()

	Authn(client)(okHandler(new(Principal))).ServeHTTP(rec, req)

	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty", rec.Body.String())
	}
}

func TestAuthnKeepsTokenOutOfResponse(t *testing.T) {
	client, _ := stubAccounts(t, func(w http.ResponseWriter, _ *http.Request) {
		writeSession(w, http.StatusOK, fullSession())
	})

	rec, _ := serveAuthn(t, client, "super-secret-token")

	if strings.Contains(rec.Body.String(), "super-secret-token") {
		t.Errorf("response leaks token: %s", rec.Body.String())
	}
}

func TestRequireAllowsOnlyRole(t *testing.T) {
	cases := []struct {
		name       string
		principal  *Principal
		wantStatus int
	}{
		{"admin", &Principal{UserID: 1, Login: "admin", Role: RoleAdmin}, http.StatusOK},
		{"user", &Principal{UserID: 7, Login: "ivanov", Role: RoleUser}, http.StatusForbidden},
		{"anonymous", nil, http.StatusUnauthorized},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var denied []string

			audit := func(_ context.Context, _ *int64, op, ref string, ok bool) {
				denied = append(denied, op+"|"+ref)
				if ok {
					t.Error("ok = true, want false for denied access")
				}
			}

			req := httptest.NewRequest(http.MethodGet, "/api/users", http.NoBody)
			if tc.principal != nil {
				req = req.WithContext(WithPrincipal(req.Context(), *tc.principal))
			}

			rec := httptest.NewRecorder()
			Require(RoleAdmin, audit)(okHandler(new(Principal))).ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}

			wantDenied := 0
			if tc.wantStatus == http.StatusForbidden {
				wantDenied = 1
			}

			if len(denied) != wantDenied {
				t.Fatalf("audit records = %v, want %d", denied, wantDenied)
			}
		})
	}
}

func TestRequireTruncatesAuditRef(t *testing.T) {
	var ref string

	audit := func(_ context.Context, _ *int64, _, value string, _ bool) { ref = value }

	req := httptest.NewRequest(http.MethodGet, "/api/users/"+strings.Repeat("a", 300), http.NoBody)
	req = req.WithContext(WithPrincipal(req.Context(), Principal{UserID: 7, Role: RoleUser}))

	Require(RoleAdmin, audit)(okHandler(new(Principal))).ServeHTTP(httptest.NewRecorder(), req)

	if len(ref) != maxAuditRefLen {
		t.Errorf("ref length = %d, want %d", len(ref), maxAuditRefLen)
	}
}

func TestRequireWithoutAuditStillDenies(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/users", http.NoBody)
	req = req.WithContext(WithPrincipal(req.Context(), Principal{UserID: 7, Login: "ivanov", Role: RoleUser}))

	rec := httptest.NewRecorder()
	Require(RoleAdmin, nil)(okHandler(new(Principal))).ServeHTTP(rec, req)

	assertErrorResponse(t, rec, http.StatusForbidden, httpx.CodeForbidden)
}

func TestPrincipalFromEmptyContext(t *testing.T) {
	if _, ok := PrincipalFrom(context.Background()); ok {
		t.Error("PrincipalFrom() reported a principal for an empty context")
	}
}

// serveAuthn прогоняет Authn с токеном token и возвращает ответ и увиденного пользователя.
func serveAuthn(t *testing.T, client *AccountsClient, token string) (*httptest.ResponseRecorder, Principal) {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/api/users", http.NoBody)
	req.Header.Set(headerAuthorization, "Bearer "+token)

	rec := httptest.NewRecorder()
	var seen Principal

	Authn(client)(okHandler(&seen)).ServeHTTP(rec, req)

	return rec, seen
}
