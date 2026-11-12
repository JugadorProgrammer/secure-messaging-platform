package opslog

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/httpx"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/ports"
)

func newTestHandler(st ports.Store) *Handler {
	return NewHandler(st, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

// doList выполняет GET /api/ops-log с указанным путём и возвращает ответ.
func doList(t *testing.T, h *Handler, target string) *httptest.ResponseRecorder {
	t.Helper()

	rec := httptest.NewRecorder()
	h.List(rec, httptest.NewRequest(http.MethodGet, target, http.NoBody))

	return rec
}

func TestListUsesDefaultPage(t *testing.T) {
	st := &fakeStore{}

	rec := doList(t, newTestHandler(st), "/api/ops-log")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	if st.gotLimit != DefaultLimit || st.gotStart != 0 {
		t.Errorf("page = limit %d offset %d, want %d/0", st.gotLimit, st.gotStart, DefaultLimit)
	}

	ct := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type = %q, want json", ct)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", cc)
	}
}

func TestListPassesPageToStore(t *testing.T) {
	tests := []struct {
		target string
		limit  int
		start  int
	}{
		{"/api/ops-log?limit=5&offset=10", 5, 10},
		{"/api/ops-log?limit=100", 100, 0},
		{"/api/ops-log?limit=&offset=", DefaultLimit, 0},
		{"/api/ops-log?offset=40", DefaultLimit, 40},
	}

	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			st := &fakeStore{}

			rec := doList(t, newTestHandler(st), tt.target)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
			}
			if st.gotLimit != tt.limit || st.gotStart != tt.start {
				t.Errorf("page = limit %d offset %d, want %d/%d", st.gotLimit, st.gotStart, tt.limit, tt.start)
			}
		})
	}
}

func TestListRendersEntries(t *testing.T) {
	at := time.Date(2026, 1, 8, 14, 22, 3, 0, time.UTC)
	st := &fakeStore{entries: []ports.OpLogEntry{
		{ID: 2, At: at, UserID: int64Ptr(7), Op: OpAccessDenied, Ref: "GET /api/users", OK: false},
		{ID: 1, At: at, UserID: nil, Op: OpSignatureVerify, Ref: "sha256:0123456789abcdef", OK: true},
	}}

	rec := doList(t, newTestHandler(st), "/api/ops-log")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}

	var got []EntryDTO
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body %q: %v", rec.Body.String(), err)
	}
	if len(got) != 2 {
		t.Fatalf("len(entries) = %d, want 2", len(got))
	}

	if got[0].ID != 2 || got[0].Op != OpAccessDenied || got[0].Ref != "GET /api/users" || got[0].OK {
		t.Errorf("entry[0] = %+v, want the denied access record", got[0])
	}
	if got[0].UserID == nil || *got[0].UserID != 7 {
		t.Errorf("entry[0].UserID = %v, want 7", got[0].UserID)
	}
	if !got[0].At.Equal(at) {
		t.Errorf("entry[0].At = %v, want %v", got[0].At, at)
	}

	// Анонимная запись должна остаться null, а не превратиться в нулевой id.
	if !strings.Contains(rec.Body.String(), `"user_id":null`) {
		t.Errorf("body = %q, want user_id null for the anonymous record", rec.Body.String())
	}
	if got[1].UserID != nil {
		t.Errorf("entry[1].UserID = %d, want nil", *got[1].UserID)
	}
}

func TestListEmptyJournalIsAnArray(t *testing.T) {
	rec := doList(t, newTestHandler(&fakeStore{}), "/api/ops-log")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	if body := rec.Body.String(); body != "[]" {
		t.Errorf("body = %q, want []", body)
	}
}

func TestListRejectsBadPageParams(t *testing.T) {
	tests := []struct {
		name   string
		target string
	}{
		{"limit не число", "/api/ops-log?limit=abc"},
		{"limit дробный", "/api/ops-log?limit=1.5"},
		{"limit нулевой", "/api/ops-log?limit=0"},
		{"limit отрицательный", "/api/ops-log?limit=-1"},
		{"limit больше ста", "/api/ops-log?limit=101"},
		{"limit с пробелом", "/api/ops-log?limit=%205"},
		{"offset не число", "/api/ops-log?offset=xyz"},
		{"offset отрицательный", "/api/ops-log?offset=-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := &fakeStore{entries: []ports.OpLogEntry{{ID: 1}}}

			rec := doList(t, newTestHandler(st), tt.target)

			assertErrCode(t, rec, http.StatusBadRequest, httpx.CodeValidation)
			if st.gotLimit != 0 {
				t.Errorf("store was called with limit %d, want no call at all", st.gotLimit)
			}
		})
	}
}

func TestListStoreRejectsPageAsInvalidArgument(t *testing.T) {
	st := &fakeStore{listErr: fmt.Errorf("store: list ops: limit out of range: %w", ports.ErrInvalidArgument)}

	rec := doList(t, newTestHandler(st), "/api/ops-log?limit=50")

	assertErrCode(t, rec, http.StatusBadRequest, httpx.CodeValidation)
	if strings.Contains(rec.Body.String(), "store") {
		t.Errorf("body = %q, want the store error text hidden", rec.Body.String())
	}
}

func TestListStoreFailureIsInternalError(t *testing.T) {
	st := &fakeStore{listErr: errors.New("dial tcp 10.0.0.5:5432: connect: refused")}

	rec := doList(t, newTestHandler(st), "/api/ops-log")

	assertErrCode(t, rec, http.StatusInternalServerError, httpx.CodeInternal)
	if strings.Contains(rec.Body.String(), "5432") {
		t.Errorf("body = %q, want no details of the storage failure", rec.Body.String())
	}
}

func TestNewHandlerWithoutLoggerDoesNotPanic(t *testing.T) {
	h := NewHandler(&fakeStore{}, nil)

	rec := doList(t, h, "/api/ops-log")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func assertErrCode(t *testing.T, rec *httptest.ResponseRecorder, wantStatus int, wantCode string) {
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
