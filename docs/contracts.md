# Контракты Координационного сервиса (общие для Dev A и Dev B)

> Единый источник истины для стыковки пакетов.
> Любое изменение этого файла — отдельный PR с пометкой «изменение контракта» в описании.
> Файл должен лежать в ветке `COORD-develop` в каталоге `docs/`, иначе агенты его не увидят.
>
> Задачи, ссылающиеся на этот файл: `COORD-2` (создание), далее все задачи T1…T15.
>
> Порядок работ и критерии приёмки — в `work-plan.md`. Правила написания кода и текста —
> в `styleguide.md`.

---

## 1. Структура проекта

```
cmd/coord/main.go              сборка зависимостей, запуск, graceful shutdown
cmd/healthcheck/main.go        бинарь для HEALTHCHECK в distroless

internal/config/               конфигурация из env                        (T1, Dev A)
internal/httpx/                writeErr, writeJSON, Recover, RequestID,
                               Logging, CORS, BodyLimit                    (T1, Dev A)
internal/api/                  сборка маршрутов, Register(mux, deps)       (T1, Dev A)

internal/auth/                 клиент Accounts, Authn, Require, /me,
                               /roles, reverse-proxy                       (T4, T5, Dev A)
internal/methods/              GET/PATCH методов, MethodEnabled            (T6, Dev A)
internal/opslog/               журнал операций                             (T9, Dev A)
internal/messages/             DTO, чтение сообщений                       (T10, Dev A)
internal/verify/               POST /api/verify                            (T13, Dev A)

internal/keys/                 EnsureUserKey, системный ключ               (T7, Dev B)
internal/cryptoclient/         HTTP-клиент Крипто-модуля                   (T3, Dev B)
internal/store/                PostgreSQL                                  (T2, Dev B)
internal/messages/             запись, дешифрование, PDF                   (T8, T11, T12, Dev B)

internal/ports/                типы, интерфейсы, sentinel-ошибки           (COORD-2)
internal/sealed/               формат запечатывания ключей v1              (COORD-2)
internal/pdfsig/               контейнер подписи PDF v1                    (COORD-2)

migrations/001_init.sql        схема БД                                    (T2, Dev B)
migrations/migrations.go       пакет с //go:embed *.sql                    (T2, Dev B)
docs/                          coordination-*.md, contracts.md
```

### Правила разграничения

1. Пакет-потребитель объявляет нужный ему интерфейс в `internal/ports`; реализация живёт
   в `internal/store` и `internal/cryptoclient`.
2. `internal/store` и `internal/cryptoclient` **не импортируют** `auth`, `messages`,
   `methods`, `verify`, `keys`.
3. `internal/messages` содержит файлы обоих разработчиков. Чтобы не конфликтовать по файлам,
   они разделены по именам:

   | Файл | Владелец | Содержимое |
   |---|---|---|
   | `dto.go` | COORD-2 | DTO и преобразование доменных типов |
   | `read.go`, `read_test.go` | Dev A (T10) | `GET /api/messages`, `GET /api/messages/{id}` |
   | `write.go`, `write_test.go` | Dev B (T8) | `POST /api/messages` |
   | `decrypt.go`, `decrypt_test.go` | Dev B (T11) | `POST /api/decrypt` |
   | `pdf.go`, `pdf_test.go` | Dev B (T12) | `GET /api/messages/{id}/pdf` |

   Файл `dto.go` меняется только через PR с пометкой «изменение контракта».
4. `Register` каждого пакета принимает свои зависимости. Общий `Deps` не создаётся —
   это порождает циклические импорты и конфликты при параллельной работе.
5. Сборка маршрутов разделена по владельцам, чтобы не конфликтовать по одному файлу:

   | Файл | Владелец | Содержимое |
   |---|---|---|
   | `internal/api/router.go` | Dev A | `New(...)`, `GET /health`, `GET /api/health`, вызов `registerDevB` |
   | `internal/api/routes_deva.go` | Dev A | регистрация `auth`, `methods`, `messages` (read), `verify` |
   | `internal/api/routes_devb.go` | Dev B | `registerDevB(mux, deps)` → `messages` (write, decrypt, pdf) |

   В `COORD-2` создаётся `routes_devb.go` с пустой реализацией `registerDevB`.
   Dev B наполняет его в своих задачах, Dev A файл не трогает.

