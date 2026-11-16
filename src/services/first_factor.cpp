#include "auth/services/first_factor.h"

#include <openssl/rand.h>

#include <cstdint>
#include <limits>
#include <memory>
#include <stdexcept>
#include <utility>

namespace auth::services {

namespace {

std::string generate_email_code() {
    constexpr std::uint32_t range = 1000000;
    constexpr auto limit = std::numeric_limits<std::uint32_t>::max() / range * range;
    std::uint32_t value;
    do {
        if (RAND_bytes(reinterpret_cast<unsigned char*>(&value), sizeof(value)) != 1) {
            throw std::runtime_error("Login challenge random generation failed");
        }
    } while (value >= limit);

    auto code = std::to_string(value % range);
    return std::string(6 - code.size(), '0') + code;
}

}  // namespace

FirstFactor::FirstFactor(drogon::orm::DbClientPtr database,
                         std::shared_ptr<password::IPasswordHasher> hasher,
                         SmtpMailer mailer)
    : database_(std::move(database)),
      users_(database_),
      hasher_(std::move(hasher)),
      mailer_(std::move(mailer)),
      dummy_password_hash_(hasher_->hash_password("unused login challenge password")) {
}

std::optional<std::string> FirstFactor::start(std::string_view login, std::string_view password) const {
    const auto user = users_.find_by_login(login);
    const auto& stored_hash = user ? user->password_hash : dummy_password_hash_;
    if (!hasher_->verify_password(password, stored_hash) || !user) {
        return std::nullopt;
    }

    auto token = generate_login_token();
    const auto token_hash = login_token_hash(token);
    const auto code = generate_email_code();

    database_->execSqlSync("DELETE FROM login_challenges WHERE expires_at <= now()");
    database_->execSqlSync(
        R"(
            INSERT INTO login_challenges (token_hash, user_id, email_code_hash, expires_at)
            VALUES ($1, $2::uuid, $3, now() + $4::int * interval '1 second')
        )",
        token_hash, user->id, login_email_code_hash(token, code), kLoginChallengeLifetimeSeconds
    );

    try {
        mailer_.send_verification_code(user->email, code);
    } catch (...) {
        database_->execSqlSync("DELETE FROM login_challenges WHERE token_hash = $1", token_hash);
        throw;
    }

    return token;
}

}  // namespace auth::services
