#include "auth/services/login_token.h"

#include <openssl/evp.h>
#include <openssl/hmac.h>
#include <openssl/rand.h>

#include <algorithm>
#include <array>
#include <stdexcept>

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

}  // namespace

std::string generate_login_token() {
    Digest bytes{};
    if (RAND_bytes(bytes.data(), static_cast<int>(bytes.size())) != 1) {
        throw std::runtime_error("Login token generation failed");
    }
    return to_hex(bytes);
}

bool is_login_token(std::string_view token) {
    return token.size() == kDigestSize * 2 &&
        std::all_of(token.begin(), token.end(), [](char ch) {
            return (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f');
        });
}

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

}  // namespace auth::services