---

## 2. Доменные типы (`internal/ports`)

```go
package ports

import (
	"context"
	"errors"
	"time"
)

type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

type Method string

const (
	MethodRSA        Method = "rsa"
	MethodKuznyechik Method = "kuznyechik"
)

const (
	KeyStatusGenerating = "generating"
	KeyStatusReady      = "ready"
	KeyStatusFailed     = "failed"
)

// Sentinel-ошибки. Возвращаются реализациями, проверяются через errors.Is.
var (
	ErrNotFound          = errors.New("not found")
	ErrConflict          = errors.New("conflict")
	ErrCryptoBadInput    = errors.New("crypto: bad input")
	ErrCryptoUnavailable = errors.New("crypto: unavailable")
	ErrUnauthorized      = errors.New("unauthorized")
)

type MethodRow struct {
	Code    string
	Title   string
	KeySize int
	Enabled bool
}

type UserKey struct {
	UserID        int64
	Bits          int
	PublicKeyPEM  string
	PrivateSealed []byte // формат sealed v1, раздел 6
	KeygenMs      int64
	Status        string
	UpdatedAt     time.Time
}

type Message struct {
	ID           int64
	OwnerID      int64
	OwnerLogin   string
	Method       string
	KeySize      int
	Plaintext    string
	Ciphertext   []byte
	KuzKeySealed []byte // формат sealed v1
	KuzIV        []byte // IV шифрования сообщения, НЕ для sealed
	KeygenMs     int64
	EncryptMs    int64
	DecryptMs    int64
	CreatedAt    time.Time
}

type Principal struct {
	UserID int64
	Login  string
	Role   Role
}
```

`MethodRow` называется так намеренно: имя `Method` уже занято строковым типом.

---

## 3. Интерфейсы

```go
type Store interface {
	Methods(ctx context.Context) ([]MethodRow, error)
	// SetMethodEnabled возвращает ErrNotFound, если code отсутствует.
	SetMethodEnabled(ctx context.Context, code string, on bool) error
	// UserKey возвращает ErrNotFound, если записи нет.
	UserKey(ctx context.Context, userID int64) (*UserKey, error)
	SaveUserKey(ctx context.Context, k *UserKey) error
	CreateMessage(ctx context.Context, m *Message) (int64, error)
	GetMessage(ctx context.Context, id int64) (*Message, error)
	// ownerID == nil — все сообщения (admin). Иначе только свои.
	// Сортировка: created_at DESC, id DESC. limit 1..100, offset >= 0.
	ListMessages(ctx context.Context, ownerID *int64, limit, offset int) ([]*Message, error)
	SetDecryptMs(ctx context.Context, id, ms int64) error
	// userID == nil — анонимный субъект (NULL в БД), НЕ 0.
	LogOp(ctx context.Context, userID *int64, op, ref string, ok bool) error
}

type CryptoClient interface {
	KuzKeys(ctx context.Context) (keyHex, ivHex string, err error)
	KuzEncrypt(ctx context.Context, keyHex, ivHex string, data []byte) (ct []byte, ms int64, err error)
	KuzDecrypt(ctx context.Context, keyHex, ivHex string, ct []byte) (data []byte, ms int64, err error)
	RSAKeys(ctx context.Context, bits int) (pub, priv string, ms int64, err error)
	RSAEncrypt(ctx context.Context, pubPEM string, data []byte) (ct []byte, ms int64, err error)
	RSADecrypt(ctx context.Context, privPEM string, ct []byte) (data []byte, ms int64, err error)
	Streebog512(ctx context.Context, data []byte) ([]byte, error)
	// SignHash/VerifyHash: hash уже посчитан вызывающей стороной.
	// Крипто-модуль НЕ хеширует повторно.
	SignHash(ctx context.Context, privPEM string, hash []byte) ([]byte, error)
	VerifyHash(ctx context.Context, pubPEM string, hash, sig []byte) (bool, error)
}

type AccountsClient interface {
	// Introspect возвращает ports.ErrUnauthorized при active=false, stage!=full, 401/403.
	// При сети/таймауте/5xx/429/битом JSON — auth.ErrAccountsUnavailable.
	Introspect(ctx context.Context, token string) (Principal, error)
}
```

