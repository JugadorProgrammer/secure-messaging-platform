// Package auth проверяет токены Сервиса учётных записей, пробрасывает
// аутентифицированного пользователя в контекст запроса и проксирует к этому
// сервису маршруты входа, регистрации и управления пользователями.
//
// Типы Principal и Role объявлены здесь и позже переедут в internal/ports,
// когда пакет контрактов появится в ветке разработки.
package auth

import (
	"context"
	"errors"
)

// Sentinel-ошибки. Вызываются реализациями, проверяются через errors.Is.
var (
	// ErrUnauthorized — токен недействителен, сессия неполная или роль неизвестна.
	// Хендлер отображает её в 401.
	ErrUnauthorized = errors.New("auth: unauthorized")

	// ErrAccountsUnavailable — Сервис учётных записей недоступен: сеть, таймаут,
	// 5xx или неразбираемый ответ. Хендлер отображает её в 503, а не в 401:
	// недоступность зависимости не означает, что пользователь не авторизован.
	ErrAccountsUnavailable = errors.New("auth: accounts service unavailable")
)

// Role — роль пользователя из ответа Сервиса учётных записей.
type Role string

// Роли пользователей.
const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

// Principal — аутентифицированный пользователь запроса.
//
// Значения берутся только из ответа introspect; заголовкам клиента о себе
// (X-User-Id, X-User-Role) сервис не доверяет и удаляет их при проксировании.
type Principal struct {
	UserID int64
	Login  string
	Role   Role
}

// principalKey — типизированный ключ контекста. Пустой struct не конфликтует
// с ключами других пакетов.
type principalKey struct{}

// WithPrincipal кладёт пользователя в контекст запроса.
func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}

// PrincipalFrom возвращает пользователя из контекста. Второй результат — false,
// если Authn не выполнялся.
func PrincipalFrom(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalKey{}).(Principal)

	return p, ok
}
