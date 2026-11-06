package cryptoclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/httpx"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/ports"
)

// Фикстуры заведомо недействительны: это не ключи, а метки, по которым
// тест ищет утечки в логе.
const (
	testPubPEM  = "-----BEGIN RSA PUBLIC KEY-----\nUFVCLUZJWFRVUkU=\n-----END RSA PUBLIC KEY-----\n"
	testPrivPEM = "-----BEGIN RSA PRIVATE KEY-----\nUFJJVi1GSVhUVVJF\n-----END RSA PRIVATE KEY-----\n"
	testKeyHex  = "abababababababababababababababababababababababababababababababab"
	testIVHex   = "00112233445566778899aabbccddeeff"

	testText    = "привет"
	testTextB64 = "0L/RgNC40LLQtdGC"
	testCT      = "\x01\x02\x03\x04"
	testCTB64   = "AQIDBA=="
	testSigB64  = "CQkJ"

	defaultTestTimeout = 2 * time.Second
)

var testHash = bytes.Repeat([]byte{0x5a}, hashLen)

func testHashHex() string { return strings.Repeat("5a", hashLen) }

// newTestClient поднимает httptest.Server с обработчиком h и клиент к нему.
// Лог пишется в буфер на уровне Debug, чтобы тест видел всё, что клиент логирует.
func newTestClient(t *testing.T, h http.Handler, timeout, keygenTimeout time.Duration) (*Client, *bytes.Buffer) {
	t.Helper()

	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	base, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}

	var logBuf bytes.Buffer

	log := slog.New(slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	return New(base, timeout, keygenTimeout, log), &logBuf
}

// waitClientGone вычитывает тело запроса. Только после этого сервер начинает
// следить за соединением и отменяет r.Context(), когда клиент его закрыл;
// без этого обработчик, ждущий отмены, висел бы до конца теста.
func waitClientGone(r *http.Request) {
	_, _ = io.Copy(io.Discard, r.Body)
}

