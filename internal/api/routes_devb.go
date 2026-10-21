package api

import (
	"log/slog"
	"net/http"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/config"
)

// registerDevB регистрирует маршруты зоны Dev B: запись сообщений,
// расшифрование и PDF.
//
// Файл принадлежит Dev B: Dev A его не меняет. На шаге каркаса маршрутов нет,
// они добавляются в COORD-10, COORD-13 и COORD-14.
func registerDevB(_ *http.ServeMux, _ *config.Config, _ *slog.Logger) {}