`ErrAccountsUnavailable` объявляется в `internal/auth`, чтобы не смешивать с крипто-ошибками.

Проверка доступности метода (`internal/methods`, экспортируется для `COORD-10`):

```go
// MethodEnabled возвращает (false, nil) для неизвестного кода.
// Ошибка БД пробрасывается как есть.
func MethodEnabled(ctx context.Context, store ports.Store, code string) (bool, error)
```

Применяется **только к шифрованию**. Расшифрование выключенным методом разрешено:
пользователь должен прочитать ранее созданные сообщения.

Журнал (`internal/opslog`):

```go
func (l *Logger) Log(ctx context.Context, userID *int64, op, ref string, ok bool)
```

---

## 4. HTTP: ошибки

Формат ответа:

```json
{"error":{"code":"method_disabled","message":"Метод шифрования выключен администратором"}}
```

| HTTP | `code` | Когда |
|---|---|---|
| 400 | `validation` | неверное тело, пустой текст, плохой limit/offset, нет `enabled`, битый multipart |
| 400 | `bad_key` | битый PEM, неверная длина ключа/IV |
| 400 | `bad_ciphertext` | шифртекст не hex / пустой |
| 401 | `unauthorized` | нет/битый токен, сессия неполная |
| 403 | `forbidden` | не хватает роли |
| 404 | `not_found` | нет ресурса или чужое сообщение |
| 405 | `method_not_allowed` | метод не поддержан маршрутом |
| 409 | `method_disabled` | метод выключен администратором |
| 409 | `key_generating` | RSA-ключ ещё генерируется |
| 413 | `payload_too_large` | тело/файл больше лимита |
| 415 | `unsupported_media_type` | не JSON / не multipart |
| 429 | `rate_limited` | превышен лимит на публичный `/api/verify` |
| 500 | `internal` | паника, ошибка БД, ошибка распечатывания ключа |
| 503 | `accounts_unavailable` | Сервис учётных записей недоступен |
| 503 | `crypto_unavailable` | Крипто-модуль недоступен |

Требования к `writeErr`:

- `Content-Type: application/json; charset=utf-8`
- `Cache-Control: no-store`
- `X-Content-Type-Options: nosniff`
- не писать тело, если заголовки уже отправлены;
- никогда не использовать `http.Error` — он отдаёт `text/plain`;
- `405`, `413`, `415`, `500` тоже отдаются в едином JSON-формате.

---

## 5. HTTP: DTO

```go
type RoleDTO struct {
	Code        Role     `json:"code"`
	Title       string   `json:"title"`
	Screens     []string `json:"screens"`
	Permissions []string `json:"permissions"`
}

type MethodDTO struct {
	Code    Method `json:"code"`
	Title   string `json:"title"`
	KeySize int    `json:"key_size"`
	Enabled bool   `json:"enabled"`
}

type MethodPatchRequest struct {
	Enabled *bool `json:"enabled"` // nil -> 400 validation
}

type EncryptRequest struct {
	Text   string `json:"text"`
	Method Method `json:"method"`
}

type MessageDTO struct {
	ID         int64     `json:"id"`
	Owner      string    `json:"owner"`
	CreatedAt  time.Time `json:"created_at"`
	Method     Method    `json:"method"`
	KeySize    int       `json:"key_size"`
	Plaintext  string    `json:"plaintext"`
	Ciphertext string    `json:"ciphertext"` // lowercase hex
	Keys       *KeysDTO  `json:"keys,omitempty"`
	Metrics    Metrics   `json:"metrics"`
}

type KeysDTO struct {
	PublicKeyPEM  string `json:"public_key_pem,omitempty"`
	PrivateKeyPEM string `json:"private_key_pem,omitempty"`
	KeyHex        string `json:"key_hex,omitempty"`
	IVHex         string `json:"iv_hex,omitempty"`
}

type Metrics struct {
	KeygenMs  int64 `json:"keygen_ms"`
	EncryptMs int64 `json:"encrypt_ms"`
	DecryptMs int64 `json:"decrypt_ms"`
}

type DecryptRequest struct {
	Method        Method `json:"method"`
	Ciphertext    string `json:"ciphertext"`
	PrivateKeyPEM string `json:"private_key_pem,omitempty"`
	KeyHex        string `json:"key_hex,omitempty"`
	IVHex         string `json:"iv_hex,omitempty"`
	MessageID     int64  `json:"message_id,omitempty"`
}

type DecryptResponse struct {
	Plaintext string `json:"plaintext"`
	DecryptMs int64  `json:"decrypt_ms"`
}

type VerifyResponse struct {
	Valid    bool       `json:"valid"`
	Signer   string     `json:"signer,omitempty"`
	SignedAt *time.Time `json:"signed_at,omitempty"` // omitempty для time.Time не работает
}
```

