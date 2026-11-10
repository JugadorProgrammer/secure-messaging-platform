package methods

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"mime"
	"net/http"
	"unicode/utf8"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/auth"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/httpx"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/ports"
)

// opMethodToggle — действие в журнале операций.
const opMethodToggle = "method.toggle"

// maxAuditRefLen — предельная длина ref в журнале: значение приходит из пути,
// поэтому его обрезает тот, кто пишет запись.
const maxAuditRefLen = 200

// Тексты ошибок видит клиент, они одинаковы для всех ответов с этим кодом.
const (
	msgMethodsUnavailable = "Не удалось получить список методов"
	msgMethodNotFound     = "Метод не найден"
	msgEnabledRequired    = "Поле enabled обязательно"
	msgBodyNotJSON        = "Ожидается тело в формате application/json"
	msgBodyTooLarge       = "Тело запроса слишком велико"
)

// AuditFunc — запись в журнал операций (совпадает с ports.Store.LogOp по
// сигнатуре). Тип объявлен здесь, а не импортируется, чтобы методы не
// зависели от пакета авторизации. nil — журнал не ведётся, ответ не меняется.
type AuditFunc func(ctx context.Context, userID *int64, op, ref string, ok bool)

// Handler обслуживает маршруты методов шифрования.
type Handler struct {
	svc   *Service
	audit AuditFunc
	log   *slog.Logger
}

// NewHandler собирает обработчик. log == nil допустим: используется slog.Default.
func NewHandler(svc *Service, audit AuditFunc, log *slog.Logger) *Handler {
	if log == nil {
		log = slog.Default()
	}

	return &Handler{svc: svc, audit: audit, log: log}
}

// List отвечает GET /api/methods. Доступ публичный: список нужен интерфейсу
// до входа, чтобы показать доступные способы шифрования.
//
// @Summary  Список методов шифрования
// @Tags     methods
// @Produce  json
// @Success  200 {array} MethodDTO
// @Failure  500 {object} httpx.ErrorResponse
// @Router   /api/methods [get]
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	dtos, err := h.svc.List(r.Context())
	if err != nil {
		h.log.ErrorContext(r.Context(), "methods list failed", "err", err)
		httpx.WriteErr(w, http.StatusInternalServerError, httpx.CodeInternal, msgMethodsUnavailable)

		return
	}

	httpx.WriteJSON(w, http.StatusOK, dtos)
}

// Patch отвечает PATCH /api/methods/{code} и меняет флаг enabled.
//
// @Summary  Включить или выключить метод шифрования
// @Tags     methods
// @Accept   json
// @Produce  json
// @Param    code path string true "Код метода" Enums(rsa, kuznyechik)
// @Param    body body MethodPatchRequest true "Новое состояние метода"
// @Success  200 {object} MethodDTO
// @Failure  400 {object} httpx.ErrorResponse
// @Failure  404 {object} httpx.ErrorResponse
// @Failure  415 {object} httpx.ErrorResponse
// @Router   /api/methods/{code} [patch]
func (h *Handler) Patch(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")

	if !isJSONRequest(r) {
		httpx.WriteErr(w, http.StatusUnsupportedMediaType, httpx.CodeUnsupportedMediaType, msgBodyNotJSON)

		return
	}

	// Код проверяется до разбора тела: заведомо неизвестный метод не должен
	// превращаться в 400 из-за формата тела.
	if !IsKnownCode(code) {
		h.logToggle(r, code, false)
		httpx.WriteErr(w, http.StatusNotFound, httpx.CodeNotFound, msgMethodNotFound)

		return
	}

	dto, ok := h.applyToggle(w, r, code)
	if !ok {
		return
	}

	httpx.WriteJSON(w, http.StatusOK, dto)
}

// applyToggle разбирает тело, применяет изменение и пишет запись журнала.
// Второй результат — false, если ответ уже отправлен.
func (h *Handler) applyToggle(w http.ResponseWriter, r *http.Request, code string) (MethodDTO, bool) {
	var req MethodPatchRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var maxBytes *http.MaxBytesError
		if errors.As(err, &maxBytes) {
			httpx.WriteErr(w, http.StatusRequestEntityTooLarge, httpx.CodePayloadTooLarge, msgBodyTooLarge)

			return MethodDTO{}, false
		}

		h.log.WarnContext(r.Context(), "method patch body rejected", "err", err)
		httpx.WriteErr(w, http.StatusBadRequest, httpx.CodeValidation, msgBodyNotJSON)

		return MethodDTO{}, false
	}

	// nil означает «поле не передано». Без указателя это было бы false,
	// и пустое тело выключало бы метод молча.
	if req.Enabled == nil {
		httpx.WriteErr(w, http.StatusBadRequest, httpx.CodeValidation, msgEnabledRequired)

		return MethodDTO{}, false
	}

	dto, err := h.svc.SetEnabled(r.Context(), code, *req.Enabled)

	switch {
	case errors.Is(err, ports.ErrNotFound):
		h.logToggle(r, code, false)
		httpx.WriteErr(w, http.StatusNotFound, httpx.CodeNotFound, msgMethodNotFound)

		return MethodDTO{}, false
	case err != nil:
		h.log.ErrorContext(r.Context(), "method toggle failed", "code", code, "err", err)
		h.logToggle(r, code, false)
		httpx.WriteErr(w, http.StatusInternalServerError, httpx.CodeInternal, msgMethodsUnavailable)

		return MethodDTO{}, false
	}

	h.logToggle(r, code, true)

	return dto, true
}

// logToggle пишет результат переключения метода в журнал best-effort:
// сбой записи не меняет ответ клиента.
//
// Журнал касается только попыток изменить метод: 415, 400 и 413 — ошибки
// формата запроса, события безопасности в них нет.
func (h *Handler) logToggle(r *http.Request, code string, ok bool) {
	if h.audit == nil {
		return
	}

	var userID *int64
	if principal, found := auth.PrincipalFrom(r.Context()); found {
		userID = &principal.UserID
	}

	h.audit(r.Context(), userID, opMethodToggle, truncateRef(code), ok)
}

// truncateRef обрезает ref по длине журнала и по границе UTF-8 символа:
// значение приходит из пути запроса, битую последовательность в таблицу
// писать нельзя.
func truncateRef(ref string) string {
	if len(ref) <= maxAuditRefLen {
		return ref
	}

	cut := maxAuditRefLen
	for cut > 0 && !utf8.RuneStart(ref[cut]) {
		cut--
	}

	return ref[:cut]
}

// isJSONRequest проверяет, что Content-Type — application/json с любыми
// параметрами (например, charset). Пустой заголовок тоже отклоняется:
// у PATCH без тела поле enabled не задано, и ответ должен быть 415, а не 400.
func isJSONRequest(r *http.Request) bool {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		return false
	}

	return mediaType == "application/json"
}
