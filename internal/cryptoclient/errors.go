package cryptoclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/ports"
)

// maxErrorMsgLen ограничивает текст ошибки модуля, который попадает в лог:
// модуль может вернуть длинный текст, и строка лога не должна разрастаться.
const maxErrorMsgLen = 200

// BadInputError — Крипто-модуль отверг входные данные (ответ 4xx).
//
// Code и Msg взяты из тела ответа модуля и предназначены только для лога.
// Клиенту сервиса уходит собственный текст из httpx.WriteErr.
type BadInputError struct {
	Op     string
	Status int
	Code   string
	Msg    string
}

func (e *BadInputError) Error() string {
	return fmt.Sprintf("cryptoclient: %s: bad input: status %d, code %q: %s", e.Op, e.Status, e.Code, e.Msg)
}

// Unwrap позволяет проверять ошибку через errors.Is(err, ports.ErrCryptoBadInput).
func (e *BadInputError) Unwrap() error { return ports.ErrCryptoBadInput }

// UnavailableError — модуль недоступен, ответил 5xx или ответ не соответствует
// контракту: битый JSON, неверный Content-Type, превышен размер.
type UnavailableError struct {
	Op  string
	Err error
}

func (e *UnavailableError) Error() string {
	return fmt.Sprintf("cryptoclient: %s: unavailable: %v", e.Op, e.Err)
}

// Unwrap отдаёт и sentinel, и исходную причину: errors.Is находит
// ports.ErrCryptoUnavailable, а errors.As — например, *net.OpError.
func (e *UnavailableError) Unwrap() []error { return []error{ports.ErrCryptoUnavailable, e.Err} }

// errorBody покрывает два формата ошибки модуля: {"error":{"code","message"}}
// и плоский {"error":"...","code":"..."}. Контракт формат не фиксирует,
// поэтому разбор нестрогий: не разобрали — остаются только статус и Op.
type errorBody struct {
	Error   json.RawMessage `json:"error"`
	Code    string          `json:"code"`
	Message string          `json:"message"`
}

type nestedError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func newBadInputError(op string, status int, raw []byte) *BadInputError {
	e := &BadInputError{Op: op, Status: status}

	var body errorBody
	if json.Unmarshal(raw, &body) != nil {
		return e
	}

	e.Code, e.Msg = body.Code, body.Message

	switch trimmed := bytes.TrimSpace(body.Error); {
	case len(trimmed) == 0:
	case trimmed[0] == '{':
		var nested nestedError
		if json.Unmarshal(trimmed, &nested) == nil {
			e.Code, e.Msg = nested.Code, nested.Message
		}
	case trimmed[0] == '"':
		var msg string
		if json.Unmarshal(trimmed, &msg) == nil {
			e.Msg = msg
		}
	}

	e.Code = truncate(e.Code)
	e.Msg = truncate(e.Msg)

	return e
}

// truncate обрезает строку до maxErrorMsgLen байт, не разрывая символ UTF-8.
func truncate(s string) string {
	if len(s) <= maxErrorMsgLen {
		return s
	}

	cut := maxErrorMsgLen
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}

	return strings.ToValidUTF8(s[:cut], "") + "…"
}
