package messages

import (
	"context"
	"fmt"
	"net/http"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/auth"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/httpx"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/ports"
)

// List отвечает на GET /api/getAllMessages: пользователь получает свои
// сообщения, администратор — сообщения всех пользователей. Новые сверху.
//
// Ключи отдаются вместе с сообщениями: схема фронта требует их у каждого
// сообщения, а журнал администратора по ТЗ показывает сообщения целиком.
//
// @Summary  Список сообщений
// @Tags     messages
// @Produce  json
// @Security SessionCookie
// @Success  200 {array}  MessageDTO
// @Failure  401 {object} httpx.ErrorResponse "unauthorized"
// @Failure  503 {object} httpx.ErrorResponse "crypto_unavailable"
// @Router   /api/getAllMessages [get]
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	p, ok := h.principal(w, r)
	if !ok {
		return
	}

	msgs, err := h.list(r.Context(), p)
	if err != nil {
		h.fail(w, r, p, "", err)

		return
	}

	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusOK, msgs)
}

func (h *Handler) list(ctx context.Context, p auth.Principal) ([]MessageDTO, error) {
	var ownerID *int64
	if p.Role != auth.RoleAdmin {
		ownerID = &p.UserID
	}

	stored, err := h.allMessages(ctx, ownerID)
	if err != nil {
		return nil, err
	}

	keys := newKeyCache(h)
	out := make([]MessageDTO, 0, len(stored))

	for _, m := range stored {
		k, err := keys.keysFor(ctx, m)
		if err != nil {
			return nil, err
		}

		out = append(out, toDTO(m, k))
	}

	return out, nil
}

// allMessages читает сообщения постранично: ListMessages отдаёт не больше
// listPageSize за раз, а фронт ждёт весь список одним ответом.
func (h *Handler) allMessages(ctx context.Context, ownerID *int64) ([]*ports.Message, error) {
	var all []*ports.Message

	for offset := 0; offset < maxListedMessages; offset += listPageSize {
		page, err := h.store.ListMessages(ctx, ownerID, listPageSize, offset)
		if err != nil {
			return nil, internalErr(fmt.Errorf("list messages: %w", err))
		}

		all = append(all, page...)

		if len(page) < listPageSize {
			break
		}
	}

	return all, nil
}