---

## 6. Формат запечатывания ключей (sealed v1)

Закрытые ключи RSA и ключи Кузнечика хранятся в БД только запечатанными под `MASTER_KEY`.

```
sealed = 0x01 | iv[16] | ciphertext | mac[32]
mac    = HMAC-SHA256(macKey, sealed[0 : len(sealed)-32])
macKey = HMAC-SHA256(MASTER_KEY, "smp/sealed/v1")
```

- `MASTER_KEY` — 32 байта (64 hex-символа), валидируется на старте.
- IV — случайный, 16 байт; берётся из `POST /kuznyechik/keys` (сгенерированный ключ
  отбрасывается, используется только `iv_hex`).
- Шифрование — `POST /kuznyechik/encrypt` с `key_hex = MASTER_KEY`.
- MAC даёт целостность контейнера средствами stdlib и не заменяет Крипто-модуль.
  Если модуль предоставит AEAD (Кузнечик-GCM), MAC убирается, версия становится `0x02`.
- Сравнение MAC — `hmac.Equal` (constant time).
- `len(sealed) < 1+16+1+32` → ошибка формата.
- Ошибка распечатывания → **500 internal**, не `400 bad_key`
  (пользователь ключ не передавал, это внутренняя проблема).

API:

```go
package sealed

func Seal(ctx context.Context, c ports.CryptoClient, masterKeyHex string, plain []byte) ([]byte, error)
func Unseal(ctx context.Context, c ports.CryptoClient, masterKeyHex string, blob []byte) ([]byte, error)
```

`kuz_iv` в таблице `messages` — IV **шифрования сообщения**, к sealed-контейнеру
отношения не имеет. Это разные вещи, путать нельзя.

---

## 7. Контейнер подписи PDF (pdfsig v1)

`rawPDF` — PDF, заканчивающийся `%%EOF` (допускается завершающий `\r\n`).

```text
meta = {
  "version": 1,
  "signer": "<login>",
  "signed_at": "<RFC3339 UTC>",
  "key_fingerprint": "<sha256 hex от DER SPKI>",
  "pdf_hash_hex": "<Стрибог-512 от rawPDF>",
  "hash_alg": "streebog512",
  "sig_alg": "rsa-pkcs1v15"
}
metaJSON = json.Marshal(meta)              // struct, порядок полей фиксирован
digest   = Streebog512(pdfHash || metaJSON)
sig      = SignHash(systemPriv, digest)
```

Блок дописывается после `%%EOF`:

```text
%SMP-SIG-V1-META
%<base64(metaJSON), строки по 76 символов, каждая с префиксом %>
%SMP-SIG-V1-SIG
%<base64(sig), строки по 76 символов, каждая с префиксом %>
%%SMP-SIG-END
```

Правила разбора (`pdfsig.Parse`):

