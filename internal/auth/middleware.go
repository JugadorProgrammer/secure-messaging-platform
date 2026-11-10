package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/httpx"
)

const (
	// headerAuthorization — заголовок с Bearer-токеном.
	headerAuthorization = "Authorization"

	// maxTokenLen — предельная длина токена: больше — точно не наш формат.
	maxTokenLen = 4096

	// maxAuditRefLen — предельная длина ref в записи журнала операций.
	maxAuditRefLen = 200

	msgUnauthorized        = "Требуется вход"
	msgInvalidSession      = "Сессия недействительна"
	msgForbidden           = "Недостаточно прав"
	msgAccountsUnavailable = "Сервис учётных записей недоступен"

	opAccessDenied = "access.denied"
)

// AuditFunc — запись в журнал операций (совпадает с store.LogOp).
// userID == nil означает анонимного субъекта, а не нулевой идентификатор.
type AuditFunc func(ctx context.Context, userID *int64, op, ref string, ok bool)

// Authn проверяет Bearer-токен через introspect и кладёт Principal в контекст.
//
// Ответы различаются: отсутствие или неверный формат токена и недействительная
// сессия дают 401, недоступность Сервиса учётных записей — 503. Токен в лог
// не попадает.
func Authn(c *AccountsClient) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := bearerToken(r)
			if !ok {
				httpx.WriteErr(w, http.StatusUnauthorized, httpx.CodeUnauthorized, msgUnauthorized)

				return
			}

			principal, err := c.Introspect(r.Context(), token)
			if errors.Is(err, context.Canceled) {
				// Клиент ушёл, писать ответ некуда.
				return
			}

			switch {
			case errors.Is(err, ErrUnauthorized):
				httpx.WriteErr(w, http.StatusUnauthorized, httpx.CodeUnauthorized, msgInvalidSession)
			case err != nil:
				httpx.WriteErr(w, http.StatusServiceUnavailable, httpx.CodeAccountsUnavailable, msgAccountsUnavailable)
			default:
				next.ServeHTTP(w, r.WithContext(WithPrincipal(r.Context(), principal)))
			}
		})
	}
}

// Require пропускает запрос только с ролью role. Отказ попадает в журнал
// best-effort: сбой записи не меняет ответ клиента.
func Require(role Role, audit AuditFunc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, ok := PrincipalFrom(r.Context())
			if !ok {
				httpx.WriteErr(w, http.StatusUnauthorized, httpx.CodeUnauthorized, msgUnauthorized)

				return
			}

			if principal.Role != role {
				logDenied(r.Context(), audit, principal, r)
				httpx.WriteErr(w, http.StatusForbidden, httpx.CodeForbidden, msgForbidden)

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// bearerToken извлекает токен: ровно один заголовок, схема Bearer без учёта
// регистра, непустой токен разумной длины.
func bearerToken(r *http.Request) (string, bool) {
	values := r.Header.Values(headerAuthorization)
	if len(values) != 1 {
		return "", false
	}

	scheme, token, found := strings.Cut(values[0], " ")
	if !found || !strings.EqualFold(scheme, "Bearer") {
		return "", false
	}

	token = strings.TrimSpace(token)
	if token == "" || len(token) > maxTokenLen || strings.ContainsAny(token, " \t") {
		return "", false
	}

	return token, true
}

// logDenied пишет отказ в доступе в журнал. Контекст запроса к этому моменту
// может отменяться, поэтому запись уходит без привязки к нему.
func logDenied(ctx context.Context, audit AuditFunc, p Principal, r *http.Request) {
	if audit == nil {
		return
	}

	ref := r.Method + " " + r.URL.Path
	if len(ref) > maxAuditRefLen {
		ref = ref[:maxAuditRefLen]
	}

	userID := p.UserID
	audit(context.WithoutCancel(ctx), &userID, opAccessDenied, ref, false)
}
