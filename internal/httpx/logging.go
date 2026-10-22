package httpx

import (
	"log/slog"
	"net/http"
	"time"
)

// Logging writes one record per request: method, path, status, response size,
// duration and request_id. The request body and the Authorization header are
// never logged.
func Logging(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(sw, r)

			level := slog.LevelInfo
			switch {
			case sw.status >= http.StatusInternalServerError:
				level = slog.LevelError
			case sw.status >= http.StatusBadRequest:
				level = slog.LevelWarn
			}

			log.LogAttrs(r.Context(), level, "http request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", sw.status),
				slog.Int64("bytes", sw.bytes),
				slog.Duration("duration", time.Since(start)),
				slog.String("request_id", RequestIDFrom(r.Context())),
			)
		})
	}
}

// statusWriter records the status and the response size while keeping access to
// the original ResponseWriter through Unwrap. Flush is needed for streaming and
// proxying.
type statusWriter struct {
	http.ResponseWriter

	status int
	bytes  int64
	wrote  bool
}

func (w *statusWriter) WriteHeader(code int) {
	if w.wrote {
		return
	}

	w.status = code
	w.wrote = true
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(p []byte) (int, error) {
	if !w.wrote {
		w.WriteHeader(http.StatusOK)
	}

	n, err := w.ResponseWriter.Write(p)
	w.bytes += int64(n)

	return n, err
}

// WroteHeader reports whether the response headers have been sent.
func (w *statusWriter) WroteHeader() bool {
	return w.wrote
}

// Unwrap returns the original ResponseWriter for http.NewResponseController.
func (w *statusWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// Flush forwards the buffer flush when the underlying ResponseWriter supports it.
func (w *statusWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