1. Искать последнее вхождение `%%SMP-SIG-END` в последних 64 КиБ файла.
2. После маркера допускаются только `\r` и `\n`.
3. Найти `%SMP-SIG-V1-SIG` и `%SMP-SIG-V1-META` строго в этом порядке.
4. `signedBytes = file[:начало %SMP-SIG-V1-META]`; проверить, что `signedBytes`
   оканчивается на `%%EOF`.
5. Декодировать `metaJSON` и `sig`; проверить `version == 1`, `hash_alg`, `sig_alg`,
   длину `sig` = размер ключа / 8.
6. Проверить `meta.pdf_hash_hex == Streebog512(signedBytes)`.
7. Проверить `meta.key_fingerprint` против системного публичного ключа.
8. Пересчитать `digest` и вызвать `VerifyHash`.
9. Любое отклонение (нет блока, несколько блоков, мусор после маркера, неверная длина,
   битый base64) → `valid: false`, HTTP `200`.

API:

```go
package pdfsig

type Meta struct {
	Version        int    `json:"version"`
	Signer         string `json:"signer"`
	SignedAt       string `json:"signed_at"`
	KeyFingerprint string `json:"key_fingerprint"`
	PDFHashHex     string `json:"pdf_hash_hex"`
	HashAlg        string `json:"hash_alg"`
	SigAlg         string `json:"sig_alg"`
}

type Signed struct {
	Raw  []byte
	Meta Meta
	Sig  []byte
}

func (m Meta) JSON() ([]byte, error)
func Attach(rawPDF []byte, meta Meta, sig []byte) ([]byte, error)
func Parse(file []byte) (*Signed, error)
func Fingerprint(pubKeyPEM string) (string, error) // sha256(DER SPKI), hex
```

Ограничение: это **не PAdES**, а собственный контейнер после `%%EOF`.
Часть просмотрщиков может показать предупреждение о структуре файла — это ожидаемо
и должно быть указано в README.

---

## 8. Контракт `POST /internal/introspect`

Запрос:

```http
POST /internal/introspect
Content-Type: application/json

{"token":"<token>"}
```

Ответ `200`:

```json
{"active":true,"user_id":7,"login":"ivanov","role":"user","stage":"full"}
```

или

```json
{"active":false}
```

Правила `Authn`:

- Принимается только `active == true` **и** `stage == "full"`.
  Поле `stage` обязательно: отсутствие, пустая строка, `password`, `email` → `401`.
- `user_id > 0`, `login` непустой, `role` ∈ {`user`,`admin`} — иначе `401` (fail-closed).
- `200 active=false` → `401 unauthorized` «Сессия недействительна».
- `401`/`403` от Accounts → `401 unauthorized`.
- сеть, таймаут, `5xx`, `429`, битый JSON → `503 accounts_unavailable`.
- `context.Canceled` → ответ не пишется.
- Токен, тело и заголовок `Authorization` не логируются.
- Таймаут `3s`, лимит ответа `64 КиБ`, без редиректов, без ретраев.

---

## 9. Правила проксирования (`/api/auth/*`, `/api/users`)

- `httputil.ReverseProxy` с `Rewrite` (не только `Director`).
- Удалить входящие `X-Forwarded-For`, `Forwarded`, `X-Real-IP`, `X-User-Id`,
  `X-User-Role`, `X-Original-URL`, `X-Rewrite-URL`, `Proxy`.
- Затем `pr.SetXForwarded()` и `X-Request-ID`.
- `Host` = хост целевого сервиса.
- `Transport` с `DialContext`, `ResponseHeaderTimeout`, `IdleConnTimeout`.
- `ErrorHandler` → `503 accounts_unavailable` в JSON, иначе будет `502` + `text/plain`.
- `Authorization` пробрасывается как есть.
- Логи: метод, путь, статус, длительность, `request_id`; **без** тела, query для `/auth/*`,
  `Authorization`, `Cookie`.
- Лимиты тела: `/api/auth/*` — 1 МиБ, `POST /api/users` — 64 КиБ.
- `POST /api/auth/logout` — **защищённый** (`Authn`), затем проксирование.
  Это уточнение к `CS-G7`. При ином решении со стороны Accounts контракт меняется.

---

## 10. Журнал операций (`ops_log`)

