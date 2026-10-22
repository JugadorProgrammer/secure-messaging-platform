package httpx

import (
	"errors"
	"log/slog"
	"net/http"
	"runtime/debug"
)

// Recover превращает панику в ответ 500 в едином формате.
//
// http.ErrAbortHandler пробрасывается дальше: httputil.ReverseProxy использует
// его для обрыва соединения, и его перехват сломал бы прокси.
func Recover(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			//nolint:contextcheck // отложенная функция выполняется внутри обработчика, контекст берётся из запроса
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

				WriteErr(w, http.StatusInternalServerError, CodeInternal, msgInternalError)
			}()

			next.ServeHTTP(w, r)
		})
	}
}
