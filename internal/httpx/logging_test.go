package httpx

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLoggingWritesRequestRecord(t *testing.T) {
	var buf bytes.Buffer

	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	h := RequestID(Logging(log)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		WriteJSON(w, http.StatusTeapot, map[string]string{"status": "teapot"})
	})))

	req := httptest.NewRequest(http.MethodGet, "/api/health?x=1", http.NoBody)
	req.Header.Set("Authorization", "Bearer super-secret-token")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	out := buf.String()

	for _, want := range []string{
		"method=GET",
		"path=/api/health",
		"status=418",
		"bytes=",
		"duration=",
		"request_id=",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("log record missing %q: %s", want, out)
		}
	}
	if strings.Contains(out, "super-secret-token") {
		t.Errorf("log record leaks Authorization: %s", out)
	}
	if strings.Contains(out, "x=1") {
		t.Errorf("log record contains query string: %s", out)
	}
}

func TestLoggingLevels(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		wantLevel string
	}{
		{name: "ok", status: http.StatusOK, wantLevel: "level=INFO"},
		{name: "client error", status: http.StatusNotFound, wantLevel: "level=WARN"},
		{name: "server error", status: http.StatusInternalServerError, wantLevel: "level=ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
			h := Logging(log)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
			}))

			h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", http.NoBody))

			if !strings.Contains(buf.String(), tt.wantLevel) {
				t.Errorf("log record = %s, want level %s", buf.String(), tt.wantLevel)
			}
		})
	}
}

func TestStatusWriterDefaults(t *testing.T) {
	rec := httptest.NewRecorder()
	sw := newStatusWriter(rec)

	if _, err := sw.Write([]byte("abc")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if sw.status != http.StatusOK {
		t.Errorf("status = %d, want %d", sw.status, http.StatusOK)
	}
	if sw.bytes != 3 {
		t.Errorf("bytes = %d, want 3", sw.bytes)
	}
	if !sw.WroteHeader() {
		t.Error("WroteHeader() = false, want true")
	}
	if got := sw.Unwrap(); got != rec {
		t.Error("Unwrap() did not return the original ResponseWriter")
	}
}

func TestStatusWriterIgnoresSecondHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	sw := newStatusWriter(rec)

	sw.WriteHeader(http.StatusCreated)
	sw.WriteHeader(http.StatusInternalServerError)

	if sw.status != http.StatusCreated {
		t.Errorf("status = %d, want %d", sw.status, http.StatusCreated)
	}
	if rec.Code != http.StatusCreated {
		t.Errorf("recorder status = %d, want %d", rec.Code, http.StatusCreated)
	}
}

func TestStatusWriterFlush(t *testing.T) {
	rec := httptest.NewRecorder()
	sw := newStatusWriter(rec)

	sw.Flush()

	if !rec.Flushed {
		t.Error("Flush() was not passed to the underlying writer")
	}
}
