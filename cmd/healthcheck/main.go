// Command healthcheck проверяет координационный сервис по HTTP.
//
// В итоговом образе нет ни shell, ни curl, ни wget, поэтому HEALTHCHECK
// в Dockerfile вызывает этот бинарник.
package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"
)

const (
	defaultAddr    = ":8080"
	healthPath     = "/health"
	healthTimeout  = 3 * time.Second
	loopbackHost   = "127.0.0.1"
	unspecifiedAny = "0.0.0.0"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "healthcheck:", err)
		os.Exit(1)
	}
}

// run запрашивает /health и считает успехом только статус 200.
func run() error {
	target, err := healthURL(os.Getenv("HTTP_ADDR"))
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), healthTimeout)
	defer cancel()

	//nolint:gosec // G704: адрес задаёт оператор через HTTP_ADDR, а не запрос
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, http.NoBody)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req) //nolint:gosec // адрес задаёт оператор через HTTP_ADDR
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d from %s", resp.StatusCode, target)
	}

	return nil
}

// healthURL строит адрес проверки из HTTP_ADDR сервиса.
//
// Адрес вида ":8080" или "0.0.0.0:8080" означает прослушивание всех интерфейсов,
// поэтому проверка идёт на loopback.
func healthURL(addr string) (string, error) {
	target := addr
	if target == "" {
		target = defaultAddr
	}

	host, port, err := net.SplitHostPort(target)
	if err != nil {
		return "", fmt.Errorf("parse HTTP_ADDR %q: %w", target, err)
	}
	if port == "" {
		return "", fmt.Errorf("parse HTTP_ADDR %q: port is empty", target)
	}

	if host == "" || host == unspecifiedAny || host == "::" {
		host = loopbackHost
	}

	u := url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort(host, port),
		Path:   healthPath,
	}

	return u.String(), nil
}
