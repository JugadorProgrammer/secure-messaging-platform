#include "cbc_mode.h"

#include <cstring>

#include "block_cipher.h"

namespace kuznyechik {

std::vector<std::uint8_t> AddPadding(const std::vector<std::uint8_t>& data) {
    // Если длина уже кратна блоку, добавляется целый блок из 0x10.
    const std::size_t padding = kBlockSize - data.size() % kBlockSize;
    std::vector<std::uint8_t> result = data;
    result.insert(result.end(), padding, static_cast<std::uint8_t>(padding));
    return result;
}

void RemovePadding(std::vector<std::uint8_t>& data) {
    if (data.empty() || data.size() % kBlockSize != 0) {
        throw PaddingError("padded data must be a positive multiple of 16 bytes");
    }
    const std::size_t padding = data.back();
    if (padding == 0 || padding > kBlockSize || padding > data.size()) {
        throw PaddingError("invalid padding length");
    }
    for (std::size_t index = data.size() - padding; index < data.size(); ++index) {
        if (data[index] != padding) {
            throw PaddingError("inconsistent padding bytes");
        }
    }
    data.resize(data.size() - padding);
}

std::vector<std::uint8_t> EncryptCbc(const std::vector<std::uint8_t>& key,
                                     const std::vector<std::uint8_t>& iv,
                                     const std::vector<std::uint8_t>& data) {
    if (key.size() != kKeySize) {
        throw std::invalid_argument("key must be 32 bytes");
    }
    if (iv.size() != kBlockSize) {
        throw std::invalid_argument("iv must be 16 bytes");
    }

    const BlockCipher cipher(key.data());
    const std::vector<std::uint8_t> padded = AddPadding(data);
    std::vector<std::uint8_t> ciphertext(padded.size());

    // C[i] = E(P[i] xor C[i-1]), C[-1] = IV.
    std::uint8_t previous[kBlockSize];
    std::memcpy(previous, iv.data(), kBlockSize);

    for (std::size_t offset = 0; offset < padded.size(); offset += kBlockSize) {
        std::uint8_t block[kBlockSize];
        for (std::size_t index = 0; index < kBlockSize; ++index) {
            block[index] = static_cast<std::uint8_t>(padded[offset + index] ^ previous[index]);
        }

        std::uint8_t encrypted[kBlockSize];
        cipher.EncryptBlock(block, encrypted);
        std::memcpy(&ciphertext[offset], encrypted, kBlockSize);
        std::memcpy(previous, encrypted, kBlockSize);
    }
    return ciphertext;
}

std::vector<std::uint8_t> DecryptCbc(const std::vector<std::uint8_t>& key,
                                     const std::vector<std::uint8_t>& iv,
                                     const std::vector<std::uint8_t>& ciphertext) {
    if (key.size() != kKeySize) {
        throw std::invalid_argument("key must be 32 bytes");
    }
    if (iv.size() != kBlockSize) {
        throw std::invalid_argument("iv must be 16 bytes");
    }
    if (ciphertext.empty() || ciphertext.size() % kBlockSize != 0) {
        throw std::invalid_argument("ciphertext must be a positive multiple of 16 bytes");
    }

    const BlockCipher cipher(key.data());
    std::vector<std::uint8_t> plaintext(ciphertext.size());

    // P[i] = D(C[i]) xor C[i-1], C[-1] = IV.
    std::uint8_t previous[kBlockSize];
    std::memcpy(previous, iv.data(), kBlockSize);

    for (std::size_t offset = 0; offset < ciphertext.size(); offset += kBlockSize) {
        std::uint8_t decrypted[kBlockSize];
        cipher.DecryptBlock(&ciphertext[offset], decrypted);

        std::uint8_t block[kBlockSize];
        for (std::size_t index = 0; index < kBlockSize; ++index) {
            block[index] = static_cast<std::uint8_t>(decrypted[index] ^ previous[index]);
        }
        std::memcpy(&plaintext[offset], block, kBlockSize);

        // Предыдущий блок — это текущий блок шифртекста, а не открытого текста.
        std::memcpy(previous, &ciphertext[offset], kBlockSize);
    }

    RemovePadding(plaintext);
    return plaintext;
}

}  // namespace kuznyechik
