package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
}

func TestCORS(t *testing.T) {
	tests := []struct {
		name             string
		origins          []string
		method           string
		origin           string
		requestMethod    string
		wantStatus       int
		wantAllowOrigin  string
		wantVary         string
		wantAllowMethods string
		wantAllowHeaders string
		wantEmptyBody    bool
	}{
		{
			name:       "CORS выключен",
			origins:    nil,
			method:     http.MethodGet,
			origin:     "http://localhost:5173",
			wantStatus: http.StatusOK,
		},
		{
			name:             "разрешённый origin",
			origins:          []string{"http://localhost:5173"},
			method:           http.MethodGet,
			origin:           "http://localhost:5173",
			wantStatus:       http.StatusOK,
			wantAllowOrigin:  "http://localhost:5173",
			wantVary:         "Origin",
			wantAllowMethods: corsAllowedMethods,
			wantAllowHeaders: corsAllowedHeaders,
		},
		{
			name:       "чужой origin",
			origins:    []string{"http://localhost:5173"},
			method:     http.MethodGet,
			origin:     "http://evil.example",
			wantStatus: http.StatusOK,
		},
		{
			name:             "wildcard разрешает любой origin",
			origins:          []string{"*"},
			method:           http.MethodGet,
			origin:           "http://any.example",
			wantStatus:       http.StatusOK,
			wantAllowOrigin:  "*",
			wantVary:         "Origin",
			wantAllowMethods: corsAllowedMethods,
			wantAllowHeaders: corsAllowedHeaders,
		},
		{
			name:             "preflight OPTIONS",
			origins:          []string{"http://localhost:5173"},
			method:           http.MethodOptions,
			origin:           "http://localhost:5173",
			requestMethod:    http.MethodPost,
			wantStatus:       http.StatusNoContent,
			wantAllowOrigin:  "http://localhost:5173",
			wantVary:         "Origin",
			wantAllowMethods: corsAllowedMethods,
			wantAllowHeaders: corsAllowedHeaders,
			wantEmptyBody:    true,
		},
		{
			name:       "OPTIONS без Access-Control-Request-Method",
			origins:    []string{"http://localhost:5173"},
			method:     http.MethodOptions,
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := CORS(tt.origins)(okHandler())

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, "/api/messages", http.NoBody)
			if tt.origin != "" {
				req.Header.Set(headerOrigin, tt.origin)
			}
			if tt.requestMethod != "" {
				req.Header.Set(headerRequestMethod, tt.requestMethod)
			}

			h.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if got := rec.Header().Get(headerAllowOrigin); got != tt.wantAllowOrigin {
				t.Errorf("Allow-Origin = %q, want %q", got, tt.wantAllowOrigin)
			}
			if got := rec.Header().Get(headerVary); got != tt.wantVary {
				t.Errorf("Vary = %q, want %q", got, tt.wantVary)
			}
			if got := rec.Header().Get(headerAllowMethods); got != tt.wantAllowMethods {
				t.Errorf("Allow-Methods = %q, want %q", got, tt.wantAllowMethods)
			}
			if got := rec.Header().Get(headerAllowHeaders); got != tt.wantAllowHeaders {
				t.Errorf("Allow-Headers = %q, want %q", got, tt.wantAllowHeaders)
			}
			if tt.wantEmptyBody && rec.Body.Len() != 0 {
				t.Errorf("body = %q, want empty", rec.Body.String())
			}
		})
	}
}
