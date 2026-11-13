
#include "auth/repositories/database.h"

#include <cstdlib>
#include <exception>
#include <iostream>
#include <string>

namespace auth::repositories {

namespace {

std::string get_env(const char* name, const char* fallback) {
    const char* value = std::getenv(name);

    return value != nullptr && value[0] != '\0'
        ? std::string(value)
        : std::string(fallback);
}

}  // namespace

drogon::orm::DbClientPtr create_database_client() {
    const std::string host =
        get_env("AUTH_DB_HOST", "127.0.0.1");

    const std::string port =
        get_env("AUTH_DB_PORT", "5432");

    const std::string dbname =
        get_env("AUTH_DB_NAME", "auth_db");

    const std::string user =
        get_env("AUTH_DB_USER", "auth_user");

    // Пароль передаётся libpq через PGPASSWORD.
    const std::string conn_info =
        "host=" + host +
        " port=" + port +
        " dbname=" + dbname +
        " user=" + user +
        " connect_timeout=5";

    return drogon::orm::DbClient::newPgClient(
        conn_info, 1
    );
}

bool check_database_connection() {
    try {
        auto client = create_database_client();

        auto result = client->execSqlSync(
            "SELECT 1 AS connection_ok"
        );

        if (result.empty() ||
            result[0]["connection_ok"].as<int>() != 1) {
            std::cerr
                << "Database check returned unexpected value\n";
            return false;
        }

        std::cout << "PostgreSQL connection: OK\n";
        return true;

    } catch (const std::exception&) {
        std::cerr << "PostgreSQL connection failed\n";
        return false;
    }
}

}  // namespace auth::repositories
