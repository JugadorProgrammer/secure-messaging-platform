package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealth(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		wantCode   int
		wantStatus string
	}{
		{name: "get", method: http.MethodGet, wantCode: http.StatusOK, wantStatus: "ok"},
		{name: "post not allowed", method: http.MethodPost, wantCode: http.StatusMethodNotAllowed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			newMux().ServeHTTP(rec, httptest.NewRequest(tt.method, "/api/health", http.NoBody))

			if rec.Code != tt.wantCode {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantCode)
			}
			if tt.wantStatus == "" {
				return
			}
			if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
				t.Fatalf("Content-Type = %q, want application/json", ct)
			}
			var got healthResponse
			if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if got.Status != tt.wantStatus {
				t.Fatalf("status field = %q, want %q", got.Status, tt.wantStatus)
			}
		})
	}
}
