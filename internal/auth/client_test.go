package auth

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// stubAccounts создаёт сервер-заглушку introspect и возвращает клиента на него
// вместе со счётчиком обращений.
func stubAccounts(t *testing.T, handler http.HandlerFunc) (client *AccountsClient, calls *int) {
	t.Helper()

	calls = new(int)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*calls++
		handler(w, r)
	}))
	t.Cleanup(server.Close)

	return NewAccountsClient(mustParseURL(t, server.URL), 2*time.Second, discardLogger()), calls
}

func fullSession() introspectResponse {
	return introspectResponse{Active: true, Stage: stageFull, UserID: 7, Login: "ivanov", Role: string(RoleUser)}
}

// writeSession отдаёт ответ introspect как JSON.
func writeSession(w http.ResponseWriter, status int, body introspectResponse) {
	w.Header().Set("Content-Type", contentTypeJSON)
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(body); err != nil {
		// Заголовок уже отправлен, сообщить об ошибке некуда.
		_ = err
	}
}

// writeRaw отдаёт тело как есть — для битых и нежелательных ответов.
func writeRaw(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", contentTypeJSON)
	w.WriteHeader(status)

	_, _ = w.Write([]byte(body))
}

func TestIntrospectSuccess(t *testing.T) {
	client, calls := stubAccounts(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != introspectPath {
			t.Errorf("path = %q, want %q", r.URL.Path, introspectPath)
		}

		var req introspectRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Token != "good-token" {
			t.Errorf("token = %q, want good-token", req.Token)
		}

		writeSession(w, http.StatusOK, fullSession())
	})

	principal, err := client.Introspect(context.Background(), "good-token")
	if err != nil {
		t.Fatalf("Introspect() error = %v", err)
	}

	want := Principal{UserID: 7, Login: "ivanov", Role: RoleUser}
	if principal != want {
		t.Errorf("principal = %+v, want %+v", principal, want)
	}
	if *calls != 1 {
		t.Errorf("calls = %d, want 1: повторов у introspect быть не должно", *calls)
	}
}

func TestIntrospectUnauthorized(t *testing.T) {
	cases := []struct {
		name string
		body introspectResponse
	}{
		{"inactive", introspectResponse{Active: false, Stage: stageFull, UserID: 7, Login: "ivanov", Role: "user"}},
		{"stage password", introspectResponse{Active: true, Stage: "password", UserID: 7, Login: "ivanov", Role: "user"}},
		{"stage missing", introspectResponse{Active: true, UserID: 7, Login: "ivanov", Role: "user"}},
		{"zero user id", introspectResponse{Active: true, Stage: stageFull, UserID: 0, Login: "ivanov", Role: "user"}},
		{"negative user id", introspectResponse{Active: true, Stage: stageFull, UserID: -1, Login: "ivanov", Role: "user"}},
		{"empty login", introspectResponse{Active: true, Stage: stageFull, UserID: 7, Login: "", Role: "user"}},
		{"unknown role", introspectResponse{Active: true, Stage: stageFull, UserID: 7, Login: "ivanov", Role: "root"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client, _ := stubAccounts(t, func(w http.ResponseWriter, _ *http.Request) {
				writeSession(w, http.StatusOK, tc.body)
			})

			if _, err := client.Introspect(context.Background(), "token"); !errors.Is(err, ErrUnauthorized) {
				t.Fatalf("error = %v, want ErrUnauthorized", err)
			}
		})
	}
}

func TestIntrospectUnavailable(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
	}{
		{"broken json", http.StatusOK, "{not json"},
		{"empty body", http.StatusOK, ""},
		{"too large body", http.StatusOK, `{"active":true,"stage":"full","user_id":7,"login":"` + strings.Repeat("x", maxIntrospectBody) + `","role":"user"}`},
		{"bad request", http.StatusBadRequest, "{}"},
		{"not found", http.StatusNotFound, "{}"},
		{"rate limited", http.StatusTooManyRequests, "{}"},
		{"server error", http.StatusInternalServerError, "{}"},
		{"bad gateway", http.StatusBadGateway, "{}"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			client, _ := stubAccounts(t, func(w http.ResponseWriter, _ *http.Request) {
				writeRaw(w, tc.status, tc.body)
			})

			if _, err := client.Introspect(context.Background(), "token"); !errors.Is(err, ErrAccountsUnavailable) {
				t.Fatalf("error = %v, want ErrAccountsUnavailable", err)
			}
		})
	}
}