func writeJSON(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

// jsonEqual сравнивает JSON по значению, а не по порядку полей.
func jsonEqual(t *testing.T, got []byte, want string) bool {
	t.Helper()

	var g, w any
	if err := json.Unmarshal(got, &g); err != nil {
		t.Fatalf("request body is not JSON: %v", err)
	}

	if err := json.Unmarshal([]byte(want), &w); err != nil {
		t.Fatalf("want is not JSON: %v", err)
	}

	return reflect.DeepEqual(g, w)
}

func TestClientEndpoints(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		wantReq  string
		response string
		call     func(t *testing.T, c *Client)
	}{
		{
			name:     "ключи кузнечика",
			path:     pathKuzKeys,
			wantReq:  `{}`,
			response: `{"key_hex":"` + testKeyHex + `","iv_hex":"` + testIVHex + `"}`,
			call: func(t *testing.T, c *Client) {
				key, iv, err := c.KuzKeys(t.Context())
				if err != nil {
					t.Fatalf("KuzKeys() error = %v", err)
				}
				if key != testKeyHex || iv != testIVHex {
					t.Errorf("KuzKeys() = %q, %q", key, iv)
				}
			},
		},
		{
			name:     "шифрование кузнечиком",
			path:     pathKuzEncrypt,
			wantReq:  `{"key_hex":"` + testKeyHex + `","iv_hex":"` + testIVHex + `","data":"` + testTextB64 + `"}`,
			response: `{"ciphertext":"` + testCTB64 + `","duration_ms":7}`,
			call: func(t *testing.T, c *Client) {
				ct, ms, err := c.KuzEncrypt(t.Context(), testKeyHex, testIVHex, []byte(testText))
				if err != nil {
					t.Fatalf("KuzEncrypt() error = %v", err)
				}
				if string(ct) != testCT || ms != 7 {
					t.Errorf("KuzEncrypt() = %v, %d", ct, ms)
				}
			},
		},
		{
			name:     "расшифрование кузнечиком",
			path:     pathKuzDecrypt,
			wantReq:  `{"key_hex":"` + testKeyHex + `","iv_hex":"` + testIVHex + `","ciphertext":"` + testCTB64 + `"}`,
			response: `{"data":"` + testTextB64 + `","duration_ms":3}`,
			call: func(t *testing.T, c *Client) {
				data, ms, err := c.KuzDecrypt(t.Context(), testKeyHex, testIVHex, []byte(testCT))
				if err != nil {
					t.Fatalf("KuzDecrypt() error = %v", err)
				}
				if string(data) != testText || ms != 3 {
					t.Errorf("KuzDecrypt() = %q, %d", data, ms)
				}
			},
		},
		{
			name:    "ключи rsa",
			path:    pathRSAKeys,
			wantReq: `{"bits":2048}`,
			response: `{"public_key_pem":` + jsonString(testPubPEM) + `,"private_key_pem":` + jsonString(testPrivPEM) +
				`,"duration_ms":1500}`,
			call: func(t *testing.T, c *Client) {
				pub, priv, ms, err := c.RSAKeys(t.Context(), 2048)
				if err != nil {
					t.Fatalf("RSAKeys() error = %v", err)
				}
				if pub != testPubPEM || priv != testPrivPEM || ms != 1500 {
					t.Errorf("RSAKeys() = %q, %q, %d", pub, priv, ms)
				}
			},
		},
		{
			name:     "шифрование rsa",
			path:     pathRSAEncrypt,
			wantReq:  `{"public_key_pem":` + jsonString(testPubPEM) + `,"data":"` + testTextB64 + `"}`,
			response: `{"ciphertext":"` + testCTB64 + `","duration_ms":11}`,
			call: func(t *testing.T, c *Client) {
				ct, ms, err := c.RSAEncrypt(t.Context(), testPubPEM, []byte(testText))
				if err != nil {
					t.Fatalf("RSAEncrypt() error = %v", err)
				}
				if string(ct) != testCT || ms != 11 {
					t.Errorf("RSAEncrypt() = %v, %d", ct, ms)
				}
			},
		},
		{
			name:     "расшифрование rsa",
			path:     pathRSADecrypt,
			wantReq:  `{"private_key_pem":` + jsonString(testPrivPEM) + `,"ciphertext":"` + testCTB64 + `"}`,
			response: `{"data":"` + testTextB64 + `","duration_ms":42}`,
			call: func(t *testing.T, c *Client) {
				data, ms, err := c.RSADecrypt(t.Context(), testPrivPEM, []byte(testCT))
				if err != nil {
					t.Fatalf("RSADecrypt() error = %v", err)
				}
				if string(data) != testText || ms != 42 {
					t.Errorf("RSADecrypt() = %q, %d", data, ms)
				}
			},
		},
		{
			name:     "стрибог-512",
			path:     pathStreebog,
			wantReq:  `{"data":"` + testTextB64 + `"}`,
			response: `{"hash_hex":"` + testHashHex() + `"}`,
			call: func(t *testing.T, c *Client) {
				hash, err := c.Streebog512(t.Context(), []byte(testText))
				if err != nil {
					t.Fatalf("Streebog512() error = %v", err)
				}
				if !bytes.Equal(hash, testHash) {
					t.Errorf("Streebog512() = %x", hash)
				}
			},
		},
		{
			name:     "подпись хеша",
			path:     pathSignHash,
			wantReq:  `{"private_key_pem":` + jsonString(testPrivPEM) + `,"hash_hex":"` + testHashHex() + `","hash_alg":"streebog512"}`,
			response: `{"signature":"` + testSigB64 + `"}`,
			call: func(t *testing.T, c *Client) {
				sig, err := c.SignHash(t.Context(), testPrivPEM, testHash)
				if err != nil {
					t.Fatalf("SignHash() error = %v", err)
				}
				if !bytes.Equal(sig, []byte{9, 9, 9}) {
					t.Errorf("SignHash() = %v", sig)
				}
			},
		},
		{
			name: "проверка подписи хеша",
			path: pathVerifyHash,
			wantReq: `{"public_key_pem":` + jsonString(testPubPEM) + `,"hash_hex":"` + testHashHex() +
				`","hash_alg":"streebog512","signature":"` + testSigB64 + `"}`,
			response: `{"valid":true}`,
			call: func(t *testing.T, c *Client) {
				ok, err := c.VerifyHash(t.Context(), testPubPEM, testHash, []byte{9, 9, 9})
				if err != nil {
					t.Fatalf("VerifyHash() error = %v", err)
				}
				if !ok {
					t.Error("VerifyHash() = false, want true")
				}
			},
		},
		{
			name: "неверная подпись не ошибка",
			path: pathVerifyHash,
			wantReq: `{"public_key_pem":` + jsonString(testPubPEM) + `,"hash_hex":"` + testHashHex() +
				`","hash_alg":"streebog512","signature":"` + testSigB64 + `"}`,
			response: `{"valid":false}`,
			call: func(t *testing.T, c *Client) {
				ok, err := c.VerifyHash(t.Context(), testPubPEM, testHash, []byte{9, 9, 9})
				if err != nil {
					t.Fatalf("VerifyHash() error = %v", err)
				}
				if ok {
					t.Error("VerifyHash() = true, want false")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("method = %s, want POST", r.Method)
				}
				if r.URL.Path != tt.path {
					t.Errorf("path = %s, want %s", r.URL.Path, tt.path)
				}
				if ct := r.Header.Get("Content-Type"); ct != contentTypeJSON {
					t.Errorf("Content-Type = %q, want %q", ct, contentTypeJSON)
				}

				var body bytes.Buffer
				_, _ = body.ReadFrom(r.Body)

				if !jsonEqual(t, body.Bytes(), tt.wantReq) {
					t.Errorf("request body = %s, want %s", body.String(), tt.wantReq)
				}

				writeJSON(w, http.StatusOK, tt.response)
			})

			c, _ := newTestClient(t, h, defaultTestTimeout, defaultTestTimeout)
			tt.call(t, c)
		})
	}
}

