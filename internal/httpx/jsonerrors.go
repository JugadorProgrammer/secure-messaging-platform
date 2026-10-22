package httpx

import (
	"encoding/json"
	"net/http"
	"strings"
)

// UniformErrors replaces the plain-text 404 and 405 responses produced by
// http.ServeMux with JSON in the uniform format from contracts.md, section 4.
//
// Responses whose Content-Type is already set by a handler (including errors
// written through WriteErr) are left untouched.
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
		// The response is already sent, so there is nothing left to report.
		_ = err
	}
}

func (w *uniformWriter) Write(p []byte) (int, error) {
	if w.intercepted {
		return len(p), nil
	}

	return w.ResponseWriter.Write(p)
}

// Unwrap returns the original ResponseWriter for http.NewResponseController.
func (w *uniformWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// shouldReplace reports whether a plain-text response must be replaced with JSON.
func (w *uniformWriter) shouldReplace(code int) bool {
	if code != http.StatusNotFound && code != http.StatusMethodNotAllowed {
		return false
	}

	return strings.HasPrefix(w.Header().Get("Content-Type"), "text/plain")
}
