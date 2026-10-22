package api

import (
	"log/slog"
	"net/http"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/config"
)

// registerDevB registers the Dev B routes: message write, decryption and PDF.
//
// The file belongs to Dev B and Dev A never edits it. The skeleton has no routes
// yet: they arrive in COORD-10, COORD-13 and COORD-14.
func registerDevB(_ *http.ServeMux, _ *config.Config, _ *slog.Logger) {}
