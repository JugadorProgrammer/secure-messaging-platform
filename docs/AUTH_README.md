# AUTH — сервис учётных записей

## Назначение

AUTH — микросервис платформы Secure Messaging Platform, отвечающий за учётные записи пользователей и аутентификацию.

На текущем этапе AUTH-4 реализован инфраструктурный каркас сервиса. Регистрация пользователей, авторизация, управление сессиями и ролями будут добавлены в последующих задачах.

## Технологии

- **C++20** — язык реализации.
- **Drogon** — HTTP REST API.
- **PostgreSQL** — основная СУБД.
- **CMake** — система сборки.
- **Docker / Docker Compose** — контейнеризация и локальное развёртывание.
- **vcpkg** — управление зависимостями для Windows.
- **PowerShell** — локальная интеграционная проверка HTTP.

## Архитектура

Сервис разделён на несколько логических уровней:

- `http` — обработка HTTP-запросов и формирование ответов.
- `services` — бизнес-логика сервиса.
- `repositories` — взаимодействие с базой данных.

Точка входа расположена в `src/main.cpp`.

Текущая реализация проверяет подключение к PostgreSQL при запуске и регистрирует HTTP-маршрут `/health`.

## Структура проекта

```text
config/
    config.example.json
    config.docker.json

include/auth/
    http/
    repositories/
    services/

src/
    http/
    repositories/
    services/
    main.cpp

tests/
    check_health.ps1

CMakeLists.txt
Dockerfile
compose.yaml
.env.example
.dockerignore
```

## Конфигурация

Настройки HTTP-сервера хранятся в JSON-файлах Drogon.

Переменная `AUTH_CONFIG_PATH` позволяет выбрать конфигурационный файл. Если она отсутствует, используется `config/config.example.json`.

Настройки базы данных:

| Переменная | Назначение |
|---|---|
| `AUTH_DB_HOST` | Адрес PostgreSQL |
| `AUTH_DB_PORT` | Порт PostgreSQL |
| `AUTH_DB_NAME` | Имя базы данных |
| `AUTH_DB_USER` | Имя пользователя |
| `PGPASSWORD` | Пароль подключения |
| `AUTH_CONFIG_PATH` | Путь к JSON-конфигурации |

Пароли не должны сохраняться в исходном коде или попадать в Git.

## Запуск через Docker Compose

Требуется установленный и запущенный Docker Desktop или Docker Engine с поддержкой Docker Compose.

1. Создать локальный файл `.env` на основе `.env.example`.
2. Указать пароль PostgreSQL.
3. Выполнить:

```bash
docker compose up -d --build
```

Проверить состояние:

```bash
docker compose ps
```

Посмотреть журнал AUTH:

```bash
docker compose logs auth --tail 50
```

Остановить окружение:

```bash
docker compose down
```

Данные PostgreSQL хранятся в отдельном Docker volume и сохраняются при обычном `docker compose down`.

## Сборка на Windows

Для локальной сборки требуются Visual Studio Build Tools 2022, CMake и vcpkg с поддержкой Drogon и PostgreSQL.

Установка зависимости через vcpkg:

```powershell
D:\vcpkg\vcpkg.exe install "drogon[postgres]:x64-windows" "curl:x64-windows" --recurse
```

Конфигурация проекта:

```powershell
cmake -S . -B build -G "Visual Studio 17 2022" -A x64 -DCMAKE_TOOLCHAIN_FILE=D:/vcpkg/scripts/buildsystems/vcpkg.cmake
```

Сборка:

```powershell
cmake --build build --config Debug
```

Запуск из корня репозитория:

```powershell
.\build\Debug\smp_auth.exe
```

Для локального запуска PostgreSQL должна быть доступна, а пароль — передан через `PGPASSWORD`.

## HTTP API

### GET /health

Проверка работоспособности HTTP-сервера.

Пример запроса:

```bash
curl -i http://127.0.0.1:8080/health
```

Успешный ответ — HTTP 200:

```json
{
  "status": "ok",
  "service": "auth"
}
```

Маршрут `/health` пока не выполняет отдельную проверку состояния БД на каждом запросе. Подключение к PostgreSQL проверяется при старте приложения.

### Управление пользователями

