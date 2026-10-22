// Package api wires the service HTTP routes.
package api

import (
	"log/slog"
	"net/http"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/config"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/httpx"
)

// New builds the service handler: routes, middleware and the uniform error format.
//
// Middleware order: RequestID → Logging → Recover → CORS → UniformErrors.
// Recover sits above the handlers, so a panic becomes a 500 while Logging still
// sees the final status.
func New(cfg *config.Config, log *slog.Logger) http.Handler {
	mux := http.NewServeMux()

	registerHealth(mux)
	registerDevA(mux, cfg, log)
	registerDevB(mux, cfg, log)

	var handler http.Handler = mux
	handler = httpx.UniformErrors(handler)
	handler = httpx.CORS(cfg.CORSOrigins)(handler)
	handler = httpx.Recover(log)(handler)
	handler = httpx.Logging(log)(handler)
	handler = httpx.RequestID(handler)

	return handler
}

// registerHealth registers the service availability checks. They touch neither
// the database nor external services.
func registerHealth(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", health)
	mux.HandleFunc("GET /api/health", health)
}

// healthResponse is the availability check response body.
type healthResponse struct {
	Status string `json:"status" example:"ok"`
}

// health reports that the process is running and accepting requests.
//
// @Summary  Проверка доступности сервиса
// @Tags     service
// @Produce  json
// @Success  200 {object} healthResponse
// @Router   /health [get]
// @Router   /api/health [get]
func health(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, healthResponse{Status: "ok"})
}
