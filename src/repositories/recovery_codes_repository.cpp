#include "auth/repositories/recovery_codes_repository.h"


#include <json/json.h>
#include <openssl/evp.h>
#include <array>
#include <cctype>

#include <stdexcept>
#include <string>
#include <unordered_set>

namespace auth::repositories {
namespace {
std::string hash_recovery_code(std::string_view code) {
    if (code.size() != 35) {
        throw std::invalid_argument("Invalid recovery code format");
    }

    std::string canonical;
    canonical.reserve(32);
    for (std::size_t i = 0; i < code.size(); ++i) {
        if (i == 8 || i == 17 || i == 26) {
            if (code[i] != '-') {
                throw std::invalid_argument("Invalid recovery code format");
            }
            continue;
        }
        const unsigned char ch = static_cast<unsigned char>(code[i]);
        if (!std::isxdigit(ch)) {
            throw std::invalid_argument("Invalid recovery code format");
        }
        canonical += static_cast<char>(std::toupper(ch));
    }

    const std::string material = "smp-auth/recovery/v1:" + canonical;
    std::array<unsigned char, EVP_MAX_MD_SIZE> digest{};
    unsigned int length = 0;
    if (EVP_Digest(material.data(), material.size(), digest.data(), &length, EVP_sha256(), nullptr) != 1 || length != 32) {
        throw std::runtime_error("Failed to hash recovery code");
    }

    constexpr char hex[] = "0123456789abcdef";
    std::string result;
    result.reserve(64);
    for (unsigned int i = 0; i < length; ++i) {
        result += hex[digest[i] >> 4];
        result += hex[digest[i] & 0x0F];
    }
    return result;
}

}


void save_recovery_codes(const drogon::orm::DbClientPtr& db, const std::string& user_id, const std::vector<std::string>& codes) {
    if (!db || user_id.empty() || codes.empty()) {
        throw std::invalid_argument("Database, user ID and recovery codes are required");
    }

    Json::Value hashes(Json::arrayValue);
    std::unordered_set<std::string> unique_hashes;
    for (const auto& code : codes) {
        auto hash = hash_recovery_code(code);
        if (!unique_hashes.insert(hash).second) {
            throw std::invalid_argument("Duplicate recovery code");
        }
        hashes.append(hash);
    }

    Json::StreamWriterBuilder writer;
    writer["indentation"] = "";
    const auto payload = Json::writeString(writer, hashes);

    db->execSqlSync(
        "INSERT INTO recovery_codes (user_id, code_hash) "
        "SELECT $1::uuid, decode(hashes.hash, 'hex') "
        "FROM jsonb_array_elements_text($2::jsonb) AS hashes(hash)",
        user_id, payload);
}

bool consume_recovery_code(const drogon::orm::DbClientPtr& db, const std::string& user_id, std::string_view code) {
    if (!db || user_id.empty()) {
        throw std::invalid_argument("Database and user ID are required");
    }

    std::string hash;
    try {
        hash = hash_recovery_code(code);
    } catch (const std::invalid_argument&) {
        return false;
    }

    const auto result = db->execSqlSync(
        "UPDATE recovery_codes SET used_at = now() "
        "WHERE user_id = $1::uuid AND code_hash = decode($2, 'hex') AND used_at IS NULL "
        "RETURNING 1",
        user_id, hash);
    return result.size() == 1;
}

}
