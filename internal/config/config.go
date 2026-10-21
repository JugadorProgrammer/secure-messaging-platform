// Package config загружает и проверяет конфигурацию сервиса из переменных окружения.
package config

import (
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Значения по умолчанию из contracts.md, раздел 12.
const (
	defaultRSABits = 32768
	minRSABits     = 2048

	defaultReadHeaderTimeout = 5 * time.Second
	defaultReadTimeout       = 15 * time.Second
	defaultWriteTimeout      = 60 * time.Second
	defaultIdleTimeout       = 60 * time.Second
	defaultShutdownTimeout   = 15 * time.Second
	defaultAccountsTimeout   = 3 * time.Second
	defaultCryptoTimeout     = 10 * time.Second
	defaultRSAKeygenTimeout  = 60 * time.Minute

	defaultMaxBodyAuth     = 1 << 20 // 1 MiB
	defaultMaxBodyUsers    = 64 << 10
	defaultMaxMessageBytes = 64 << 10
	defaultMaxPDFUpload    = 10 << 20 // 10 MiB

	masterKeyLen = 32
	redacted     = "[redacted]"
)

// Config — проверенная конфигурация сервиса.
//
// Мастер-ключ хранится массивом фиксированной длины; String и LogValue
// не выводят его и DSN.
type Config struct {
	HTTPAddr string
	DSN      string

	MasterKey [masterKeyLen]byte

	AccountsURL *url.URL
	CryptoURL   *url.URL

	RSABits     int
	CORSOrigins []string
	LogLevel    slog.Level

	HTTPReadHeaderTimeout time.Duration
	HTTPReadTimeout       time.Duration
	HTTPWriteTimeout      time.Duration
	HTTPIdleTimeout       time.Duration
	ShutdownTimeout       time.Duration

	AccountsTimeout  time.Duration
	CryptoTimeout    time.Duration
	RSAKeygenTimeout time.Duration

	MaxBodyAuth     int64
	MaxBodyUsers    int64
	MaxMessageBytes int64
	MaxPDFUpload    int64
}

// Load читает конфигурацию через getenv. Если обязательная переменная
// отсутствует или значение недопустимо, возвращается ошибка со всеми
// найденными проблемами.
func Load(getenv func(string) string) (*Config, error) {
	if getenv == nil {
		return nil, errors.New("config: getenv must not be nil")
	}

	l := &loader{getenv: getenv}
	cfg := &Config{
		HTTPAddr:    l.required("HTTP_ADDR"),
		DSN:         l.required("DSN"),
		MasterKey:   l.masterKey("MASTER_KEY"),
		AccountsURL: l.url("ACCOUNTS_URL"),
		CryptoURL:   l.url("CRYPTO_URL"),
		RSABits:     l.rsaBits("RSA_BITS"),
		CORSOrigins: l.origins("CORS_ORIGINS"),
		LogLevel:    l.logLevel("LOG_LEVEL"),

		HTTPReadHeaderTimeout: l.duration("HTTP_READ_HEADER_TIMEOUT", defaultReadHeaderTimeout),
		HTTPReadTimeout:       l.duration("HTTP_READ_TIMEOUT", defaultReadTimeout),
		HTTPWriteTimeout:      l.duration("HTTP_WRITE_TIMEOUT", defaultWriteTimeout),
		HTTPIdleTimeout:       l.duration("HTTP_IDLE_TIMEOUT", defaultIdleTimeout),
		ShutdownTimeout:       l.duration("SHUTDOWN_TIMEOUT", defaultShutdownTimeout),

		AccountsTimeout:  l.duration("ACCOUNTS_TIMEOUT", defaultAccountsTimeout),
		CryptoTimeout:    l.duration("CRYPTO_TIMEOUT", defaultCryptoTimeout),
		RSAKeygenTimeout: l.duration("RSA_KEYGEN_TIMEOUT", defaultRSAKeygenTimeout),

		MaxBodyAuth:     l.size("MAX_BODY_AUTH", defaultMaxBodyAuth),
		MaxBodyUsers:    l.size("MAX_BODY_USERS", defaultMaxBodyUsers),
		MaxMessageBytes: l.size("MAX_MESSAGE_BYTES", defaultMaxMessageBytes),
		MaxPDFUpload:    l.size("MAX_PDF_UPLOAD", defaultMaxPDFUpload),
	}

	if err := errors.Join(l.errs...); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}

	return cfg, nil
}

