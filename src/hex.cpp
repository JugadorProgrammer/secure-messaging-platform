#include "hex.h"

#include <stdexcept>

namespace kuznyechik {
namespace {

// Возвращает значение hex-цифры или -1, если символ не hex.
int HexDigit(char symbol) {
    if (symbol >= '0' && symbol <= '9') {
        return symbol - '0';
    }
    if (symbol >= 'a' && symbol <= 'f') {
        return symbol - 'a' + 10;
    }
    if (symbol >= 'A' && symbol <= 'F') {
        return symbol - 'A' + 10;
    }
    return -1;
}

}  // namespace

std::vector<std::uint8_t> BytesFromHex(const std::string& text) {
    if (text.size() % 2 != 0) {
        throw std::invalid_argument("hex string has odd length");
    }

    std::vector<std::uint8_t> bytes;
    bytes.reserve(text.size() / 2);
    for (std::size_t index = 0; index < text.size(); index += 2) {
        const int high = HexDigit(text[index]);
        const int low = HexDigit(text[index + 1]);
        if (high < 0 || low < 0) {
            throw std::invalid_argument("hex string contains a non-hex character");
        }
        bytes.push_back(static_cast<std::uint8_t>(high * 16 + low));
    }
    return bytes;
}

std::string HexFromBytes(const std::vector<std::uint8_t>& bytes) {
    const char* const kDigits = "0123456789abcdef";
    std::string text;
    text.reserve(bytes.size() * 2);
    for (const std::uint8_t byte : bytes) {
        text.push_back(kDigits[byte >> 4]);
        text.push_back(kDigits[byte & 0x0F]);
    }
    return text;
}

}  // namespace kuznyechik
