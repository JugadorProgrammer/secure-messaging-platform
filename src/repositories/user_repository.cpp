
#include "auth/repositories/user_repository.h"

#include <stdexcept>
#include <string>
#include <utility>

namespace auth::repositories {

UserRepository::UserRepository(
    drogon::orm::DbClientPtr client
) : client_(std::move(client)) {
    if (!client_) {
        throw std::invalid_argument(
            "Database client must not be null"
        );
    }
}

std::optional<models::User> UserRepository::find_by_login(
    std::string_view login
) const {
    // $1 — параметр, а не часть SQL-строки.
    const auto result = client_->execSqlSync(
        R"(
            SELECT
                id::text AS id,
                login,
                email,
                password_hash
            FROM users
            WHERE lower(login) = lower($1)
            LIMIT 1
        )",
        std::string(login)
    );

    if (result.empty()) {
        return std::nullopt;
    }

    const auto& row = result[0];

    models::User user;
    user.id = row["id"].as<std::string>();
    user.login = row["login"].as<std::string>();
    user.email = row["email"].as<std::string>();
    user.password_hash =
        row["password_hash"].as<std::string>();

    // Получаем роли найденного пользователя.
    const auto roles = client_->execSqlSync(
        R"(
            SELECT r.code
            FROM user_roles ur
            JOIN roles r ON r.id = ur.role_id
            WHERE ur.user_id = $1::uuid
            ORDER BY r.code
        )",
        user.id
    );

    for (const auto& role : roles) {
        user.roles.push_back(
            role["code"].as<std::string>()
        );
    }

    return user;
}


std::optional<models::User> UserRepository::find_by_email(
    std::string_view email
) const {
    const auto result = client_->execSqlSync(
        R"(
            SELECT
                id::text AS id,
                login,
                email,
                password_hash
            FROM users
            WHERE lower(email) = lower($1)
            LIMIT 1
        )",
        std::string(email)
    );

    if (result.empty()) {
        return std::nullopt;
    }

    const auto& row = result[0];

    models::User user;
    user.id = row["id"].as<std::string>();
    user.login = row["login"].as<std::string>();
    user.email = row["email"].as<std::string>();
    user.password_hash =
        row["password_hash"].as<std::string>();

    const auto roles = client_->execSqlSync(
        R"(
            SELECT r.code
            FROM user_roles ur
            JOIN roles r ON r.id = ur.role_id
            WHERE ur.user_id = $1::uuid
            ORDER BY r.code
        )",
        user.id
    );

    for (const auto& role : roles) {
        user.roles.push_back(
            role["code"].as<std::string>()
        );
    }

    return user;
}


}  // namespace auth::repositories
