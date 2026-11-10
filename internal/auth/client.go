package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// introspectPath — путь эндпоинта проверки токена.
	introspectPath = "/internal/introspect"

	// stageFull — обязательное значение stage: сессия прошла все факторы.
	// Сессия с другим stage или без него не даёт доступа, иначе 3FA обходится.
	stageFull = "full"

	// maxIntrospectBody — лимит ответа introspect. Больше — считаем ответ битым.
	maxIntrospectBody = 64 << 10

	contentTypeJSON = "application/json"
)

// AccountsClient — клиент эндпоинта introspect Сервиса учётных записей.
//
// Редиректы не отслеживаются, повторов нет: ответ introspect зависит от
// одноразового состояния сессии, и повторный запрос не улучшает результат.
type AccountsClient struct {
	base   *url.URL
	client *http.Client
	log    *slog.Logger
}

// introspectRequest — тело запроса introspect.
type introspectRequest struct {
	Token string `json:"token"`
}

// introspectResponse — ответ introspect. Поля обязательны: отсутствие stage
// означает неполную сессию.
type introspectResponse struct {
	Active bool   `json:"active"`
	Stage  string `json:"stage"`
	UserID int64  `json:"user_id"`
	Login  string `json:"login"`
	Role   string `json:"role"`
}

// NewAccountsClient строит клиента для сервиса по адресу base с таймаутом
// на весь вызов.
func NewAccountsClient(base *url.URL, timeout time.Duration, log *slog.Logger) *AccountsClient {
	if log == nil {
		log = slog.Default()
	}

	return &AccountsClient{
		base: base,
		client: &http.Client{
			Timeout: timeout,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		log: log,
	}
}

// Introspect проверяет токен и возвращает пользователя полной сессии.
//
// Ошибки различимы вызывающим: ErrUnauthorized — токен не принят,
// ErrAccountsUnavailable — зависимость недоступна. Отмена контекста
// возвращается как есть, чтобы middleware не писал ответ ушедшему клиенту.
func (c *AccountsClient) Introspect(ctx context.Context, token string) (Principal, error) {
	if c.base == nil {
		return Principal{}, fmt.Errorf("%w: accounts url is not configured", ErrAccountsUnavailable)
	}

	payload, err := json.Marshal(introspectRequest{Token: token})
	if err != nil {
		return Principal{}, fmt.Errorf("auth: encode introspect request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.introspectURL(), bytes.NewReader(payload))
	if err != nil {
		return Principal{}, fmt.Errorf("auth: build introspect request: %w", err)
	}
	req.Header.Set("Content-Type", contentTypeJSON)
	req.Header.Set("Accept", contentTypeJSON)

	resp, err := c.client.Do(req)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return Principal{}, err
		}
		c.log.ErrorContext(ctx, "accounts introspect request failed", "err", err)

		return Principal{}, ErrAccountsUnavailable
	}

	defer drainAndClose(resp)

	return c.principalFromResponse(ctx, resp)
}

// principalFromResponse разбирает ответ по коду состояния: 401 и 403 означают
// недействительный токен, всё остальное кроме 200 — недоступность сервиса.
func (c *AccountsClient) principalFromResponse(ctx context.Context, resp *http.Response) (Principal, error) {
	switch resp.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return Principal{}, ErrUnauthorized
	case http.StatusOK:
		// Разбор ответа ниже.
	default:
		c.log.ErrorContext(ctx, "accounts introspect unexpected status", "status", resp.StatusCode)

		return Principal{}, ErrAccountsUnavailable
	}

	var ir introspectResponse
	if err := decodeLimitedJSON(resp.Body, &ir); err != nil {
		c.log.ErrorContext(ctx, "accounts introspect unreadable response", "err", err)

		return Principal{}, ErrAccountsUnavailable
	}

	return ir.principal()
}

// principal проверяет ответ по контракту: активная сессия, полный набор
// факторов, осмысленные пользователь и роль.
func (ir introspectResponse) principal() (Principal, error) {
	if !ir.Active || ir.Stage != stageFull {
		return Principal{}, ErrUnauthorized
	}

	role := Role(ir.Role)
	if ir.UserID <= 0 || ir.Login == "" || (role != RoleUser && role != RoleAdmin) {
		return Principal{}, ErrUnauthorized
	}

	return Principal{UserID: ir.UserID, Login: ir.Login, Role: role}, nil
}

// introspectURL собирает адрес эндпоинта, сохраняя путь из ACCOUNTS_URL.
func (c *AccountsClient) introspectURL() string {
	u := *c.base
	u.Path = strings.TrimSuffix(u.Path, "/") + introspectPath
	u.RawQuery = ""
	u.Fragment = ""

	return u.String()
}

// decodeLimitedJSON читает не более maxIntrospectBody байт и разбирает JSON.
// Обрезанный или битый ответ даёт ошибку — вызывающий считает её
// недоступностью сервиса.
func decodeLimitedJSON(r io.Reader, v any) error {
	b, err := io.ReadAll(io.LimitReader(r, maxIntrospectBody))
	if err != nil {
		return fmt.Errorf("auth: read response: %w", err)
	}

	if err := json.Unmarshal(b, v); err != nil {
		return fmt.Errorf("auth: decode response: %w", err)
	}

	return nil
}

// drainAndClose освобождает соединение: дочитывает остаток тела и закрывает его.
func drainAndClose(resp *http.Response) {
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxIntrospectBody))

	_ = resp.Body.Close()
}
