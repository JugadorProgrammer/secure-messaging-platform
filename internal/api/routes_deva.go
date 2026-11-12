package api

import (
	"log/slog"
	"net/http"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/auth"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/httpx"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/methods"
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

// registerDevA регистрирует маршруты Dev A: auth, methods, messages (чтение),
// журнал операций и verify.
//
// st — хранилище сервиса; nil означает, что работать с базой нечем: отказ в
// доступе тогда не записывается в журнал, а маршруты, которым нужна база
// (методы шифрования и GET /api/ops-log), не регистрируются.
func registerDevA(mux *http.ServeMux, cfg Config, st ports.Store, log *slog.Logger) {
	journal := opslog.New(st, log)
	gw := newAccountsGateway(cfg, journal, log)

	registerAccountsRoutes(mux, cfg, gw)
	registerMethodRoutes(mux, st, gw, journal, log)
	registerOpsLogRoutes(mux, gw, st, log)
}

// accountsGateway — прокси в Сервис учётных записей и цепочки проверки
// доступа. Поля равны nil, если ACCOUNTS_URL не настроен: без проверки токена
// защищённый маршрут открывать нельзя.
type accountsGateway struct {
	// proxy — обработчик прокси, срезающий /api. nil при отсутствии
	// ACCOUNTS_URL: без адреса учётных записей запросы до Accounts не доходят.
	proxy http.Handler
	// authn требует активную сессию любого пользователя.
	authn func(http.Handler) http.Handler
	// admin требует роль администратора.
	admin func(http.Handler) http.Handler
}

// newAccountsGateway собирает прокси и middleware проверки доступа.
//
// journal пишет отказ на административном маршруте (access.denied).
// Nil-журнал — хранилища нет — не пишет ничего, ответ от этого не меняется.
func newAccountsGateway(cfg Config, journal *opslog.Logger, log *slog.Logger) accountsGateway {
	base := cfg.AccountsBase()
	if base == nil {
		log.Error("accounts url is not configured: proxy and admin routes are not registered")

		return accountsGateway{}
	}

	client := auth.NewAccountsClient(base, cfg.AccountsCallTimeout(), log)
	authn := auth.Authn(client)
	requireAdmin := auth.Require(auth.RoleAdmin, journal.Log)

	return accountsGateway{
		proxy: http.StripPrefix(accountsPathPrefix, auth.NewAccountsProxy(base, log)),
		authn: authn,
		admin: func(next http.Handler) http.Handler {
			return authn(requireAdmin(next))
		},
	}
}

// registerAccountsRoutes проксирует вход, выход и управление пользователями.
func registerAccountsRoutes(mux *http.ServeMux, cfg Config, gw accountsGateway) {
	if gw.proxy == nil {
		return
	}

	for _, pattern := range accountsPublicRoutes {
		mux.Handle(pattern, httpx.BodyLimit(cfg.AuthBodyLimit())(gw.proxy))
	}

	// Выход требует активную сессию любого пользователя: без токена запрос
	// до Accounts не доходит.
	mux.Handle("POST /api/auth/logout", httpx.BodyLimit(cfg.AuthBodyLimit())(gw.authn(gw.proxy)))

	mux.Handle("GET /api/users", gw.admin(gw.proxy))
	mux.Handle("POST /api/users", httpx.BodyLimit(cfg.UsersBodyLimit())(gw.admin(gw.proxy)))
}

// registerMethodRoutes регистрирует список методов и его переключение.
//
// GET публичный и работает даже без настроенного Accounts: интерфейсу список
// нужен до входа. PATCH без проверки роли не регистрируется.
func registerMethodRoutes(mux *http.ServeMux, st ports.Store, gw accountsGateway, journal *opslog.Logger, log *slog.Logger) {
	if st == nil {
		log.Error("store is not configured: methods routes are not registered")

		return
	}

	h := methods.NewHandler(methods.NewService(st), journal.Log, log)

	mux.Handle("GET /api/methods", http.HandlerFunc(h.List))

	if gw.admin == nil {
		return
	}

	mux.Handle("PATCH /api/methods/{code}",
		httpx.BodyLimit(methods.MaxPatchBodyBytes)(gw.admin(http.HandlerFunc(h.Patch))))
}

// registerOpsLogRoutes открывает журнал администратору. Чтение возможно
// только при подключённом хранилище и настроенном Accounts: иначе пустой
// ответ выдавал бы «записей нет» там, где читать нечего.
func registerOpsLogRoutes(mux *http.ServeMux, gw accountsGateway, st ports.Store, log *slog.Logger) {
	if st == nil || gw.admin == nil {
		return
	}

	handler := opslog.NewHandler(st, log)
	mux.Handle("GET /api/ops-log", gw.admin(http.HandlerFunc(handler.List)))
}
