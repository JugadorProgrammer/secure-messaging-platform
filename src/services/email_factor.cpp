
#include "auth/services/email_factor.h"

#include <future>
#include <memory>
#include <stdexcept>
#include <string>

#include "auth/services/login_token.h"
#include "auth/services/recovery_factor.h"

namespace auth::services {

std::optional<std::string> complete_email_factor(
    const drogon::orm::DbClientPtr& database,
    std::string_view pending_token,
    std::string_view email_code
) {
    // Проверка формата временного токена.
    if (!is_login_token(pending_token)) {
        return std::nullopt;
    }

    // Email-код строго шестизначный.
    if (email_code.size() != 6) {
        return std::nullopt;
    }

    for (const char ch : email_code) {
        if (ch < '0' || ch > '9') {
            return std::nullopt;
        }
    }

    if (!database) {
        throw std::invalid_argument(
            "Database client is null"
        );
    }

    const auto token_hash =
        login_token_hash(pending_token);

    const auto code_hash =
        login_email_code_hash(pending_token, email_code);

    auto commit = std::make_shared<std::promise<bool>>();
    auto commit_result = commit->get_future();

    std::string recovery_token;

    {
        // Одна транзакция на весь переход 2 -> 3.
        const auto transaction = database->newTransaction(
            [commit](bool success) {
                commit->set_value(success);
            }
        );

        if (!transaction) {
            throw std::runtime_error(
                "Cannot start email factor transaction"
            );
        }

        try {
            // Блокируем пользователя.
            // Проверяем срок действия и оба хэша.
            const auto account = transaction->execSqlSync(
                R"(
                    SELECT u.id::text AS user_id
                    FROM users u
                    JOIN login_challenges c
                        ON c.user_id = u.id
                    WHERE c.token_hash = $1
                      AND c.email_code_hash = $2
                      AND c.expires_at > now()
                    FOR UPDATE OF u
                )",
                token_hash,
                code_hash
            );

            if (account.empty()) {
                transaction->rollback();
                return std::nullopt;
            }

            const std::string user_id =
                account[0]["user_id"].as<std::string>();

            // Погашаем временную попытку входа.
            // DELETE RETURNING исключает повторное
            // успешное использование.
            const auto consumed = transaction->execSqlSync(
                R"(
                    DELETE FROM login_challenges
                    WHERE token_hash = $1
                      AND email_code_hash = $2
                      AND expires_at > now()
                    RETURNING user_id::text AS user_id
                )",
                token_hash,
                code_hash
            );

            if (consumed.empty()) {
                transaction->rollback();
                return std::nullopt;
            }

            // Создаём третий фактор в той же транзакции.
            recovery_token = create_recovery_challenge(
                transaction,
                user_id
            );

        } catch (...) {
            transaction->rollback();
            throw;
        }
    }

    // Токен выдаём только после COMMIT.
    if (!commit_result.get()) {
        throw std::runtime_error(
            "Email factor transaction was not committed"
        );
    }

    return recovery_token;
}

}  // namespace auth::services
