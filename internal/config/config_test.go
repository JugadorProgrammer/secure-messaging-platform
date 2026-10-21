package config

import (
	"log/slog"
	"strings"
	"testing"
	"time"
)

const testMasterKey = "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"

// baseEnv — минимальный набор обязательных переменных.
func baseEnv() map[string]string {
	return map[string]string{
		"HTTP_ADDR":    ":8080",
		"DSN":          "postgres://coord:coord_local@localhost:5432/coord?sslmode=disable",
		"MASTER_KEY":   testMasterKey,
		"ACCOUNTS_URL": "http://localhost:8081",
		"CRYPTO_URL":   "http://localhost:8082",
	}
}

func getenvFrom(env map[string]string) func(string) string {
	return func(name string) string {
		return env[name]
	}
}

func TestLoadFull(t *testing.T) {
	env := baseEnv()
	env["RSA_BITS"] = "4096"
	env["CORS_ORIGINS"] = "http://localhost:5173, https://example.org"
	env["LOG_LEVEL"] = "debug"
	env["HTTP_READ_HEADER_TIMEOUT"] = "1s"
	env["HTTP_READ_TIMEOUT"] = "2s"
	env["HTTP_WRITE_TIMEOUT"] = "3s"
	env["HTTP_IDLE_TIMEOUT"] = "4s"
	env["SHUTDOWN_TIMEOUT"] = "5s"
	env["ACCOUNTS_TIMEOUT"] = "6s"
	env["CRYPTO_TIMEOUT"] = "7s"
	env["RSA_KEYGEN_TIMEOUT"] = "90m"
	env["MAX_BODY_AUTH"] = "2MiB"
	env["MAX_BODY_USERS"] = "32KiB"
	env["MAX_MESSAGE_BYTES"] = "128KiB"
	env["MAX_PDF_UPLOAD"] = "20MiB"

	cfg, err := Load(getenvFrom(env))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	checks := []struct {
		name string
		got  any
		want any
	}{
		{"HTTPAddr", cfg.HTTPAddr, ":8080"},
		{"DSN", cfg.DSN, env["DSN"]},
		{"MasterKeyHex", cfg.MasterKeyHex(), testMasterKey},
		{"AccountsURL", cfg.AccountsURL.String(), "http://localhost:8081"},
		{"CryptoURL", cfg.CryptoURL.String(), "http://localhost:8082"},
		{"RSABits", cfg.RSABits, 4096},
		{"LogLevel", cfg.LogLevel, slog.LevelDebug},
		{"HTTPReadHeaderTimeout", cfg.HTTPReadHeaderTimeout, time.Second},
		{"HTTPReadTimeout", cfg.HTTPReadTimeout, 2 * time.Second},
		{"HTTPWriteTimeout", cfg.HTTPWriteTimeout, 3 * time.Second},
		{"HTTPIdleTimeout", cfg.HTTPIdleTimeout, 4 * time.Second},
		{"ShutdownTimeout", cfg.ShutdownTimeout, 5 * time.Second},
		{"AccountsTimeout", cfg.AccountsTimeout, 6 * time.Second},
		{"CryptoTimeout", cfg.CryptoTimeout, 7 * time.Second},
		{"RSAKeygenTimeout", cfg.RSAKeygenTimeout, 90 * time.Minute},
		{"MaxBodyAuth", cfg.MaxBodyAuth, int64(2 << 20)},
		{"MaxBodyUsers", cfg.MaxBodyUsers, int64(32 << 10)},
		{"MaxMessageBytes", cfg.MaxMessageBytes, int64(128 << 10)},
		{"MaxPDFUpload", cfg.MaxPDFUpload, int64(20 << 20)},
	}

	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}

	if len(cfg.CORSOrigins) != 2 || cfg.CORSOrigins[0] != "http://localhost:5173" ||
		cfg.CORSOrigins[1] != "https://example.org" {
		t.Errorf("CORSOrigins = %v", cfg.CORSOrigins)
	}
}

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load(getenvFrom(baseEnv()))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	checks := []struct {
		name string
		got  any
		want any
	}{
		{"RSABits", cfg.RSABits, defaultRSABits},
		{"CORSOrigins", len(cfg.CORSOrigins), 0},
		{"LogLevel", cfg.LogLevel, slog.LevelInfo},
		{"HTTPReadHeaderTimeout", cfg.HTTPReadHeaderTimeout, 5 * time.Second},
		{"HTTPReadTimeout", cfg.HTTPReadTimeout, 15 * time.Second},
		{"HTTPWriteTimeout", cfg.HTTPWriteTimeout, 60 * time.Second},
		{"HTTPIdleTimeout", cfg.HTTPIdleTimeout, 60 * time.Second},
		{"ShutdownTimeout", cfg.ShutdownTimeout, 15 * time.Second},
		{"AccountsTimeout", cfg.AccountsTimeout, 3 * time.Second},
		{"CryptoTimeout", cfg.CryptoTimeout, 10 * time.Second},
		{"RSAKeygenTimeout", cfg.RSAKeygenTimeout, 60 * time.Minute},
		{"MaxBodyAuth", cfg.MaxBodyAuth, int64(1 << 20)},
		{"MaxBodyUsers", cfg.MaxBodyUsers, int64(64 << 10)},
		{"MaxMessageBytes", cfg.MaxMessageBytes, int64(64 << 10)},
		{"MaxPDFUpload", cfg.MaxPDFUpload, int64(10 << 20)},
	}

	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}
}

