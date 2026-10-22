package httpx

import "net/http"

// responseWriter хранит исходный ResponseWriter доступным через Unwrap.
//
// Обе обёртки встраивают его, поэтому метод объявлен один раз, а не
// копируется в каждую обёртку.
type responseWriter struct {
	http.ResponseWriter
}

// Unwrap возвращает исходный ResponseWriter для http.NewResponseController.
func (w responseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