AUTH принимает `GET /users` и `POST /users`. Через COORD используются `GET /api/users` и `POST /api/users`.
Оба метода требуют действующую полную сессию администратора в заголовке `Authorization: Bearer <token>`.
Запрос с отсутствующей или недействительной сессией получает `401`, а с ролью `user` — `403`.

- `GET` возвращает `200` и массив объектов с полями `login`, `email`, `role`. Для учётной записи с обеими ролями возвращается `admin`, для записи без назначенной роли — `null`.
- `POST` принимает JSON с полями `login`, `email`, `password` и `role` (`user` или `admin`). При успехе возвращает `201` и массив из десяти кодов восстановления. Некорректные поля дают `400`, занятый логин или email — `409`.

На новой установке зарегистрируйте первую учётную запись через `POST /auth/signup` и назначьте ей роль `admin` из доверенного подключения к базе AUTH. В примере замените `first_admin` на логин этой записи.

```sql
INSERT INTO user_roles (user_id, role_id)
SELECT u.id, r.id
FROM users u CROSS JOIN roles r
WHERE u.login = 'first_admin' AND r.code = 'admin'
ON CONFLICT DO NOTHING;
```

## Проверка работоспособности

При запущенном сервисе на Windows:

```powershell
powershell.exe -NoProfile -File .\tests\check_health.ps1
```

Проверяется успешный HTTP-ответ и содержимое JSON.

Проверка является интеграционной и требует предварительно запущенного AUTH. Полноценные модульные тесты и их интеграция с CTest/CI будут добавляться отдельно.

## Текущее состояние

В рамках AUTH-4 подготовлены:

- каркас C++20-приложения;
- HTTP-сервер Drogon;
- маршрут `/health`;
- разделение каталогов по архитектурным слоям;
- проверка соединения с PostgreSQL;
- Dockerfile;
- запуск через Docker Compose;
- базовая HTTP-проверка.

Бизнес-логика регистрации, авторизации, управления пользователями и сессиями в рамках данной задачи не реализуется.

## Единый формат ошибок HTTP API

Согласно частному техническому заданию AUTH, прикладные ошибки возвращаются в едином JSON-формате:

```json
{
  "error": {
    "code": "INVALID_REQUEST",
    "message": "Invalid request parameters"
  }
}
```

Назначение полей:

- `error.code` — строковый машиночитаемый идентификатор ошибки.
- `error.message` — текстовое описание ошибки.

HTTP-статус должен соответствовать характеру ошибки и передаваться отдельно от JSON-тела.

Для формирования ответов используется общая функция:

```cpp
auth::http::make_error_response(
    drogon::k400BadRequest,
    "INVALID_REQUEST",
    "Invalid request parameters"
);
```

Функция объявлена в `include/auth/http/error_response.h`, реализация расположена в `src/http/error_response.cpp`.

### Правила использования

- Все прикладные HTTP-контроллеры AUTH должны использовать единый формат ошибок.
- Контроллеры не должны дублировать код формирования JSON-ошибок.
- В HTTP-ответах запрещено раскрывать пароли, токены, SQL-запросы, внутренние исключения и другие чувствительные сведения.
- Коды ошибок должны быть стабильными и пригодными для программной обработки клиентами.
- Конкретный перечень прикладных кодов ошибок и соответствующие HTTP-статусы согласуются с COORD и FRONT при реализации соответствующих API.

### Тестирование

Команда запуска автоматических тестов:

```bash
ctest --test-dir build -C Debug --output-on-failure
```

Для Linux-сборки:

```bash
ctest --test-dir build --output-on-failure
```

Тест `auth_error_response` проверяет структуру JSON, значения `code` и `message`, а также установленный HTTP-статус.

Текущая реализация предоставляет общий механизм формирования прикладных ошибок. Подключение этого механизма к маршрутам регистрации, аутентификации и управления пользователями выполняется при реализации соответствующих контроллеров.

## AUTH-5. Хранение пользователей и ролей

Схема `db/001_users.sql` создаёт таблицу `users` с UUID, логином, email,
хэшем пароля и ролью (`user` или `admin`). Логин и email уникальны без
учёта регистра. Роль по умолчанию — `user`.