func TestIntrospectRejects(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			client, _ := stubAccounts(t, func(w http.ResponseWriter, _ *http.Request) {
				writeRaw(w, status, "{}")
			})

			if _, err := client.Introspect(context.Background(), "token"); !errors.Is(err, ErrUnauthorized) {
				t.Fatalf("error = %v, want ErrUnauthorized", err)
			}
		})
	}
}

func TestIntrospectTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Millisecond)
		writeSession(w, http.StatusOK, fullSession())
	}))
	defer server.Close()

	client := NewAccountsClient(mustParseURL(t, server.URL), 10*time.Millisecond, discardLogger())

	if _, err := client.Introspect(context.Background(), "token"); !errors.Is(err, ErrAccountsUnavailable) {
		t.Fatalf("error = %v, want ErrAccountsUnavailable", err)
	}
}

func TestIntrospectConnectionRefused(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	target := mustParseURL(t, server.URL)
	server.Close()

	client := NewAccountsClient(target, time.Second, discardLogger())

	if _, err := client.Introspect(context.Background(), "token"); !errors.Is(err, ErrAccountsUnavailable) {
		t.Fatalf("error = %v, want ErrAccountsUnavailable", err)
	}
}

func TestIntrospectCanceledContext(t *testing.T) {
	client, _ := stubAccounts(t, func(w http.ResponseWriter, _ *http.Request) {
		writeSession(w, http.StatusOK, fullSession())
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := client.Introspect(ctx, "token"); !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}

func TestIntrospectDoesNotFollowRedirect(t *testing.T) {
	var hits int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++

		if r.URL.Path == introspectPath {
			http.Redirect(w, r, "/elsewhere", http.StatusTemporaryRedirect)

			return
		}

		writeSession(w, http.StatusOK, fullSession())
	}))
	defer server.Close()

	client := NewAccountsClient(mustParseURL(t, server.URL), time.Second, discardLogger())

	if _, err := client.Introspect(context.Background(), "token"); !errors.Is(err, ErrAccountsUnavailable) {
		t.Fatalf("error = %v, want ErrAccountsUnavailable", err)
	}
	if hits != 1 {
		t.Errorf("server hits = %d, want 1: редирект не должен обходиться", hits)
	}
}

func TestIntrospectWithoutURL(t *testing.T) {
	client := NewAccountsClient(nil, time.Second, discardLogger())

	if _, err := client.Introspect(context.Background(), "token"); !errors.Is(err, ErrAccountsUnavailable) {
		t.Fatalf("error = %v, want ErrAccountsUnavailable", err)
	}
}

func TestIntrospectKeepsBasePath(t *testing.T) {
	var path string

	client, _ := stubAccounts(t, func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		writeSession(w, http.StatusOK, fullSession())
	})

	client.base.Path = "/api-v2"

	if _, err := client.Introspect(context.Background(), "token"); err != nil {
		t.Fatalf("Introspect() error = %v", err)
	}
	if path != "/api-v2"+introspectPath {
		t.Errorf("path = %q, want %q", path, "/api-v2"+introspectPath)
	}
}

func TestIntrospectSendsTokenInBody(t *testing.T) {
	const secret = "super-secret-token"

	var body string

	client, _ := stubAccounts(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		body = string(b)
		writeSession(w, http.StatusOK, fullSession())
	})

	if _, err := client.Introspect(context.Background(), secret); err != nil {
		t.Fatalf("Introspect() error = %v", err)
	}

	if !strings.Contains(body, secret) {
		t.Errorf("request body = %q, want token sent to accounts", body)
	}
}
