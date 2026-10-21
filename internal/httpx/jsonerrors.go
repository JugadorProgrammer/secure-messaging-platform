package httpx

import (
	"encoding/json"
	"net/http"
	"strings"
)

// UniformErrors заменяет текстовые ответы 404 и 405, которые отдаёт
// http.ServeMux, на JSON в едином формате contracts.md, раздел 4.
//
// Ответы, у которых Content-Type уже задан обработчиком (в том числе ошибки
// из WriteErr), не трогаются.
func UniformErrors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(&uniformWriter{ResponseWriter: w}, r)
	})
}

type uniformWriter struct {
	http.ResponseWriter

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
		// Ответ уже отправлен, остаётся только не мешать обработке запроса.
		_ = err
	}
}

func (w *uniformWriter) Write(p []byte) (int, error) {
	if w.intercepted {
		return len(p), nil
	}

	return w.ResponseWriter.Write(p)
}

// Unwrap возвращает исходный ResponseWriter для http.NewResponseController.
func (w *uniformWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// shouldReplace сообщает, нужно ли заменить текстовый ответ на JSON.
func (w *uniformWriter) shouldReplace(code int) bool {
	if code != http.StatusNotFound && code != http.StatusMethodNotAllowed {
		return false
	}

	return strings.HasPrefix(w.Header().Get("Content-Type"), "text/plain")
}
