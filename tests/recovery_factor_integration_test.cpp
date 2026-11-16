#include "auth/repositories/database.h"
#include "auth/repositories/recovery_codes_repository.h"
#include "auth/repositories/session_repository.h"
#include "auth/services/login_token.h"
#include "auth/services/recovery_codes.h"
#include "auth/services/recovery_factor.h"

#include <future>
#include <iostream>
#include <memory>
#include <stdexcept>

namespace {

void require(bool condition, const char* message) {
    if (!condition) {
        throw std::runtime_error(message);
    }
}

std::string open_recovery_challenge(const drogon::orm::DbClientPtr& database, const std::string& user_id) {
    auto commit = std::make_shared<std::promise<bool>>();
    auto committed = commit->get_future();
    auto transaction = database->newTransaction([commit](bool success) {
        commit->set_value(success);
    });
    const auto token = auth::services::create_recovery_challenge(transaction, user_id);
    transaction.reset();
    require(committed.get(), "Could not commit test recovery challenge");
    return token;
}

void test_recovery_factor(const drogon::orm::DbClientPtr& database, const std::string& id) {
    database->execSqlSync(
        "INSERT INTO user_roles (user_id, role_id) "
        "SELECT $1::uuid, id FROM roles WHERE code = 'user'",
        id
    );

    const auto codes = auth::services::generate_recovery_codes(3);
    auth::repositories::save_recovery_codes(database, id, codes);
    const auto pending = open_recovery_challenge(database, id);

    require(!auth::services::complete_recovery_factor(database, pending, "wrong-code"),
            "An invalid code was accepted");
    const auto session = auth::services::complete_recovery_factor(database, pending, codes[0]);
    require(session.has_value(), "A valid recovery code did not create a session");
    const auth::repositories::SessionRepository sessions(database);
    require(sessions.find_active_principal(*session).has_value(), "The new session is inactive");
    require(!auth::services::complete_recovery_factor(database, pending, codes[1]),
            "An already completed challenge was accepted");

    const auto retry = open_recovery_challenge(database, id);
    require(!auth::services::complete_recovery_factor(database, retry, codes[0]),
            "An already used code was accepted");
    require(auth::services::complete_recovery_factor(database, retry, codes[1]).has_value(),
            "A rejected code deleted the pending challenge");

    const auto expired = open_recovery_challenge(database, id);
    database->execSqlSync("UPDATE recovery_challenges SET expires_at = now() WHERE token_hash = $1",
                          auth::services::login_token_hash(expired));
    require(!auth::services::complete_recovery_factor(database, expired, codes[2]),
            "An expired challenge was accepted");
}

}  // namespace

int main() {
    try {
        const auto database = auth::repositories::create_database_client();
        const auto login = "recovery-test-" + auth::services::generate_login_token();
        const auto inserted = database->execSqlSync(
            "INSERT INTO users (login, email, password_hash) VALUES ($1, $2, 'test-only') "
            "RETURNING id::text AS id",
            login, login + "@example.test"
        );
        const auto id = inserted[0]["id"].as<std::string>();

        try {
            test_recovery_factor(database, id);
        } catch (...) {
            database->execSqlSync("DELETE FROM users WHERE id = $1::uuid", id);
            throw;
        }
        database->execSqlSync("DELETE FROM users WHERE id = $1::uuid", id);
        return 0;
    } catch (const std::exception& error) {
        std::cerr << "Recovery factor integration test failed: " << error.what() << '\n';
        return 1;
    }
}
