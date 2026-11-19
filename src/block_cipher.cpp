#include "block_cipher.h"

#include <cstring>

#include "kuz_tables.h"

namespace kuznyechik {
namespace {

// Умножение в GF(2^8) по модулю p(x) = x^8 + x^7 + x^6 + x + 1.
// Многочлену соответствует 0x1C3, поэтому при переполнении байта
// складываем с 0xC3 (старший бит уже ушёл за разрядную сетку).
std::uint8_t MultiplyInField(std::uint8_t left, std::uint8_t right) {
    std::uint8_t result = 0;
    while (right != 0) {
        if ((right & 1U) != 0U) {
            result ^= left;
        }
        const bool overflow = (left & 0x80U) != 0U;
        left = static_cast<std::uint8_t>(left << 1);
        if (overflow) {
            left ^= 0xC3U;
        }
        right = static_cast<std::uint8_t>(right >> 1);
    }
    return result;
}

// Линейная функция l(a_15, ..., a_0): сумма байтов с коэффициентами.
std::uint8_t LinearSum(const std::uint8_t* block) {
    std::uint8_t sum = 0;
    for (std::size_t index = 0; index < kBlockSize; ++index) {
        sum ^= MultiplyInField(block[index], kLinearCoefficients[index]);
    }
    return sum;
}

}  // namespace

void Substitute(const std::uint8_t* input, std::uint8_t* output) {
    for (std::size_t index = 0; index < kBlockSize; ++index) {
        output[index] = kSbox[input[index]];
    }
}

void SubstituteInverse(const std::uint8_t* input, std::uint8_t* output) {
    for (std::size_t index = 0; index < kBlockSize; ++index) {
        output[index] = kSboxInverse[input[index]];
    }
}

void XorWithKey(std::uint8_t* state, const std::uint8_t* key) {
    for (std::size_t index = 0; index < kBlockSize; ++index) {
        state[index] ^= key[index];
    }
}

// R(a_15, ..., a_0) = l(a_15, ..., a_0) || a_15 || ... || a_1.
void R(const std::uint8_t* input, std::uint8_t* output) {
    std::uint8_t result[kBlockSize];
    result[0] = LinearSum(input);
    for (std::size_t index = 1; index < kBlockSize; ++index) {
        result[index] = input[index - 1];
    }
    std::memcpy(output, result, kBlockSize);
}

// R^-1(a_15, ..., a_0) = a_14 || ... || a_0 || l(a_14, ..., a_0, a_15).
void RInverse(const std::uint8_t* input, std::uint8_t* output) {
    std::uint8_t rotated[kBlockSize];
    for (std::size_t index = 0; index + 1 < kBlockSize; ++index) {
        rotated[index] = input[index + 1];
    }
    rotated[kBlockSize - 1] = input[0];

    std::uint8_t result[kBlockSize];
    for (std::size_t index = 0; index + 1 < kBlockSize; ++index) {
        result[index] = input[index + 1];
    }
    result[kBlockSize - 1] = LinearSum(rotated);
    std::memcpy(output, result, kBlockSize);
}

// L = R^16: сдвиг на 16 байт возвращает исходную расстановку.
void L(const std::uint8_t* input, std::uint8_t* output) {
    std::uint8_t current[kBlockSize];
    std::memcpy(current, input, kBlockSize);
    for (std::size_t step = 0; step < kBlockSize; ++step) {
        std::uint8_t next[kBlockSize];
        R(current, next);
        std::memcpy(current, next, kBlockSize);
    }
    std::memcpy(output, current, kBlockSize);
}

// L^-1 = (R^-1)^16.
void LInverse(const std::uint8_t* input, std::uint8_t* output) {
    std::uint8_t current[kBlockSize];
    std::memcpy(current, input, kBlockSize);
    for (std::size_t step = 0; step < kBlockSize; ++step) {
        std::uint8_t next[kBlockSize];
        RInverse(current, next);
        std::memcpy(current, next, kBlockSize);
    }
    std::memcpy(output, current, kBlockSize);
}

void LS(const std::uint8_t* input, std::uint8_t* output) {
    std::uint8_t substituted[kBlockSize];
    Substitute(input, substituted);
    L(substituted, output);
}

void LSInverse(const std::uint8_t* input, std::uint8_t* output) {
    // Обратное к L∘S — это S^-1∘L^-1: сначала L^-1, потом S^-1.
    std::uint8_t linear[kBlockSize];
    LInverse(input, linear);
    SubstituteInverse(linear, output);
}

// C_index = L(Vec_128(index)): единица в последнем байте блока.
void RoundConstant(int index, std::uint8_t* output) {
    std::uint8_t vector[kBlockSize] = {0};
    vector[kBlockSize - 1] = static_cast<std::uint8_t>(index);
    L(vector, output);
}

// Развертывание ключа: K1 и K2 берутся из ключа, каждая следующая пара
// получается восемью применениями F[C_i](a1, a0) = (LSX[C_i](a1) xor a0, a1).
void ExpandKey(const std::uint8_t* key, std::uint8_t round_keys[kRoundCount][kBlockSize]) {
    std::uint8_t first[kBlockSize];
    std::uint8_t second[kBlockSize];
    std::memcpy(first, key, kBlockSize);
    std::memcpy(second, key + kBlockSize, kBlockSize);
    std::memcpy(round_keys[0], first, kBlockSize);
    std::memcpy(round_keys[1], second, kBlockSize);

    for (int group = 0; group < 4; ++group) {
        for (int step = 0; step < 8; ++step) {
            std::uint8_t constant[kBlockSize];
            RoundConstant(group * 8 + step + 1, constant);

            std::uint8_t transformed[kBlockSize];
            std::memcpy(transformed, first, kBlockSize);
            XorWithKey(transformed, constant);
            LS(transformed, transformed);
            XorWithKey(transformed, second);

            // Новое состояние: (LSX(a1) xor a0, a1).
            std::memcpy(second, first, kBlockSize);
            std::memcpy(first, transformed, kBlockSize);
        }
        std::memcpy(round_keys[2 * group + 2], first, kBlockSize);
        std::memcpy(round_keys[2 * group + 3], second, kBlockSize);
    }
}

BlockCipher::BlockCipher(const std::uint8_t* key) {
    ExpandKey(key, round_keys_);
}

// E = X[K_10] LS X[K_9] ... LS X[K_1].
void BlockCipher::EncryptBlock(const std::uint8_t* input, std::uint8_t* output) const {
    std::uint8_t state[kBlockSize];
    std::memcpy(state, input, kBlockSize);

    XorWithKey(state, round_keys_[0]);
    for (std::size_t round = 1; round + 1 < kRoundCount; ++round) {
        LS(state, state);
        XorWithKey(state, round_keys_[round]);
    }
    LS(state, state);
    XorWithKey(state, round_keys_[kRoundCount - 1]);

    std::memcpy(output, state, kBlockSize);
}

// D = X[K_1] L^-1 S^-1 X[K_2] ... L^-1 S^-1 X[K_10].
void BlockCipher::DecryptBlock(const std::uint8_t* input, std::uint8_t* output) const {
    std::uint8_t state[kBlockSize];
    std::memcpy(state, input, kBlockSize);

    XorWithKey(state, round_keys_[kRoundCount - 1]);
    for (std::size_t round = kRoundCount - 1; round > 0; --round) {
        LSInverse(state, state);
        XorWithKey(state, round_keys_[round - 1]);
    }

    std::memcpy(output, state, kBlockSize);
}

}  // namespace kuznyechik
