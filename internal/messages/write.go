package messages

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/auth"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/httpx"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/methods"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/ports"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/sealed"
)

// opEncrypt — действие в журнале операций.
const opEncrypt = "message.encrypt"

// kuzKeyBits — размер ключа Кузнечика, он фиксирован.
const kuzKeyBits = 256

// bodyOverhead — запас тела запроса сверх MAX_MESSAGE_BYTES на JSON-обёртку
// и экранирование: кириллица в виде \uXXXX занимает втрое больше места.
const bodyOverhead = 128 << 10

// Тексты ошибок видит клиент (contracts.md, раздел 4).
const (
	msgBodyNotJSON       = "Ожидается тело в формате application/json"
	msgBodyTooLarge      = "Тело запроса слишком велико"
	msgBodyInvalid       = "Некорректное тело запроса"
	msgTextEmpty         = "Текст сообщения не может быть пустым"
	msgTextTooLongFmt    = "Текст сообщения превышает %d КиБ"
	msgUnknownMethod     = "Неизвестный метод шифрования"
	msgMethodDisabled    = "Метод шифрования выключен администратором"
	msgKeyGenerating     = "Ключ шифрования генерируется, повторите позже"
	msgCryptoBadInput    = "Текст не удалось зашифровать выбранным методом"
	msgCryptoUnavailable = "Крипто-модуль недоступен, повторите позже"
	msgInternal          = "Внутренняя ошибка сервера"
)

// AuditFunc — запись в журнал операций, как opslog.Logger.Log. nil — журнал
// не ведётся, ответ от этого не меняется.
type AuditFunc func(ctx context.Context, userID *int64, op, ref string, ok bool)

// KeyManager выдаёт ключ RSA пользователя и при необходимости запускает его
// генерацию. keys.Manager реализует его.
type KeyManager interface {
	EnsureUserKey(ctx context.Context, userID int64) (*ports.UserKey, error)
}

// WriteConfig — параметры записи сообщений из конфигурации сервиса.
type WriteConfig struct {
	// MasterKeyHex — MASTER_KEY в hex: под ним запечатывается ключ Кузнечика
	// и распечатывается закрытый ключ RSA.
	MasterKeyHex string
	// MaxTextBytes — MAX_MESSAGE_BYTES, предельный размер текста.
	MaxTextBytes int64
}

// WriteHandler обслуживает POST /api/messages.
type WriteHandler struct {
	store        ports.Store
	crypto       ports.CryptoClient
	keys         KeyManager
	audit        AuditFunc
	masterKeyHex string
	maxTextBytes int64
	log          *slog.Logger
}

// NewWriteHandler собирает обработчик. log == nil допустим: используется slog.Default.
func NewWriteHandler(
	store ports.Store, crypto ports.CryptoClient, keys KeyManager, audit AuditFunc, cfg WriteConfig, log *slog.Logger,
) *WriteHandler {
	if log == nil {
		log = slog.Default()
	}

	return &WriteHandler{
		store:        store,
		crypto:       crypto,
		keys:         keys,
		audit:        audit,
		masterKeyHex: cfg.MasterKeyHex,
		maxTextBytes: cfg.MaxTextBytes,
		log:          log,
	}
}

// BodyLimit — предельный размер тела POST /api/messages. Маршрут оборачивается
// в httpx.BodyLimit с этим значением.
func (h *WriteHandler) BodyLimit() int64 {
	return h.maxTextBytes + bodyOverhead
}

// encrypted — результат шифрования до сохранения.
type encrypted struct {
	ciphertext   []byte
	keySize      int
	keygenMs     int64
	encryptMs    int64
	kuzKeySealed []byte
	kuzIV        []byte
	keys         *KeysDTO
}

