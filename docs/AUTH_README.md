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
