package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/config"
)

func testConfig() *config.Config {
	return &config.Config{
		HTTPAddr:              "127.0.0.1:0",
		HTTPReadHeaderTimeout: time.Second,
		HTTPReadTimeout:       time.Second,
		HTTPWriteTimeout:      time.Second,
		HTTPIdleTimeout:       time.Second,
		ShutdownTimeout:       2 * time.Second,
	}
}

func TestClosedAsNil(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantNil bool
	}{
		{name: "nil", err: nil, wantNil: true},
		{name: "server closed", err: http.ErrServerClosed, wantNil: true},
		{name: "other error", err: errors.New("boom"), wantNil: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := closedAsNil(tt.err)
			if tt.wantNil && got != nil {
				t.Fatalf("closedAsNil(%v) = %v, want nil", tt.err, got)
			}
			if !tt.wantNil && got == nil {
				t.Fatalf("closedAsNil(%v) = nil, want error", tt.err)
			}
		})
	}
}

func TestServeStopsOnContextCancel(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- serve(ctx, testConfig(), log)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("serve() error = %v, want nil", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("serve() did not stop after context cancel")
	}
}
