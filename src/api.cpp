#include "kuznyechik.h"

#include <cstring>
#include <limits>
#include <vector>

#include "block_cipher.h"
#include "cbc_mode.h"
#include "hex.h"

namespace {

// Переводит пару «указатель + длина» в vector. Вызывается только после
// проверки указателя: с NULL и нулевой длиной vector не строится.
std::vector<std::uint8_t> ToVector(const std::uint8_t* data, std::size_t size) {
    if (size == 0) {
        return std::vector<std::uint8_t>();
    }
    return std::vector<std::uint8_t>(data, data + size);
}

// Копирует результат в буфер вызывающей стороны. Размер пишется всегда,
// чтобы вызывающий мог повторить вызов с буфером нужного размера.
KuzStatus CopyOutput(const std::vector<std::uint8_t>& result, std::uint8_t* output,
                     std::size_t output_capacity, std::size_t* output_size) {
    *output_size = result.size();
    if (result.size() > output_capacity) {
        return KUZ_STATUS_BUFFER_TOO_SMALL;
    }
    if (output == nullptr && !result.empty()) {
        return KUZ_STATUS_INVALID_ARGUMENT;
    }
    if (!result.empty()) {
        std::memcpy(output, result.data(), result.size());
    }
    return KUZ_STATUS_OK;
}

// Проверки, одинаковые для шифрования и расшифровки.
KuzStatus CheckParameters(const std::uint8_t* key, std::size_t key_size,
                          const std::uint8_t* iv, std::size_t iv_size,
                          std::size_t* output_size) {
    if (output_size == nullptr) {
        return KUZ_STATUS_INVALID_ARGUMENT;
    }
    *output_size = 0;
    if (key == nullptr || iv == nullptr) {
        return KUZ_STATUS_INVALID_ARGUMENT;
    }
    if (key_size != kuznyechik::kKeySize) {
        return KUZ_STATUS_INVALID_KEY;
    }
    if (iv_size != kuznyechik::kBlockSize) {
        return KUZ_STATUS_INVALID_IV;
    }
    return KUZ_STATUS_OK;
}

}  // namespace

const char* KuzVersion(void) {
    return "1.0.0";
}

const char* KuzStatusMessage(KuzStatus status) {
    switch (status) {
        case KUZ_STATUS_OK:
            return "ok";
        case KUZ_STATUS_INVALID_KEY:
            return "invalid key size, expected 32 bytes";
        case KUZ_STATUS_INVALID_IV:
            return "invalid iv size, expected 16 bytes";
        case KUZ_STATUS_INVALID_ARGUMENT:
            return "invalid argument";
        case KUZ_STATUS_BUFFER_TOO_SMALL:
            return "output buffer is too small";
        case KUZ_STATUS_INVALID_PADDING:
            return "invalid padding: wrong key, wrong iv or corrupted data";
        case KUZ_STATUS_INTERNAL_ERROR:
            return "internal error";
    }
    return "unknown status";
}

size_t KuzCiphertextSize(size_t data_size) {
    const std::size_t block = kuznyechik::kBlockSize;
    if (data_size > std::numeric_limits<std::size_t>::max() - block) {
        return 0;
    }
    return (data_size / block + 1) * block;
}

