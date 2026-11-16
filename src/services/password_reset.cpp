#include "auth/services/password_reset.h"

#include <algorithm>
#include <future>
#include <stdexcept>
#include <utility>

#include "auth/services/login_token.h"

namespace auth::services {

namespace {

constexpr std::size_t kNonceLength = 64;

bool is_pending_token(std::string_view token) {
    return token.size() == 2 * kNonceLength && is_login_token(token.substr(0, kNonceLength))
        && is_login_token(token.substr(kNonceLength));
}

}  // namespace

PasswordReset::PasswordReset(drogon::orm::DbClientPtr database,
                             std::shared_ptr<password::IPasswordHasher> hasher,
                             SmtpMailer mailer)
    : database_(std::move(database)), hasher_(std::move(hasher)), mailer_(std::move(mailer)) {
}

PasswordResetRequest PasswordReset::request_code(
    std::string_view email, std::string_view previous_token
) const {
    constexpr int kMaxCodeAttempts = 5;
    constexpr int kResendSeconds = 60;
    std::string address(email);
    std::transform(address.begin(), address.end(), address.begin(), [](char ch) {
        return ch >= 'A' && ch <= 'Z' ? static_cast<char>(ch - 'A' + 'a') : ch;
    });
    // Cookie rotation depends on the submitted address and browser, independent of account existence.
    auto token = std::string(previous_token);
    const auto previous_nonce = previous_token.substr(0, kNonceLength);
    if (!is_pending_token(previous_token)
        || previous_token.substr(kNonceLength) != login_email_code_hash(previous_nonce, address)) {
        token = generate_login_token();
        token += login_email_code_hash(token, address);
    }
    const auto nonce = std::string_view(token).substr(0, kNonceLength);
    auto code = generate_email_code();
    // A single outstanding request per account limits delivery and serializes resends.
    const auto result = database_->execSqlSync(
        R"(
            WITH account AS (
                SELECT id, email FROM users WHERE lower(email) = lower($1)
            ), requested AS (
                INSERT INTO password_reset_challenges
                    (user_id, token_hash, code_hash, attempts_remaining, resend_after, expires_at)
                SELECT id, $2, $3, $4, now() + $5::int * interval '1 second',
                       now() + $6::int * interval '1 second'
                FROM account
                ON CONFLICT (user_id) DO UPDATE
                SET token_hash = EXCLUDED.token_hash,
                    code_hash = EXCLUDED.code_hash,
                    attempts_remaining = EXCLUDED.attempts_remaining,
                    resend_after = EXCLUDED.resend_after,
                    expires_at = EXCLUDED.expires_at
                WHERE password_reset_challenges.resend_after <= now()
                    AND (password_reset_challenges.code_hash IS NOT NULL
                         OR password_reset_challenges.expires_at <= now())
                RETURNING user_id
            )
            SELECT email FROM account JOIN requested ON requested.user_id = account.id
        )",
        std::string(email), login_token_hash(token), login_email_code_hash(nonce, code),
        kMaxCodeAttempts, kResendSeconds, kPasswordResetLifetimeSeconds
    );

    if (result.empty()) {
        return {std::move(token), std::nullopt};
    }
    return {std::move(token), PasswordResetDelivery{result[0]["email"].as<std::string>(), std::move(code)}};
}

void PasswordReset::send_code(const PasswordResetRequest& request) const {
    if (!request.delivery) {
        return;
    }
    try {
        mailer_.send_password_reset_code(request.delivery->email, request.delivery->code);
    } catch (...) {
        const auto nonce = std::string_view(request.pending_token).substr(0, kNonceLength);
        database_->execSqlSync(
            R"(
                UPDATE password_reset_challenges SET attempts_remaining = 0
                WHERE token_hash = $1 AND code_hash = $2
            )",
            login_token_hash(request.pending_token), login_email_code_hash(nonce, request.delivery->code)
        );
        throw;
    }
}

std::optional<std::string> PasswordReset::confirm_code(
    std::string_view pending_token, std::string_view code
) const {
    if (!is_pending_token(pending_token)) {
        return std::nullopt;
    }

    auto grant = generate_login_token();
    const auto result = database_->execSqlSync(
        R"(
            UPDATE password_reset_challenges
            SET token_hash = CASE WHEN code_hash = $2 THEN $3 ELSE token_hash END,
                expires_at = CASE WHEN code_hash = $2
                    THEN now() + $4::int * interval '1 second' ELSE expires_at END,
                code_hash = CASE WHEN code_hash = $2 THEN NULL ELSE code_hash END,
                attempts_remaining = attempts_remaining - 1
            WHERE token_hash = $1 AND code_hash IS NOT NULL
                AND expires_at > now() AND attempts_remaining > 0
            RETURNING code_hash IS NULL AS confirmed
        )",
        login_token_hash(pending_token), login_email_code_hash(pending_token.substr(0, kNonceLength), code),
        login_token_hash(grant), kPasswordResetLifetimeSeconds
    );

    if (result.empty() || !result[0]["confirmed"].as<bool>()) {
        return std::nullopt;
    }
    return grant;
}

bool PasswordReset::change_password(std::string_view grant_token, std::string_view password) const {
    if (!is_login_token(grant_token)) {
        return false;
    }

    const auto token_hash = login_token_hash(grant_token);
    const auto password_hash = hasher_->hash_password(password);
    auto commit = std::make_shared<std::promise<bool>>();
    auto commit_result = commit->get_future();
    {
        const auto transaction = database_->newTransaction([commit](bool success) {
            commit->set_value(success);
        });
        if (!transaction) {
            throw std::runtime_error("Cannot start password reset transaction");
        }

        try {
            // Use the same user-row lock as login and final session issuance.
            const auto account = transaction->execSqlSync(
                R"(
                    SELECT u.id::text AS user_id
                    FROM users u JOIN password_reset_challenges c ON c.user_id = u.id
                    WHERE c.token_hash = $1 AND c.code_hash IS NULL AND c.expires_at > now()
                    FOR UPDATE OF u
                )", token_hash
            );
            if (account.empty()) {
                transaction->rollback();
                return false;
            }

            const auto user_id = account[0]["user_id"].as<std::string>();
            const auto consumed = transaction->execSqlSync(
                R"(
                    DELETE FROM password_reset_challenges
                    WHERE user_id = $1::uuid AND token_hash = $2 AND code_hash IS NULL
                        AND expires_at > now()
                    RETURNING user_id
                )", user_id, token_hash
            );
            if (consumed.empty()) {
                transaction->rollback();
                return false;
            }

            transaction->execSqlSync(
                R"(
                    UPDATE users
                    SET password_hash = $2, failed_login_attempts = 0,
                        last_failed_login_at = NULL, login_locked_until = NULL
                    WHERE id = $1::uuid
                )", user_id, password_hash
            );
            transaction->execSqlSync("DELETE FROM auth_sessions WHERE user_id = $1::uuid", user_id);
            transaction->execSqlSync("DELETE FROM login_challenges WHERE user_id = $1::uuid", user_id);
            transaction->execSqlSync("DELETE FROM recovery_challenges WHERE user_id = $1::uuid", user_id);
        } catch (...) {
            transaction->rollback();
            throw;
        }
    }

    if (!commit_result.get()) {
        throw std::runtime_error("Password reset transaction was not committed");
    }
    return true;
}

}  // namespace auth::services