// MasterKeyHex возвращает мастер-ключ в hex-виде для пакета sealed.
func (c *Config) MasterKeyHex() string {
	return hex.EncodeToString(c.MasterKey[:])
}

// String возвращает представление конфигурации без секретов.
func (c *Config) String() string {
	if c == nil {
		return "config(nil)"
	}

	return fmt.Sprintf(
		"config{http_addr=%q dsn=%s master_key=%s accounts_url=%s crypto_url=%s rsa_bits=%d cors_origins=%v log_level=%s}",
		c.HTTPAddr, redacted, redacted, urlString(c.AccountsURL), urlString(c.CryptoURL),
		c.RSABits, c.CORSOrigins, c.LogLevel,
	)
}

// LogValue скрывает DSN и мастер-ключ при логировании через slog.
func (c *Config) LogValue() slog.Value {
	if c == nil {
		return slog.StringValue("config(nil)")
	}

	return slog.GroupValue(
		slog.String("http_addr", c.HTTPAddr),
		slog.String("dsn", redacted),
		slog.String("master_key", redacted),
		slog.String("accounts_url", urlString(c.AccountsURL)),
		slog.String("crypto_url", urlString(c.CryptoURL)),
		slog.Int("rsa_bits", c.RSABits),
		slog.Any("cors_origins", c.CORSOrigins),
		slog.String("log_level", c.LogLevel.String()),
		slog.Duration("http_read_header_timeout", c.HTTPReadHeaderTimeout),
		slog.Duration("http_read_timeout", c.HTTPReadTimeout),
		slog.Duration("http_write_timeout", c.HTTPWriteTimeout),
		slog.Duration("http_idle_timeout", c.HTTPIdleTimeout),
		slog.Duration("shutdown_timeout", c.ShutdownTimeout),
		slog.Duration("accounts_timeout", c.AccountsTimeout),
		slog.Duration("crypto_timeout", c.CryptoTimeout),
		slog.Duration("rsa_keygen_timeout", c.RSAKeygenTimeout),
		slog.Int64("max_body_auth", c.MaxBodyAuth),
		slog.Int64("max_body_users", c.MaxBodyUsers),
		slog.Int64("max_message_bytes", c.MaxMessageBytes),
		slog.Int64("max_pdf_upload", c.MaxPDFUpload),
	)
}

func urlString(u *url.URL) string {
	if u == nil {
		return ""
	}

	return u.String()
}

// loader накапливает ошибки, чтобы вернуть их все сразу.
type loader struct {
	getenv func(string) string
	errs   []error
}

func (l *loader) value(name string) string {
	return strings.TrimSpace(l.getenv(name))
}

func (l *loader) required(name string) string {
	v := l.value(name)
	if v == "" {
		l.errs = append(l.errs, fmt.Errorf("%s is required", name))
	}

	return v
}

func (l *loader) masterKey(name string) [masterKeyLen]byte {
	var key [masterKeyLen]byte

	raw := l.value(name)
	if raw == "" {
		l.errs = append(l.errs, fmt.Errorf("%s is required", name))

		return key
	}

	b, err := hex.DecodeString(raw)
	if err != nil {
		l.errs = append(l.errs, fmt.Errorf("%s must be hex encoded: %w", name, err))

		return key
	}
	if len(b) != masterKeyLen {
		l.errs = append(l.errs, fmt.Errorf("%s must be %d hex characters (%d bytes), got %d bytes",
			name, masterKeyLen*2, masterKeyLen, len(b)))

		return key
	}

	copy(key[:], b)

	return key
}

