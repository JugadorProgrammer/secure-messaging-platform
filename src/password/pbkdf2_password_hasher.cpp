#include "auth/password/pbkdf2_password_hasher.h"

#include <openssl/crypto.h>
#include <openssl/evp.h>
#include <openssl/rand.h>

#include <array>
#include <limits>
#include <stdexcept>

namespace auth::password {

namespace {

constexpr int kIterations = 600000;
constexpr std::string_view kPrefix = "pbkdf2-sha256$600000$";
constexpr std::string_view kHex = "0123456789abcdef";
using Salt = std::array<unsigned char, 16>;
using Digest = std::array<unsigned char, 32>;

template <std::size_t Size>
std::string to_hex(const std::array<unsigned char, Size>& bytes) {
    std::string hex;
    hex.reserve(Size * 2);
    for (const auto byte : bytes) {
        hex += kHex[byte >> 4];
        hex += kHex[byte & 0xf];
    }
    return hex;
}

template <std::size_t Size>
bool from_hex(std::string_view hex, std::array<unsigned char, Size>& bytes) {
    if (hex.size() != Size * 2) {
        return false;
    }

    for (std::size_t i = 0; i < Size; ++i) {
        const auto high = kHex.find(hex[i * 2]);
        const auto low = kHex.find(hex[i * 2 + 1]);
        if (high == kHex.npos || low == kHex.npos) {
            return false;
        }
        bytes[i] = static_cast<unsigned char>(high * 16 + low);
    }
    return true;
}

Digest derive_key(std::string_view password, const Salt& salt) {
    if (password.size() > std::numeric_limits<int>::max()) {
        throw std::invalid_argument("Password is too long");
    }

    Digest key{};
    if (PKCS5_PBKDF2_HMAC(
            password.data(), static_cast<int>(password.size()),
            salt.data(), static_cast<int>(salt.size()), kIterations, EVP_sha256(),
            static_cast<int>(key.size()), key.data()) != 1) {
        throw std::runtime_error("Password key derivation failed");
    }
    return key;
}

}  // namespace

std::string Pbkdf2PasswordHasher::hash_password(std::string_view password) {
    Salt salt{};
    if (RAND_bytes(salt.data(), static_cast<int>(salt.size())) != 1) {
        throw std::runtime_error("Password salt generation failed");
    }
    return std::string(kPrefix) + to_hex(salt) + "$" + to_hex(derive_key(password, salt));
}

bool Pbkdf2PasswordHasher::verify_password(std::string_view password, std::string_view stored_hash) {
    Salt salt{};
    Digest expected{};
    const auto salt_size = salt.size() * 2;
    const auto digest_size = expected.size() * 2;

    if (!stored_hash.starts_with(kPrefix)) {
        return false;
    }
    stored_hash.remove_prefix(kPrefix.size());
    if (stored_hash.size() != salt_size + 1 + digest_size ||
        stored_hash[salt_size] != '$' ||
        !from_hex(stored_hash.substr(0, salt_size), salt) ||
        !from_hex(stored_hash.substr(salt_size + 1), expected)) {
        return false;
    }

    const auto actual = derive_key(password, salt);
    return CRYPTO_memcmp(actual.data(), expected.data(), actual.size()) == 0;
}

}  // namespace auth::password
