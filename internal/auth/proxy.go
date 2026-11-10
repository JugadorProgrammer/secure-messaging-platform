package auth

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/httpx"
)

// Параметры соединения с Сервисом учётных записей.
const (
	proxyDialTimeout           = 3 * time.Second
	proxyKeepAlive             = 30 * time.Second
	proxyIdleConnTimeout       = 90 * time.Second
	proxyResponseHeaderTimeout = 15 * time.Second
	proxyExpectContinueTimeout = 1 * time.Second

	proxyMaxIdleConns        = 100
	proxyMaxIdleConnsPerHost = 20

	msgBodyTooLarge = "Тело запроса слишком велико"
)

// identityHeaders — заголовки, которые клиенту передавать нельзя: адрес
// источника, «личность» пользователя и обходные указания прокси.
// Сервис учётных записей получает их только от этого сервиса.
var identityHeaders = []string{
	"X-Forwarded-For",
	"Forwarded",
	"X-Real-IP",
	"X-User-Id",
	"X-User-Role",
	"X-Original-URL",
	"X-Rewrite-URL",
	"Proxy",
}

// NewAccountsProxy строит reverse-proxy в Сервис учётных записей.
//
// Префикс /api срезает вызывающий (http.StripPrefix): прокси знает только
// адрес назначения. Стандартный ReverseProxy без ErrorHandler отвечает
// 502 текстом, поэтому здесь ошибка зависимости превращается в
// 503 accounts_unavailable в едином JSON-формате.
//
// FlushInterval остаётся значением по умолчанию: стриминга в этих маршрутах нет.
func NewAccountsProxy(target *url.URL, logger *slog.Logger) *httputil.ReverseProxy {
	if logger == nil {
		logger = slog.Default()
	}

	return &httputil.ReverseProxy{
		Rewrite:      rewriteAccountsRequest(target),
		Transport:    newAccountsTransport(),
		ErrorHandler: accountsErrorHandler(logger),
		ErrorLog:     accountsErrorLog(logger),
	}
}

// accountsErrorLog переводит сообщения ReverseProxy о внутренних ошибках
// в slog: без этого они уходили бы в стандартный log текстом без уровня.
func accountsErrorLog(logger *slog.Logger) *log.Logger {
	return log.New(slogWriter{logger: logger}, "accounts proxy: ", 0)
}

// slogWriter пишет строки стандартного log в slog с уровнем error.
type slogWriter struct {
	logger *slog.Logger
}

func (w slogWriter) Write(p []byte) (int, error) {
	if msg := strings.TrimSpace(string(p)); msg != "" {
		w.logger.Error(msg)
	}

	return len(p), nil
}

// rewriteAccountsRequest переписывает запрос на целевой сервис и приводит
// заголовки к безопасному виду.
func rewriteAccountsRequest(target *url.URL) func(*httputil.ProxyRequest) {
	return func(pr *httputil.ProxyRequest) {
		pr.Out.URL.Scheme = target.Scheme
		pr.Out.URL.Host = target.Host
		pr.Out.Host = target.Host

		// Сначала снимаем клиентские заголовки, затем ставим свои: SetXForwarded
		// дополняет X-Forwarded-For, и подделанное значение попало бы в цепочку.
		stripIdentityHeaders(pr.Out.Header)
		pr.SetXForwarded()

		if id := httpx.RequestIDFrom(pr.In.Context()); id != "" {
			pr.Out.Header.Set(httpx.HeaderRequestID, id)
		}
	}
}

// stripIdentityHeaders удаляет заголовки, которые не должны доходить до
// Сервиса учётных записей от клиента. Authorization не удаляется: вход и
// выход проверяются на той стороне.
func stripIdentityHeaders(h http.Header) {
	for _, name := range identityHeaders {
		h.Del(name)
	}
}

// accountsErrorHandler отдаёт единый JSON об недоступности зависимости.
func accountsErrorHandler(logger *slog.Logger) func(http.ResponseWriter, *http.Request, error) {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		if errors.Is(err, context.Canceled) {
			// Клиент ушёл до ответа: писать некуда.
			return
		}

		var maxBytes *http.MaxBytesError
		if errors.As(err, &maxBytes) {
			httpx.WriteErr(w, http.StatusRequestEntityTooLarge, httpx.CodePayloadTooLarge, msgBodyTooLarge)

			return
		}

		logger.ErrorContext(r.Context(), "accounts proxy error", "err", err)
		httpx.WriteErr(w, http.StatusServiceUnavailable, httpx.CodeAccountsUnavailable, msgAccountsUnavailable)
	}
}

// newAccountsTransport строит транспорт с таймаутами, которые не позволяют
// зависшему сервису удерживать соединения клиента.
func newAccountsTransport() *http.Transport {
	return &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   proxyDialTimeout,
			KeepAlive: proxyKeepAlive,
		}).DialContext,
		MaxIdleConns:          proxyMaxIdleConns,
		MaxIdleConnsPerHost:   proxyMaxIdleConnsPerHost,
		IdleConnTimeout:       proxyIdleConnTimeout,
		ResponseHeaderTimeout: proxyResponseHeaderTimeout,
		ExpectContinueTimeout: proxyExpectContinueTimeout,
	}
}