C++-модель находится в `include/auth/models/user.h`. Реализация регистрации
и запросов к таблице из C++ относится к следующим задачам.

При первом создании тома PostgreSQL схема запускается автоматически.
Для уже существующего тома её можно применить вручную:

```bash
docker compose exec -T postgres sh -c 'psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" -f /docker-entrypoint-initdb.d/001_users.sql'
```

SQL-проверка ограничений без сохранения тестовых записей:

```bash
docker compose exec -T postgres sh -c 'psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB"' < tests/users_schema.sql
```
Хэширование паролей (AUTH-10)

В сервисе AUTH предусмотрен интерфейс для формирования хэшей паролей пользователей.

Криптографическая реализация разрабатывается отдельно командой CRYPTO с использованием алгоритма «Стрибог».

Архитектура

В рамках AUTH-10 подготовлены следующие компоненты:

include/auth/password/i_password_hasher.h — абстрактный интерфейс хэширования.

include/auth/password/stub_password_hasher.h — объявление временной заглушки.

src/password/stub_password_hasher.cpp — реализация заглушки.

tests/password_hasher_test.cpp — автоматический тест.

Интерфейс предоставляет метод:

std::string hash_password(std::string_view password);

Метод предназначен для формирования значения, пригодного для хранения в PostgreSQL.

Текущие ограничения

На текущем этапе алгоритм хэширования не реализован в AUTH.

Временная заглушка StubPasswordHasher выбрасывает исключение std::runtime_error при вызове hash_password(). Это исключает случайное сохранение пароля в открытом виде или использование фиктивного хэша.

После согласования контракта с CRYPTO заглушка должна быть заменена адаптером, вызывающим криптографическую реализацию.

Формат хэша, генерация индивидуальной соли и параметры защиты от перебора должны быть согласованы с CRYPTO до использования механизма в рабочей системе.

Тестирование

Тест auth_password_hasher проверяет, что заглушка не возвращает фиктивное значение вместо безопасного хэша.

Запуск тестов в Windows:

ctest --test-dir build -C Debug --output-on-failure

В Docker тесты выполняются на этапе сборки образа.

Статус: интерфейс и заглушка готовы к интеграции. Реальное хэширование паролей пока недоступно.


## Валидация данных регистрации (AUTH-12)

В сервисе AUTH реализован независимый компонент проверки входных данных регистрации.

### Файлы

- `include/auth/validation/registration_validator.h` — структуры данных и публичный интерфейс.
- `src/validation/registration_validator.cpp` — реализация проверок.
- `tests/registration_validator_test.cpp` — автоматические тесты.

### Проверяемые поля

- `login` — длина и допустимые символы.
- `email` — наличие и корректность основных элементов адреса.
- `password` — длина и наличие необходимых групп символов.

Текущая политика является предварительной и должна быть подтверждена командой AUTH, поскольку исходное ЧТЗ не устанавливает точные регулярные выражения и численные ограничения.

### Интерфейс

```cpp
auth::validation::RegistrationData data{
    "brilds",
    "user@example.com",
    "SecurePass123!"
};

auto result =
    auth::validation::validate_registration(data);

if (!result.is_valid()) {
    // Обработка ошибки в вызывающем сервисе.
}
```

Компонент не зависит от Drogon, PostgreSQL или CRYPTO.

Валидация не проверяет уникальность логина и email в БД. Эта проверка относится к AUTH-13.

### Тестирование

```bash
ctest --test-dir build -C Debug --output-on-failure
```

При сборке Docker-образа тесты запускаются автоматически через CTest.

### Интеграция

Подключение валидатора к HTTP-маршруту `POST /auth/signup` предусмотрено в AUTH-16.

Обработка ошибок на HTTP-уровне должна использовать общий механизм AUTH-17.

## Поиск пользователя по логину (AUTH-8)

Реализован компонент `UserRepository` для получения учётной записи пользователя из PostgreSQL.

### Структура

- `include/auth/repositories/user_repository.h` — интерфейс репозитория.
- `src/repositories/user_repository.cpp` — реализация SQL-запросов.
- `tests/user_repository_integration_test.cpp` — интеграционный тест.

### Использование

