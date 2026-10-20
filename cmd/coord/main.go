// Command coord runs the coordination service.
package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"time"
)

// @title       COORD API
// @description Координационный сервис secure-messaging-platform.
// @BasePath    /
func main() {
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           newMux(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	slog.Info("listening", "addr", addr)
	if err := srv.ListenAndServe(); err != nil {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

func newMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", health)
	return mux
}

type healthResponse struct {
	Status string `json:"status" example:"ok"`
}

// health reports that the process is up.
//
// @Summary  Проверка доступности сервиса
// @Tags     service
// @Produce  json
// @Success  200 {object} healthResponse
// @Router   /api/health [get]
func health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(healthResponse{Status: "ok"}); err != nil {
		slog.Error("write health response", "err", err)
	}
}
