#include "auth/services/first_factor.h"

#include <openssl/evp.h>
#include <openssl/hmac.h>
#include <openssl/rand.h>

#include <array>
#include <cstdint>
#include <limits>
#include <memory>
#include <stdexcept>
#include <utility>

namespace auth::services {

namespace {

constexpr std::size_t kDigestSize = 32;
using Digest = std::array<unsigned char, kDigestSize>;

std::string to_hex(const Digest& bytes) {
    constexpr std::string_view digits = "0123456789abcdef";
    std::string hex;
    hex.reserve(bytes.size() * 2);
    for (const auto byte : bytes) {
        hex += digits[byte >> 4];
        hex += digits[byte & 0xf];
    }
    return hex;
}

void fill_random(unsigned char* bytes, int size) {
    if (RAND_bytes(bytes, size) != 1) {
        throw std::runtime_error("Login challenge random generation failed");
    }
}

std::string generate_email_code() {
    constexpr std::uint32_t range = 1000000;
    constexpr auto limit = std::numeric_limits<std::uint32_t>::max() / range * range;
    std::uint32_t value;
    do {
        fill_random(reinterpret_cast<unsigned char*>(&value), sizeof(value));
    } while (value >= limit);

    auto code = std::to_string(value % range);
    return std::string(6 - code.size(), '0') + code;
}

}  // namespace

std::string login_token_hash(std::string_view token) {
    Digest hash{};
    std::size_t size = hash.size();
    if (EVP_Q_digest(nullptr, "SHA256", nullptr, token.data(), token.size(), hash.data(), &size) != 1) {
        throw std::runtime_error("Login token hashing failed");
    }
    return to_hex(hash);
}

std::string login_email_code_hash(std::string_view token, std::string_view code) {
    if (token.size() != kDigestSize * 2) {
        throw std::invalid_argument("Invalid login challenge token size");
    }

    Digest hash{};
    unsigned int size = hash.size();
    const auto* code_bytes = reinterpret_cast<const unsigned char*>(code.data());
    if (HMAC(EVP_sha256(), token.data(), static_cast<int>(token.size()),
             code_bytes, code.size(), hash.data(), &size) == nullptr) {
        throw std::runtime_error("Login email code hashing failed");
    }
    return to_hex(hash);
}

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

    Digest random_token{};
    fill_random(random_token.data(), static_cast<int>(random_token.size()));
    auto token = to_hex(random_token);
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
