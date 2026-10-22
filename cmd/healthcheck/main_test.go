package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthURL(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    string
		wantErr bool
	}{
		{name: "port only", raw: ":8080", want: "http://127.0.0.1:8080/health"},
		{name: "empty means default", raw: "", want: "http://127.0.0.1:8080/health"},
		{name: "all interfaces", raw: "0.0.0.0:9090", want: "http://127.0.0.1:9090/health"},
		{name: "ipv6 any", raw: "[::]:8080", want: "http://127.0.0.1:8080/health"},
		{name: "explicit host", raw: "10.1.2.3:8080", want: "http://10.1.2.3:8080/health"},
		{name: "localhost", raw: "localhost:8080", want: "http://localhost:8080/health"},
		{name: "no port", raw: "localhost", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := healthURL(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("healthURL(%q) error = nil, want error", tt.raw)
				}

				return
			}
			if err != nil {
				t.Fatalf("healthURL(%q) error = %v", tt.raw, err)
			}
			if got != tt.want {
				t.Errorf("healthURL(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestRunAgainstHealthyServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != healthPath {
			t.Errorf("path = %q, want %q", r.URL.Path, healthPath)
		}

		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	t.Setenv("HTTP_ADDR", srv.Listener.Addr().String())

	if err := run(); err != nil {
		t.Fatalf("run() error = %v", err)
	}
}

func TestRunAgainstUnhealthyServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(srv.Close)

	t.Setenv("HTTP_ADDR", srv.Listener.Addr().String())

	if err := run(); err == nil {
		t.Fatal("run() error = nil, want error")
	}
}

func TestRunAgainstClosedServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	addr := srv.Listener.Addr().String()
	srv.Close()

	t.Setenv("HTTP_ADDR", addr)

	if err := run(); err == nil {
		t.Fatal("run() error = nil, want connection error")
	}
}

func TestRunWithBadAddr(t *testing.T) {
	t.Setenv("HTTP_ADDR", "not-an-address")

	if err := run(); err == nil {
		t.Fatal("run() error = nil, want error")
	}
}