func TestLoadErrors(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(map[string]string)
		wantErr string
	}{
		{
			name:    "no HTTP_ADDR",
			mutate:  func(env map[string]string) { delete(env, "HTTP_ADDR") },
			wantErr: "HTTP_ADDR is required",
		},
		{
			name:    "no DSN",
			mutate:  func(env map[string]string) { delete(env, "DSN") },
			wantErr: "DSN is required",
		},
		{
			name:    "no MASTER_KEY",
			mutate:  func(env map[string]string) { delete(env, "MASTER_KEY") },
			wantErr: "MASTER_KEY is required",
		},
		{
			name:    "no ACCOUNTS_URL",
			mutate:  func(env map[string]string) { delete(env, "ACCOUNTS_URL") },
			wantErr: "ACCOUNTS_URL is required",
		},
		{
			name:    "no CRYPTO_URL",
			mutate:  func(env map[string]string) { delete(env, "CRYPTO_URL") },
			wantErr: "CRYPTO_URL is required",
		},
		{
			name:    "MASTER_KEY not hex",
			mutate:  func(env map[string]string) { env["MASTER_KEY"] = strings.Repeat("zz", 32) },
			wantErr: "MASTER_KEY must be hex encoded",
		},
		{
			name:    "MASTER_KEY too short",
			mutate:  func(env map[string]string) { env["MASTER_KEY"] = "00112233" },
			wantErr: "MASTER_KEY must be 64 hex characters",
		},
		{
			name:    "RSA_BITS not a number",
			mutate:  func(env map[string]string) { env["RSA_BITS"] = "big" },
			wantErr: "RSA_BITS must be an integer",
		},
		{
			name:    "RSA_BITS too small",
			mutate:  func(env map[string]string) { env["RSA_BITS"] = "1024" },
			wantErr: "RSA_BITS must be at least 2048",
		},
		{
			name:    "RSA_BITS not multiple of 8",
			mutate:  func(env map[string]string) { env["RSA_BITS"] = "2050" },
			wantErr: "RSA_BITS must be at least 2048",
		},
		{
			name:    "bad ACCOUNTS_URL scheme",
			mutate:  func(env map[string]string) { env["ACCOUNTS_URL"] = "localhost:8081" },
			wantErr: "ACCOUNTS_URL must use http or https scheme",
		},
		{
			name:    "ACCOUNTS_URL without host",
			mutate:  func(env map[string]string) { env["ACCOUNTS_URL"] = "http://" },
			wantErr: "ACCOUNTS_URL must contain a host",
		},
		{
			name:    "bad LOG_LEVEL",
			mutate:  func(env map[string]string) { env["LOG_LEVEL"] = "trace" },
			wantErr: "LOG_LEVEL must be one of debug, info, warn, error",
		},
		{
			name:    "bad duration",
			mutate:  func(env map[string]string) { env["HTTP_READ_TIMEOUT"] = "soon" },
			wantErr: "HTTP_READ_TIMEOUT is not a valid duration",
		},
		{
			name:    "negative duration",
			mutate:  func(env map[string]string) { env["SHUTDOWN_TIMEOUT"] = "-1s" },
			wantErr: "SHUTDOWN_TIMEOUT must be positive",
		},
		{
			name:    "bad size",
			mutate:  func(env map[string]string) { env["MAX_BODY_AUTH"] = "1X" },
			wantErr: "MAX_BODY_AUTH: invalid size",
		},
		{
			name:    "zero size",
			mutate:  func(env map[string]string) { env["MAX_PDF_UPLOAD"] = "0" },
			wantErr: "MAX_PDF_UPLOAD must be positive",
		},
		{
			name:    "bad origin",
			mutate:  func(env map[string]string) { env["CORS_ORIGINS"] = "localhost" },
			wantErr: "CORS_ORIGINS contains invalid origin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := baseEnv()
			tt.mutate(env)

			_, err := Load(getenvFrom(env))
			if err == nil {
				t.Fatal("Load() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("Load() error = %v, want substring %q", err, tt.wantErr)
			}
		})
	}
}

