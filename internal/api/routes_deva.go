package api

import (
	"log/slog"
	"net/http"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/auth"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/httpx"
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
func registerDevA(mux *http.ServeMux, cfg Config, log *slog.Logger) {
	base := cfg.AccountsBase()
	if base == nil {
		log.Error("accounts routes are not registered: accounts url is not configured")

		return
	}

	proxy := http.StripPrefix(accountsPathPrefix, auth.NewAccountsProxy(base, log))
	client := auth.NewAccountsClient(base, cfg.AccountsCallTimeout(), log)
	authn := auth.Authn(client)

	// audit == nil до появления хранилища: Require тогда пропускает отказ
	// без записи в журнал, ответ от этого не меняется.
	requireAdmin := auth.Require(auth.RoleAdmin, nil)

	for _, pattern := range accountsPublicRoutes {
		mux.Handle(pattern, httpx.BodyLimit(cfg.AuthBodyLimit())(proxy))
	}

	// Выход требует активную сессию: без токена запрос до Accounts не доходит.
	mux.Handle("POST /api/auth/logout", httpx.BodyLimit(cfg.AuthBodyLimit())(authn(proxy)))

	mux.Handle("GET /api/users", authn(requireAdmin(proxy)))
	mux.Handle("POST /api/users", httpx.BodyLimit(cfg.UsersBodyLimit())(authn(requireAdmin(proxy))))
}
