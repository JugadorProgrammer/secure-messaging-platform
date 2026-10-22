// Command healthcheck probes the coordination service over HTTP.
//
// The final image has no shell, curl or wget, so the Dockerfile HEALTHCHECK
// calls this binary instead.
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

// run requests /health and succeeds only on a 200 status.
func run() error {
	target, err := healthURL(os.Getenv("HTTP_ADDR"))
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), healthTimeout)
	defer cancel()

	//nolint:gosec // G704: the address is set by the operator through HTTP_ADDR, not by a request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, http.NoBody)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req) //nolint:gosec // the address is set by the operator through HTTP_ADDR
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

// healthURL builds the probe address from the service HTTP_ADDR.
//
// An address like ":8080" or "0.0.0.0:8080" means listening on all interfaces,
// so the probe goes to loopback.
func healthURL(raw string) (string, error) {
	if raw == "" {
		raw = defaultAddr
	}

	host, port, err := net.SplitHostPort(raw)
	if err != nil {
		return "", fmt.Errorf("parse HTTP_ADDR %q: %w", raw, err)
	}
	if port == "" {
		return "", fmt.Errorf("parse HTTP_ADDR %q: port is empty", raw)
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
