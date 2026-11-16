#include "auth/services/recovery_factor.h"

#include <future>
#include <memory>
#include <stdexcept>

#include "auth/repositories/recovery_codes_repository.h"
#include "auth/repositories/session_repository.h"
#include "auth/services/login_token.h"

namespace auth::services {

std::string create_recovery_challenge(
    const std::shared_ptr<drogon::orm::Transaction>& transaction,
    std::string_view user_id
) {
    auto token = generate_login_token();
    transaction->execSqlSync("DELETE FROM recovery_challenges WHERE expires_at <= now()");
    transaction->execSqlSync(
        R"(
            INSERT INTO recovery_challenges (token_hash, user_id, expires_at)
            VALUES ($1, $2::uuid, now() + $3::int * interval '1 second')
        )",
        login_token_hash(token), std::string(user_id), kRecoveryChallengeLifetimeSeconds
    );
    return token;
}

std::optional<std::string> complete_recovery_factor(
    const drogon::orm::DbClientPtr& database,
    std::string_view pending_token,
    std::string_view code
) {
    if (!is_login_token(pending_token)) {
        return std::nullopt;
    }

    const auto pending_hash = login_token_hash(pending_token);
    auto commit = std::make_shared<std::promise<bool>>();
    auto commit_result = commit->get_future();
    std::string session_token;
    {
        const auto transaction = database->newTransaction([commit](bool success) {
            commit->set_value(success);
        });
        if (!transaction) {
            throw std::runtime_error("Cannot start recovery factor transaction");
        }

        try {
            const auto challenge = transaction->execSqlSync(
                R"(
                    DELETE FROM recovery_challenges
                    WHERE token_hash = $1 AND expires_at > now()
                    RETURNING user_id::text AS user_id
                )",
                pending_hash
            );
            if (challenge.empty()) {
                transaction->rollback();
                return std::nullopt;
            }

            const auto user_id = challenge[0]["user_id"].as<std::string>();
            if (!repositories::consume_recovery_code(transaction, user_id, code)) {
                transaction->rollback();
                return std::nullopt;
            }

            const repositories::SessionRepository sessions(transaction);
            session_token = sessions.create_full_session(user_id);
        } catch (...) {
            transaction->rollback();
            throw;
        }
    }

    // Drogon выполняет COMMIT при уничтожении Transaction.
    if (!commit_result.get()) {
        throw std::runtime_error("Recovery factor transaction was not committed");
    }

    return session_token;
}

}  // namespace auth::services
