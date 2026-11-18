package messages

import (
	"context"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/auth"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/httpx"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/methods"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/ports"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/sealed"
)

// encrypted — результат шифрования до сохранения.
type encrypted struct {
	ciphertext   []byte
	keySize      int
	keygenMs     int64
	encryptMs    int64
	kuzKeySealed []byte
	kuzIV        []byte
	keys         KeysDTO
}

// Encrypt отвечает на POST /api/encrypt: шифрует текст выбранным методом,
// сохраняет сообщение и возвращает его с ключами.
//
// Текст сообщения и ключи не логируются (styleguide.md, раздел 10).
//
// @Summary  Зашифровать и сохранить сообщение
// @Tags     messages
// @Accept   json
// @Produce  json
// @Security SessionCookie
// @Param    request body     EncryptRequest true "Текст и название метода"
// @Success  201     {object} MessageDTO
// @Failure  400     {object} httpx.ErrorResponse "validation"
// @Failure  401     {object} httpx.ErrorResponse "unauthorized"
// @Failure  409     {object} httpx.ErrorResponse "method_disabled, key_generating"
// @Failure  413     {object} httpx.ErrorResponse "payload_too_large"
// @Failure  415     {object} httpx.ErrorResponse "unsupported_media_type"
// @Failure  503     {object} httpx.ErrorResponse "crypto_unavailable"
// @Router   /api/encrypt [post]
func (h *Handler) Encrypt(w http.ResponseWriter, r *http.Request) {
	p, ok := h.principal(w, r)
	if !ok {
		return
	}

	msg, err := h.encrypt(r.Context(), p, r)
	if err != nil {
		h.fail(w, r, p, opEncrypt, err)

		return
	}

	id := strconv.FormatInt(msg.ID, 10)
	h.logOp(r.Context(), p.UserID, opEncrypt, id, true)

	w.Header().Set("Cache-Control", "no-store")
	httpx.WriteJSON(w, http.StatusCreated, msg)
}

func (h *Handler) encrypt(ctx context.Context, p auth.Principal, r *http.Request) (MessageDTO, error) {
	text, method, err := h.decodeEncrypt(r)
	if err != nil {
		return MessageDTO{}, err
	}

	if err := h.checkMethodEnabled(ctx, method); err != nil {
		return MessageDTO{}, err
	}

	var enc *encrypted

	switch method {
	case ports.MethodKuznyechik:
		enc, err = h.encryptKuznyechik(ctx, text)
	default:
		enc, err = h.encryptRSA(ctx, p.UserID, text)
	}

	if err != nil {
		return MessageDTO{}, err
	}

	m := &ports.Message{
		OwnerID:      p.UserID,
		OwnerLogin:   p.Login,
		Method:       string(method),
		KeySize:      enc.keySize,
		Plaintext:    text,
		Ciphertext:   enc.ciphertext,
		KuzKeySealed: enc.kuzKeySealed,
		KuzIV:        enc.kuzIV,
		KeygenMs:     enc.keygenMs,
		EncryptMs:    enc.encryptMs,
	}

	// CreateMessage заполняет m.ID и m.CreatedAt значениями из базы.
	if _, err := h.store.CreateMessage(ctx, m); err != nil {
		return MessageDTO{}, internalErr(fmt.Errorf("save message: %w", err))
	}

	return toDTO(m, enc.keys), nil
}

// decodeEncrypt проверяет тело запроса. Возвращается первая проблема,
// а не список (styleguide.md, раздел 8).
func (h *Handler) decodeEncrypt(r *http.Request) (string, ports.Method, error) {
	var req EncryptRequest
	if err := decodeJSON(r, &req); err != nil {
		return "", "", err
	}

	method, known := methods.CodeByName(req.Method)

	switch {
	case strings.TrimSpace(req.Plaintext) == "":
		return "", "", validationErr(msgTextEmpty)
	case int64(len(req.Plaintext)) > h.maxTextBytes:
		return "", "", validationErr(fmt.Sprintf(msgTextTooLongFmt, h.maxTextBytes>>10))
	case !known:
		return "", "", validationErr(msgUnknownMethod)
	}

	return req.Plaintext, method, nil
}

func (h *Handler) checkMethodEnabled(ctx context.Context, method ports.Method) error {
	enabled, err := methods.MethodEnabled(ctx, h.store, string(method))
	if err != nil {
		return internalErr(err)
	}

	if !enabled {
		return &apiError{status: http.StatusConflict, code: httpx.CodeMethodDisabled, message: msgMethodDisabled}
	}

	return nil
}

// encryptKuznyechik шифрует новым ключом и IV: они генерируются на каждое
// сообщение. Ключ сохраняется запечатанным под MASTER_KEY, IV — как есть.
func (h *Handler) encryptKuznyechik(ctx context.Context, text string) (*encrypted, error) {
	keyHex, ivHex, err := h.crypto.KuzKeys(ctx)
	if err != nil {
		return nil, cryptoErr(err, validationErr(msgEncryptBadInput))
	}

	ct, encryptMs, err := h.crypto.KuzEncrypt(ctx, keyHex, ivHex, []byte(text))
	if err != nil {
		return nil, cryptoErr(err, validationErr(msgEncryptBadInput))
	}

	keySealed, err := sealed.Seal(ctx, h.crypto, h.masterKeyHex, []byte(keyHex))
	if err != nil {
		return nil, cryptoErr(err, validationErr(msgEncryptBadInput))
	}

	iv, err := hex.DecodeString(ivHex)
	if err != nil {
		// cryptoclient проверяет hex в ответе модуля, сюда попасть нельзя.
		return nil, internalErr(fmt.Errorf("decode iv: %w", err))
	}

	return &encrypted{
		ciphertext:   ct,
		keySize:      kuzKeyBits,
		encryptMs:    encryptMs,
		kuzKeySealed: keySealed,
		kuzIV:        iv,
		keys:         KeysDTO{Key: keyHex, IV: ivHex},
	}, nil
}

// encryptRSA шифрует открытым ключом пользователя. Пока ключ генерируется,
// отвечает 409 key_generating: генерация RSA-32768 занимает десятки минут.
func (h *Handler) encryptRSA(ctx context.Context, userID int64, text string) (*encrypted, error) {
	key, err := h.keys.EnsureUserKey(ctx, userID)
	if err != nil {
		return nil, internalErr(fmt.Errorf("ensure user key: %w", err))
	}

	if key.Status != ports.KeyStatusReady {
		return nil, &apiError{status: http.StatusConflict, code: httpx.CodeKeyGenerating, message: msgKeyGenerating}
	}

	// Ключи раскрываются до шифрования: если контейнер повреждён, сообщение
	// не должно сохраниться без возможности показать владельцу ключ.
	keys, err := h.rsaKeys(ctx, key)
	if err != nil {
		return nil, err
	}

	ct, encryptMs, err := h.crypto.RSAEncrypt(ctx, key.PublicKeyPEM, []byte(text))
	if err != nil {
		return nil, cryptoErr(err, validationErr(msgEncryptBadInput))
	}

	return &encrypted{
		ciphertext: ct,
		keySize:    key.Bits,
		keygenMs:   key.KeygenMs,
		encryptMs:  encryptMs,
		keys:       keys,
	}, nil
}