func jsonString(s string) string {
	b, _ := json.Marshal(s)

	return string(b)
}

// allCalls вызывает каждый из девяти эндпоинтов с корректными аргументами.
var allCalls = []struct {
	name string
	call func(ctx context.Context, c *Client) error
}{
	{"ключи кузнечика", func(ctx context.Context, c *Client) error {
		_, _, err := c.KuzKeys(ctx)
		return err
	}},
	{"шифрование кузнечиком", func(ctx context.Context, c *Client) error {
		_, _, err := c.KuzEncrypt(ctx, testKeyHex, testIVHex, []byte(testText))
		return err
	}},
	{"расшифрование кузнечиком", func(ctx context.Context, c *Client) error {
		_, _, err := c.KuzDecrypt(ctx, testKeyHex, testIVHex, []byte(testCT))
		return err
	}},
	{"ключи rsa", func(ctx context.Context, c *Client) error {
		_, _, _, err := c.RSAKeys(ctx, 2048)
		return err
	}},
	{"шифрование rsa", func(ctx context.Context, c *Client) error {
		_, _, err := c.RSAEncrypt(ctx, testPubPEM, []byte(testText))
		return err
	}},
	{"расшифрование rsa", func(ctx context.Context, c *Client) error {
		_, _, err := c.RSADecrypt(ctx, testPrivPEM, []byte(testCT))
		return err
	}},
	{"стрибог-512", func(ctx context.Context, c *Client) error {
		_, err := c.Streebog512(ctx, []byte(testText))
		return err
	}},
	{"подпись хеша", func(ctx context.Context, c *Client) error {
		_, err := c.SignHash(ctx, testPrivPEM, testHash)
		return err
	}},
	{"проверка подписи хеша", func(ctx context.Context, c *Client) error {
		_, err := c.VerifyHash(ctx, testPubPEM, testHash, []byte{9, 9, 9})
		return err
	}},
}

// TestAllEndpointsMapStatus проверяет на каждом эндпоинте, что 4xx даёт
// ErrCryptoBadInput, а 5xx — ErrCryptoUnavailable.
func TestAllEndpointsMapStatus(t *testing.T) {
	for _, tt := range allCalls {
		t.Run(tt.name, func(t *testing.T) {
			var calls atomic.Int32

			h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if calls.Add(1) == 1 {
					writeJSON(w, http.StatusBadRequest, `{"error":{"code":"bad_key","message":"invalid key"}}`)
					return
				}
				writeJSON(w, http.StatusInternalServerError, `{"error":"boom"}`)
			})

			c, _ := newTestClient(t, h, defaultTestTimeout, defaultTestTimeout)

			if err := tt.call(t.Context(), c); !errors.Is(err, ports.ErrCryptoBadInput) {
				t.Errorf("400: error = %v, want ErrCryptoBadInput", err)
			}

			if err := tt.call(t.Context(), c); !errors.Is(err, ports.ErrCryptoUnavailable) {
				t.Errorf("500: error = %v, want ErrCryptoUnavailable", err)
			}
		})
	}
}

