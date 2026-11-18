package methods

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/auth"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/httpx"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/ports"
)

const adminID = int64(1)

// newTestHandler собирает обработчик над хранилищем в памяти.
func newTestHandler(st ports.Store, audit AuditFunc) *Handler {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	return NewHandler(NewService(st), audit, log)
}

const (
	listPath = "/api/getAllEncryptionMethods"
	setPath  = "/api/setEncryptionMethodAvailability"
)

// newTestMux — маршруты так же, как их регистрирует api: изменение под лимит тела.
func newTestMux(h *Handler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("GET "+listPath, http.HandlerFunc(h.List))
	mux.Handle("POST "+setPath,
		httpx.BodyLimit(MaxSetBodyBytes)(http.HandlerFunc(h.SetAvailability)))

	return mux
}

// withName добавляет поле name в JSON-объект body.
func withName(t *testing.T, name, body string) string {
	t.Helper()

	encoded, err := json.Marshal(name)
	if err != nil {
		t.Fatalf("encode name: %v", err)
	}

	rest := strings.TrimPrefix(body, "{")
	if strings.TrimSpace(rest) == "}" {
		return `{"name":` + string(encoded) + `}`
	}

	return `{"name":` + string(encoded) + `,` + rest
}

// newSet строит POST изменения метода с JSON-телом и, если нужно,
// пользователем в контексте. name — название или код метода.
func newSet(t *testing.T, name, body string, principal *auth.Principal) *http.Request {
	t.Helper()

	if body == "" {
		return httptest.NewRequest(http.MethodPost, setPath, http.NoBody)
	}

	req := httptest.NewRequest(http.MethodPost, setPath, strings.NewReader(withName(t, name, body)))
	req.Header.Set("Content-Type", "application/json")

	if principal != nil {
		req = req.WithContext(auth.WithPrincipal(req.Context(), *principal))
	}

	return req
}

func adminPrincipal() *auth.Principal {
	return &auth.Principal{UserID: adminID, Login: "admin", Role: auth.RoleAdmin}
}

func TestHandlerListReturnsSortedMethods(t *testing.T) {
	h := newTestHandler(newFakeStore(seedRows()...), nil)

	rec := httptest.NewRecorder()
	newTestMux(h).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, listPath, http.NoBody))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}

	var got []EncryptionMethodDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body %q: %v", rec.Body.String(), err)
	}

	want := []EncryptionMethodDTO{
		{Name: NameKuznyechik, Enabled: true},
		{Name: NameRSA, Enabled: true},
	}

	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("body = %+v, want %+v", got, want)
	}
}

func TestHandlerListEmptyIsJSONArray(t *testing.T) {
	h := newTestHandler(newFakeStore(), nil)

	rec := httptest.NewRecorder()
	newTestMux(h).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, listPath, http.NoBody))

	if body := strings.TrimSpace(rec.Body.String()); body != "[]" {
		t.Errorf("body = %q, want []", body)
	}
}

func TestHandlerListStoreErrorIsInternal(t *testing.T) {
	h := newTestHandler(&fakeStore{listErr: errors.New("connection reset")}, nil)

	rec := httptest.NewRecorder()
	newTestMux(h).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, listPath, http.NoBody))

	assertMethodError(t, rec, http.StatusInternalServerError, httpx.CodeInternal)

	if strings.Contains(rec.Body.String(), "connection reset") {
		t.Errorf("ответ раскрывает текст ошибки базы: %s", rec.Body.String())
	}
}

func TestHandlerSetDisablesMethod(t *testing.T) {
	st := newFakeStore(seedRows()...)
	h := newTestHandler(st, func(ctx context.Context, userID *int64, op, ref string, ok bool) {
		_ = st.LogOp(ctx, userID, op, ref, ok)
	})

	rec := httptest.NewRecorder()
	newTestMux(h).ServeHTTP(rec, newSet(t, "rsa", `{"enabled":false}`, adminPrincipal()))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}

	var dto EncryptionMethodDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &dto); err != nil {
		t.Fatalf("decode body %q: %v", rec.Body.String(), err)
	}

	if dto.Name != NameRSA || dto.Enabled {
		t.Errorf("dto = %+v, want rsa с enabled:false", dto)
	}

	ops := st.ops
	if len(ops) != 1 {
		t.Fatalf("ops_log записей = %d, want 1", len(ops))
	}

	if ops[0].op != opMethodToggle || ops[0].ref != "rsa" || !ops[0].ok {
		t.Errorf("ops_log = %+v, want method.toggle rsa ok=true", ops[0])
	}

	if ops[0].userID == nil || *ops[0].userID != adminID {
		t.Errorf("ops_log userID = %v, want %d", ops[0].userID, adminID)
	}
}

