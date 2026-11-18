package messages

import (
	"errors"
	"net/http"
	"testing"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/auth"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/httpx"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/methods"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/ports"
)

const listPath = "/api/getAllMessages"

func (e *env) list(t *testing.T, p auth.Principal) []MessageDTO {
	t.Helper()

	rec := e.do(t, p, http.MethodGet, listPath, "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}

	if cc := rec.Header().Get("Cache-Control"); cc != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", cc)
	}

	return decodeBody[[]MessageDTO](t, rec)
}

// seedMessages: два сообщения пользователя (RSA, Кузнечик) и одно чужое.
func seedMessages(t *testing.T, e *env) {
	t.Helper()

	e.encrypt(t, userPrincipal(), "первое", methods.NameRSA)
	e.encrypt(t, userPrincipal(), "второе", methods.NameKuznyechik)
	e.encrypt(t, otherPrincipal(), "чужое", methods.NameKuznyechik)
}

func TestListOwnMessages(t *testing.T) {
	e := newEnv(t)
	seedMessages(t, e)

	got := e.list(t, userPrincipal())

	if len(got) != 2 || got[0].Plaintext != "второе" || got[1].Plaintext != "первое" {
		t.Fatalf("messages = %+v, want own messages newest first", got)
	}

	if got[0].Keys != (KeysDTO{Key: testKuzKey, IV: testKuzIV}) {
		t.Errorf("kuznyechik keys = %+v", got[0].Keys)
	}

	if got[1].Keys != (KeysDTO{PublicKey: testPubPEM, PrivateKey: testPrivPEM}) {
		t.Errorf("rsa keys = %+v", got[1].Keys)
	}

	if owner := e.store.listOwners[0]; owner == nil || *owner != testUserID {
		t.Errorf("ListMessages owner = %v, want %d", owner, testUserID)
	}
}

func TestListAdminSeesAll(t *testing.T) {
	e := newEnv(t)
	seedMessages(t, e)

	got := e.list(t, adminPrincipal())

	if len(got) != 3 || got[0].User.Name != otherLogin {
		t.Errorf("messages = %+v, want all three", got)
	}

	if e.store.listOwners[0] != nil {
		t.Errorf("ListMessages owner = %v, want nil for admin", *e.store.listOwners[0])
	}
}

func TestListEmptyIsArray(t *testing.T) {
	e := newEnv(t)

	rec := e.do(t, userPrincipal(), http.MethodGet, listPath, "", "")
	if body := rec.Body.String(); body != "[]\n" && body != "[]" {
		t.Errorf("body = %q, want []", body)
	}
}

// Сообщения читаются страницами: больше одной страницы тоже доходит целиком.
func TestListPages(t *testing.T) {
	e := newEnv(t)

	for range listPageSize + 5 {
		e.store.created = append(e.store.created, &ports.Message{
			ID: int64(len(e.store.created) + 1), OwnerID: testUserID, Method: string(ports.MethodRSA), Ciphertext: []byte{1},
		})
	}

	if got := e.list(t, userPrincipal()); len(got) != listPageSize+5 {
		t.Errorf("len = %d, want %d", len(got), listPageSize+5)
	}
}

// Без ключа владельца RSA-сообщение отдаётся без ключей, список не ломается.
func TestListWithoutOwnerKey(t *testing.T) {
	e := newEnv(t)
	seedMessages(t, e)
	delete(e.store.userKeys, testUserID)

	got := e.list(t, userPrincipal())

	if len(got) != 2 || got[1].Keys != (KeysDTO{}) {
		t.Errorf("messages = %+v, want rsa message without keys", got)
	}
}

func TestListErrors(t *testing.T) {
	tests := []struct {
		name  string
		setup func(e *env)
	}{
		{"ошибка списка", func(e *env) { e.store.listErr = errors.New("db down") }},
		{"ошибка ключа", func(e *env) { e.store.keyErr = errors.New("db down") }},
		{"повреждённый ключ Кузнечика", func(e *env) { e.store.created[1].KuzKeySealed[0] ^= 1 }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			seedMessages(t, e)
			tt.setup(e)

			rec := e.do(t, userPrincipal(), http.MethodGet, listPath, "", "")
			assertError(t, rec, http.StatusInternalServerError, httpx.CodeInternal)
		})
	}
}
