package api

import (
	"log/slog"
	"net/http"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/config"
)

// registerDevA регистрирует маршруты зоны Dev A: auth, methods,
// messages (чтение) и verify.
//
// На шаге каркаса маршрутов нет: они добавляются в COORD-7, COORD-6,
// COORD-8, COORD-12 и COORD-15.
func registerDevA(_ *http.ServeMux, _ *config.Config, _ *slog.Logger) {}
