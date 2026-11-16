
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


std::vector<UserSummary> UserRepository::list_users() const {
    const auto result = client_->execSqlSync(
        R"(
            SELECT u.login, u.email,
                CASE
                    WHEN bool_or(r.code = 'admin') THEN 'admin'
                    WHEN bool_or(r.code = 'user') THEN 'user'
                END AS role
            FROM users u
            LEFT JOIN user_roles ur ON u.id = ur.user_id
            LEFT JOIN roles r ON r.id = ur.role_id
            GROUP BY u.id
            ORDER BY lower(u.login)
        )"
    );

    std::vector<UserSummary> users;
    users.reserve(result.size());
    for (const auto& row : result) {
        const auto role = row["role"];
        users.push_back(UserSummary{
            row["login"].as<std::string>(),
            row["email"].as<std::string>(),
            role.isNull() ? std::nullopt : std::optional{role.as<std::string>()}
        });
    }
    return users;
}

std::optional<std::string> UserRepository::create_user(
    std::string_view login,
    std::string_view email,
    std::string_view password_hash,
    std::string_view role
) const {
    const auto result = client_->execSqlSync(
        R"(
            WITH new_user AS (
                INSERT INTO users (login, email, password_hash)
                VALUES ($1, $2, $3)
                ON CONFLICT DO NOTHING
                RETURNING id
            )
            INSERT INTO user_roles (user_id, role_id)
            SELECT id, (SELECT id FROM roles WHERE code = $4)
            FROM new_user
            RETURNING user_id::text AS id
        )",
        std::string(login), std::string(email), std::string(password_hash), std::string(role)
    );

    if (result.empty()) {
        return std::nullopt;
    }
    return result[0]["id"].as<std::string>();
}

}  // namespace auth::repositories