func TestLoadReportsAllErrors(t *testing.T) {
	env := map[string]string{
		"RSA_BITS":  "1024",
		"LOG_LEVEL": "trace",
	}

	_, err := Load(getenvFrom(env))
	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}

	for _, want := range []string{"HTTP_ADDR", "DSN", "MASTER_KEY", "ACCOUNTS_URL", "CRYPTO_URL", "RSA_BITS", "LOG_LEVEL"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("Load() error = %v, want substring %q", err, want)
		}
	}
}

func TestLoadNilGetenv(t *testing.T) {
	if _, err := Load(nil); err == nil {
		t.Fatal("Load(nil) error = nil, want error")
	}
}

func TestConfigDoesNotLeakSecrets(t *testing.T) {
	env := baseEnv()
	env["DSN"] = "postgres://user:secret-password@localhost:5432/coord"

	cfg, err := Load(getenvFrom(env))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	for _, s := range []string{cfg.String(), cfg.LogValue().String()} {
		if strings.Contains(s, testMasterKey) {
			t.Errorf("config representation leaks master key: %s", s)
		}
		if strings.Contains(s, "secret-password") {
			t.Errorf("config representation leaks DSN password: %s", s)
		}
		if !strings.Contains(s, redacted) {
			t.Errorf("config representation has no redaction marker: %s", s)
		}
	}
}

func TestConfigNilRepresentations(t *testing.T) {
	var cfg *Config

	if got := cfg.String(); got != "config(nil)" {
		t.Errorf("String() = %q", got)
	}
	if got := cfg.LogValue().String(); got != "config(nil)" {
		t.Errorf("LogValue() = %q", got)
	}
}

func TestParseSize(t *testing.T) {
	tests := []struct {
		in      string
		want    int64
		wantErr bool
	}{
		{in: "1024", want: 1024},
		{in: "1KiB", want: 1 << 10},
		{in: "64KiB", want: 64 << 10},
		{in: "10MiB", want: 10 << 20},
		{in: "1GiB", want: 1 << 30},
		{in: "1KB", want: 1000},
		{in: "1MB", want: 1_000_000},
		{in: "1B", want: 1},
		{in: "1KiB ", want: 1 << 10},
		{in: "abc", wantErr: true},
		{in: "MiB", wantErr: true},
		{in: "999999999999999999999GiB", wantErr: true},
	}

	for _, tt := range tests {
		got, err := parseSize(tt.in)
		if tt.wantErr {
			if err == nil {
				t.Errorf("parseSize(%q) error = nil, want error", tt.in)
			}

			continue
		}
		if err != nil {
			t.Errorf("parseSize(%q) error = %v", tt.in, err)

			continue
		}
		if got != tt.want {
			t.Errorf("parseSize(%q) = %d, want %d", tt.in, got, tt.want)
		}
	}
}
