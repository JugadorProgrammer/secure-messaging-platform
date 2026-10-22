package httpx

import (
	"encoding/json"
	"net/http"
	"strings"
)

// UniformErrors заменяет текстовые ответы 404 и 405, которые отдаёт
// http.ServeMux, на JSON в едином формате из contracts.md, раздел 4.
//
// Ответы, у которых Content-Type уже выставлен обработчиком (включая ошибки,
// записанные через WriteErr), остаются без изменений.
func UniformErrors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(&uniformWriter{responseWriter: responseWriter{ResponseWriter: w}}, r)
	})
}

type uniformWriter struct {
	responseWriter

	intercepted bool
}

func (w *uniformWriter) WriteHeader(code int) {
	if !w.shouldReplace(code) {
		w.ResponseWriter.WriteHeader(code)

		return
	}

	w.intercepted = true
	setJSONHeaders(w.ResponseWriter)
	w.ResponseWriter.WriteHeader(code)

	body := ErrorResponse{Error: ErrorBody{Code: CodeNotFound, Message: "Ресурс не найден"}}
	if code == http.StatusMethodNotAllowed {
		body.Error = ErrorBody{Code: CodeMethodNotAllowed, Message: "Метод не поддерживается"}
	}

	if err := json.NewEncoder(w.ResponseWriter).Encode(body); err != nil {
		// Ответ уже отправлен, сообщать некуда.
		_ = err
	}
}

func (w *uniformWriter) Write(p []byte) (int, error) {
	if w.intercepted {
		return len(p), nil
	}

	return w.ResponseWriter.Write(p)
}

// shouldReplace сообщает, нужно ли заменить текстовый ответ на JSON.
func (w *uniformWriter) shouldReplace(code int) bool {
	if code != http.StatusNotFound && code != http.StatusMethodNotAllowed {
		return false
	}

	return strings.HasPrefix(w.Header().Get("Content-Type"), "text/plain")
}