| `op` | `ref` | `ok` |
|---|---|---|
| `message.encrypt` | `id` сообщения | `true` / `false` |
| `message.decrypt` | `id` сообщения или пусто | `true` / `false` |
| `message.pdf` | `id` сообщения | `true` / `false` |
| `signature.verify` | `sha256:<16 hex>` от файла | результат проверки |
| `method.toggle` | код метода | `true` / `false` |
| `access.denied` | `<METHOD> <path>` (без query, ≤ 200 символов) | `false` |

- `user_id` = `NULL` для анонимного субъекта (`POST /api/verify` без токена).
  Значение `0` зарезервировано под системный ключ, использовать его нельзя.
- Запись выполняется через `context.WithoutCancel(ctx)` + таймаут `2s`.
- Ошибка записи журнала не меняет HTTP-ответ; логируется на уровне `error`.
- Запрещено писать: тексты сообщений, шифртекст, ключи, токены, `MASTER_KEY`, PEM.

---

## 11. Пагинация и выборки

- `limit` по умолчанию `20`, максимум `100`; `offset >= 0`; иначе `400 validation`.
- Сортировка: `created_at DESC, id DESC` — стабильная пагинация.
- `ownerID == nil` — все сообщения (только `admin`), иначе фильтр по владельцу.
- Поле `keys` отдаётся владельцу и в списке, и в карточке. Для `admin` по чужим — `nil`.
  Если размер ответа станет проблемой, вводится флаг `MESSAGES_KEYS_IN_LIST=false`
  отдельным PR с согласованием Front.

---

## 12. Конфигурация (env)

| Переменная | Обязательна | По умолчанию | Примечание |
|---|---|---|---|
| `HTTP_ADDR` | да | — | `:8080` |
| `DSN` | да | — | PostgreSQL DSN |
| `MASTER_KEY` | да | — | 64 hex-символа (32 байта) |
| `ACCOUNTS_URL` | да | — | `http://auth:8081` |
| `CRYPTO_URL` | да | — | `http://crypto:8082` |
| `RSA_BITS` | нет | `32768` | ≥ 2048, кратно 8 |
| `CORS_ORIGINS` | нет | пусто | через запятую; пусто = CORS выключен |
| `LOG_LEVEL` | нет | `info` | `debug`/`info`/`warn`/`error` |
| `HTTP_READ_HEADER_TIMEOUT` | нет | `5s` | |
| `HTTP_READ_TIMEOUT` | нет | `15s` | |
| `HTTP_WRITE_TIMEOUT` | нет | `60s` | |
| `HTTP_IDLE_TIMEOUT` | нет | `60s` | |
| `SHUTDOWN_TIMEOUT` | нет | `15s` | |
| `ACCOUNTS_TIMEOUT` | нет | `3s` | |
| `CRYPTO_TIMEOUT` | нет | `10s` | |
| `RSA_KEYGEN_TIMEOUT` | нет | `60m` | верхняя граница фоновой генерации |
| `MAX_BODY_AUTH` | нет | `1MiB` | |
| `MAX_BODY_USERS` | нет | `64KiB` | |
| `MAX_MESSAGE_BYTES` | нет | `64KiB` | размер исходного текста |
| `MAX_PDF_UPLOAD` | нет | `10MiB` | `POST /api/verify` |

`MASTER_KEY` не логируется и не попадает в `/health` и `/api/me`.

В `Config` мастер-ключ хранится как `MasterKey [32]byte`. Пакет `sealed` принимает
hex-строку, поэтому `Config` предоставляет:

```go
func (c *Config) MasterKeyHex() string // hex.EncodeToString(c.MasterKey[:])
```

Метод `String()` у `Config` и любой его лог-вывод обязаны скрывать `DSN` и `MasterKey`.

---

## 13. Схема БД (T2)

