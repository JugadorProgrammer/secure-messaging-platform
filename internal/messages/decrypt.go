package messages

import (
	"context"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/auth"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/httpx"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/methods"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/ports"
)

const (
	kuzKeyLen = 32
	kuzIVLen  = 16
)

// decryption — разобранный запрос расшифрования.
type decryption struct {
	method     ports.Method
	ciphertext []byte
	privatePEM string
	keyHex     string
	ivHex      string
}

// Decrypt отвечает на POST /api/decrypt: расшифровывает шифртекст ключом из
// запроса. Расшифрование не зависит от того, включён ли метод: иначе
// сохранённые сообщения стали бы нечитаемыми после выключения метода.
//
// IV Кузнечика берётся из запроса, а без него — из сохранённого сообщения
// с тем же шифртекстом. Если это сообщение принадлежит пользователю, у него
// обновляется decrypt_ms (CS-M8).
//
// @Summary  Расшифровать сообщение
// @Tags     messages
// @Accept   json
// @Produce  json
// @Security SessionCookie
// @Param    request body     DecryptRequest true "Метод, шифртекст в hex и ключ"
// @Success  200     {object} DecryptResponse
// @Failure  400     {object} httpx.ErrorResponse "validation, bad_key, bad_ciphertext"
// @Failure  401     {object} httpx.ErrorResponse "unauthorized"
// @Failure  503     {object} httpx.ErrorResponse "crypto_unavailable"
// @Router   /api/decrypt [post]
func (h *Handler) Decrypt(w http.ResponseWriter, r *http.Request) {
	p, ok := h.principal(w, r)
	if !ok {
		return
	}

	resp, ref, err := h.decrypt(r.Context(), p, r)
	if err != nil {
		h.fail(w, r, p, opDecrypt, err)

		return
	}

	h.logOp(r.Context(), p.UserID, opDecrypt, ref, true)

	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusOK, resp)
}

// decrypt возвращает ответ и ref для журнала: идентификатор сообщения, если
// шифртекст принадлежит сохранённому сообщению.
func (h *Handler) decrypt(ctx context.Context, p auth.Principal, r *http.Request) (DecryptResponse, string, error) {
	d, err := decodeDecrypt(r)
	if err != nil {
		return DecryptResponse{}, "", err
	}

	stored, err := h.storedMessage(ctx, d)
	if err != nil {
		return DecryptResponse{}, "", err
	}

	var (
		plain []byte
		ms    int64
	)

	switch d.method {
	case ports.MethodKuznyechik:
		ivHex := d.ivHex
		if ivHex == "" {
			if stored == nil {
				return DecryptResponse{}, "", badCiphertextErr(msgIVNotFound)
			}

			ivHex = hex.EncodeToString(stored.KuzIV)
		}

		plain, ms, err = h.crypto.KuzDecrypt(ctx, d.keyHex, ivHex, d.ciphertext)
	default:
		plain, ms, err = h.crypto.RSADecrypt(ctx, d.privatePEM, d.ciphertext)
	}

	if err != nil {
		return DecryptResponse{}, "", cryptoErr(err, badKeyErr(msgDecryptFailed))
	}

	var ref string

	if stored != nil {
		ref = strconv.FormatInt(stored.ID, 10)
		h.recordDecryptMs(ctx, p, stored, ms)
	}

	return DecryptResponse{Plaintext: string(plain)}, ref, nil
}

// decodeDecrypt проверяет запрос до обращения к Крипто-модулю: формат ключа
// и шифртекста проверяется здесь, а не превращается в 503 или 500.
func decodeDecrypt(r *http.Request) (decryption, error) {
	var req DecryptRequest
	if err := decodeJSON(r, &req); err != nil {
		return decryption{}, err
	}

	method, known := methods.CodeByName(req.Method)
	if !known {
		return decryption{}, validationErr(msgUnknownMethod)
	}

	ct, err := hex.DecodeString(strings.TrimSpace(req.Ciphertext))
	if err != nil || len(ct) == 0 {
		return decryption{}, badCiphertextErr(msgCiphertextInvalid)
	}

	d := decryption{method: method, ciphertext: ct}

	if method == ports.MethodRSA {
		if block, _ := pem.Decode([]byte(req.PrivateKey)); block == nil {
			return decryption{}, badKeyErr(msgPrivateKeyInvalid)
		}

		d.privatePEM = req.PrivateKey

		return d, nil
	}

	d.keyHex = strings.ToLower(strings.TrimSpace(req.Key))
	if !isHexOfLen(d.keyHex, kuzKeyLen) {
		return decryption{}, badKeyErr(msgKuzKeyInvalid)
	}

	d.ivHex = strings.ToLower(strings.TrimSpace(req.IV))
	if d.ivHex != "" && !isHexOfLen(d.ivHex, kuzIVLen) {
		return decryption{}, badKeyErr(msgIVInvalid)
	}

	return d, nil
}

// storedMessage находит сохранённое сообщение с тем же шифртекстом или nil.
func (h *Handler) storedMessage(ctx context.Context, d decryption) (*ports.Message, error) {
	m, err := h.store.MessageByCiphertext(ctx, string(d.method), d.ciphertext)

	switch {
	case errors.Is(err, ports.ErrNotFound):
		return nil, nil //nolint:nilnil // отсутствие сообщения — нормальный исход: шифртекст мог прийти извне
	case err != nil:
		return nil, internalErr(fmt.Errorf("find message by ciphertext: %w", err))
	default:
		return m, nil
	}
}

// recordDecryptMs обновляет время расшифрования своего сообщения. Сбой
// записи не меняет ответ: текст уже расшифрован.
func (h *Handler) recordDecryptMs(ctx context.Context, p auth.Principal, m *ports.Message, ms int64) {
	if m.OwnerID != p.UserID {
		return
	}

	if err := h.store.SetDecryptMs(context.WithoutCancel(ctx), m.ID, ms); err != nil {
		h.log.WarnContext(ctx, "save decrypt_ms", "message_id", m.ID, "err", err)
	}
}

func isHexOfLen(s string, n int) bool {
	b, err := hex.DecodeString(s)

	return err == nil && len(b) == n
}