func (l *loader) url(name string) *url.URL {
	raw := l.value(name)
	if raw == "" {
		l.errs = append(l.errs, fmt.Errorf("%s is required", name))

		return nil
	}

	u, err := url.Parse(raw)
	if err != nil {
		l.errs = append(l.errs, fmt.Errorf("%s is not a valid URL: %w", name, err))

		return nil
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		l.errs = append(l.errs, fmt.Errorf("%s must use http or https scheme, got %q", name, u.Scheme))

		return nil
	}
	if u.Host == "" {
		l.errs = append(l.errs, fmt.Errorf("%s must contain a host", name))

		return nil
	}

	return u
}

func (l *loader) rsaBits(name string) int {
	raw := l.value(name)
	if raw == "" {
		return defaultRSABits
	}

	n, err := strconv.Atoi(raw)
	if err != nil {
		l.errs = append(l.errs, fmt.Errorf("%s must be an integer: %w", name, err))

		return defaultRSABits
	}
	if n < minRSABits || n%8 != 0 {
		l.errs = append(l.errs, fmt.Errorf("%s must be at least %d and a multiple of 8, got %d",
			name, minRSABits, n))

		return defaultRSABits
	}

	return n
}

func (l *loader) origins(name string) []string {
	raw := l.value(name)
	if raw == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	origins := make([]string, 0, len(parts))

	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if p != "*" {
			u, err := url.Parse(p)
			if err != nil || u.Scheme == "" || u.Host == "" {
				l.errs = append(l.errs, fmt.Errorf("%s contains invalid origin %q", name, p))

				continue
			}
		}

		origins = append(origins, p)
	}

	return origins
}

func (l *loader) logLevel(name string) slog.Level {
	switch strings.ToLower(l.value(name)) {
	case "":
		return slog.LevelInfo
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		l.errs = append(l.errs, fmt.Errorf("%s must be one of debug, info, warn, error, got %q",
			name, l.value(name)))

		return slog.LevelInfo
	}
}

func (l *loader) duration(name string, def time.Duration) time.Duration {
	raw := l.value(name)
	if raw == "" {
		return def
	}

	d, err := time.ParseDuration(raw)
	if err != nil {
		l.errs = append(l.errs, fmt.Errorf("%s is not a valid duration: %w", name, err))

		return def
	}
	if d <= 0 {
		l.errs = append(l.errs, fmt.Errorf("%s must be positive, got %s", name, raw))

		return def
	}

	return d
}

func (l *loader) size(name string, def int64) int64 {
	raw := l.value(name)
	if raw == "" {
		return def
	}

	n, err := parseSize(raw)
	if err != nil {
		l.errs = append(l.errs, fmt.Errorf("%s: %w", name, err))

		return def
	}
	if n <= 0 {
		l.errs = append(l.errs, fmt.Errorf("%s must be positive, got %s", name, raw))

		return def
	}

	return n
}

var sizeUnits = []struct {
	suffix string
	mult   int64
}{
	{"GiB", 1 << 30},
	{"MiB", 1 << 20},
	{"KiB", 1 << 10},
	{"GB", 1_000_000_000},
	{"MB", 1_000_000},
	{"KB", 1_000},
	{"B", 1},
}

// parseSize разбирает размер вида "64KiB", "10MiB", "1024" или "1MB".
func parseSize(raw string) (int64, error) {
	raw = strings.TrimSpace(raw)

	for _, u := range sizeUnits {
		if !strings.HasSuffix(raw, u.suffix) {
			continue
		}

		num := strings.TrimSpace(strings.TrimSuffix(raw, u.suffix))
		n, err := strconv.ParseInt(num, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid size %q", raw)
		}
		if n > math.MaxInt64/u.mult {
			return 0, fmt.Errorf("size %q is too large", raw)
		}

		return n * u.mult, nil
	}

	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size %q", raw)
	}

	return n, nil
}
