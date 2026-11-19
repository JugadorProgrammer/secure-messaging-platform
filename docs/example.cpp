// Минимальный пример работы с DLL «Кузнечик».
//
// Сборка (MinGW):
//   g++ -std=c++17 -I. example.cpp libkuznyechik.dll.a -static -o example.exe
// -static нужен, чтобы примеру не требовались libstdc++-6.dll и libgcc_s_seh-1.dll.
// Запуск: kuznyechik.dll должна лежать рядом с example.exe.

#include <cstdint>
#include <cstdio>
#include <string>
#include <vector>

#include "kuznyechik.h"

namespace {

int HexDigit(char symbol) {
    if (symbol >= '0' && symbol <= '9') {
        return symbol - '0';
    }
    if (symbol >= 'a' && symbol <= 'f') {
        return symbol - 'a' + 10;
    }
    return symbol - 'A' + 10;
}

std::vector<std::uint8_t> FromHex(const std::string& text) {
    std::vector<std::uint8_t> bytes;
    bytes.reserve(text.size() / 2);
    for (std::size_t index = 0; index + 1 < text.size(); index += 2) {
        bytes.push_back(static_cast<std::uint8_t>(HexDigit(text[index]) * 16 +
                                                  HexDigit(text[index + 1])));
    }
    return bytes;
}

std::string ToHex(const std::vector<std::uint8_t>& bytes) {
    const char* const kDigits = "0123456789abcdef";
    std::string text;
    for (const std::uint8_t byte : bytes) {
        text.push_back(kDigits[byte >> 4]);
        text.push_back(kDigits[byte & 0x0F]);
    }
    return text;
}

}  // namespace

int main() {
    // Ключ и IV — в проекте их готовит Крипто-модуль и передаёт в hex.
    const std::vector<std::uint8_t> key = FromHex(
        "8899aabbccddeeff0011223344556677fedcba98765432100123456789abcdef");
    const std::vector<std::uint8_t> iv = FromHex("1234567890abcdef1234567890abcdef");

    const std::string message = "hello kuznyechik";
    const std::vector<std::uint8_t> plaintext(message.begin(), message.end());

    // 1. Шифрование.
    std::vector<std::uint8_t> ciphertext(KuzCiphertextSize(plaintext.size()));
    std::size_t ciphertext_size = 0;
    KuzStatus status = KuzEncrypt(key.data(), key.size(), iv.data(), iv.size(),
                                  plaintext.data(), plaintext.size(), ciphertext.data(),
                                  ciphertext.size(), &ciphertext_size);
    if (status != KUZ_STATUS_OK) {
        std::printf("encrypt: %s\n", KuzStatusMessage(status));
        return 1;
    }
    ciphertext.resize(ciphertext_size);
    std::printf("ciphertext: %s\n", ToHex(ciphertext).c_str());

    // 2. Расшифровка.
    std::vector<std::uint8_t> restored(ciphertext.size());
    std::size_t restored_size = 0;
    status = KuzDecrypt(key.data(), key.size(), iv.data(), iv.size(), ciphertext.data(),
                        ciphertext.size(), restored.data(), restored.size(), &restored_size);
    if (status != KUZ_STATUS_OK) {
        std::printf("decrypt: %s\n", KuzStatusMessage(status));
        return 1;
    }
    restored.resize(restored_size);

    if (std::string(restored.begin(), restored.end()) != message) {
        std::printf("round trip mismatch\n");
        return 1;
    }

    std::printf("version: %s\n", KuzVersion());
    std::printf("selftest: %s\n", KuzStatusMessage(KuzSelfTest()));
    std::printf("example ok\n");
    return 0;
}