func TestHandlerSetEnablesMethod(t *testing.T) {
	st := newFakeStore(ports.MethodRow{Code: "rsa", Title: "RSA", KeySize: 2048, Enabled: false})
	h := newTestHandler(st, nil)

	rec := httptest.NewRecorder()
	newTestMux(h).ServeHTTP(rec, newSet(t, "rsa", `{"enabled":true}`, adminPrincipal()))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}

	var dto EncryptionMethodDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &dto); err != nil {
		t.Fatalf("decode body %q: %v", rec.Body.String(), err)
	}

	if !dto.Enabled {
		t.Errorf("dto = %+v, want enabled:true", dto)
	}
}

// Без указателя в DTO отсутствие поля стало бы false и метод выключился бы молча.
func TestHandlerSetWithoutEnabledField(t *testing.T) {
	st := newFakeStore(seedRows()...)

	var denied int

	audit := func(context.Context, *int64, string, string, bool) { denied++ }

	h := newTestHandler(st, audit)

	rec := httptest.NewRecorder()
	newTestMux(h).ServeHTTP(rec, newSet(t, "rsa", `{}`, adminPrincipal()))

	assertMethodError(t, rec, http.StatusBadRequest, httpx.CodeValidation)

	if len(st.setCalls) != 0 {
		t.Errorf("setCalls = %v, want пустой: метод не изменён", st.setCalls)
	}

	for _, row := range st.rows {
		if !row.Enabled {
			t.Errorf("row %+v выключен, want enabled:true", row)
		}
	}

	// Ошибка формата запроса — не событие журнала.
	if denied != 0 {
		t.Errorf("записей в журнал = %d, want 0", denied)
	}
}

func TestHandlerSetMalformedJSON(t *testing.T) {
	st := newFakeStore(seedRows()...)
	h := newTestHandler(st, nil)

	rec := httptest.NewRecorder()
	newTestMux(h).ServeHTTP(rec, newSet(t, "rsa", `{"enabled": "no"}`, adminPrincipal()))

	assertMethodError(t, rec, http.StatusBadRequest, httpx.CodeValidation)

	if len(st.setCalls) != 0 {
		t.Errorf("setCalls = %v, want пустой", st.setCalls)
	}
}

func TestHandlerSetRejectsNonJSONContentType(t *testing.T) {
	cases := []struct {
		name        string
		contentType string
	}{
		{"пустой Content-Type", ""},
		{"text/plain", "text/plain"},
		{"form", "application/x-www-form-urlencoded"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			st := newFakeStore(seedRows()...)
			h := newTestHandler(st, nil)

			req := httptest.NewRequest(http.MethodPost, setPath, strings.NewReader(`{"name":"rsa","enabled":false}`))
			if tc.contentType != "" {
				req.Header.Set("Content-Type", tc.contentType)
			}

			rec := httptest.NewRecorder()
			newTestMux(h).ServeHTTP(rec, req)

			assertMethodError(t, rec, http.StatusUnsupportedMediaType, httpx.CodeUnsupportedMediaType)

			if len(st.setCalls) != 0 {
				t.Errorf("setCalls = %v, want пустой", st.setCalls)
			}
		})
	}
}

func TestHandlerSetAcceptsContentTypeWithParams(t *testing.T) {
	st := newFakeStore(seedRows()...)
	h := newTestHandler(st, nil)

	req := httptest.NewRequest(http.MethodPost, setPath, strings.NewReader(`{"name":"rsa","enabled":false}`))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	rec := httptest.NewRecorder()
	newTestMux(h).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
}