func TestClientErrors(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
		want    error
	}{
		{
			name: "400 неверный ключ",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				writeJSON(w, http.StatusBadRequest, `{"error":{"code":"bad_key","message":"bad"}}`)
			},
			want: ports.ErrCryptoBadInput,
		},
		{
			name: "422 без тела",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusUnprocessableEntity)
			},
			want: ports.ErrCryptoBadInput,
		},
		{
			name: "429 перегрузка модуля",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				writeJSON(w, http.StatusTooManyRequests, `{}`)
			},
			want: ports.ErrCryptoUnavailable,
		},
		{
			name: "503",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				writeJSON(w, http.StatusServiceUnavailable, `{}`)
			},
			want: ports.ErrCryptoUnavailable,
		},
		{
			name: "редирект не выполняется",
			handler: func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, "/elsewhere", http.StatusFound)
			},
			want: ports.ErrCryptoUnavailable,
		},
		{
			name: "битый json",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				writeJSON(w, http.StatusOK, `{"ciphertext":`)
			},
			want: ports.ErrCryptoUnavailable,
		},
		{
			name: "ответ не json",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				_, _ = w.Write([]byte(`{"ciphertext":"AQIDBA==","duration_ms":1}`))
			},
			want: ports.ErrCryptoUnavailable,
		},
		{
			name: "ответ больше лимита",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				writeJSON(w, http.StatusOK, `{"ciphertext":"`+strings.Repeat("A", maxResponseBytes)+`"}`)
			},
			want: ports.ErrCryptoUnavailable,
		},
		{
			name: "обрыв соединения",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				hj, ok := w.(http.Hijacker)
				if !ok {
					t.Error("response writer does not support hijacking")
					return
				}
				conn, _, err := hj.Hijack()
				if err != nil {
					t.Errorf("hijack: %v", err)
					return
				}
				_ = conn.Close()
			},
			want: ports.ErrCryptoUnavailable,
		},
		{
			name: "таймаут",
			handler: func(w http.ResponseWriter, r *http.Request) {
				waitClientGone(r)
				select {
				case <-time.After(time.Second):
				case <-r.Context().Done():
				}
				writeJSON(w, http.StatusOK, `{"ciphertext":"AQIDBA==","duration_ms":1}`)
			},
			want: ports.ErrCryptoUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := newTestClient(t, tt.handler, 100*time.Millisecond, defaultTestTimeout)

			_, _, err := c.KuzEncrypt(t.Context(), testKeyHex, testIVHex, []byte(testText))
			if !errors.Is(err, tt.want) {
				t.Errorf("error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestClientBadResponses(t *testing.T) {
	tests := []struct {
		name     string
		response string
		call     func(ctx context.Context, c *Client) error
	}{
		{
			name:     "ключ кузнечика не hex",
			response: `{"key_hex":"zz","iv_hex":"` + testIVHex + `"}`,
			call:     allCalls[0].call,
		},
		{
			name:     "ключ кузнечика не 256 бит",
			response: `{"key_hex":"abab","iv_hex":"` + testIVHex + `"}`,
			call:     allCalls[0].call,
		},
		{
			name:     "пустой iv",
			response: `{"key_hex":"` + testKeyHex + `","iv_hex":""}`,
			call:     allCalls[0].call,
		},
		{
			name:     "шифртекст не base64",
			response: `{"ciphertext":"%%%","duration_ms":1}`,
			call:     allCalls[1].call,
		},
		{
			name:     "пустые данные",
			response: `{"data":"","duration_ms":1}`,
			call:     allCalls[2].call,
		},
		{
			name:     "открытый ключ не pem",
			response: `{"public_key_pem":"not pem","private_key_pem":` + jsonString(testPrivPEM) + `}`,
			call:     allCalls[3].call,
		},
		{
			name:     "закрытый ключ не pem",
			response: `{"public_key_pem":` + jsonString(testPubPEM) + `,"private_key_pem":""}`,
			call:     allCalls[3].call,
		},
		{
			name:     "хеш короче 64 байт",
			response: `{"hash_hex":"5a5a"}`,
			call:     allCalls[6].call,
		},
		{
			name:     "пустая подпись",
			response: `{"signature":""}`,
			call:     allCalls[7].call,
		},
		{
			name:     "нет поля valid",
			response: `{}`,
			call:     allCalls[8].call,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				writeJSON(w, http.StatusOK, tt.response)
			})

			c, _ := newTestClient(t, h, defaultTestTimeout, defaultTestTimeout)

			if err := tt.call(t.Context(), c); !errors.Is(err, ports.ErrCryptoUnavailable) {
				t.Errorf("error = %v, want ErrCryptoUnavailable", err)
			}
		})
	}
}

