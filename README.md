# COORD — координационный сервис secure-messaging-platform

Сервис принимает сообщения, шифрует их через Крипто-модуль, хранит ключи
в запечатанном виде, отдаёт сообщения владельцу и администратору, формирует
PDF с подписью и проверяет подпись загруженного файла.

Требования и контракты лежат в `docs/` (`contracts.md` — источник истины),
порядок работ — в `docs/work-plan.md`, правила ветвления — в `gitflow.md`.

## Состояние ветки

Реализован каркас сервиса: конфигурация, единый формат ответов и ошибок,
middleware, проверки доступности. Прикладных маршрутов пока нет.

| Маршрут | Назначение |
|---|---|
| `GET /health` | проверка доступности процесса |
| `GET /api/health` | то же для внешнего шлюза |

Оба маршрута не обращаются к базе и внешним сервисам.

## Требования

- Go 1.27+
- Docker и Docker Compose (для стенда)
- `make` — необязательно, все цели повторяются командами `go`

## Конфигурация

Скопируйте `.env.example` в `.env` и заполните значения. Обязательны
`HTTP_ADDR`, `DSN`, `MASTER_KEY`, `ACCOUNTS_URL`, `CRYPTO_URL`; остальные
переменные имеют значения по умолчанию.

| Переменная | Обязательна | По умолчанию |
|---|---|---|
| `HTTP_ADDR` | да | — |
| `DSN` | да | — |
| `MASTER_KEY` | да | — (64 hex-символа) |
| `ACCOUNTS_URL` | да | — |
| `CRYPTO_URL` | да | — |
| `RSA_BITS` | нет | `32768` |
| `CORS_ORIGINS` | нет | пусто (CORS выключен) |
| `LOG_LEVEL` | нет | `info` |
| `HTTP_READ_HEADER_TIMEOUT` | нет | `5s` |
| `HTTP_READ_TIMEOUT` | нет | `15s` |
| `HTTP_WRITE_TIMEOUT` | нет | `60s` |
| `HTTP_IDLE_TIMEOUT` | нет | `60s` |
| `SHUTDOWN_TIMEOUT` | нет | `15s` |
| `ACCOUNTS_TIMEOUT` | нет | `3s` |
| `CRYPTO_TIMEOUT` | нет | `10s` |
| `RSA_KEYGEN_TIMEOUT` | нет | `60m` |
| `MAX_BODY_AUTH` | нет | `1MiB` |
| `MAX_BODY_USERS` | нет | `64KiB` |
| `MAX_MESSAGE_BYTES` | нет | `64KiB` |
| `MAX_PDF_UPLOAD` | нет | `10MiB` |

Мастер-ключ и `DSN` не попадают ни в лог, ни в строковое представление
конфигурации. Сгенерировать локальный ключ:

```powershell
-join ((1..32) | ForEach-Object { '{0:x2}' -f (Get-Random -Max 256) })
```

Без обязательных переменных сервис завершается с ненулевым кодом и печатает
список всех найденных проблем.

## Запуск

Сервис читает только переменные окружения, `.env` сам он не разбирает.
Поэтому локально нужен один из двух способов ниже.

### Через make

`make` подключает `.env` из корня и экспортирует переменные в окружение:

```powershell
make run
```

### Без make

Переменные загружаются из `.env` в текущую сессию PowerShell:

```powershell
Get-Content .env | Where-Object { $_ -match '^\s*[^#].*=' } | ForEach-Object {
    $k, $v = $_ -split '=', 2
    Set-Item -Path "env:$($k.Trim())" -Value $v.Trim()
}
go run ./cmd/coord
```

Либо те же значения задаются явно:

```powershell
$env:HTTP_ADDR=":8080"
$env:DSN="postgres://coord:coord_local@localhost:5432/coord?sslmode=disable"
$env:MASTER_KEY="<64 hex-символа>"
$env:ACCOUNTS_URL="http://localhost:8081"
$env:CRYPTO_URL="http://localhost:8082"
$env:RSA_BITS="2048"
go run ./cmd/coord
```

Проверка в другом терминале:

```powershell
Invoke-RestMethod http://localhost:8080/health
Invoke-RestMethod http://localhost:8080/api/health
```

Ожидаемый ответ на оба адреса: `{"status":"ok"}`.

### В контейнерах

PostgreSQL и сервис вместе:

```powershell
make up          # или: docker compose up -d --build
docker compose ps
make down
```

Только PostgreSQL, если сервис запускается локально:

```powershell
make infra
```

В образе нет shell и пользователя root, а `HEALTHCHECK` вызывает бинарь
`/healthcheck`: он читает `HTTP_ADDR` и запрашивает `/health`.

## Проверки

```powershell
go build ./...
go vet ./...
gofmt -l .
go test ./...
```

Те же проверки через `make`:

```powershell
make build
make fmt-check
make vet
make lint
make test
make cover
make check       # fmt-check + vet + lint + test
```

`make test` и `make cover` используют `-race`, для этого нужен cgo:
на Windows — компилятор C (`gcc`) и `CGO_ENABLED=1`.

## Структура

```
cmd/coord/           запуск, graceful shutdown
cmd/healthcheck/     проверка доступности для HEALTHCHECK
internal/config/     конфигурация из env и её валидация
internal/httpx/      единый формат ответов и ошибок, middleware
internal/api/        сборка маршрутов; routes_deva.go и routes_devb.go — зоны разработчиков
```

Пакеты `internal/ports`, `internal/sealed`, `internal/pdfsig`, `internal/store`,
`internal/cryptoclient` и остальные прикладные пакеты появятся в следующих задачах.

## Формат ошибок

Все ошибки отдаются в JSON:

```json
{"error":{"code":"validation","message":"Недопустимое значение limit"}}
```

Коды и соответствующие им статусы зафиксированы в `docs/contracts.md`, раздел 4.

## Ограничения

- Контейнер подписи PDF — собственный формат после `%%EOF`, не PAdES.
  Часть просмотрщиков может показать предупреждение о структуре файла.
- Генерация ключа RSA рабочего размера занимает десятки минут.
  Первый запрос на шифрование в это время получает `409 key_generating`.
- Ротации мастер-ключа нет.
