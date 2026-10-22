package httpx

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRequestIDGeneratesID(t *testing.T) {
	var got string

	h := RequestID(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = RequestIDFrom(r.Context())
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", http.NoBody))

	if got == "" {
		t.Fatal("RequestIDFrom(ctx) = empty")
	}
	if !validRequestID(got) {
		t.Errorf("generated id %q is not valid", got)
	}
	if header := rec.Header().Get(HeaderRequestID); header != got {
		t.Errorf("response header = %q, want %q", header, got)
	}
}

func TestRequestIDKeepsValidIncomingID(t *testing.T) {
	const id = "abc-123_XYZ.9"

	var got string

	h := RequestID(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = RequestIDFrom(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	req.Header.Set(HeaderRequestID, "  "+id+"  ")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if got != id {
		t.Errorf("RequestIDFrom(ctx) = %q, want %q", got, id)
	}
	if header := rec.Header().Get(HeaderRequestID); header != id {
		t.Errorf("response header = %q, want %q", header, id)
	}
}

func TestRequestIDRejectsUnsafeIncomingID(t *testing.T) {
	tests := []struct {
		name string
		id   string
	}{
		{name: "too long", id: strings.Repeat("a", maxRequestIDLen+1)},
		{name: "newline", id: "abc\ndef"},
		{name: "space inside", id: "abc def"},
		{name: "angle brackets", id: "<script>"},
		{name: "unicode", id: "запрос"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got string

			h := RequestID(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				got = RequestIDFrom(r.Context())
			}))

			req := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
			req.Header.Set(HeaderRequestID, tt.id)
			h.ServeHTTP(httptest.NewRecorder(), req)

			if got == tt.id {
				t.Fatalf("unsafe id %q was accepted", tt.id)
			}
			if got == "" {
				t.Fatal("id was not generated")
			}
		})
	}
}

func TestRequestIDFromEmptyContext(t *testing.T) {
	if got := RequestIDFrom(t.Context()); got != "" {
		t.Errorf("RequestIDFrom(empty ctx) = %q, want empty", got)
	}
}
