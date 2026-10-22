package api

import (
	"log/slog"
	"net/http"
)

// registerDevB регистрирует маршруты Dev B: запись сообщений, дешифрование и PDF.
//
// Файл принадлежит Dev B, и Dev A его не правит. В каркасе маршрутов пока нет:
// они появятся в COORD-10, COORD-13 и COORD-14.
func registerDevB(_ *http.ServeMux, _ Config, _ *slog.Logger) {}