func TestHandlerSetUnknownCode(t *testing.T) {
	st := newFakeStore(seedRows()...)

	var last opRecord

	audit := func(_ context.Context, _ *int64, op, ref string, ok bool) {
		last = opRecord{op: op, ref: ref, ok: ok}
	}

	h := newTestHandler(st, audit)

	rec := httptest.NewRecorder()
	newTestMux(h).ServeHTTP(rec, newSet(t, "gost", `{"enabled":false}`, adminPrincipal()))

	assertMethodError(t, rec, http.StatusNotFound, httpx.CodeNotFound)

	if len(st.setCalls) != 0 {
		t.Errorf("setCalls = %v, want пустой: в базу не пишем", st.setCalls)
	}

	if last.ref != "gost" || last.ok {
		t.Errorf("ops_log = %+v, want ref=gost ok=false", last)
	}
}

func TestHandlerSetLongCodeIsTruncatedInAudit(t *testing.T) {
	var last opRecord

	audit := func(_ context.Context, _ *int64, _, ref string, _ bool) { last = opRecord{ref: ref} }

	h := newTestHandler(newFakeStore(seedRows()...), audit)

	long := strings.Repeat("a", maxAuditRefLen+100)

	rec := httptest.NewRecorder()
	newTestMux(h).ServeHTTP(rec, newSet(t, long, `{"enabled":false}`, adminPrincipal()))

	assertMethodError(t, rec, http.StatusNotFound, httpx.CodeNotFound)

	if len(last.ref) != maxAuditRefLen {
		t.Errorf("длина ref = %d, want %d", len(last.ref), maxAuditRefLen)
	}
}

func TestHandlerSetAuditRefKeepsValidUTF8(t *testing.T) {
	var last opRecord

	audit := func(_ context.Context, _ *int64, _, ref string, _ bool) { last = opRecord{ref: ref} }

	h := newTestHandler(newFakeStore(seedRows()...), audit)

	// Обрезка по байтам не должна рвать многобайтовый символ.
	long := strings.Repeat("а", maxAuditRefLen) + "bb"

	rec := httptest.NewRecorder()
	newTestMux(h).ServeHTTP(rec, newSet(t, long, `{"enabled":false}`, adminPrincipal()))

	if !utf8.ValidString(last.ref) {
		t.Errorf("ref содержит битую последовательность: %q", last.ref)
	}
}

func TestHandlerSetBodyTooLarge(t *testing.T) {
	st := newFakeStore(seedRows()...)
	h := newTestHandler(st, nil)

	body := `{"name":"rsa","enabled":false,"padding":"` + strings.Repeat("x", int(MaxSetBodyBytes)+1024) + `"}`

	req := httptest.NewRequest(http.MethodPost, setPath, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// Без ContentLength лимит обнаруживается при чтении, и это тоже 413.
	req.ContentLength = -1

	rec := httptest.NewRecorder()
	newTestMux(h).ServeHTTP(rec, req)

	assertMethodError(t, rec, http.StatusRequestEntityTooLarge, httpx.CodePayloadTooLarge)

	if len(st.setCalls) != 0 {
		t.Errorf("setCalls = %v, want пустой", st.setCalls)
	}
}

func TestHandlerSetDeclaredBodyTooLarge(t *testing.T) {
	h := newTestHandler(newFakeStore(seedRows()...), nil)

	body := strings.Repeat("x", int(MaxSetBodyBytes)+1)

	req := httptest.NewRequest(http.MethodPost, setPath, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	newTestMux(h).ServeHTTP(rec, req)

	assertMethodError(t, rec, http.StatusRequestEntityTooLarge, httpx.CodePayloadTooLarge)
}

func TestHandlerSetStoreNotFound(t *testing.T) {
	st := newFakeStore(ports.MethodRow{Code: "kuznyechik", Title: "Кузнечик", KeySize: 256, Enabled: true})

	var last opRecord

	audit := func(_ context.Context, _ *int64, op, ref string, ok bool) {
		last = opRecord{op: op, ref: ref, ok: ok}
	}

	h := newTestHandler(st, audit)

	rec := httptest.NewRecorder()
	newTestMux(h).ServeHTTP(rec, newSet(t, "rsa", `{"enabled":false}`, adminPrincipal()))

	assertMethodError(t, rec, http.StatusNotFound, httpx.CodeNotFound)

	if last.ok {
		t.Errorf("ops_log ok = true, want false: изменение не применено")
	}
}

func TestHandlerSetStoreError(t *testing.T) {
	failure := errors.New("deadlock detected")
	st := newFakeStore(seedRows()...)
	st.setErr = failure

	var last opRecord

	audit := func(_ context.Context, _ *int64, op, ref string, ok bool) {
		last = opRecord{op: op, ref: ref, ok: ok}
	}

	h := newTestHandler(st, audit)

	rec := httptest.NewRecorder()
	newTestMux(h).ServeHTTP(rec, newSet(t, "rsa", `{"enabled":false}`, adminPrincipal()))

	assertMethodError(t, rec, http.StatusInternalServerError, httpx.CodeInternal)

	if strings.Contains(rec.Body.String(), "deadlock") {
		t.Errorf("ответ раскрывает текст ошибки базы: %s", rec.Body.String())
	}

	if last.ok {
		t.Errorf("ops_log ok = true, want false при ошибке базы")
	}
}

func TestHandlerSetWithoutPrincipalLogsAnonymous(t *testing.T) {
	var last opRecord

	audit := func(_ context.Context, userID *int64, _, _ string, _ bool) {
		last = opRecord{userID: userID, op: opMethodToggle, ref: "rsa", ok: true}
	}

	h := newTestHandler(newFakeStore(seedRows()...), audit)

	rec := httptest.NewRecorder()
	newTestMux(h).ServeHTTP(rec, newSet(t, "rsa", `{"enabled":false}`, nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}

	// nil, а не 0: ноль — это системный ключ, и аудит соврал бы о субъекте.
	if last.userID != nil {
		t.Errorf("ops_log userID = %d, want NULL для анонимного запроса", *last.userID)
	}
}

func TestHandlerSetWithoutAuditStillWorks(t *testing.T) {
	h := newTestHandler(newFakeStore(seedRows()...), nil)

	rec := httptest.NewRecorder()
	newTestMux(h).ServeHTTP(rec, newSet(t, "kuznyechik", `{"enabled":false}`, adminPrincipal()))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
}

func TestHandlerListRequiresNoPrincipal(t *testing.T) {
	h := NewHandler(NewService(newFakeStore(seedRows()...)), nil, nil)

	rec := httptest.NewRecorder()
	http.HandlerFunc(h.List).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, listPath, http.NoBody))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: список методов публичный", rec.Code)
	}
}

