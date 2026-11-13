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

// registerDevB регистрирует маршруты Dev B: запись сообщений, дешифрование и PDF.
//
// Файл принадлежит Dev B, и Dev A его не правит. Маршрут записи требует
// хранилище, крипто-клиент, менеджер ключей и проверку токена: без любой из
// этих зависимостей он не регистрируется, как и маршруты Dev A без своих.
func registerDevB(mux *http.ServeMux, cfg Config, st ports.Store, devB *DevB, log *slog.Logger) {
	if devB == nil || st == nil || devB.Crypto == nil || devB.Keys == nil {
		return
	}

	// Без адреса Сервиса учётных записей токен проверить нечем. Об этом уже
	// пишет registerDevA, второе сообщение в логе не нужно.
	if cfg.AccountsBase() == nil {
		return
	}

	journal := opslog.New(st, log)
	gw := newAccountsGateway(cfg, journal, log)

	write := messages.NewWriteHandler(st, devB.Crypto, devB.Keys, journal.Log, messages.WriteConfig{
		MasterKeyHex: devB.MasterKeyHex,
		MaxTextBytes: devB.MaxMessageBytes,
	}, log)

	mux.Handle("POST /api/messages", httpx.BodyLimit(write.BodyLimit())(gw.authn(http.HandlerFunc(write.Create))))
}
