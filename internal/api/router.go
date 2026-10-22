// Package api собирает HTTP-маршруты сервиса.
package api

import (
	"log/slog"
	"net/http"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/httpx"
)

// Config — та часть конфигурации сервиса, которая нужна роутеру.
//
// Интерфейс объявлен у потребителя: api не должен зависеть от всего пакета
// config ради одного значения.
type Config interface {
	// CORSOrigins возвращает origin, разрешённые CORS.
	CORSOrigins() []string
}

// New собирает обработчик сервиса: маршруты, middleware и единый формат ошибок.
//
// Порядок middleware: RequestID → Logging → Recover → CORS → UniformErrors.
// Recover стоит выше обработчиков, поэтому паника превращается в 500, а Logging
// всё равно видит итоговый статус.
func New(cfg Config, log *slog.Logger) http.Handler {
	mux := http.NewServeMux()

	registerHealth(mux)
	registerDevA(mux, cfg, log)
	registerDevB(mux, cfg, log)

	var handler http.Handler = mux
	handler = httpx.UniformErrors(handler)
	handler = httpx.CORS(cfg.CORSOrigins())(handler)
	handler = httpx.Recover(log)(handler)
	handler = httpx.Logging(log)(handler)
	handler = httpx.RequestID(handler)

	return handler
}

// registerHealth регистрирует проверки доступности сервиса. Они не трогают
// ни базу, ни внешние сервисы.
func registerHealth(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", health)
	mux.HandleFunc("GET /api/health", health)
}

// healthResponse — тело ответа проверки доступности.
type healthResponse struct {
	Status string `json:"status" example:"ok"`
}

// health сообщает, что процесс запущен и принимает запросы.
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