func assertMethodError(t *testing.T, rec *httptest.ResponseRecorder, wantStatus int, wantCode string) {
	t.Helper()

	if rec.Code != wantStatus {
		t.Fatalf("status = %d, want %d (body %q)", rec.Code, wantStatus, rec.Body.String())
	}

	var got httpx.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body %q: %v", rec.Body.String(), err)
	}

	if got.Error.Code != wantCode {
		t.Errorf("code = %q, want %q (message %q)", got.Error.Code, wantCode, got.Error.Message)
	}
}

// TestHandlerSetAcceptsFrontName: фронт присылает название метода, а не код.
func TestHandlerSetAcceptsFrontName(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		wantCode string
	}{
		{"название rsa", NameRSA, "rsa"},
		{"название кузнечика", NameKuznyechik, "kuznyechik"},
		{"код вместо названия", "kuznyechik", "kuznyechik"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := newFakeStore(seedRows()...)
			h := newTestHandler(st, nil)

			rec := httptest.NewRecorder()
			newTestMux(h).ServeHTTP(rec, newSet(t, tt.method, `{"enabled":false}`, adminPrincipal()))

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
			}

			if len(st.setCalls) != 1 || st.setCalls[0] != tt.wantCode {
				t.Errorf("setCalls = %+v, want %s", st.setCalls, tt.wantCode)
			}
		})
	}
}

func TestHandlerSetWithoutName(t *testing.T) {
	h := newTestHandler(newFakeStore(seedRows()...), nil)

	req := httptest.NewRequest(http.MethodPost, setPath, strings.NewReader(`{"enabled":false}`))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	newTestMux(h).ServeHTTP(rec, req)

	assertMethodError(t, rec, http.StatusBadRequest, httpx.CodeValidation)
}

func TestName(t *testing.T) {
	if Name(ports.MethodRSA) != NameRSA || Name(ports.MethodKuznyechik) != NameKuznyechik || Name("aes") != "aes" {
		t.Errorf("Name() mapping is wrong")
	}

	if _, ok := CodeByName("AES"); ok {
		t.Error("CodeByName(AES) ok = true, want false")
	}
}