func TestClientCanceled(t *testing.T) {
	started := make(chan struct{})

	h := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		waitClientGone(r)
		close(started)
		<-r.Context().Done()
	})

	c, _ := newTestClient(t, h, defaultTestTimeout, defaultTestTimeout)

	ctx, cancel := context.WithCancel(t.Context())
	go func() {
		<-started
		cancel()
	}()

	_, err := c.Streebog512(ctx, []byte(testText))
	if !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want context.Canceled", err)
	}

	if errors.Is(err, ports.ErrCryptoUnavailable) {
		t.Errorf("error = %v: cancellation must not be reported as unavailable", err)
	}
}

func TestClientCallerDeadline(t *testing.T) {
	h := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		waitClientGone(r)
		<-r.Context().Done()
	})

	c, _ := newTestClient(t, h, defaultTestTimeout, defaultTestTimeout)

	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()

	_, err := c.Streebog512(ctx, []byte(testText))
	if !errors.Is(err, ports.ErrCryptoUnavailable) || !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("error = %v, want ErrCryptoUnavailable wrapping DeadlineExceeded", err)
	}
}

func TestRSAKeysTimeouts(t *testing.T) {
	tests := []struct {
		name          string
		timeout       time.Duration
		keygenTimeout time.Duration
		delay         time.Duration
		wantErr       error
	}{
		{
			name:          "общий таймаут не обрывает генерацию",
			timeout:       50 * time.Millisecond,
			keygenTimeout: defaultTestTimeout,
			delay:         200 * time.Millisecond,
		},
		{
			name:          "таймаут генерации обрывает",
			timeout:       defaultTestTimeout,
			keygenTimeout: 50 * time.Millisecond,
			delay:         time.Second,
			wantErr:       ports.ErrCryptoUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				waitClientGone(r)
				select {
				case <-time.After(tt.delay):
				case <-r.Context().Done():
					return
				}
				writeJSON(w, http.StatusOK,
					`{"public_key_pem":`+jsonString(testPubPEM)+`,"private_key_pem":`+jsonString(testPrivPEM)+`,"duration_ms":1}`)
			})

			c, _ := newTestClient(t, h, tt.timeout, tt.keygenTimeout)

			_, _, _, err := c.RSAKeys(t.Context(), 2048)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("RSAKeys() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestHashLengthChecked(t *testing.T) {
	tests := []struct {
		name string
		hash []byte
		call func(ctx context.Context, c *Client, hash []byte) error
	}{
		{"подпись пустого хеша", nil, signHash},
		{"подпись 63 байт", bytes.Repeat([]byte{1}, hashLen-1), signHash},
		{"подпись 65 байт", bytes.Repeat([]byte{1}, hashLen+1), signHash},
		{"проверка 32 байт", bytes.Repeat([]byte{1}, 32), verifyHash},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls atomic.Int32

			h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				calls.Add(1)
				writeJSON(w, http.StatusOK, `{}`)
			})

			c, _ := newTestClient(t, h, defaultTestTimeout, defaultTestTimeout)

			if err := tt.call(t.Context(), c, tt.hash); !errors.Is(err, ports.ErrInvalidArgument) {
				t.Errorf("error = %v, want ErrInvalidArgument", err)
			}

			if n := calls.Load(); n != 0 {
				t.Errorf("crypto module called %d times, want 0", n)
			}
		})
	}
}

func signHash(ctx context.Context, c *Client, hash []byte) error {
	_, err := c.SignHash(ctx, testPrivPEM, hash)
	return err
}

func verifyHash(ctx context.Context, c *Client, hash []byte) error {
	_, err := c.VerifyHash(ctx, testPubPEM, hash, []byte{9})
	return err
}

