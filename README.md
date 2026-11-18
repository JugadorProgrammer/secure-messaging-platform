# COORD — координационный сервис secure-messaging-platform

Сервис принимает сообщения, шифрует их через Крипто-модуль, хранит ключи
в запечатанном виде, отдаёт сообщения владельцу и администратору, формирует
PDF с подписью и проверяет подпись загруженного файла.

Требования и контракты лежат в `docs/` (`contracts.md` — источник истины),
порядок работ — в `docs/work-plan.md`, правила ветвления — в `gitflow.md`.

## Состояние ветки

Реализован каркас сервиса: конфигурация, единый формат ответов и ошибок,
middleware, проверки доступности. Добавлено проксирование маршрутов входа
и управления пользователями в Сервис учётных записей, список методов
шифрования и его переключение администратором, журнал операций с чтением
администратором.

| Маршрут | Назначение |
|---|---|
| `GET /health` | проверка доступности процесса |
| `GET /api/health` | то же для внешнего шлюза |
| `POST /api/auth/signup` | регистрация, прокси → учётки |
| `POST /api/auth/login` | фактор 1 (пароль), прокси → учётки |
| `POST /api/auth/login/email-code` | фактор 2 (код из письма), прокси → учётки |
| `POST /api/auth/login/recovery-code` | фактор 3 (одноразовый код), выдаёт токен |
| `POST /api/auth/password/restore` | восстановление пароля, прокси → учётки |
| `POST /api/auth/password/confirm` | подтверждение восстановления, прокси → учётки |
| `POST /api/auth/logout` | выход; требует токен |
| `GET /api/users` | список пользователей; только роль `admin` |
| `POST /api/users` | создание пользователя; только роль `admin` |
| `GET /api/roles` | справочник ролей (`1` user, `2` admin); публичный |
| `GET /api/getAllEncryptionMethods` | методы шифрования `{name, enabled}`; публичный |
| `POST /api/setEncryptionMethodAvailability` | включить или выключить метод; только роль `admin` |
| `POST /api/encrypt` | зашифровать текст, сохранить сообщение; требует сессию |
| `POST /api/decrypt` | расшифровать шифртекст ключом из запроса; требует сессию |
| `GET /api/getAllMessages` | сообщения с ключами: свои, у `admin` — все; требует сессию |
| `POST /api/exportPdfWithSignature` | PDF своего сообщения с подписью системным ключом |
| `GET /api/ops-log` | журнал операций; только роль `admin` |

Сессия передаётся cookie `session` (её ставит Сервис авторизации) или
заголовком `Authorization: Bearer <token>`; заголовок, если он есть, важнее.
Методы называются так же, как в интерфейсе: `RSA-32768` и `Кузнечик`; коды
`rsa` и `kuznyechik` тоже принимаются.

Маршруты здоровья не обращаются к базе и внешним сервисам. Маршруты `/api/auth/*`
и `/api/users` требуют `ACCOUNTS_URL`: префикс `/api` срезается, до учётных
записей запрос доходит как `/auth/...` или `/users`. Тела запросов и ответов
в лог не пишутся.

Маршруты методов читают таблицу `methods`, поэтому без подключённого хранилища
не регистрируются. Переключение требует поля `name` и `enabled` (без них —
`400`, иначе пропущенное поле выключило бы метод), неизвестное имя — `404`,
тело не больше 4 КиБ, пишется в `ops_log` как `method.toggle`.

### Сообщения

`POST /api/decrypt` принимает `privateKey` для RSA и `key` для Кузнечика.
Интерфейс IV не передаёт: он берётся из сохранённого сообщения с тем же
шифртекстом (hash-индекс `messages_ciphertext_idx`), для внешнего шифртекста
`iv` можно передать явно. Время расшифрования записывается в метрики, только
если сообщение принадлежит пользователю.

`GET /api/getAllMessages` раскрывает ключи каждого сообщения, потому что их
показывает интерфейс. Администратор видит все сообщения, а значит и все ключи.

`POST /api/exportPdfWithSignature` отдаёт PDF только владельцу (чужое
сообщение — `404` и запись `access.denied`). Подпись — контейнер `pdfsig v1`
после `%%EOF` (contracts.md, раздел 7), не PAdES: просмотрщики PDF её не
показывают. Без готового системного ключа — `503`.

### Журнал операций

