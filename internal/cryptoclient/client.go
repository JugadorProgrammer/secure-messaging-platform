// Package cryptoclient реализует ports.CryptoClient поверх HTTP API
// Крипто-модуля (contracts.md, раздел 14).
//
// Кодировки на проводе: данные, шифртекст и подпись — base64 (standard,
// с padding), ключ и IV Кузнечика и хеш — hex, ключи RSA — PEM (PKCS#1).
// Повторов нет: крипто-операции дорогие, и повтор только удлиняет ожидание.
package cryptoclient

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/JugadorProgrammer/secure-messaging-platform/internal/httpx"
	"github.com/JugadorProgrammer/secure-messaging-platform/internal/ports"
)

// Проверка на этапе компиляции: Client реализует контракт целиком.
var _ ports.CryptoClient = (*Client)(nil)

const (
	// maxResponseBytes — лимит тела ответа. PEM ключа RSA-32768 занимает около
	// 25 КБ, шифртекст сообщения 64 КиБ в base64 — около 90 КБ; 4 МиБ с запасом.
	maxResponseBytes = 4 << 20

	maxIdleConnsPerHost = 20
	idleConnTimeout     = 90 * time.Second
	dialTimeout         = 5 * time.Second
	keepAlive           = 30 * time.Second

	// hashAlg передаётся явно, чтобы модуль не подставил свой алгоритм по умолчанию.
	hashAlg = "streebog512"
	hashLen = 64

	kuzKeyLen = 32

	contentTypeJSON = "application/json"
)

// Пути эндпоинтов Крипто-модуля.
const (
	pathRSAKeys    = "/rsa/keys"
	pathRSAEncrypt = "/rsa/encrypt"
	pathRSADecrypt = "/rsa/decrypt"
	pathKuzKeys    = "/kuznyechik/keys"
	pathKuzEncrypt = "/kuznyechik/encrypt"
	pathKuzDecrypt = "/kuznyechik/decrypt"
	pathStreebog   = "/streebog512"
	pathSignHash   = "/sign-hash"
	pathVerifyHash = "/verify-hash"
)

// errResponseTooLarge — ответ модуля больше maxResponseBytes.
var errResponseTooLarge = errors.New("response exceeds size limit")

// Client — HTTP-клиент Крипто-модуля. Безопасен для конкурентного использования.
type Client struct {
	base *url.URL
	// http ограничен общим таймаутом CRYPTO_TIMEOUT.
	http *http.Client
	// keygen обслуживает только /rsa/keys: ключ 32768 бит генерируется
	// десятки минут, и общий таймаут оборвал бы его.
	keygen        *http.Client
	keygenTimeout time.Duration
	log           *slog.Logger
}

// New создаёт клиент. timeout — CRYPTO_TIMEOUT для всех операций, кроме
// генерации ключа RSA; keygenTimeout — RSA_KEYGEN_TIMEOUT для /rsa/keys.
func New(base *url.URL, timeout, keygenTimeout time.Duration, log *slog.Logger) *Client {
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}

	return &Client{
		base: base,
		http: &http.Client{
			Transport:     newTransport(0),
			Timeout:       timeout,
			CheckRedirect: noRedirect,
		},
		keygen: &http.Client{
			Transport:     newTransport(keygenTimeout),
			CheckRedirect: noRedirect,
		},
		keygenTimeout: keygenTimeout,
		log:           log,
	}
}

func newTransport(responseHeaderTimeout time.Duration) *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   dialTimeout,
			KeepAlive: keepAlive,
		}).DialContext,
		MaxIdleConnsPerHost:   maxIdleConnsPerHost,
		IdleConnTimeout:       idleConnTimeout,
		ResponseHeaderTimeout: responseHeaderTimeout,
		ForceAttemptHTTP2:     true,
	}
}

// noRedirect запрещает переходы: модуль внутренний, редирект означает
// неверный CRYPTO_URL, и тело запроса с ключом не должно уйти на другой адрес.
func noRedirect(*http.Request, []*http.Request) error {
	return http.ErrUseLastResponse
}

type kuzKeysRequest struct{}

type kuzKeysResponse struct {
	KeyHex string `json:"key_hex"`
	IVHex  string `json:"iv_hex"`
}

type kuzEncryptRequest struct {
	KeyHex string `json:"key_hex"`
	IVHex  string `json:"iv_hex"`
	Data   string `json:"data"`
}

