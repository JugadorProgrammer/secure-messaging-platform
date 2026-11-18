package api

import (
	"log/slog"
	"net/http"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/httpx"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/messages"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/opslog"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/ports"
)

// DevB — зависимости маршрутов Dev B, которых нет у остальных маршрутов.
// Передаются в New через WithDevB.
type DevB struct {
	Crypto ports.CryptoClient
	Keys   messages.KeyManager
	// MasterKeyHex — MASTER_KEY в hex для запечатывания ключей.
	MasterKeyHex string
	// MaxMessageBytes — MAX_MESSAGE_BYTES, предельный размер текста сообщения.
	MaxMessageBytes int64
}

// Option настраивает New.
type Option func(*options)

type options struct {
	devB *DevB
}

// WithDevB подключает маршруты Dev B. Без этой опции они не регистрируются:
// так тесты маршрутов Dev A не обязаны собирать крипто-клиент и ключи.
func WithDevB(d DevB) Option {
	return func(o *options) {
		o.devB = &d
	}
}

// registerDevB регистрирует маршруты сообщений: шифрование, расшифрование,
// список и PDF с подписью.
//
// Маршрутам нужны хранилище с поиском по шифртексту, крипто-клиент, менеджер
// ключей и проверка сессии: без любой из этих зависимостей они не
// регистрируются, как и маршруты Dev A без своих.
func registerDevB(mux *http.ServeMux, cfg Config, st ports.Store, devB *DevB, log *slog.Logger) {
	if devB == nil || st == nil || devB.Crypto == nil || devB.Keys == nil {
		return
	}

	store, ok := st.(messages.Store)
	if !ok {
		log.Error("store does not support message lookup: message routes are not registered")

		return
	}

	// Без адреса Сервиса учётных записей сессию проверить нечем. Об этом уже
	// пишет registerDevA, второе сообщение в логе не нужно.
	if cfg.AccountsBase() == nil {
		return
	}

	journal := opslog.New(st, log)
	gw := newAccountsGateway(cfg, journal, log)

	h := messages.NewHandler(store, devB.Crypto, devB.Keys, journal.Log, messages.Config{
		MasterKeyHex: devB.MasterKeyHex,
		MaxTextBytes: devB.MaxMessageBytes,
	}, log)

	routes := []struct {
		pattern string
		limit   int64
		handler http.HandlerFunc
	}{
		{"POST /api/encrypt", h.EncryptBodyLimit(), h.Encrypt},
		{"POST /api/decrypt", h.DecryptBodyLimit(), h.Decrypt},
		{"POST /api/exportPdfWithSignature", h.ExportBodyLimit(), h.ExportPDF},
	}

	for _, rt := range routes {
		mux.Handle(rt.pattern, httpx.BodyLimit(rt.limit)(gw.authn(rt.handler)))
	}

	mux.Handle("GET /api/getAllMessages", gw.authn(http.HandlerFunc(h.List)))
}
