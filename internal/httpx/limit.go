package httpx

import "net/http"

// BodyLimit ограничивает размер тела запроса. Явный Content-Length выше лимита
// отклоняется сразу с 413; иначе тело оборачивается в http.MaxBytesReader,
// и чтение сверх лимита возвращает ошибку.
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
