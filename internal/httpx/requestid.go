package httpx

import (
	"context"
	"crypto/rand"
	"net/http"
	"strings"
)

const (
	// HeaderRequestID — заголовок с идентификатором запроса.
	HeaderRequestID = "X-Request-ID"

	maxRequestIDLen = 64
)

type requestIDKey struct{}

// RequestID принимает X-Request-ID от клиента, когда он безопасен, и иначе
// генерирует новый. Идентификатор кладётся в контекст и возвращается в ответе.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(r.Header.Get(HeaderRequestID))
		if !validRequestID(id) {
			id = rand.Text()
		}

		w.Header().Set(HeaderRequestID, id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey{}, id)))
	})
}

// RequestIDFrom возвращает идентификатор запроса из контекста или пустую строку.
func RequestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)

	return id
}

// validRequestID принимает только короткие идентификаторы из безопасных символов.
func validRequestID(id string) bool {
	if id == "" || len(id) > maxRequestIDLen {
		return false
	}

	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-', r == '_', r == '.':
		default:
			return false
		}
	}

	return true
}
