package httpx

import (
	"net/http"
	"strings"
)

const (
	headerOrigin        = "Origin"
	headerVary          = "Vary"
	headerAllowOrigin   = "Access-Control-Allow-Origin"
	headerAllowMethods  = "Access-Control-Allow-Methods"
	headerAllowHeaders  = "Access-Control-Allow-Headers"
	headerMaxAge        = "Access-Control-Max-Age"
	headerRequestMethod = "Access-Control-Request-Method"

	corsAllowedMethods = "GET, POST, PATCH, DELETE, OPTIONS"
	corsAllowedHeaders = "Authorization, Content-Type, X-Request-ID"
	corsMaxAgeSeconds  = "600"
	corsWildcard       = "*"
)

// CORS разрешает запросы с перечисленных origin. Пустой список отключает CORS.
func CORS(origins []string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(origins))
	wildcard := false

	for _, o := range origins {
		o = strings.TrimSpace(o)
		if o == "" {
			continue
		}
		if o == corsWildcard {
			wildcard = true

			continue
		}

		allowed[o] = struct{}{}
	}

	if !wildcard && len(allowed) == 0 {
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get(headerOrigin)
			_, originAllowed := allowed[origin]

			if origin != "" && (wildcard || originAllowed) {
				h := w.Header()
				h.Add(headerVary, headerOrigin)

				if wildcard {
					h.Set(headerAllowOrigin, corsWildcard)
				} else {
					h.Set(headerAllowOrigin, origin)
				}

				h.Set(headerAllowMethods, corsAllowedMethods)
				h.Set(headerAllowHeaders, corsAllowedHeaders)
				h.Set(headerMaxAge, corsMaxAgeSeconds)
			}

			if r.Method == http.MethodOptions && r.Header.Get(headerRequestMethod) != "" {
				w.WriteHeader(http.StatusNoContent)

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