```sql
CREATE TABLE IF NOT EXISTS methods (
    code     TEXT PRIMARY KEY CHECK (code IN ('rsa','kuznyechik')),
    title    TEXT NOT NULL,
    key_size INT  NOT NULL CHECK (key_size > 0),
    enabled  BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE IF NOT EXISTS user_keys (
    user_id        BIGINT PRIMARY KEY CHECK (user_id >= 0),
    bits           INT    NOT NULL,
    public_key_pem TEXT,
    private_sealed BYTEA,
    keygen_ms      BIGINT NOT NULL DEFAULT 0,
    status         TEXT   NOT NULL CHECK (status IN ('generating','ready','failed')),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS messages (
    id             BIGSERIAL PRIMARY KEY,
    owner_id       BIGINT NOT NULL,
    owner_login    TEXT   NOT NULL,
    method         TEXT   NOT NULL REFERENCES methods(code),
    key_size       INT    NOT NULL,
    plaintext      TEXT   NOT NULL,
    ciphertext     BYTEA  NOT NULL,
    kuz_key_sealed BYTEA,
    kuz_iv         BYTEA,
    keygen_ms      BIGINT NOT NULL DEFAULT 0,
    encrypt_ms     BIGINT NOT NULL DEFAULT 0,
    decrypt_ms     BIGINT NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS messages_owner_created_idx
    ON messages (owner_id, created_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS ops_log (
    id      BIGSERIAL PRIMARY KEY,
    at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    user_id BIGINT,
    op      TEXT NOT NULL,
    ref     TEXT,
    ok      BOOLEAN NOT NULL
);

CREATE INDEX IF NOT EXISTS ops_log_at_idx ON ops_log (at DESC);
```

Начальные записи:

```sql
INSERT INTO methods (code, title, key_size, enabled)
VALUES ('rsa', 'RSA', $1, TRUE),
       ('kuznyechik', 'Кузнечик', 256, TRUE)
ON CONFLICT (code) DO UPDATE SET key_size = EXCLUDED.key_size;
```

`$1` = `RSA_BITS`. `enabled` при конфликте не перезаписывается: админ мог выключить метод.

---

## 14. Крипто-модуль: запрашиваемые эндпоинты

Бинарные данные — base64, ключи и IV Кузнечика — hex, ключи RSA — PEM (PKCS#1).

| Метод и путь | Запрос | Ответ |
|---|---|---|
| `POST /rsa/keys` | `{"bits":32768}` | `{"public_key_pem","private_key_pem","duration_ms"}` |
| `POST /rsa/encrypt` | `{"public_key_pem","data"}` | `{"ciphertext","duration_ms"}` |
| `POST /rsa/decrypt` | `{"private_key_pem","ciphertext"}` | `{"data","duration_ms"}` |
| `POST /kuznyechik/keys` | `{}` | `{"key_hex","iv_hex"}` |
| `POST /kuznyechik/encrypt` | `{"key_hex","iv_hex","data"}` | `{"ciphertext","duration_ms"}` |
| `POST /kuznyechik/decrypt` | `{"key_hex","iv_hex","ciphertext"}` | `{"data","duration_ms"}` |
| `POST /streebog512` | `{"data"}` | `{"hash_hex"}` |
| `POST /sign-hash` | `{"private_key_pem","hash_hex","hash_alg":"streebog512"}` | `{"signature"}` |
| `POST /verify-hash` | `{"public_key_pem","hash_hex","hash_alg":"streebog512","signature"}` | `{"valid"}` |

`/sign-hash` и `/verify-hash` — требование к Крипто-модулю, устраняющее двойное хеширование.
Если модуль их не предоставляет, используется `/sign` и `/verify` с `data = digest`
**только после письменного подтверждения**, что модуль не хеширует внутри.

Ошибки: `4xx` → `ErrCryptoBadInput` (`400 bad_key` / `bad_ciphertext`),
сеть/таймаут/`5xx` → `ErrCryptoUnavailable` (`503 crypto_unavailable`).

---

## 15. Порядок изменения контрактов

1. Изменение оформляется отдельным PR: `docs/contracts.md` + код в `internal/ports`,
   `internal/sealed`, `internal/pdfsig`.
2. PR получает ревью обоих разработчиков; при затрагивании внешнего контракта —
   ещё и владельцев смежных сервисов.
3. Только после мержа задачи, зависящие от контракта, снимаются с блокировки.
