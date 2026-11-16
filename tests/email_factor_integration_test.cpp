
#include "auth/repositories/database.h"
#include "auth/services/email_factor.h"
#include "auth/services/login_token.h"

#include <iostream>
#include <stdexcept>
#include <string>

namespace {

void require(bool condition, const char* message) {
    if (!condition) {
        throw std::runtime_error(message);
    }
}

}  // namespace

int main() {
    using namespace auth::services;

    auto database = auth::repositories::create_database_client();

    // Уникальные данные для каждого запуска теста.
    const auto suffix = generate_login_token().substr(0, 16);
    const std::string login = "auth20_" + suffix;
    const std::string email = login + "@example.test";

    std::string user_id;

    try {
        const auto inserted = database->execSqlSync(
            R"(
                INSERT INTO users (login, email, password_hash)
                VALUES ($1, $2, $3)
                RETURNING id::text AS id
            )",
            login, email, "integration-test-only"
        );

        user_id = inserted[0]["id"].as<std::string>();

        // Создаёт попытку входа, аналогичную AUTH-19.
        auto create_attempt = [&](const std::string& token,
                                  const std::string& code,
                                  int lifetime_seconds) {
            database->execSqlSync(
                R"(
                    INSERT INTO login_challenges
                        (token_hash, user_id,
                         email_code_hash, expires_at)
                    VALUES (
                        $1,
                        $2::uuid,
                        $3,
                        now() + $4::int * interval '1 second'
                    )
                )",
                login_token_hash(token),
                user_id,
                login_email_code_hash(token, code),
                lifetime_seconds
            );
        };

        const std::string email_code = "123456";

        // 1. Неправильный код не должен проходить.
        const auto token = generate_login_token();
        create_attempt(token, email_code, 300);

        const auto wrong = complete_email_factor(
            database, token, "999999"
        );

        require(!wrong.has_value(),
                "Wrong email code was accepted");

        // Ошибочная попытка не должна погасить
        // действительный email-код.
        const auto correct = complete_email_factor(
            database, token, email_code
        );

        require(correct.has_value(),
                "Correct email code was rejected");

        require(is_login_token(*correct),
                "Invalid recovery token format");

        // 2. Email challenge должен быть погашен.
        const auto remaining = database->execSqlSync(
            R"(
                SELECT COUNT(*) AS total
                FROM login_challenges
                WHERE token_hash = $1
            )",
            login_token_hash(token)
        );

        require(remaining[0]["total"].as<int>() == 0,
                "Email challenge was not consumed");

        // 3. Проверяем создание третьего фактора.
        const auto recovery = database->execSqlSync(
            R"(
                SELECT COUNT(*) AS total
                FROM recovery_challenges
                WHERE token_hash = $1
                  AND user_id = $2::uuid
                  AND expires_at > now()
            )",
            login_token_hash(*correct),
            user_id
        );

        require(recovery[0]["total"].as<int>() == 1,
                "Recovery challenge was not created");

        // 4. Повторный ввод не должен работать.
        const auto repeated = complete_email_factor(
            database, token, email_code
        );

        require(!repeated.has_value(),
                "Email code was reused");

        // 5. Просроченная попытка отклоняется.
        const auto expired_token = generate_login_token();
        create_attempt(expired_token, email_code, -60);

        const auto expired = complete_email_factor(
            database, expired_token, email_code
        );

        require(!expired.has_value(),
                "Expired email code was accepted");

        // 6. Несуществующий токен отклоняется.
        const auto unknown = complete_email_factor(
            database, generate_login_token(), email_code
        );

        require(!unknown.has_value(),
                "Unknown login attempt was accepted");

        // 7. Неверный формат кода отклоняется.
        const auto invalid = complete_email_factor(
            database, generate_login_token(), "abcdef"
        );

        require(!invalid.has_value(),
                "Invalid email code format was accepted");

        // 8. Второй фактор не выдаёт полную сессию.
        const auto sessions = database->execSqlSync(
            R"(
                SELECT COUNT(*) AS total
                FROM auth_sessions
                WHERE user_id = $1::uuid
            )",
            user_id
        );

        require(sessions[0]["total"].as<int>() == 0,
                "Full session created before third factor");

        // Удаление пользователя каскадно удаляет
        // связанные временные записи.
        database->execSqlSync(
            "DELETE FROM users WHERE id = $1::uuid",
            user_id
        );

        std::cout
            << "AUTH-20 integration test: PASSED\n";

        return 0;

    } catch (const std::exception& e) {
        std::cerr
            << "AUTH-20 integration test: FAILED: "
            << e.what() << '\n';

        if (!user_id.empty()) {
            try {
                database->execSqlSync(
                    "DELETE FROM users WHERE id = $1::uuid",
                    user_id
                );
            } catch (...) {
                std::cerr
                    << "Warning: test cleanup failed\n";
            }
        }

        return 1;
    }
}
