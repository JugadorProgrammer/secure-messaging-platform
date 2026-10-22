package httpx

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	rec := httptest.NewRecorder()

	WriteJSON(rec, http.StatusCreated, map[string]string{"status": "ok"})

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}

	h := rec.Header()
	if ct := h.Get("Content-Type"); ct != contentTypeJSON {
		t.Errorf("Content-Type = %q, want %q", ct, contentTypeJSON)
	}
	if cc := h.Get("Cache-Control"); cc != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", cc)
	}
	if xcto := h.Get("X-Content-Type-Options"); xcto != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want nosniff", xcto)
	}

	var got map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if got["status"] != "ok" {
		t.Errorf("body = %v", got)
	}
}

func TestWriteJSONNilBody(t *testing.T) {
	rec := httptest.NewRecorder()

	WriteJSON(rec, http.StatusNoContent, nil)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty", rec.Body.String())
	}
}

func TestWriteErr(t *testing.T) {
	tests := []struct {
		name   string
		status int
		code   string
	}{
		{name: "validation", status: http.StatusBadRequest, code: CodeValidation},
		{name: "unauthorized", status: http.StatusUnauthorized, code: CodeUnauthorized},
		{name: "not found", status: http.StatusNotFound, code: CodeNotFound},
		{name: "method not allowed", status: http.StatusMethodNotAllowed, code: CodeMethodNotAllowed},
		{name: "payload too large", status: http.StatusRequestEntityTooLarge, code: CodePayloadTooLarge},
		{name: "unsupported media type", status: http.StatusUnsupportedMediaType, code: CodeUnsupportedMediaType},
		{name: "internal", status: http.StatusInternalServerError, code: CodeInternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()

			WriteErr(rec, tt.status, tt.code, "message")

			if rec.Code != tt.status {
				t.Fatalf("status = %d, want %d", rec.Code, tt.status)
			}
			if ct := rec.Header().Get("Content-Type"); ct != contentTypeJSON {
				t.Errorf("Content-Type = %q, want %q", ct, contentTypeJSON)
			}

			var got ErrorResponse
			if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if got.Error.Code != tt.code {
				t.Errorf("code = %q, want %q", got.Error.Code, tt.code)
			}
			if got.Error.Message != "message" {
				t.Errorf("message = %q", got.Error.Message)
			}
		})
	}
}

func TestWriteJSONMarshalError(t *testing.T) {
	rec := httptest.NewRecorder()

	// A function cannot be encoded, so WriteJSON must report 500 instead of 200.
	WriteJSON(rec, http.StatusOK, func() {})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}

	var got ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if got.Error.Code != CodeInternal {
		t.Errorf("code = %q, want %q", got.Error.Code, CodeInternal)
	}
}

func TestWriteErrAfterHeadersSent(t *testing.T) {
	rec := httptest.NewRecorder()
	sw := &statusWriter{ResponseWriter: rec, status: http.StatusOK}

	sw.WriteHeader(http.StatusAccepted)
	WriteErr(sw, http.StatusInternalServerError, CodeInternal, "boom")

	if sw.status != http.StatusAccepted {
		t.Errorf("status = %d, want %d", sw.status, http.StatusAccepted)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty", rec.Body.String())
	}
}
