#include "auth/services/first_factor.h"

#include <utility>

namespace auth::services {

FirstFactor::FirstFactor(drogon::orm::DbClientPtr database,
                         std::shared_ptr<password::IPasswordHasher> hasher,
                         SmtpMailer mailer)
    : database_(std::move(database)),
      hasher_(std::move(hasher)),
      mailer_(std::move(mailer)),
      dummy_password_hash_(hasher_->hash_password("unused login challenge password")) {
}

std::optional<std::string> FirstFactor::start(std::string_view login, std::string_view password) const {
    constexpr int kMaxFailures = 5;
    constexpr int kLockSeconds = 900;
    const auto users = database_->execSqlSync(
        R"(
            SELECT id::text AS id, password_hash, COALESCE(login_locked_until > now(), FALSE) AS locked
            FROM users WHERE lower(login) = lower($1)
        )", std::string(login)
    );
    const bool available = !users.empty() && !users[0]["locked"].as<bool>();
    const auto stored_hash = available ? users[0]["password_hash"].as<std::string>() : dummy_password_hash_;
    const auto valid_password = hasher_->verify_password(password, stored_hash);
    if (!available) {
        return std::nullopt;
    }

    const auto user_id = users[0]["id"].as<std::string>();
    if (!valid_password) {
        // PostgreSQL serializes updates to the row and rechecks the WHERE clause after waiting.
        database_->execSqlSync(
            R"(
                UPDATE users
                SET failed_login_attempts = CASE
                        WHEN login_locked_until IS NULL
                            AND last_failed_login_at > now() - $4::int * interval '1 second'
                        THEN failed_login_attempts + 1 ELSE 1 END,
                    login_locked_until = CASE
                        WHEN login_locked_until IS NULL AND failed_login_attempts >= $3::int - 1
                            AND last_failed_login_at > now() - $4::int * interval '1 second'
                        THEN now() + $4::int * interval '1 second' END,
                    last_failed_login_at = now()
                WHERE id = $1::uuid AND password_hash = $2
                    AND (login_locked_until IS NULL OR login_locked_until <= now())
            )", user_id, stored_hash, kMaxFailures, kLockSeconds
        );
        return std::nullopt;
    }

    auto token = generate_login_token();
    const auto token_hash = login_token_hash(token);
    const auto code = generate_email_code();

    database_->execSqlSync("DELETE FROM login_challenges WHERE expires_at <= now()");
    const auto challenge = database_->execSqlSync(
        R"(
            WITH account AS (
                UPDATE users
                SET failed_login_attempts = 0, last_failed_login_at = NULL, login_locked_until = NULL
                WHERE id = $1::uuid AND password_hash = $2
                    AND (login_locked_until IS NULL OR login_locked_until <= now())
                RETURNING id, email
            ), issued AS (
                INSERT INTO login_challenges (token_hash, user_id, email_code_hash, expires_at)
                SELECT $3, id, $4, now() + $5::int * interval '1 second' FROM account
                RETURNING user_id
            )
            SELECT email FROM account JOIN issued ON issued.user_id = account.id
        )",
        user_id, stored_hash, token_hash, login_email_code_hash(token, code), kLoginChallengeLifetimeSeconds
    );
    if (challenge.empty()) {
        return std::nullopt;
    }

    try {
        mailer_.send_verification_code(challenge[0]["email"].as<std::string>(), code);
    } catch (...) {
        database_->execSqlSync("DELETE FROM login_challenges WHERE token_hash = $1", token_hash);
        throw;
    }

    return token;
}

}  // namespace auth::services