`GET /api/ops-log` отдаёт страницу журнала: `limit` — 20 по умолчанию,
максимум 100; `offset` — от 0. Значение вне этих границ — `400 validation`.
Сортировка: новые записи сверху (`at DESC, id DESC`), пустой журнал — `[]`.

Запись `user_id` равна `null` для анонимной операции (проверка подписи без
токена): идентификатор `0` занят системным ключом.

Журнал пишется по принципу «не помешать ответу»: сбой базы меняет только
строку в логе сервиса, но не код ответа. Запись отвязывается от контекста
запроса (`context.WithoutCancel`) и живёт до 2 секунд, поэтому обрыв
соединения клиента не теряет событие. Поля `op` и `ref` содержат только код
операции и ссылку на объект — коды методов, идентификаторы сообщений,
`"METHOD /path"` для отказа. Текстов сообщений, ключей и токенов в журнале
не бывает.

Записываемые операции: `message.encrypt`, `message.decrypt`, `message.pdf`,
`signature.verify`, `method.toggle`, `access.denied`. Подключены `access.denied`
(отказ роли на административном маршруте) и `method.toggle` (переключение
метода); остальные появятся вместе с шифрованием, PDF и проверкой подписи.

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
| `RSA_BITS` | нет | `32768` (от 2048 до 32768, кратно 8) |
| `CORS_ORIGINS` | нет | пусто (CORS выключен) |
| `LOG_LEVEL` | нет | `info` |
| `HTTP_READ_HEADER_TIMEOUT` | нет | `5s` |
| `HTTP_READ_TIMEOUT` | нет | `15s` |
| `HTTP_WRITE_TIMEOUT` | нет | `60s` |
| `HTTP_IDLE_TIMEOUT` | нет | `60s` |
| `SHUTDOWN_TIMEOUT` | нет | `15s` |
| `ACCOUNTS_TIMEOUT` | нет | `3s` |
| `CRYPTO_TIMEOUT` | нет | `30s` |
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

Тесты `internal/store` работают с настоящим PostgreSQL и требуют `TEST_DSN`.
Без переменной они пропускаются. Отдельная база для тестов:

```powershell
docker compose exec postgres createdb -U coord coord_test
$env:TEST_DSN="postgres://coord:coord_local@localhost:5432/coord_test?sslmode=disable"
go test -race ./internal/store/...
```

### CI

`.github/workflows/coord-ci.yml` прогоняет линтер, тесты и сборку образа на
каждой ветке `COORD-*` и PR. После влития в `COORD-develop` образ
публикуется в Docker Hub как `<DOCKERHUB_USERNAME>/smp-coord:test`, после
влития в `COORD-master` — как `:stable`, в обоих случаях ещё и
`:sha-<commit>`. Нужны секреты репозитория `DOCKERHUB_USERNAME` и
`DOCKERHUB_TOKEN` (access token Docker Hub с правом Read & Write).

## Миграции

Схема описана в `migrations/*.sql` и встроена в бинарь. При старте сервис
применяет файлы, которых ещё нет в таблице `schema_migrations`, в порядке имён,
каждый в отдельной транзакции. Ошибка миграции останавливает запуск.

Вместе с первой миграцией заполняется справочник `methods`: `rsa` с размером
ключа из `RSA_BITS` и `kuznyechik` с ключом 256 бит.

Новая миграция — новый файл со следующим номером, например `002_add_index.sql`.
Применённые файлы не редактируются.

## Структура

```text
cmd/coord/           запуск, graceful shutdown
cmd/healthcheck/     проверка доступности для HEALTHCHECK
internal/config/     конфигурация из env и её валидация
internal/httpx/      единый формат ответов и ошибок, middleware
internal/auth/       проверка токена (introspect), роли, reverse-proxy к учётным записям
internal/api/        сборка маршрутов; routes_deva.go и routes_devb.go — зоны разработчиков
internal/methods/    список методов шифрования и его переключение
internal/opslog/     запись журнала операций и его чтение администратором
internal/ports/      доменные типы, интерфейс Store, sentinel-ошибки
internal/store/      реализация Store на PostgreSQL (pgx), миграции
migrations/          SQL-миграции, встроенные через //go:embed
```

Пакеты `internal/sealed`, `internal/pdfsig`, `internal/cryptoclient`
и остальные прикладные пакеты появятся в следующих задачах.

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