func TestClientRequestHeadersAndBasePath(t *testing.T) {
	const requestID = "req-123"

	var gotID, gotPath string

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotID = r.Header.Get(httpx.HeaderRequestID)
		gotPath = r.URL.Path
		writeJSON(w, http.StatusOK, `{"hash_hex":"`+testHashHex()+`"}`)
	})

	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	base, err := url.Parse(srv.URL + "/crypto")
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}

	c := New(base, defaultTestTimeout, defaultTestTimeout, nil)

	// Идентификатор кладётся в контекст тем же middleware, что и в сервисе.
	var ctx context.Context

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody)
	req.Header.Set(httpx.HeaderRequestID, requestID)
	httpx.RequestID(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		ctx = r.Context()
	})).ServeHTTP(rec, req)

	if _, err := c.Streebog512(ctx, []byte(testText)); err != nil {
		t.Fatalf("Streebog512() error = %v", err)
	}

	if gotID != requestID {
		t.Errorf("X-Request-ID = %q, want %q", gotID, requestID)
	}

	if gotPath != "/crypto"+pathStreebog {
		t.Errorf("path = %q, want %q", gotPath, "/crypto"+pathStreebog)
	}
}

func TestSecretsNotLogged(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case pathKuzKeys:
			writeJSON(w, http.StatusOK, `{"key_hex":"`+testKeyHex+`","iv_hex":"`+testIVHex+`"}`)
		case pathRSAKeys:
			writeJSON(w, http.StatusOK,
				`{"public_key_pem":`+jsonString(testPubPEM)+`,"private_key_pem":`+jsonString(testPrivPEM)+`,"duration_ms":1}`)
		case pathKuzEncrypt, pathRSAEncrypt:
			writeJSON(w, http.StatusOK, `{"ciphertext":"`+testCTB64+`","duration_ms":1}`)
		case pathKuzDecrypt, pathRSADecrypt:
			writeJSON(w, http.StatusOK, `{"data":"`+testTextB64+`","duration_ms":1}`)
		case pathStreebog:
			writeJSON(w, http.StatusOK, `{"hash_hex":"`+testHashHex()+`"}`)
		case pathSignHash:
			writeJSON(w, http.StatusOK, `{"signature":"`+testSigB64+`"}`)
		default:
			writeJSON(w, http.StatusBadRequest, `{"error":{"code":"bad_signature","message":"rejected"}}`)
		}
	})

	c, logBuf := newTestClient(t, h, defaultTestTimeout, defaultTestTimeout)

	for _, call := range allCalls {
		_ = call.call(t.Context(), c)
	}

	logged := logBuf.String()
	if !strings.Contains(logged, "crypto call") {
		t.Fatalf("log is empty, nothing was checked: %q", logged)
	}

	for _, secret := range []string{"UFJJVi1GSVhUVVJF", "UFVCLUZJWFRVUkU=", testKeyHex, testIVHex, testTextB64, testCTB64, testText} {
		if strings.Contains(logged, secret) {
			t.Errorf("log contains secret %q", secret)
		}
	}
}

func TestNewBadInputError(t *testing.T) {
	long := strings.Repeat("ж", maxErrorMsgLen)

	tests := []struct {
		name     string
		body     string
		wantCode string
		wantMsg  string
	}{
		{"вложенный формат", `{"error":{"code":"bad_key","message":"invalid key"}}`, "bad_key", "invalid key"},
		{"плоский формат", `{"code":"bad_ciphertext","message":"broken"}`, "bad_ciphertext", "broken"},
		{"ошибка строкой", `{"error":"invalid iv"}`, "", "invalid iv"},
		{"тело не json", `bad request`, "", ""},
		{"пустое тело", ``, "", ""},
		{"длинный текст обрезается", `{"error":{"message":"` + long + `"}}`, "", long[:maxErrorMsgLen] + "…"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newBadInputError(pathRSADecrypt, http.StatusBadRequest, []byte(tt.body))

			if e.Code != tt.wantCode || e.Msg != tt.wantMsg {
				t.Errorf("got code %q msg %q, want %q %q", e.Code, e.Msg, tt.wantCode, tt.wantMsg)
			}

			if !utf8.ValidString(e.Msg) {
				t.Errorf("msg is not valid UTF-8: %q", e.Msg)
			}

			if e.Status != http.StatusBadRequest || !strings.Contains(e.Error(), pathRSADecrypt) {
				t.Errorf("Error() = %q", e.Error())
			}
		})
	}
}

func TestUnavailableErrorUnwrap(t *testing.T) {
	cause := errors.New("connection refused")
	err := error(&UnavailableError{Op: pathRSAEncrypt, Err: cause})

	if !errors.Is(err, ports.ErrCryptoUnavailable) || !errors.Is(err, cause) {
		t.Errorf("errors.Is failed for %v", err)
	}

	if !strings.Contains(err.Error(), pathRSAEncrypt) {
		t.Errorf("Error() = %q, want op in text", err.Error())
	}
}
