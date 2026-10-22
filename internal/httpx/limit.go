package httpx

import "net/http"

// BodyLimit caps the request body size. An explicit Content-Length above the
// limit is rejected immediately with 413; otherwise the body is wrapped into
// http.MaxBytesReader and reading past the limit returns an error.
func BodyLimit(limit int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if limit > 0 {
				if r.ContentLength > limit {
					WriteErr(w, http.StatusRequestEntityTooLarge, CodePayloadTooLarge,
						"Тело запроса слишком велико")

					return
				}

				r.Body = http.MaxBytesReader(w, r.Body, limit)
			}

			next.ServeHTTP(w, r)
		})
	}
}
