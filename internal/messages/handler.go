package messages

import (
	"context"
	"log/slog"
	"time"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/ports"
)

// Действия в журнале операций.
const (
	opEncrypt      = "message.encrypt"
	opDecrypt      = "message.decrypt"
	opPDF          = "message.pdf"
	opAccessDenied = "access.denied"
)

const (
	// kuzKeyBits — размер ключа Кузнечика, он фиксирован.
	kuzKeyBits = 256

	// bodyOverhead — запас тела запроса сверх MAX_MESSAGE_BYTES на JSON-обёртку
	// и экранирование: кириллица в виде \uXXXX занимает втрое больше места.
	bodyOverhead = 128 << 10

	// maxExportBodyBytes — тело запроса PDF: только идентификатор.
	maxExportBodyBytes = 4 << 10

	// listPageSize — размер страницы при чтении всех сообщений: больше
	// ListMessages не отдаёт за раз.
	listPageSize = 100
	// maxListedMessages — верхняя граница ответа GET /api/getAllMessages.
	// Фронт пагинацию не поддерживает; граница защищает от ответа на сотни
	// мегабайт, если сообщений накопится очень много.
	maxListedMessages = 10_000
)

// AuditFunc — запись в журнал операций, как opslog.Logger.Log. nil — журнал
// не ведётся, ответ от этого не меняется.
type AuditFunc func(ctx context.Context, userID *int64, op, ref string, ok bool)

// KeyManager выдаёт ключ RSA пользователя и при необходимости запускает его
// генерацию. keys.Manager реализует его.
type KeyManager interface {
	EnsureUserKey(ctx context.Context, userID int64) (*ports.UserKey, error)
}

// Store — хранилище сообщений. store.Store реализует его; поиск по
// шифртексту нужен только этому пакету и в общий ports.Store не вынесен.
type Store interface {
	ports.Store
	// MessageByCiphertext возвращает ports.ErrNotFound, если сообщения нет.
	MessageByCiphertext(ctx context.Context, method string, ciphertext []byte) (*ports.Message, error)
}

// Config — параметры сообщений из конфигурации сервиса.
type Config struct {
	// MasterKeyHex — MASTER_KEY в hex: под ним запечатаны ключи Кузнечика
	// и закрытые ключи RSA.
	MasterKeyHex string
	// MaxTextBytes — MAX_MESSAGE_BYTES, предельный размер текста.
	MaxTextBytes int64
}

// Handler обслуживает маршруты сообщений: шифрование, список, расшифрование
// и PDF с подписью. Маршруты разложены по файлам write.go, read.go,
// decrypt.go и pdf.go.
type Handler struct {
	store        Store
	crypto       ports.CryptoClient
	keys         KeyManager
	audit        AuditFunc
	masterKeyHex string
	maxTextBytes int64
	log          *slog.Logger
	now          func() time.Time
}

// NewHandler собирает обработчик. log == nil допустим: используется slog.Default.
func NewHandler(store Store, crypto ports.CryptoClient, keys KeyManager, audit AuditFunc, cfg Config, log *slog.Logger) *Handler {
	if log == nil {
		log = slog.Default()
	}

	return &Handler{
		store:        store,
		crypto:       crypto,
		keys:         keys,
		audit:        audit,
		masterKeyHex: cfg.MasterKeyHex,
		maxTextBytes: cfg.MaxTextBytes,
		log:          log,
		now:          time.Now,
	}
}

// EncryptBodyLimit — предельный размер тела POST /api/encrypt.
func (h *Handler) EncryptBodyLimit() int64 {
	return h.maxTextBytes + bodyOverhead
}

// DecryptBodyLimit — предельный размер тела POST /api/decrypt. Шифртекст
// приходит в hex и вдвое длиннее текста, к нему добавляется PEM закрытого
// ключа RSA-32768 (около 25 КБ).
func (h *Handler) DecryptBodyLimit() int64 {
	return 2*h.maxTextBytes + bodyOverhead
}

// ExportBodyLimit — предельный размер тела POST /api/exportPdfWithSignature.
func (h *Handler) ExportBodyLimit() int64 {
	return maxExportBodyBytes
}

func (h *Handler) logOp(ctx context.Context, userID int64, op, ref string, ok bool) {
	if h.audit == nil {
		return
	}

	h.audit(ctx, &userID, op, ref, ok)
}