type kuzDecryptRequest struct {
	KeyHex     string `json:"key_hex"`
	IVHex      string `json:"iv_hex"`
	Ciphertext string `json:"ciphertext"`
}

type rsaKeysRequest struct {
	Bits int `json:"bits"`
}

type rsaKeysResponse struct {
	PublicKeyPEM  string `json:"public_key_pem"`
	PrivateKeyPEM string `json:"private_key_pem"`
	DurationMs    int64  `json:"duration_ms"`
}

type rsaEncryptRequest struct {
	PublicKeyPEM string `json:"public_key_pem"`
	Data         string `json:"data"`
}

type rsaDecryptRequest struct {
	PrivateKeyPEM string `json:"private_key_pem"`
	Ciphertext    string `json:"ciphertext"`
}

type ciphertextResponse struct {
	Ciphertext string `json:"ciphertext"`
	DurationMs int64  `json:"duration_ms"`
}

type dataResponse struct {
	Data       string `json:"data"`
	DurationMs int64  `json:"duration_ms"`
}

type streebogRequest struct {
	Data string `json:"data"`
}

type streebogResponse struct {
	HashHex string `json:"hash_hex"`
}

type signHashRequest struct {
	PrivateKeyPEM string `json:"private_key_pem"`
	HashHex       string `json:"hash_hex"`
	HashAlg       string `json:"hash_alg"`
}

type signHashResponse struct {
	Signature string `json:"signature"`
}

type verifyHashRequest struct {
	PublicKeyPEM string `json:"public_key_pem"`
	HashHex      string `json:"hash_hex"`
	HashAlg      string `json:"hash_alg"`
	Signature    string `json:"signature"`
}

type verifyHashResponse struct {
	// Указатель, чтобы отличить отсутствующее поле от valid=false.
	Valid *bool `json:"valid"`
}

// KuzKeys запрашивает новый ключ (256 бит) и IV Кузнечика в hex.
func (c *Client) KuzKeys(ctx context.Context) (keyHex, ivHex string, err error) {
	var resp kuzKeysResponse
	if err := c.do(ctx, c.http, pathKuzKeys, kuzKeysRequest{}, &resp); err != nil {
		return "", "", err
	}

	if err := checkHex(pathKuzKeys, "key_hex", resp.KeyHex, kuzKeyLen); err != nil {
		return "", "", err
	}

	if err := checkHex(pathKuzKeys, "iv_hex", resp.IVHex, 0); err != nil {
		return "", "", err
	}

	return resp.KeyHex, resp.IVHex, nil
}

// KuzEncrypt шифрует data Кузнечиком. ms — время операции по данным модуля.
func (c *Client) KuzEncrypt(ctx context.Context, keyHex, ivHex string, data []byte) (ct []byte, ms int64, err error) {
	req := kuzEncryptRequest{KeyHex: keyHex, IVHex: ivHex, Data: encodeB64(data)}

	return c.ciphertextCall(ctx, pathKuzEncrypt, req)
}

// KuzDecrypt расшифровывает ct Кузнечиком.
func (c *Client) KuzDecrypt(ctx context.Context, keyHex, ivHex string, ct []byte) (data []byte, ms int64, err error) {
	req := kuzDecryptRequest{KeyHex: keyHex, IVHex: ivHex, Ciphertext: encodeB64(ct)}

	return c.dataCall(ctx, pathKuzDecrypt, req)
}

// RSAKeys генерирует пару ключей RSA размером bits.
//
// Запрос идёт через отдельный клиент без общего таймаута. Если у ctx нет
// дедлайна, ставится RSA_KEYGEN_TIMEOUT: бесконечное ожидание недопустимо
// даже для фоновой генерации.
func (c *Client) RSAKeys(ctx context.Context, bits int) (pub, priv string, ms int64, err error) {
	if _, ok := ctx.Deadline(); !ok && c.keygenTimeout > 0 {
		var cancel context.CancelFunc

		ctx, cancel = context.WithTimeout(ctx, c.keygenTimeout)
		defer cancel()
	}

	var resp rsaKeysResponse
	if err := c.do(ctx, c.keygen, pathRSAKeys, rsaKeysRequest{Bits: bits}, &resp); err != nil {
		return "", "", 0, err
	}

	if err := checkPEM(pathRSAKeys, "public_key_pem", resp.PublicKeyPEM); err != nil {
		return "", "", 0, err
	}

	if err := checkPEM(pathRSAKeys, "private_key_pem", resp.PrivateKeyPEM); err != nil {
		return "", "", 0, err
	}

	return resp.PublicKeyPEM, resp.PrivateKeyPEM, resp.DurationMs, nil
}

