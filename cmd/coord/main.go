// Command coord запускает координационный сервис.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/api"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/config"
)

// @title       COORD API
// @version     0.1.0
// @description Координационный сервис secure-messaging-platform.
// @BasePath    /
func main() {
	if err := run(); err != nil {
		slog.Error("coord stopped", "err", err)
		os.Exit(1)
	}
}

// run читает конфигурацию и работает до прихода сигнала остановки.
func run() error {
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		return err
	}

	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(log)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	return serve(ctx, cfg, log)
}

// serve запускает HTTP-сервер и корректно останавливает его при отмене ctx.
func serve(ctx context.Context, cfg *config.Config, log *slog.Logger) error {
	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           api.New(cfg, log),
		ReadHeaderTimeout: cfg.HTTPReadHeaderTimeout,
		ReadTimeout:       cfg.HTTPReadTimeout,
		WriteTimeout:      cfg.HTTPWriteTimeout,
		IdleTimeout:       cfg.HTTPIdleTimeout,
	}

	errCh := make(chan error, 1)

	go func() {
		errCh <- srv.ListenAndServe()
	}()

	log.Info("coord started", "config", cfg)

	select {
	case err := <-errCh:
		return closedAsNil(err)
	case <-ctx.Done():
	}

	log.Info("coord stopping", "timeout", cfg.ShutdownTimeout)

	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}

	return closedAsNil(<-errCh)
}

// closedAsNil считает штатную остановку сервера успехом.
func closedAsNil(err error) error {
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}

	return err
}
