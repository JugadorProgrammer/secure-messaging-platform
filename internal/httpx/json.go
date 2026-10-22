package httpx

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

const contentTypeJSON = "application/json; charset=utf-8"

// ErrorResponse is the error body in the uniform format from contracts.md, section 4.
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

// ErrorBody carries the machine-readable code and the message shown to the client.
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// WriteJSON writes v as JSON with the given status code.
//
// The value is encoded before the status is sent: if encoding fails the client
// gets a 500 instead of a 200 with an empty body.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	if v == nil {
		setJSONHeaders(w)
		w.WriteHeader(status)

		return
	}

	b, err := json.Marshal(v)
	if err != nil {
		slog.Error("encode json response", "err", err)
		WriteErr(w, http.StatusInternalServerError, CodeInternal, "Внутренняя ошибка сервера")

		return
	}

	setJSONHeaders(w)
	w.WriteHeader(status)
	_, _ = w.Write(b)
}

// WriteErr writes an error in the uniform format. When the headers have already
// been sent the body is skipped: the status cannot be changed anymore.
//
// The body is encoded here rather than through WriteJSON so that a failure in
// WriteJSON cannot lead back into WriteErr.
func WriteErr(w http.ResponseWriter, status int, code, message string) {
	if wroteHeader(w) {
		return
	}

	b, err := json.Marshal(ErrorResponse{Error: ErrorBody{Code: code, Message: message}})
	if err != nil {
		b = []byte(`{"error":{"code":"internal","message":"Внутренняя ошибка сервера"}}`)
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

// headerWriter is implemented by the Logging wrapper and reports whether the
// response headers have already been sent.
type headerWriter interface {
	WroteHeader() bool
}

func wroteHeader(w http.ResponseWriter) bool {
	hw, ok := w.(headerWriter)

	return ok && hw.WroteHeader()
}
