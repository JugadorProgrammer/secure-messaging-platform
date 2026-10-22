package httpx

import (
	"log/slog"
	"net/http"
	"time"
)

// Logging пишет одну запись на запрос: метод, путь, статус, размер ответа,
// длительность и request_id. Тело запроса и заголовок Authorization
// не логируются никогда.
func Logging(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := newStatusWriter(w)

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

// statusWriter запоминает статус и размер ответа. Исходный ResponseWriter
// остаётся доступен через встроенный responseWriter.
type statusWriter struct {
	responseWriter

	status int
	bytes  int64
	wrote  bool
}

// newStatusWriter оборачивает w и считает статус равным 200, пока обработчик
// не скажет иначе.
func newStatusWriter(w http.ResponseWriter) *statusWriter {
	return &statusWriter{
		responseWriter: responseWriter{ResponseWriter: w},
		status:         http.StatusOK,
	}
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

// WroteHeader сообщает, отправлены ли заголовки ответа.
func (w *statusWriter) WroteHeader() bool {
	return w.wrote
}

// Flush передаёт сброс буфера, если исходный ResponseWriter это умеет.
func (w *statusWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
