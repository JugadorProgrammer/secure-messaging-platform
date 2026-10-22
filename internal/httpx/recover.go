package httpx

import (
	"errors"
	"log/slog"
	"net/http"
	"runtime/debug"
)

// Recover turns a panic into a 500 response in the uniform format.
//
// http.ErrAbortHandler is re-panicked: httputil.ReverseProxy uses it to abort
// a connection, and swallowing it would break the proxy.
func Recover(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			//nolint:contextcheck // the deferred func runs inside the handler, the context comes from the request
			defer func() {
				rec := recover()
				if rec == nil {
					return
				}

				if err, ok := rec.(error); ok && errors.Is(err, http.ErrAbortHandler) {
					panic(rec)
				}

				log.ErrorContext(r.Context(), "panic in handler",
					slog.Any("panic", rec),
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.String("request_id", RequestIDFrom(r.Context())),
					slog.String("stack", string(debug.Stack())),
				)

				WriteErr(w, http.StatusInternalServerError, CodeInternal, "Внутренняя ошибка сервера")
			}()

			next.ServeHTTP(w, r)
		})
	}
}