```cpp
auto client = auth::repositories::create_database_client();

auth::repositories::UserRepository repository(client);

auto user = repository.find_by_login("brilds");

if (user) {
    // Пользователь найден.
}
```

Поиск выполняется без учёта регистра логина, что соответствует индексу `users_login_unique` из AUTH-5.

В результате возвращается модель `auth::models::User`, содержащая идентификатор, логин, email, хэш пароля и роли.

Если пользователь отсутствует, возвращается `std::nullopt`. Ошибки PostgreSQL передаются вызывающему коду как исключения.

SQL-запросы параметризованы для защиты от SQL-инъекций.

### Тестирование

Для запуска интеграционных тестов необходимо включить CMake-параметр `AUTH_ENABLE_DB_INTEGRATION_TESTS=ON`, предоставить доступ к PostgreSQL и задать переменные окружения подключения.

```powershell
ctest --test-dir build -C Debug --output-on-failure
```

Интеграционный тест создаёт временную учётную запись, проверяет поиск по логину, регистронезависимость, получение ролей и отсутствие SQL-инъекций, после чего удаляет запись.

### Ограничения

- Проверка пароля относится к AUTH-11.
- Поиск по email относится к AUTH-9.
- Выдача токенов и HTTP-авторизация не входят в AUTH-8.
- `password_hash` является внутренним полем и не должен передаваться клиентам API.
- В дальнейшем рекомендуется использовать асинхронные операции БД в HTTP-обработчиках.

## AUTH-18. Отправка кода подтверждения по почте

`auth::services::SmtpMailer` отправляет шестизначный код. Вызов синхронный и может
занять до 8 секунд. В HTTP-обработчиках вызывайте его вне потока ввода-вывода.

Для локальной демонстрации скопируйте `.env.example` в `.env` и запустите:

```bash
docker compose -f compose.yaml -f compose.mailpit.yaml up -d --build
```

