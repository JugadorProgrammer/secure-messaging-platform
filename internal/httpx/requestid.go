package httpx

import (
	"context"
	"crypto/rand"
	"net/http"
	"strings"
)

const (
	// HeaderRequestID is the request identifier header.
	HeaderRequestID = "X-Request-ID"

	maxRequestIDLen = 64
)

type requestIDKey struct{}

// RequestID accepts X-Request-ID from the client when it is safe and generates
// a new one otherwise. The identifier is stored in the context and returned in
// the response.
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

// RequestIDFrom returns the request identifier from the context or an empty string.
func RequestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)

	return id
}

// validRequestID accepts only short identifiers made of safe characters.
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
