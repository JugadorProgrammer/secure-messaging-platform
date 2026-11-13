// Command coord запускает координационный сервис.
package main

import (
	"context"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/api"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/config"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/cryptoclient"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/keys"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/ports"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/store"
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

	// Без схемы сервис работать не может: ошибка миграции останавливает запуск.
	st, err := store.NewStore(ctx, cfg.DSN, log)
	if err != nil {
		return err
	}
	defer st.Close()

	if err := st.Migrate(ctx, cfg.RSABits); err != nil {
		return err
	}

	crypto := cryptoclient.New(cfg.CryptoURL, cfg.CryptoTimeout, cfg.RSAKeygenTimeout, log)

	masterKeyHex := hex.EncodeToString(cfg.MasterKey[:])

	keyManager := keys.New(st, crypto, keys.Config{
		MasterKeyHex:  masterKeyHex,
		Bits:          cfg.RSABits,
		KeygenTimeout: cfg.RSAKeygenTimeout,
	}, log)
	// Close вызывается раньше st.Close (defer в обратном порядке): прерванная
	// генерация успевает записать статус failed и перезапустится при старте.
	defer keyManager.Close()

	// Системный ключ генерируется в фоне: EnsureUserKey не ждёт генерации,
	// поэтому /health отвечает сразу, а деплой не блокируется на десятки минут.
	if _, err := keyManager.EnsureUserKey(ctx, keys.SystemUserID); err != nil {
		return err
	}

	return serve(ctx, cfg, st, log, api.WithDevB(api.DevB{
		Crypto:          crypto,
		Keys:            keyManager,
		MasterKeyHex:    masterKeyHex,
		MaxMessageBytes: cfg.MaxMessageBytes,
	}))
}

// serve запускает HTTP-сервер и корректно останавливает его при отмене ctx.
// st передаётся в маршруты: часть из них работает только с хранилищем.
// opts подключают маршруты, которым нужны другие зависимости (api.WithDevB).
func serve(ctx context.Context, cfg *config.Config, st ports.Store, log *slog.Logger, opts ...api.Option) error {
	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           api.New(cfg, st, log, opts...),
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
