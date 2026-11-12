package api

import (
	"log/slog"
	"net/http"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/auth"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/httpx"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/opslog"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/ports"
)

// accountsPathPrefix — внешний префикс маршрутов. До Сервиса учётных записей
// запрос доходит без него: /api/auth/login превращается в /auth/login.
const accountsPathPrefix = "/api"

// accountsPublicRoutes — вход, регистрация и восстановление пароля. Токена не
// требуют: пользователь получает его в ответе.
var accountsPublicRoutes = []string{
	"POST /api/auth/signup",
	"POST /api/auth/login",
	"POST /api/auth/login/email-code",
	"POST /api/auth/login/recovery-code",
	"POST /api/auth/password/restore",
	"POST /api/auth/password/confirm",
}

// registerDevA регистрирует маршруты Dev A: auth, methods, messages (чтение)
// и verify.
//
// Здесь проксируются публичные маршруты входа, выход и управление
// пользователями. Методы шифрования добавятся в следующей задаче, чтение
// сообщений и проверка подписи — после них.
//
// st — хранилище для журнала операций; nil означает, что писать и читать
// журнал некуда: отказ в доступе тогда не записывается, а GET /api/ops-log
// не регистрируется.
func registerDevA(mux *http.ServeMux, cfg Config, st ports.Store, log *slog.Logger) {
	base := cfg.AccountsBase()
	if base == nil {
		log.Error("accounts routes are not registered: accounts url is not configured")

		return
	}

	proxy := http.StripPrefix(accountsPathPrefix, auth.NewAccountsProxy(base, log))
	client := auth.NewAccountsClient(base, cfg.AccountsCallTimeout(), log)
	authn := auth.Authn(client)

	// Журнал подключается к проверке роли: отказ пользователя на
	// административном маршруте пишется как access.denied. Без хранилища
	// Logger остаётся nil и запись молча пропускается, ответ не меняется.
	journal := opslog.New(st, log)
	requireAdmin := auth.Require(auth.RoleAdmin, journal.Log)

	for _, pattern := range accountsPublicRoutes {
		mux.Handle(pattern, httpx.BodyLimit(cfg.AuthBodyLimit())(proxy))
	}

	// Выход требует активную сессию: без токена запрос до Accounts не доходит.
	mux.Handle("POST /api/auth/logout", httpx.BodyLimit(cfg.AuthBodyLimit())(authn(proxy)))

	mux.Handle("GET /api/users", authn(requireAdmin(proxy)))
	mux.Handle("POST /api/users", httpx.BodyLimit(cfg.UsersBodyLimit())(authn(requireAdmin(proxy))))

	if st != nil {
		registerOpsLogRoutes(mux, authn, requireAdmin, st, log)
	}
}

// registerOpsLogRoutes открывает журнал администратору. Чтение возможно
// только при подключённом хранилище, иначе ответ был бы пустым списком там,
// где данных просто нет.
func registerOpsLogRoutes(mux *http.ServeMux, authn, requireAdmin func(http.Handler) http.Handler, st ports.Store, log *slog.Logger) {
	handler := opslog.NewHandler(st, log)
	mux.Handle("GET /api/ops-log", authn(requireAdmin(http.HandlerFunc(handler.List))))
}
