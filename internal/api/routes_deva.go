package api

import (
	"log/slog"
	"net/http"
)

// registerDevA регистрирует маршруты Dev A: auth, methods, messages (чтение)
// и verify.
//
// В каркасе маршрутов пока нет: они появятся в COORD-7, COORD-6, COORD-8,
// COORD-12 и COORD-15.
func registerDevA(_ *http.ServeMux, _ Config, _ *slog.Logger) {}
