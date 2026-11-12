package opslog

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/httpx"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/ports"
)

const (
	// DefaultLimit — размер страницы журнала, когда limit не передан.
	DefaultLimit = 20
	// MaxLimit — предельный размер страницы. Границы задаёт контракт
	// ports.Store.ListOps, проверка в хендлере нужна, чтобы некорректный
	// запрос не уходил в базу.
	MaxLimit = 100
)

// Сообщения об ошибках разбора страницы. Текст видит клиент, поэтому он
// нейтральный и не раскрывает устройство хранилища.
const (
	msgLimitRange    = "limit должен быть целым числом от 1 до 100"
	msgOffsetRange   = "offset должен быть целым числом не меньше 0"
	msgBadPage       = "Некорректные параметры страницы"
	msgInternalError = "Внутренняя ошибка сервера"
)

// EntryDTO — запись журнала в ответе HTTP.
//
// UserID равен null для анонимной операции: идентификатор 0 занят системным
// ключом и выдать его за анонима нельзя.
type EntryDTO struct {
	ID     int64     `json:"id" example:"12"`
	At     time.Time `json:"at" example:"2026-01-08T14:22:03Z"`
	UserID *int64    `json:"user_id" example:"7"`
	Op     string    `json:"op" example:"method.toggle"`
	Ref    string    `json:"ref" example:"rsa"`
	OK     bool      `json:"ok" example:"true"`
}

// Handler читает журнал операций.
//
// Полномочия проверяет middleware: хендлер отвечает только за разбор страницы
// и формат ответа, поэтому роль администратора здесь не участвует.
type Handler struct {
	store ports.Store
	log   *slog.Logger
}

// NewHandler собирает хендлер журнала.
func NewHandler(store ports.Store, log *slog.Logger) *Handler {
	if log == nil {
		log = slog.Default()
	}

	return &Handler{store: store, log: log}
}

// List отвечает GET /api/ops-log: страница журнала, новые записи сверху.
//
// Пустой журнал отдаётся как [], а не null, чтобы клиент не разбирал два
// разных формата.
//
// @Summary  Журнал операций
// @Description Только роль admin. Сортировка: at DESC, id DESC.
// @Tags       admin
// @Produce    json
// @Param      limit  query int false "Размер страницы, 1..100, по умолчанию 20"
// @Param      offset query int false "Смещение, >= 0, по умолчанию 0"
// @Success    200 {array} EntryDTO
// @Failure    400 {object} httpx.ErrorResponse
// @Failure    401 {object} httpx.ErrorResponse
// @Failure    403 {object} httpx.ErrorResponse
// @Failure    500 {object} httpx.ErrorResponse
// @Router     /api/ops-log [get]
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	limit, offset, err := pageParams(r)
	if err != nil {
		httpx.WriteErr(w, http.StatusBadRequest, httpx.CodeValidation, err.Error())

		return
	}

	entries, err := h.store.ListOps(r.Context(), limit, offset)
	switch {
	case errors.Is(err, ports.ErrInvalidArgument):
		// Границы страницы отсекаются выше, до обращения к базе, поэтому
		// ветка — подстраховка. Текст ошибки хранилища клиенту не отдаётся.
		httpx.WriteErr(w, http.StatusBadRequest, httpx.CodeValidation, msgBadPage)
	case err != nil:
		h.log.ErrorContext(r.Context(), "ops log: list failed", "err", err)
		httpx.WriteErr(w, http.StatusInternalServerError, httpx.CodeInternal, msgInternalError)
	default:
		httpx.WriteJSON(w, http.StatusOK, toDTO(entries))
	}
}

// pageParams разбирает limit и offset. Пустое значение трактуется как
// «параметр не передан», то есть берётся по умолчанию.
func pageParams(r *http.Request) (limit, offset int, err error) {
	query := r.URL.Query()

	limit = DefaultLimit
	if raw := query.Get("limit"); raw != "" {
		n, convErr := strconv.Atoi(raw)
		if convErr != nil || n < 1 || n > MaxLimit {
			return 0, 0, errors.New(msgLimitRange)
		}

		limit = n
	}

	offset = 0
	if raw := query.Get("offset"); raw != "" {
		n, convErr := strconv.Atoi(raw)
		if convErr != nil || n < 0 {
			return 0, 0, errors.New(msgOffsetRange)
		}

		offset = n
	}

	return limit, offset, nil
}

// toDTO преобразует записи в ответ. Срез создаётся заранее пустым, чтобы
// отсутствие записей кодировалось в [] а не null.
func toDTO(entries []ports.OpLogEntry) []EntryDTO {
	dtos := make([]EntryDTO, 0, len(entries))
	for _, e := range entries {
		dtos = append(dtos, EntryDTO{
			ID: e.ID, At: e.At, UserID: e.UserID, Op: e.Op, Ref: e.Ref, OK: e.OK,
		})
	}

	return dtos
}
