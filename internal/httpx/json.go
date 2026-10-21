package httpx

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

const contentTypeJSON = "application/json; charset=utf-8"

// ErrorResponse — тело ошибки в едином формате contracts.md, раздел 4.
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

// ErrorBody — машиночитаемый код и текст ошибки.
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// WriteJSON отправляет v в формате JSON с указанным статусом.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	setJSONHeaders(w)
	w.WriteHeader(status)

	if v == nil {
		return
	}

	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("encode json response", "err", err)
	}
}

// WriteErr отправляет ошибку в едином формате. Если заголовки уже отправлены,
// тело не пишется: изменить статус всё равно нельзя.
func WriteErr(w http.ResponseWriter, status int, code, message string) {
	if wroteHeader(w) {
		return
	}

	WriteJSON(w, status, ErrorResponse{Error: ErrorBody{Code: code, Message: message}})
}

func setJSONHeaders(w http.ResponseWriter) {
	h := w.Header()
	h.Set("Content-Type", contentTypeJSON)
	h.Set("Cache-Control", "no-store")
	h.Set("X-Content-Type-Options", "nosniff")
}

// headerWriter реализуется обёрткой Logging и позволяет понять, были ли
// заголовки уже отправлены.
type headerWriter interface {
	WroteHeader() bool
}

func wroteHeader(w http.ResponseWriter) bool {
	hw, ok := w.(headerWriter)

	return ok && hw.WroteHeader()
}