// apiError — ошибка, у которой уже выбран ответ клиенту.
type apiError struct {
	status  int
	code    string
	message string
	// attempted — шифрование начиналось: неудача пишется в журнал с ok = false.
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

// Create отвечает на POST /api/messages: шифрует текст выбранным методом,
// сохраняет сообщение и возвращает его с ключами и метриками.
//
// Текст сообщения и ключи не логируются (styleguide.md, раздел 10).
//
// @Summary  Зашифровать и сохранить сообщение
// @Tags     messages
// @Accept   json
// @Produce  json
// @Security BearerAuth
// @Param    request body     EncryptRequest true "Текст и метод шифрования"
// @Success  201     {object} MessageDTO
// @Failure  400     {object} httpx.ErrorResponse "validation"
// @Failure  401     {object} httpx.ErrorResponse "unauthorized"
// @Failure  409     {object} httpx.ErrorResponse "method_disabled, key_generating"
// @Failure  413     {object} httpx.ErrorResponse "payload_too_large"
// @Failure  415     {object} httpx.ErrorResponse "unsupported_media_type"
// @Failure  503     {object} httpx.ErrorResponse "crypto_unavailable"
// @Router   /api/messages [post]
func (h *WriteHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	p, ok := auth.PrincipalFrom(ctx)
	if !ok {
		// Маршрут регистрируется только под Authn: без пользователя сюда
		// попасть нельзя, это ошибка сборки маршрутов.
		h.log.ErrorContext(ctx, "encrypt message: no principal in context")
		httpx.WriteErr(w, http.StatusInternalServerError, httpx.CodeInternal, msgInternal)

		return
	}

	msg, err := h.create(ctx, p, r)
	if err != nil {
		h.fail(w, r, p, err)

		return
	}

	h.logOp(ctx, p, strconv.FormatInt(msg.ID, 10), true)

	w.Header().Set("Location", "/api/messages/"+strconv.FormatInt(msg.ID, 10))
	httpx.WriteJSON(w, http.StatusCreated, msg)
}

func (h *WriteHandler) create(ctx context.Context, p auth.Principal, r *http.Request) (MessageDTO, error) {
	req, err := h.decode(r)
	if err != nil {
		return MessageDTO{}, err
	}

	if err := h.checkMethodEnabled(ctx, req.Method); err != nil {
		return MessageDTO{}, err
	}

	var enc *encrypted

	switch req.Method {
	case ports.MethodKuznyechik:
		enc, err = h.encryptKuznyechik(ctx, req.Text)
	default:
		enc, err = h.encryptRSA(ctx, p.UserID, req.Text)
	}

	if err != nil {
		return MessageDTO{}, err
	}

	m := &ports.Message{
		OwnerID:      p.UserID,
		OwnerLogin:   p.Login,
		Method:       string(req.Method),
		KeySize:      enc.keySize,
		Plaintext:    req.Text,
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

// decode проверяет заголовок и тело запроса. Возвращается первая проблема,
// а не список (styleguide.md, раздел 8).
func (h *WriteHandler) decode(r *http.Request) (EncryptRequest, error) {
	var req EncryptRequest

	if !isJSONRequest(r) {
		return req, &apiError{
			status: http.StatusUnsupportedMediaType, code: httpx.CodeUnsupportedMediaType, message: msgBodyNotJSON,
		}
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return req, &apiError{
				status: http.StatusRequestEntityTooLarge, code: httpx.CodePayloadTooLarge, message: msgBodyTooLarge,
			}
		}

		return req, validationErr(msgBodyInvalid)
	}

	switch {
	case strings.TrimSpace(req.Text) == "":
		return req, validationErr(msgTextEmpty)
	case int64(len(req.Text)) > h.maxTextBytes:
		return req, validationErr(fmt.Sprintf(msgTextTooLongFmt, h.maxTextBytes>>10))
	case !methods.IsKnownCode(string(req.Method)):
		return req, validationErr(msgUnknownMethod)
	}

	return req, nil
}

func (h *WriteHandler) checkMethodEnabled(ctx context.Context, method ports.Method) error {
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
func (h *WriteHandler) encryptKuznyechik(ctx context.Context, text string) (*encrypted, error) {
	keyHex, ivHex, err := h.crypto.KuzKeys(ctx)
	if err != nil {
		return nil, cryptoErr(err)
	}

	ct, encryptMs, err := h.crypto.KuzEncrypt(ctx, keyHex, ivHex, []byte(text))
	if err != nil {
		return nil, cryptoErr(err)
	}

	keySealed, err := sealed.Seal(ctx, h.crypto, h.masterKeyHex, []byte(keyHex))
	if err != nil {
		return nil, cryptoErr(err)
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
		keys:         &KeysDTO{KeyHex: keyHex, IVHex: ivHex},
	}, nil
}

// encryptRSA шифрует открытым ключом пользователя. Пока ключ генерируется,
// отвечает 409 key_generating: генерация RSA-32768 занимает десятки минут.
func (h *WriteHandler) encryptRSA(ctx context.Context, userID int64, text string) (*encrypted, error) {
	key, err := h.keys.EnsureUserKey(ctx, userID)
	if err != nil {
		return nil, internalErr(fmt.Errorf("ensure user key: %w", err))
	}

	if key.Status != ports.KeyStatusReady {
		return nil, &apiError{status: http.StatusConflict, code: httpx.CodeKeyGenerating, message: msgKeyGenerating}
	}

	// Закрытый ключ распечатывается до шифрования: если контейнер повреждён,
	// сообщение не должно сохраниться без возможности показать владельцу ключ.
	privateBytes, err := sealed.Unseal(ctx, h.crypto, h.masterKeyHex, key.PrivateSealed)
	if err != nil {
		return nil, unsealErr(err)
	}
	defer clear(privateBytes)

	ct, encryptMs, err := h.crypto.RSAEncrypt(ctx, key.PublicKeyPEM, []byte(text))
	if err != nil {
		return nil, cryptoErr(err)
	}

	return &encrypted{
		ciphertext: ct,
		keySize:    key.Bits,
		keygenMs:   key.KeygenMs,
		encryptMs:  encryptMs,
		keys:       &KeysDTO{PublicKeyPEM: key.PublicKeyPEM, PrivateKeyPEM: string(privateBytes)},
	}, nil
}

// fail пишет ответ на ошибку. Отмена запроса клиентом — не ошибка сервиса:
// ответ уже некому читать, и в лог она не пишется (styleguide.md, раздел 7).
func (h *WriteHandler) fail(w http.ResponseWriter, r *http.Request, p auth.Principal, err error) {
	ctx := r.Context()

	if errors.Is(err, context.Canceled) {
		return
	}

	var apiErr *apiError
	if !errors.As(err, &apiErr) {
		apiErr = internalErr(err)
	}

	if apiErr.attempted {
		h.logOp(ctx, p, "", false)
	}

	if apiErr.status >= http.StatusInternalServerError {
		h.log.ErrorContext(ctx, "encrypt message", "user_id", p.UserID, "err", err)
	}

	httpx.WriteErr(w, apiErr.status, apiErr.code, apiErr.message)
}

func (h *WriteHandler) logOp(ctx context.Context, p auth.Principal, ref string, ok bool) {
	if h.audit == nil {
		return
	}

	userID := p.UserID
	h.audit(ctx, &userID, opEncrypt, ref, ok)
}

func validationErr(message string) *apiError {
	return &apiError{status: http.StatusBadRequest, code: httpx.CodeValidation, message: message}
}

func internalErr(err error) *apiError {
	return &apiError{
		status: http.StatusInternalServerError, code: httpx.CodeInternal, message: msgInternal,
		attempted: true, cause: err,
	}
}

// cryptoErr: недоступность модуля — 503; отказ модуля принять данные — 400
// validation, потому что от пользователя пришёл только текст, а не ключ.
func cryptoErr(err error) *apiError {
	switch {
	case errors.Is(err, ports.ErrCryptoUnavailable):
		return &apiError{
			status: http.StatusServiceUnavailable, code: httpx.CodeCryptoUnavailable, message: msgCryptoUnavailable,
			attempted: true, cause: err,
		}
	case errors.Is(err, ports.ErrCryptoBadInput):
		return &apiError{
			status: http.StatusBadRequest, code: httpx.CodeValidation, message: msgCryptoBadInput,
			attempted: true, cause: err,
		}
	default:
		return internalErr(err)
	}
}

// unsealErr: повреждённый контейнер или чужой MASTER_KEY — внутренняя
// проблема, 500 (contracts.md, раздел 6); недоступный модуль — 503.
func unsealErr(err error) *apiError {
	if errors.Is(err, ports.ErrCryptoUnavailable) {
		return cryptoErr(err)
	}

	return internalErr(fmt.Errorf("unseal private key: %w", err))
}

// isJSONRequest проверяет, что Content-Type — application/json с любыми
// параметрами, например charset.
func isJSONRequest(r *http.Request) bool {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))

	return err == nil && mediaType == "application/json"
}