// RSAEncrypt шифрует data открытым ключом pubPEM.
func (c *Client) RSAEncrypt(ctx context.Context, pubPEM string, data []byte) (ct []byte, ms int64, err error) {
	req := rsaEncryptRequest{PublicKeyPEM: pubPEM, Data: encodeB64(data)}

	return c.ciphertextCall(ctx, pathRSAEncrypt, req)
}

// RSADecrypt расшифровывает ct закрытым ключом privPEM.
func (c *Client) RSADecrypt(ctx context.Context, privPEM string, ct []byte) (data []byte, ms int64, err error) {
	req := rsaDecryptRequest{PrivateKeyPEM: privPEM, Ciphertext: encodeB64(ct)}

	return c.dataCall(ctx, pathRSADecrypt, req)
}

// Streebog512 возвращает хеш Стрибог-512 от data (64 байта).
func (c *Client) Streebog512(ctx context.Context, data []byte) ([]byte, error) {
	var resp streebogResponse
	if err := c.do(ctx, c.http, pathStreebog, streebogRequest{Data: encodeB64(data)}, &resp); err != nil {
		return nil, err
	}

	if err := checkHex(pathStreebog, "hash_hex", resp.HashHex, hashLen); err != nil {
		return nil, err
	}

	hash, _ := hex.DecodeString(resp.HashHex) // формат проверен в checkHex

	return hash, nil
}

// SignHash подписывает готовый хеш Стрибог-512 закрытым ключом privPEM.
// Хеш не той длины — ports.ErrInvalidArgument, запрос в модуль не уходит.
func (c *Client) SignHash(ctx context.Context, privPEM string, hash []byte) ([]byte, error) {
	if err := checkHashLen(pathSignHash, hash); err != nil {
		return nil, err
	}

	req := signHashRequest{PrivateKeyPEM: privPEM, HashHex: hex.EncodeToString(hash), HashAlg: hashAlg}

	var resp signHashResponse
	if err := c.do(ctx, c.http, pathSignHash, req, &resp); err != nil {
		return nil, err
	}

	return decodeB64(pathSignHash, "signature", resp.Signature)
}

// VerifyHash проверяет подпись sig над готовым хешем открытым ключом pubPEM.
// Неверная подпись — (false, nil), а не ошибка.
func (c *Client) VerifyHash(ctx context.Context, pubPEM string, hash, sig []byte) (bool, error) {
	if err := checkHashLen(pathVerifyHash, hash); err != nil {
		return false, err
	}

	req := verifyHashRequest{
		PublicKeyPEM: pubPEM,
		HashHex:      hex.EncodeToString(hash),
		HashAlg:      hashAlg,
		Signature:    encodeB64(sig),
	}

	var resp verifyHashResponse
	if err := c.do(ctx, c.http, pathVerifyHash, req, &resp); err != nil {
		return false, err
	}

	if resp.Valid == nil {
		return false, badResponse(pathVerifyHash, "valid", "missing")
	}

	return *resp.Valid, nil
}

func (c *Client) ciphertextCall(ctx context.Context, path string, req any) (ct []byte, ms int64, err error) {
	var resp ciphertextResponse
	if err := c.do(ctx, c.http, path, req, &resp); err != nil {
		return nil, 0, err
	}

	ct, err = decodeB64(path, "ciphertext", resp.Ciphertext)
	if err != nil {
		return nil, 0, err
	}

	return ct, resp.DurationMs, nil
}

func (c *Client) dataCall(ctx context.Context, path string, req any) (data []byte, ms int64, err error) {
	var resp dataResponse
	if err := c.do(ctx, c.http, path, req, &resp); err != nil {
		return nil, 0, err
	}

	data, err = decodeB64(path, "data", resp.Data)
	if err != nil {
		return nil, 0, err
	}

	return data, resp.DurationMs, nil
}

