#include "auth/repositories/database.h"
#include "auth/repositories/session_repository.h"
#include "auth/services/login_token.h"

#include <iostream>
#include <stdexcept>

namespace {

void require(bool condition, const char* message) {
    if (!condition) {
        throw std::runtime_error(message);
    }
}

}  // namespace

int main() {
    const auto database = auth::repositories::create_database_client();
    const auto transaction = database->newTransaction();

    try {
        const auto login = "session-test-" + auth::services::generate_login_token();
        const auto user = transaction->execSqlSync(
            "INSERT INTO users (login, email, password_hash) VALUES ($1, $2, 'test-only') "
            "RETURNING id::text AS id, coord_id",
            login, login + "@example.test"
        );
        const auto id = user[0]["id"].as<std::string>();
        const auto coord_id = user[0]["coord_id"].as<std::int64_t>();
        const auth::repositories::SessionRepository sessions(transaction);
        const auto token = sessions.create_full_session(id);
        const auto hash = auth::services::login_token_hash(token);

        require(coord_id > 0, "COORD ID must be a positive integer");
        require(auth::services::is_login_token(token), "Unexpected session token format");
        const auto stored = transaction->execSqlSync(
            "SELECT token_hash FROM auth_sessions WHERE user_id = $1::uuid", id
        );
        require(stored.size() == 1 && stored[0]["token_hash"].as<std::string>() == hash,
                "Stored value must be the session token hash");
        require(!sessions.find_active_principal(token), "User without a recognized role got access");

        transaction->execSqlSync(
            "INSERT INTO user_roles (user_id, role_id) "
            "SELECT $1::uuid, id FROM roles WHERE code = 'user'",
            id
        );
        const auto user_principal = sessions.find_active_principal(token);
        require(user_principal && user_principal->user_id == coord_id &&
                user_principal->login == login && user_principal->role == "user",
                "Could not read the active user session");

        transaction->execSqlSync(
            "INSERT INTO user_roles (user_id, role_id) "
            "SELECT $1::uuid, id FROM roles WHERE code = 'admin'",
            id
        );
        const auto admin_principal = sessions.find_active_principal(token);
        require(admin_principal && admin_principal->role == "admin",
                "Admin role should take precedence");

        const auto expiring = sessions.create_full_session(id);
        transaction->execSqlSync("UPDATE auth_sessions SET expires_at = now() WHERE token_hash = $1",
                                 auth::services::login_token_hash(expiring));
        require(!sessions.find_active_principal(expiring), "Expired session got access");

        sessions.revoke(token);
        require(!sessions.find_active_principal(token), "Revoked session got access");
        require(!sessions.find_active_principal(auth::services::generate_login_token()),
                "Unknown token got access");
        require(!sessions.find_active_principal("malformed"), "Malformed token got access");

        transaction->rollback();
        return 0;
    } catch (const std::exception& error) {
        transaction->rollback();
        std::cerr << "Session integration test failed: " << error.what() << '\n';
        return 1;
    }
}
