#include <windows.h>

#include <cstdio>
#include <cstring>
#include <vector>

#include "hex.h"

namespace {

// Проверка «чёрным ящиком»: DLL грузится через LoadLibrary, поэтому
// здесь дублируются сигнатуры и код успеха, без заголовка и import library.
const int kStatusOk = 0;

using VersionFn = const char* (*)(void);
using SelfTestFn = int (*)(void);
using EncryptFn = int (*)(const std::uint8_t*, std::size_t, const std::uint8_t*, std::size_t,
                          const std::uint8_t*, std::size_t, std::uint8_t*, std::size_t,
                          std::size_t*);
using DecryptFn = int (*)(const std::uint8_t*, std::size_t, const std::uint8_t*, std::size_t,
                          const std::uint8_t*, std::size_t, std::uint8_t*, std::size_t,
                          std::size_t*);

// Приведение через void* — идиома Windows, при которой GCC не ругается
// на несовместимый тип функции.
template <typename Function>
Function Find(HMODULE module, const char* name) {
    void* address = reinterpret_cast<void*>(GetProcAddress(module, name));
    return reinterpret_cast<Function>(address);
}

}  // namespace

int main() {
    HMODULE module = LoadLibraryA("kuznyechik.dll");
    if (module == nullptr) {
        std::printf("cannot load kuznyechik.dll, error %lu\n", GetLastError());
        return 1;
    }

    const VersionFn version = Find<VersionFn>(module, "KuzVersion");
    const SelfTestFn selftest = Find<SelfTestFn>(module, "KuzSelfTest");
    const EncryptFn encrypt = Find<EncryptFn>(module, "KuzEncrypt");
    const DecryptFn decrypt = Find<DecryptFn>(module, "KuzDecrypt");

    if (version == nullptr || selftest == nullptr || encrypt == nullptr || decrypt == nullptr) {
        std::printf("some exports are missing\n");
        FreeLibrary(module);
        return 1;
    }
    std::printf("exports found, version %s\n", version());

    if (selftest() != kStatusOk) {
        std::printf("selftest failed\n");
        FreeLibrary(module);
        return 1;
    }

    const std::vector<std::uint8_t> key = kuznyechik::BytesFromHex(
        "8899aabbccddeeff0011223344556677fedcba98765432100123456789abcdef");
    const std::vector<std::uint8_t> iv(16, 0);
    const char* const text = "hello kuznyechik";
    const std::vector<std::uint8_t> plaintext(text, text + std::strlen(text));

    std::vector<std::uint8_t> ciphertext(plaintext.size() + 16);
    std::size_t ciphertext_size = 0;
    if (encrypt(key.data(), key.size(), iv.data(), iv.size(), plaintext.data(),
                plaintext.size(), ciphertext.data(), ciphertext.size(), &ciphertext_size) !=
        kStatusOk) {
        std::printf("encryption failed\n");
        FreeLibrary(module);
        return 1;
    }
    ciphertext.resize(ciphertext_size);

    std::vector<std::uint8_t> restored(ciphertext.size());
    std::size_t restored_size = 0;
    if (decrypt(key.data(), key.size(), iv.data(), iv.size(), ciphertext.data(),
                ciphertext.size(), restored.data(), restored.size(), &restored_size) !=
        kStatusOk) {
        std::printf("decryption failed\n");
        FreeLibrary(module);
        return 1;
    }
    restored.resize(restored_size);

    if (restored != plaintext) {
        std::printf("round trip mismatch\n");
        FreeLibrary(module);
        return 1;
    }

    std::printf("ciphertext: %s\n", kuznyechik::HexFromBytes(ciphertext).c_str());
    std::printf("DLL check passed\n");
    FreeLibrary(module);
    return 0;
}
