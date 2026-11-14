#include "auth/services/recovery_codes.h"

#include <openssl/rand.h>

#include <array>
#include <stdexcept>
#include <string>
#include <unordered_set>
#include <utility>

namespace auth::services {

std::vector<std::string> generate_recovery_codes(std::size_t count) {
    if (count == 0 || count > 100) {
        throw std::invalid_argument("Recovery code count must be between 1 and 100");
    }

    constexpr char hex[] = "0123456789ABCDEF";
    std::vector<std::string> codes;
    codes.reserve(count);

    std::unordered_set<std::string> unique_codes;
    while (codes.size() < count) {
        std::array<unsigned char, 16> bytes{};
        if (RAND_bytes(bytes.data(), static_cast<int>(bytes.size())) != 1) {
            throw std::runtime_error("Failed to generate recovery code");
        }

        std::string code;
        code.reserve(35);
        for (std::size_t i = 0; i < bytes.size(); ++i) {
            if (i > 0 && i % 4 == 0) {
                code += '-';
            }
            code += hex[bytes[i] >> 4];
            code += hex[bytes[i] & 0x0F];
        }
        if (unique_codes.insert(code).second) {
            codes.push_back(std::move(code));
        }
    }
    return codes;
}

}