KuzStatus KuzSelfTest(void) {
    try {
        // Вектор из ГОСТ Р 34.12-2015, приложение А.1.
        const std::vector<std::uint8_t> key = kuznyechik::BytesFromHex(
            "8899aabbccddeeff0011223344556677fedcba98765432100123456789abcdef");
        const std::vector<std::uint8_t> iv(kuznyechik::kBlockSize, 0);
        const std::vector<std::uint8_t> plaintext =
            kuznyechik::BytesFromHex("1122334455667700ffeeddccbbaa9988");
        const std::vector<std::uint8_t> expected =
            kuznyechik::BytesFromHex("7f679d90bebc24305a468d42b9d4edcd");

        const kuznyechik::BlockCipher cipher(key.data());

        std::uint8_t encrypted[kuznyechik::kBlockSize];
        cipher.EncryptBlock(plaintext.data(), encrypted);
        if (std::memcmp(encrypted, expected.data(), expected.size()) != 0) {
            return KUZ_STATUS_INTERNAL_ERROR;
        }

        std::uint8_t decrypted[kuznyechik::kBlockSize];
        cipher.DecryptBlock(encrypted, decrypted);
        if (std::memcmp(decrypted, plaintext.data(), plaintext.size()) != 0) {
            return KUZ_STATUS_INTERNAL_ERROR;
        }

        // Режим CBC: длина не кратна блоку, данные должны вернуться как были.
        const std::vector<std::uint8_t> data =
            kuznyechik::BytesFromHex("00112233445566778899aabbccddeeff0011");
        const std::vector<std::uint8_t> ciphertext = kuznyechik::EncryptCbc(key, iv, data);
        if (kuznyechik::DecryptCbc(key, iv, ciphertext) != data) {
            return KUZ_STATUS_INTERNAL_ERROR;
        }
        return KUZ_STATUS_OK;
    } catch (...) {
        return KUZ_STATUS_INTERNAL_ERROR;
    }
}

KuzStatus KuzEncrypt(const std::uint8_t* key, std::size_t key_size,
                     const std::uint8_t* iv, std::size_t iv_size,
                     const std::uint8_t* data, std::size_t data_size,
                     std::uint8_t* output, std::size_t output_capacity,
                     std::size_t* output_size) {
    const KuzStatus status = CheckParameters(key, key_size, iv, iv_size, output_size);
    if (status != KUZ_STATUS_OK) {
        return status;
    }
    if (data == nullptr && data_size != 0) {
        return KUZ_STATUS_INVALID_ARGUMENT;
    }

    // Граница DLL: исключения наружу не выходят, наружу отдаются коды.
    try {
        const std::vector<std::uint8_t> key_bytes = ToVector(key, key_size);
        const std::vector<std::uint8_t> iv_bytes = ToVector(iv, iv_size);
        const std::vector<std::uint8_t> input = ToVector(data, data_size);
        const std::vector<std::uint8_t> ciphertext =
            kuznyechik::EncryptCbc(key_bytes, iv_bytes, input);
        return CopyOutput(ciphertext, output, output_capacity, output_size);
    } catch (...) {
        return KUZ_STATUS_INTERNAL_ERROR;
    }
}

KuzStatus KuzDecrypt(const std::uint8_t* key, std::size_t key_size,
                     const std::uint8_t* iv, std::size_t iv_size,
                     const std::uint8_t* data, std::size_t data_size,
                     std::uint8_t* output, std::size_t output_capacity,
                     std::size_t* output_size) {
    const KuzStatus status = CheckParameters(key, key_size, iv, iv_size, output_size);
    if (status != KUZ_STATUS_OK) {
        return status;
    }
    if (data == nullptr && data_size != 0) {
        return KUZ_STATUS_INVALID_ARGUMENT;
    }
    if (data_size == 0 || data_size % kuznyechik::kBlockSize != 0) {
        return KUZ_STATUS_INVALID_ARGUMENT;
    }

    try {
        const std::vector<std::uint8_t> key_bytes = ToVector(key, key_size);
        const std::vector<std::uint8_t> iv_bytes = ToVector(iv, iv_size);
        const std::vector<std::uint8_t> input = ToVector(data, data_size);
        const std::vector<std::uint8_t> plaintext =
            kuznyechik::DecryptCbc(key_bytes, iv_bytes, input);
        return CopyOutput(plaintext, output, output_capacity, output_size);
    } catch (const kuznyechik::PaddingError&) {
        return KUZ_STATUS_INVALID_PADDING;
    } catch (...) {
        return KUZ_STATUS_INTERNAL_ERROR;
    }
}
