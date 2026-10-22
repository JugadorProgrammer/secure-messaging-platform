package httpx

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBodyLimit(t *testing.T) {
	tests := []struct {
		name       string
		limit      int64
		body       string
		wantStatus int
		wantCalled bool
		wantRead   int
	}{
		{
			name:       "Content-Length больше лимита",
			limit:      16,
			body:       strings.Repeat("a", 100),
			wantStatus: http.StatusRequestEntityTooLarge,
		},
		{
			name:       "тело в пределах лимита",
			limit:      16,
			body:       "12345",
			wantStatus: http.StatusOK,
			wantCalled: true,
			wantRead:   5,
		},
		{
			name:       "нулевой лимит отключает проверку",
			limit:      0,
			body:       strings.Repeat("a", 100),
			wantStatus: http.StatusOK,
			wantCalled: true,
			wantRead:   100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				called bool
				read   int
			)

			h := BodyLimit(tt.limit)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true

				b, err := io.ReadAll(r.Body)
				if err != nil {
					t.Fatalf("read body: %v", err)
				}

				read = len(b)
				WriteJSON(w, http.StatusOK, nil)
			}))

			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if ct := rec.Header().Get("Content-Type"); ct != contentTypeJSON {
				t.Errorf("Content-Type = %q, want %q", ct, contentTypeJSON)
			}
			if called != tt.wantCalled {
				t.Errorf("handler called = %v, want %v", called, tt.wantCalled)
			}
			if read != tt.wantRead {
				t.Errorf("read = %d bytes, want %d", read, tt.wantRead)
			}
			if tt.wantStatus == http.StatusRequestEntityTooLarge {
				var got ErrorResponse
				if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
					t.Fatalf("decode body: %v", err)
				}
				if got.Error.Code != CodePayloadTooLarge {
					t.Errorf("code = %q, want %q", got.Error.Code, CodePayloadTooLarge)
				}
			}
		})
	}
}
