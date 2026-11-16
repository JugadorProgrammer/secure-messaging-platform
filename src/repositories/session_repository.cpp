#include "auth/repositories/session_repository.h"

#include <stdexcept>
#include <utility>

#include "auth/services/login_token.h"

namespace auth::repositories {

SessionRepository::SessionRepository(drogon::orm::DbClientPtr client)
    : client_(std::move(client)) {
    if (!client_) {
        throw std::invalid_argument("Database client must not be null");
    }
}

std::string SessionRepository::create_full_session(std::string_view user_id) const {
    auto token = services::generate_login_token();
    client_->execSqlSync(
        R"(
            INSERT INTO auth_sessions (token_hash, user_id, expires_at)
            VALUES ($1, $2::uuid, now() + $3::int * interval '1 second')
        )",
        services::login_token_hash(token), std::string(user_id), kSessionLifetimeSeconds
    );
    return token;
}

std::optional<SessionPrincipal> SessionRepository::find_active_principal(std::string_view token) const {
    if (!services::is_login_token(token)) {
        return std::nullopt;
    }

    const auto result = client_->execSqlSync(
        R"(
            SELECT u.coord_id, u.login, r.code AS role
            FROM auth_sessions s
            JOIN users u ON u.id = s.user_id
            JOIN user_roles ur ON ur.user_id = u.id
            JOIN roles r ON r.id = ur.role_id
            WHERE s.token_hash = $1 AND s.expires_at > now() AND r.code IN ('user', 'admin')
            ORDER BY (r.code = 'admin') DESC
            LIMIT 1
        )",
        services::login_token_hash(token)
    );
    if (result.empty()) {
        return std::nullopt;
    }

    const auto& row = result[0];
    return SessionPrincipal{
        row["coord_id"].as<std::int64_t>(),
        row["login"].as<std::string>(),
        row["role"].as<std::string>()
    };
}

void SessionRepository::revoke(std::string_view token) const {
    if (services::is_login_token(token)) {
        client_->execSqlSync("DELETE FROM auth_sessions WHERE token_hash = $1",
                             services::login_token_hash(token));
    }
}

}  // namespace auth::repositories