Письма попадут в [Mailpit](http://127.0.0.1:8025). Его интерфейс доступен
на машине, где запущен Compose. SMTP-порт доступен внутри Docker-сети.

Для реальной доставки запустите обычный `docker compose up -d --build` с
настройками вашего SMTP-провайдера в `.env`. Пример для Brevo:

```dotenv
AUTH_SMTP_URL=smtp://smtp-relay.brevo.com:587
AUTH_SMTP_FROM=auth@example.com
AUTH_SMTP_USERNAME=your-smtp-login
AUTH_SMTP_PASSWORD=your-smtp-key
AUTH_SMTP_SECURITY=tls
```

Для Brevo подтвердите домен и настройте адрес отправителя. Режим `tls` требует
STARTTLS для `smtp://`, а `smtps://` открывает TLS-соединение сразу.


## AUTH-19. Логин и пароль

`POST /auth/login` (COORD) и `POST /login` (FRONT) принимают JSON
`{"login":"demo","password":"DemoPassword123!"}`. При успешной проверке
AUTH отправляет шестизначный код на email аккаунта и возвращает `204` с
`Set-Cookie: auth_pending=…`. Cookie действует 5 минут. Для локального HTTP
оверрайд Mailpit снимает атрибут `Secure`.

| HTTP | Код | Причина |
|---|---|---|
| 400 | `INVALID_REQUEST` | Поля запроса отсутствуют или имеют неверный формат |
| 401 | `INVALID_CREDENTIALS` | Логин или пароль не прошли проверку |
| 503 | `LOGIN_UNAVAILABLE` | Вход временно недоступен, например при ошибке PostgreSQL или SMTP |

Для нового тома Compose применяет миграцию автоматически. Для существующего
тома выполните её отдельно:

```bash
docker compose exec -T postgres sh -c \
  'psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB"' \
  < db/003_login_challenges.sql
```

### Быстрая проверка на локальном стенде

Команды ниже создают тестовую запись с известным паролем. Используйте чистую
локальную БД.

```bash
compose=(docker compose -f compose.yaml -f compose.mailpit.yaml)
demo_hash=$(printf '%s\n' 'DemoPassword123!' |
  "${compose[@]}" exec -T auth ./smp_auth --hash-password) &&
"${compose[@]}" exec -T postgres sh -c \
  'psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" -v "demo_hash=$1"' \
  sh "$demo_hash" <<'SQL'
INSERT INTO users (login, email, password_hash)
VALUES ('auth-demo', 'auth-demo@example.test', :'demo_hash');
SQL

curl -i http://127.0.0.1:8080/login \
  -H 'Content-Type: application/json' \
  -d '{"login":"auth-demo","password":"DemoPassword123!"}'
```

Ожидается `204` и письмо в [Mailpit](http://127.0.0.1:8025). Неверный пароль
возвращает `401`. Для текущего стенда `Pbkdf2PasswordHasher` создаёт и проверяет
хэши PBKDF2-HMAC-SHA256 с 600 000 итерациями и случайной солью. Когда будет готов
провайдер CRYPTO, подключите его через `IPasswordHasher` вместе с регистрацией
и сменой пароля. Форматы хэшей провайдеров должны быть согласованы.

### Данные для AUTH-20

Промежуточный идентификатор приходит в cookie `auth_pending`. Для поиска
в `login_challenges` используйте `auth::services::login_token_hash(token)`,
для проверки кода — `auth::services::login_email_code_hash(token, code)`. Успех
возможен при совпадении обоих хэшей и `expires_at > now()`. Погашение делается
одной операцией, например `DELETE … RETURNING user_id`, с последующим переходом
на этап одноразового recovery code. В AUTH-19 создаётся только промежуточный
идентификатор. Токен полной сессии выдаётся после третьего этапа.


## AUTH-16. Регистрация пользователя

`POST /auth/signup` и `POST /signup` принимают JSON с полями `login`, `email` и
`password`. Пользователь получает роль `user`. AUTH возвращает `201 Created`
и JSON-массив из 10 одноразовых кодов. Коды выдаются после успешного
сохранения пользователя, роли и хэшей всех кодов в одной транзакции.

```bash
curl -i http://127.0.0.1:8080/auth/signup \
  -H 'Content-Type: application/json' \
  -d '{"login":"authdemo","email":"authdemo@example.test","password":"DemoPassword123!"}'
```

Запрос с некорректными полями возвращает `400 INVALID_REQUEST`,
повтор логина или email без учёта регистра — `409 ACCOUNT_ALREADY_EXISTS`.
При ошибке записи AUTH возвращает `503 SIGNUP_UNAVAILABLE`.

Новый том PostgreSQL получает миграцию кодов восстановления при запуске.
Для существующего тома примените её из корня AUTH:

```bash
docker compose exec -T postgres sh -c \
  'psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB"' \
  < db/002_recovery_codes.sql
```


## AUTH-21. Сессии и проверка доступа для COORD

Полная сессия живёт 8 часов. В AUTH хранится хэш токена. Внутренние связи
используют UUID, а COORD получает постоянный числовой `user_id`.

COORD передаёт токен в `POST /internal/introspect` в JSON-поле `token`.
Ответ для действующей полной сессии:

```json
{"active":true,"stage":"full","user_id":1,"login":"exampleuser","role":"user"}
```

Для неизвестного, промежуточного, просроченного или отозванного токена
возвращается `200` с телом `{"active":false}`. Маршрут используется внутри
контура AUTH и COORD. `POST /auth/logout` отзывает указанную Bearer-сессию,
возвращает `204` и очищает cookie `session`.

AUTH-22 сможет создавать полную сессию методом
`SessionRepository::create_full_session`. Чтобы погасить recovery code и
создать сессию атомарно, используйте одну транзакцию для обеих операций.

На новом томе Docker Compose применяет миграцию автоматически. Для
существующего тома:

```bash
docker compose exec -T postgres sh -c \
  'psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB"' \
  < db/004_auth_sessions.sql
```

Для запуска теста репозитория сессий соберите AUTH с
`-DAUTH_ENABLE_DB_INTEGRATION_TESTS=ON` и выполните
`ctest --test-dir build -R auth_session_repository --output-on-failure`
с настроенным подключением к тестовой PostgreSQL.


## Восстановление пароля и блокировка входа (#19)

Все запросы выполняются методом `POST` с JSON. При обращении через COORD добавьте префикс `/api`.

| Маршрут AUTH | Тело и cookie | Ответ `204` |
|---|---|---|
| `/auth/password/restore` | `{"email":"user@example.test"}` | `auth_reset_pending` на 5 минут. Код для зарегистрированного адреса отправляется по SMTP. |
| `/auth/password/confirm` | `{"code":"123456"}` и `auth_reset_pending` | Одноразовая cookie `auth_reset_grant` на 5 минут. Первая cookie очищается. |
| `/auth/password/confirm` | `{"password":"NewPassword123!"}` и `auth_reset_grant` | Новый пароль сохранён, cookie очищена, старые сессии и незавершённые попытки входа отозваны. |

Первый запрос возвращает одинаковый статус для известного и неизвестного адреса. Письмо отправляется после ответа. На одну учётную запись можно отправлять письмо раз в минуту и вводить до пяти кодов на одну попытку. Неверный или просроченный код, отсутствующая cookie и использованный токен возвращают `401`. При смене пароля пользователь проходит обычный вход заново.

AUTH также принимает `/restorePassword`, `/confirmRestoreCode` и `/confirmNewPassword` с теми же телами и cookie. При интеграции прямого клиента FRONT нужно передать `Set-Cookie` от `/restorePassword` в браузер, а `Cookie` из браузера — в `/confirmRestoreCode`. Последний шаг уже передаёт cookie в AUTH.

Пять неверных паролей подряд блокируют вход для учётной записи на 15 минут. Счётчик сбрасывается при успешной проверке пароля, смене пароля или паузе в 15 минут между ошибками. Во время блокировки вход возвращает `401 INVALID_CREDENTIALS`. Восстановление пароля остаётся доступным.

Новый том PostgreSQL получает миграцию автоматически. Для существующей БД примените её перед обновлением AUTH из корня репозитория.

```bash
docker compose exec -T postgres sh -c \
  'psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB"' \
  < db/006_password_reset.sql
```

## AUTH-22. Вход по коду восстановления

`POST /auth/login/recovery-code` принимает JSON `{"code":"..."}` и cookie
`auth_recovery_pending`. Для уже существующего клиента FRONT доступен
`POST /confirmDisposableCode` с тем же контрактом.

Успешный ответ — `204` с полной cookie `session` и удалением промежуточной cookie.
Неверный или использованный код, отсутствующая или просроченная попытка входа
возвращают `401 INVALID_RECOVERY_CODE`. При ошибке записи в БД вся транзакция
откатывается, а AUTH возвращает `503 RECOVERY_UNAVAILABLE`.

### Стык с AUTH-20

После проверки email-кода вызовите `auth::services::create_recovery_challenge(tx, user_id)`
из `auth/services/recovery_factor.h` в той же транзакции, в которой погашается
email-код. `user_id` — внутренний UUID пользователя. Перед погашением email-кода
заблокируйте строку пользователя через `FOR UPDATE`, чтобы сброс пароля и смена
этапа входа обрабатывались последовательно. После коммита передайте
результат в HttpOnly-cookie `auth_recovery_pending` с путём `/`,
`SameSite=Lax`, `Max-Age=300` и настройкой `AUTH_COOKIE_SECURE` как у
`auth_pending`. На этом этапе можно удалить cookie `auth_pending`.

Новый том PostgreSQL получает миграцию автоматически. Для существующего тома:

```bash
docker compose exec -T postgres sh -c \
  'psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB"' \
  < db/005_recovery_challenges.sql
```
### AUTH-20. Второй фактор аутентификации — подтверждение email

Второй фактор аутентификации предназначен для подтверждения владения адресом электронной почты пользователя.

После успешной проверки логина и пароля (AUTH-19) сервер отправляет шестизначный код подтверждения на email и устанавливает временную HttpOnly-cookie `auth_pending` со сроком действия 300 секунд.

**Маршруты:**

- `POST /auth/login/email-code` — основной маршрут подтверждения.
- `POST /confirmEmailCode` — дополнительный маршрут совместимости.

**Формат запроса:**

```json
{
  "code": "123456"
}
```

Запрос должен содержать действительную cookie `auth_pending`.

**Алгоритм работы:**

1. Контроллер проверяет формат входного JSON и наличие шестизначного кода.
2. Из cookie извлекается временный токен попытки входа.
3. Сервис вычисляет SHA-256-хэш токена и HMAC-SHA256 от email-кода с использованием токена.
4. В PostgreSQL проверяются соответствующие значения таблицы `login_challenges` и срок действия попытки входа.
5. В транзакции блокируется запись пользователя, погашается исходная попытка входа и создаётся запись в `recovery_challenges`.
6. После успешной фиксации транзакции сервер возвращает HTTP 204 и устанавливает HttpOnly-cookie `auth_recovery_pending`.
7. Cookie `auth_pending` удаляется. Пользователь переходит к третьему фактору (AUTH-22).

**HTTP-ответы:**

| Код | Значение |
|---|---|
| 204 | Email-код подтверждён |
| 400 | Некорректный формат запроса |
| 401 | Неверный или просроченный email-код либо недействительная попытка входа |
| 503 | Временная ошибка сервиса или PostgreSQL |

**Безопасность:**

- В базе данных не сохраняется открытый email-код.
- Проверка и погашение кода выполняются транзакционно.
- Успешно использованный код нельзя применить повторно.
- Токен следующего фактора выдаётся только после успешного COMMIT.
- После второго фактора полноценная пользовательская сессия не создаётся.
- Cookie содержит `HttpOnly`, `SameSite=Lax`, `Path=/` и время действия 300 секунд.
- Атрибут `Secure` управляется параметром `AUTH_COOKIE_SECURE`.

**Тестирование:**

Интеграционный тест `auth_email_factor` проверяет корректный, неверный, просроченный и повторно использованный код, создание следующего challenge и отсутствие полной пользовательской сессии до третьего фактора.

Сквозной сценарий проверяется через Docker Compose, PostgreSQL и Mailpit.

# Локальная проверка AUTH и COORD

### Запуск и проверка одной командой (AUTH-24)

Понадобятся Bash, Docker Compose, `curl` и `jq`. Сначала подготовьте отдельное
рабочее дерево COORD. Из корня AUTH выполните:

```bash
git fetch origin COORD-develop
git worktree add --detach ../smp-coord origin/COORD-develop
bash scripts/auth-coord-local.sh ../smp-coord
```

Если рабочее дерево COORD уже есть, передайте путь к нему. Скрипт запускает
два Compose-проекта с тестовыми портами из таблицы ниже и устанавливает
`ACCOUNTS_URL` только на время запуска COORD. Он создаёт `.env` для AUTH из
шаблона, если файла ещё нет. Существующий файл сохраняется. Настройки портов
для этого запуска передаются Docker Compose через окружение.

При первом запуске с пустыми томами PostgreSQL инициализирует обе базы, а AUTH
применяет миграции `db/001_*.sql`–`db/006_*.sql`. На уже существующей локальной
базе AUTH примените недостающие миграции перед запуском. Скрипт сохраняет тома.

Проверка выполняется через COORD: регистрация возвращает десять кодов,
пароль запускает отправку письма в Mailpit, а подтверждение письма и резервного
кода выдаёт полную сессию. Запрос списка пользователей с этой сессией
подтверждает роль `user` ответом `403`. Тестовая учётная запись удаляется по
окончании, сервисы остаются запущенными. Mailpit доступен разработчику на
`http://127.0.0.1:8025`. Его SMTP-порт доступен только внутри сети AUTH.

Остановить сервисы с сохранением данных можно из соответствующих рабочих деревьев:

```bash
# В корне AUTH
docker compose -f compose.yaml -f compose.mailpit.yaml down
# В корне COORD
docker compose -f docker-compose.yml down
```

## Порты локального стенда

| Служба | Адрес с Windows-хоста | Адрес внутри Docker |
| --- | --- | --- |
| COORD HTTP | `http://127.0.0.1:8080` | `coord:8080` (COORD-сеть) |
| AUTH HTTP | `http://127.0.0.1:8081` | `auth:8080` (AUTH-сеть) |
| COORD PostgreSQL | `127.0.0.1:5432` | `postgres:5432` (COORD-сеть) |
| AUTH PostgreSQL | `127.0.0.1:5433` | `postgres:5432` (AUTH-сеть) |
| Mailpit UI | `http://127.0.0.1:8025` | `mailpit:8025` |
| Mailpit SMTP | НЕ опубликован | `mailpit:1025` (AUTH-сеть) |

Стенды запускаются из разных рабочих деревьев Git, поэтому они используют отдельные Docker Compose-проекты и разные сети. Для вызовов COORD -> AUTH из контейнера COORD используется опубликованный порт хоста: `http://host.docker.internal:8081`. Не используйте `localhost:8081` внутри контейнера COORD: это сам контейнер COORD, а не хост.

**Безопасность:** хостовый порт AUTH `8081` опубликован на интерфейсах хоста, поскольку он должен быть доступен через `host.docker.internal`. Это локальный тестовый вариант; ограничьте порт брандмауэром и не используйте такое опубликование для production. AUTH PostgreSQL и Mailpit UI привязаны только к `127.0.0.1`. SMTP порт `1025` наружу не публикуется.

## Ручной запуск AUTH

В корне AUTH:

```powershell
Copy-Item .env.example .env # только если .env ещё отсутствует
# В .env укажите свой POSTGRES_PASSWORD (НЕ закоммитьте .env)
docker compose -f compose.yaml -f compose.mailpit.yaml config --quiet
docker compose -f compose.yaml -f compose.mailpit.yaml up -d --build
docker compose -f compose.yaml -f compose.mailpit.yaml ps
Invoke-RestMethod http://127.0.0.1:8081/health
```

`config/config.docker.json` не меняется: процесс AUTH продолжает слушать `0.0.0.0:8080` внутри контейнера. Изменяется только публикация порта на хост.

Mailpit UI: http://127.0.0.1:8025. SMTP AUTH: `smtp://mailpit:1025`.

## Установка COORD (ветка COORD-develop)

Нужно **отдельное рабочее дерево/папка** COORD — не переключайте текущую ветку AUTH в том же каталоге при запущенном тестовом стенде.

```powershell
# В ДРУГОЙ папке с веткой COORD-develop:
# Скопируйте .env.example в .env, если файла ещё нет.
# Значения в .env COORD:
# HTTP_PORT=8080
# POSTGRES_PORT=5432
# ACCOUNTS_URL=http://host.docker.internal:8081
# CRYPTO_URL=http://host.docker.internal:8082
# MASTER_KEY=<64 hex characters> (локальный секрет)
docker compose -f docker-compose.yml config --quiet
docker compose -f docker-compose.yml up -d --build
Invoke-RestMethod http://127.0.0.1:8080/health
```

`COORD-develop/docker-compose.yml` уже содержит `ACCOUNTS_URL: ${ACCOUNTS_URL:-http://host.docker.internal:8081}`, `HTTP_PORT=8080` и `POSTGRES_PORT=5432` по умолчанию. Указывать `ACCOUNTS_URL=http://localhost:8081` при Docker-запуске COORD нельзя. Отдельно проверить доступность CRYPTO (запуск COORD может зависеть от криптосервиса).

## Сквозная проверка регистрации через COORD

После запуска обоих стеков, из корня AUTH:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\tests\coord_integration_e2e.ps1
```

Тест проверяет `GET /health` обоих сервисов, POST `/api/auth/signup` через COORD, 10 кодов восстановления и создание пользователя в БД AUTH, повторную регистрацию `409`. В конце удаляет только собственного временного пользователя. Запускать только на выделенной тестовой БД.

Существующие AUTH-25–29 тесты после смены хостового порта можно запускать, задав `-BaseUrl http://127.0.0.1:8081`, либо заменив в них значение параметра `BaseUrl` по умолчанию.

## Проверка портов и ошибки

```powershell
Get-NetTCPConnection -State Listen | Where-Object LocalPort -in 8080,8081,5432,5433,8025

docker compose -f compose.yaml -f compose.mailpit.yaml logs --tail 80 auth
```

Если порт занят, согласуйте другое значение соответствующего хостового порта, не меняя внутренние порты процессов. Для другого хостового порта AUTH обязательно изменить `ACCOUNTS_URL` у COORD.

## Изменения документации

К существующему `docs/AUTH_README.md` добавьте ссылку на этот документ и обновите примеры с `127.0.0.1:8080` на `127.0.0.1:8081` для запросов непосредственно к AUTH. **Не заменяйте весь AUTH_README.md:** он содержит накопленные контракты других задач.
