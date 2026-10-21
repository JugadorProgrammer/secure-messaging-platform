package httpx

import "net/http"

// BodyLimit ограничивает размер тела запроса. Явно заданный Content-Length
// больше лимита отклоняется сразу с 413; иначе тело оборачивается
// http.MaxBytesReader и чтение сверх лимита вернёт ошибку.
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
