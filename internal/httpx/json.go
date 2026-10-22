package httpx

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

const contentTypeJSON = "application/json; charset=utf-8"

// ErrorResponse — тело ошибки в едином формате из contracts.md, раздел 4.
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

// ErrorBody несёт машиночитаемый код и сообщение, которое видит клиент.
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// WriteJSON пишет v как JSON с указанным кодом состояния.
//
// Значение кодируется до отправки статуса: если кодирование не удалось,
// клиент получает 500, а не 200 с пустым телом.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	if v == nil {
		setJSONHeaders(w)
		w.WriteHeader(status)

		return
	}

	b, err := json.Marshal(v)
	if err != nil {
		slog.Error("encode json response", "err", err)
		WriteErr(w, http.StatusInternalServerError, CodeInternal, msgInternalError)

		return
	}

	setJSONHeaders(w)
	w.WriteHeader(status)
	_, _ = w.Write(b)
}

// WriteErr пишет ошибку в едином формате. Если заголовки уже отправлены,
// тело пропускается: статус изменить нельзя.
//
// Тело кодируется здесь, а не через WriteJSON, чтобы ошибка внутри WriteJSON
// не привела обратно в WriteErr.
func WriteErr(w http.ResponseWriter, status int, code, message string) {
	if wroteHeader(w) {
		return
	}

	b, err := json.Marshal(ErrorResponse{Error: ErrorBody{Code: code, Message: message}})
	if err != nil {
		b = []byte(`{"error":{"code":"` + CodeInternal + `","message":"` + msgInternalError + `"}}`)
	}

	setJSONHeaders(w)
	w.WriteHeader(status)
	_, _ = w.Write(b)
}

func setJSONHeaders(w http.ResponseWriter) {
	h := w.Header()
	h.Set("Content-Type", contentTypeJSON)
	h.Set("Cache-Control", "no-store")
	h.Set("X-Content-Type-Options", "nosniff")
}

// headerWriter реализуется обёрткой Logging и сообщает, отправлены ли
// заголовки ответа.
type headerWriter interface {
	WroteHeader() bool
}

func wroteHeader(w http.ResponseWriter) bool {
	hw, ok := w.(headerWriter)

	return ok && hw.WroteHeader()
}
