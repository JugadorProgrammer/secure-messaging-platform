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
D:\vcpkg\vcpkg.exe install "drogon[postgres]:x64-windows" --recurse
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