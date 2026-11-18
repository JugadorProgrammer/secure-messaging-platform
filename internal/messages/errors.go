package messages

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net/http"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/auth"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/httpx"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/ports"
)

// Тексты ошибок видит клиент (contracts.md, раздел 4).
const (
	msgBodyNotJSON        = "Ожидается тело в формате application/json"
	msgBodyTooLarge       = "Тело запроса слишком велико"
	msgBodyInvalid        = "Некорректное тело запроса"
	msgTextEmpty          = "Текст сообщения не может быть пустым"
	msgTextTooLongFmt     = "Текст сообщения превышает %d КиБ"
	msgUnknownMethod      = "Неизвестный метод шифрования"
	msgMethodDisabled     = "Метод шифрования выключен администратором"
	msgKeyGenerating      = "Ключ шифрования генерируется, повторите позже"
	msgEncryptBadInput    = "Текст не удалось зашифровать выбранным методом"
	msgCryptoUnavailable  = "Крипто-модуль недоступен, повторите позже"
	msgCiphertextInvalid  = "Шифртекст должен быть непустой строкой в hex"
	msgPrivateKeyInvalid  = "Закрытый ключ должен быть в формате PEM"
	msgKuzKeyInvalid      = "Ключ Кузнечика должен быть 64 hex-символа"
	msgIVInvalid          = "IV должен быть 32 hex-символа"
	msgIVNotFound         = "Не найден IV: шифртекст не принадлежит сохранённому сообщению"
	msgDecryptFailed      = "Не удалось расшифровать: проверьте ключ и шифртекст"
	msgMessageIDInvalid   = "Идентификатор сообщения должен быть положительным числом"
	msgMessageNotFound    = "Сообщение не найдено"
	msgSystemKeyNotReady  = "Системный ключ подписи недоступен"
	msgInternal           = "Внутренняя ошибка сервера"
	msgNoPrincipalLogText = "messages: no principal in context"
)

// apiError — ошибка, у которой уже выбран ответ клиенту.
type apiError struct {
	status  int
	code    string
	message string
	// attempted — операция начиналась: неудача пишется в журнал с ok = false.
	attempted bool
	cause     error
}

func (e *apiError) Error() string {
	if e.cause == nil {
		return e.code
	}

	return fmt.Sprintf("%s: %v", e.code, e.cause)
}

func (e *apiError) Unwrap() error { return e.cause }

func validationErr(message string) *apiError {
	return &apiError{status: http.StatusBadRequest, code: httpx.CodeValidation, message: message}
}

func badKeyErr(message string) *apiError {
	return &apiError{status: http.StatusBadRequest, code: httpx.CodeBadKey, message: message}
}

func badCiphertextErr(message string) *apiError {
	return &apiError{status: http.StatusBadRequest, code: httpx.CodeBadCiphertext, message: message}
}

func notFoundErr() *apiError {
	return &apiError{status: http.StatusNotFound, code: httpx.CodeNotFound, message: msgMessageNotFound}
}

func internalErr(err error) *apiError {
	return &apiError{
		status: http.StatusInternalServerError, code: httpx.CodeInternal, message: msgInternal,
		attempted: true, cause: err,
	}
}

func cryptoUnavailableErr(err error) *apiError {
	return &apiError{
		status: http.StatusServiceUnavailable, code: httpx.CodeCryptoUnavailable, message: msgCryptoUnavailable,
		attempted: true, cause: err,
	}
}

// cryptoErr: недоступность модуля — 503; отказ модуля принять данные — 400
// с кодом и текстом, которые выбрал вызывающий.
func cryptoErr(err error, badInput *apiError) *apiError {
	switch {
	case errors.Is(err, ports.ErrCryptoUnavailable):
		return cryptoUnavailableErr(err)
	case errors.Is(err, ports.ErrCryptoBadInput):
		badInput.attempted = true
		badInput.cause = err

		return badInput
	default:
		return internalErr(err)
	}
}

// unsealErr: повреждённый контейнер или чужой MASTER_KEY — внутренняя
// проблема, 500 (contracts.md, раздел 6); недоступный модуль — 503.
func unsealErr(err error) *apiError {
	if errors.Is(err, ports.ErrCryptoUnavailable) {
		return cryptoUnavailableErr(err)
	}

	return internalErr(fmt.Errorf("unseal key: %w", err))
}

// principal достаёт пользователя. Маршруты регистрируются только под
// Authn, поэтому отсутствие пользователя — ошибка сборки маршрутов.
func (h *Handler) principal(w http.ResponseWriter, r *http.Request) (auth.Principal, bool) {
	p, ok := auth.PrincipalFrom(r.Context())
	if !ok {
		h.log.ErrorContext(r.Context(), msgNoPrincipalLogText, "path", r.URL.Path)
		httpx.WriteErr(w, http.StatusInternalServerError, httpx.CodeInternal, msgInternal)
	}

	return p, ok
}

// fail пишет ответ на ошибку. Отмена запроса клиентом — не ошибка сервиса:
// ответ уже некому читать, и в лог она не пишется (styleguide.md, раздел 7).
// op — действие для журнала; пустой op — неудачи в журнал не пишутся.
func (h *Handler) fail(w http.ResponseWriter, r *http.Request, p auth.Principal, op string, err error) {
	ctx := r.Context()

	if errors.Is(err, context.Canceled) {
		return
	}

	var apiErr *apiError
	if !errors.As(err, &apiErr) {
		apiErr = internalErr(err)
	}

	if apiErr.attempted && op != "" {
		h.logOp(ctx, p.UserID, op, "", false)
	}

	if apiErr.status >= http.StatusInternalServerError {
		// Текст ошибки не содержит ни сообщений, ни ключей: cryptoclient,
		// sealed и store не включают значения полей в тексты ошибок.
		h.log.ErrorContext(ctx, "messages request failed", "path", r.URL.Path, "user_id", p.UserID, "err", err)
	}

	httpx.WriteErr(w, apiErr.status, apiErr.code, apiErr.message)
}

// decodeJSON проверяет Content-Type и разбирает тело в v.
func decodeJSON(r *http.Request, v any) error {
	if !isJSONRequest(r) {
		return &apiError{
			status: http.StatusUnsupportedMediaType, code: httpx.CodeUnsupportedMediaType, message: msgBodyNotJSON,
		}
	}

	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return &apiError{
				status: http.StatusRequestEntityTooLarge, code: httpx.CodePayloadTooLarge, message: msgBodyTooLarge,
			}
		}

		return validationErr(msgBodyInvalid)
	}

	return nil
}

// isJSONRequest проверяет, что Content-Type — application/json с любыми
// параметрами, например charset.
func isJSONRequest(r *http.Request) bool {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))

	return err == nil && mediaType == "application/json"
}
