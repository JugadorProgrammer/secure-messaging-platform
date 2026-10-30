
#include "auth/repositories/database.h"

#include <drogon/orm/DbClient.h>

#include <cstdlib>
#include <exception>
#include <iostream>
#include <string>

namespace auth::repositories {

namespace {

// Получение переменной окружения с запасным значением.
std::string get_env(const char* name, const char* fallback) {
    const char* value = std::getenv(name);
    return value != nullptr && value[0] != '\0'
        ? std::string(value)
        : std::string(fallback);
}

}  // namespace

bool check_database_connection() {
    // Настройки подключения.
    const std::string host =
        get_env("AUTH_DB_HOST", "127.0.0.1");

    const std::string port =
        get_env("AUTH_DB_PORT", "5432");

    const std::string dbname =
        get_env("AUTH_DB_NAME", "auth_db");

    const std::string user =
        get_env("AUTH_DB_USER", "auth_user");

    // Пароль читается libpq из переменной PGPASSWORD.
    // В строку подключения его не добавляем.
    const std::string conn_info =
        "host=" + host +
        " port=" + port +
        " dbname=" + dbname +
        " user=" + user +
        " connect_timeout=5";

    try {
        auto client =
            drogon::orm::DbClient::newPgClient(conn_info, 1);

        auto result = client->execSqlSync("SELECT 1 AS connection_ok");

        const int value = result[0]["connection_ok"].as<int>();

        if (value != 1) {
            std::cerr << "Database check returned unexpected value\n";
            return false;
        }

        std::cout << "PostgreSQL connection: OK\n";
        return true;

    } catch (const std::exception& e) {
        // Не выводим строку подключения или секреты.
        std::cerr << "PostgreSQL connection failed\n";
        return false;
    }
}

}  // namespace auth::repositories