// do выполняет POST path с JSON-телом req и разбирает ответ 200 в resp.
//
// Тела запроса и ответа не логируются: в них ключи, PEM и открытый текст.
func (c *Client) do(ctx context.Context, hc *http.Client, path string, req, resp any) error {
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("cryptoclient: %s: encode request: %w", path, err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base.JoinPath(path).String(), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("cryptoclient: %s: build request: %w", path, err)
	}

	httpReq.Header.Set("Content-Type", contentTypeJSON)
	httpReq.Header.Set("Accept", contentTypeJSON)

	if id := httpx.RequestIDFrom(ctx); id != "" {
		httpReq.Header.Set(httpx.HeaderRequestID, id)
	}

	start := time.Now()

	res, err := hc.Do(httpReq)
	if err != nil {
		return transportError(ctx, path, err)
	}
	defer res.Body.Close()

	raw, err := readLimited(res.Body)
	if err != nil {
		return transportError(ctx, path, err)
	}

	c.log.DebugContext(ctx, "crypto call",
		"path", path, "status", res.StatusCode, "duration_ms", time.Since(start).Milliseconds())

	switch {
	case res.StatusCode == http.StatusOK:
		return decodeResponse(path, res.Header.Get("Content-Type"), raw, resp)
	case isBadInputStatus(res.StatusCode):
		return newBadInputError(path, res.StatusCode, raw)
	default:
		return &UnavailableError{Op: path, Err: fmt.Errorf("unexpected status %d", res.StatusCode)}
	}
}

// isBadInputStatus: 4xx — модуль отверг данные. 408 и 429 говорят о состоянии
// модуля, а не о запросе, поэтому считаются недоступностью: иначе клиент
// сервиса получил бы 400 за перегрузку чужого компонента.
func isBadInputStatus(status int) bool {
	return status >= 400 && status < 500 &&
		status != http.StatusRequestTimeout && status != http.StatusTooManyRequests
}

// transportError: отмена контекста вызывающим — не сбой модуля, она
// пробрасывается как context.Canceled. Остальное, включая истёкший дедлайн, —
// недоступность.
func transportError(ctx context.Context, path string, err error) error {
	if errors.Is(ctx.Err(), context.Canceled) {
		return fmt.Errorf("cryptoclient: %s: %w", path, context.Canceled)
	}

	return &UnavailableError{Op: path, Err: err}
}

func readLimited(r io.Reader) ([]byte, error) {
	raw, err := io.ReadAll(io.LimitReader(r, maxResponseBytes+1))
	if err != nil {
		return nil, err
	}

	if len(raw) > maxResponseBytes {
		return nil, errResponseTooLarge
	}

	return raw, nil
}

func decodeResponse(path, contentType string, raw []byte, resp any) error {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil || mediaType != contentTypeJSON {
		return &UnavailableError{Op: path, Err: fmt.Errorf("unexpected content type %q", contentType)}
	}

	if err := json.Unmarshal(raw, resp); err != nil {
		return &UnavailableError{Op: path, Err: fmt.Errorf("decode response: %w", err)}
	}

	return nil
}

func encodeB64(b []byte) string {
	return base64.StdEncoding.EncodeToString(b)
}

func decodeB64(path, field, s string) ([]byte, error) {
	if s == "" {
		return nil, badResponse(path, field, "empty")
	}

	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, badResponse(path, field, "not base64")
	}

	return b, nil
}

// checkHex проверяет hex-поле ответа; wantLen > 0 задаёт длину в байтах.
func checkHex(path, field, s string, wantLen int) error {
	if s == "" {
		return badResponse(path, field, "empty")
	}

	b, err := hex.DecodeString(s)
	if err != nil {
		return badResponse(path, field, "not hex")
	}

	if wantLen > 0 && len(b) != wantLen {
		return badResponse(path, field, fmt.Sprintf("got %d bytes, want %d", len(b), wantLen))
	}

	return nil
}

func checkPEM(path, field, s string) error {
	if block, _ := pem.Decode([]byte(s)); block == nil {
		return badResponse(path, field, "not a PEM block")
	}

	return nil
}

func checkHashLen(path string, hash []byte) error {
	if len(hash) != hashLen {
		return fmt.Errorf("cryptoclient: %s: hash is %d bytes, want %d: %w", path, len(hash), hashLen, ports.ErrInvalidArgument)
	}

	return nil
}

// badResponse — ответ 200, нарушающий контракт. Значение поля в текст
// не попадает: там может быть ключ.
func badResponse(path, field, reason string) error {
	return &UnavailableError{Op: path, Err: fmt.Errorf("bad response field %s: %s", field, reason)}
}
