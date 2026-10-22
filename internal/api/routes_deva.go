package api

import (
	"log/slog"
	"net/http"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/config"
)

// registerDevA registers the Dev A routes: auth, methods, messages (read) and verify.
//
// The skeleton has no routes yet: they arrive in COORD-7, COORD-6, COORD-8,
// COORD-12 and COORD-15.
func registerDevA(_ *http.ServeMux, _ *config.Config, _ *slog.Logger) {}
