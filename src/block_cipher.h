#ifndef KUZ_BLOCK_CIPHER_H_
#define KUZ_BLOCK_CIPHER_H_

#include <cstddef>
#include <cstdint>

namespace kuznyechik {

// Размеры ГОСТ Р 34.12-2015: блок 128 бит, ключ 256 бит, 10 раундов.
constexpr std::size_t kBlockSize = 16;
constexpr std::size_t kKeySize = 32;
constexpr std::size_t kRoundCount = 10;

// Преобразования из п. 4.3. Все работают с блоком ровно 16 байт.
// Вход и выход могут указывать на один и тот же буфер.
void Substitute(const std::uint8_t* input, std::uint8_t* output);
void SubstituteInverse(const std::uint8_t* input, std::uint8_t* output);

void R(const std::uint8_t* input, std::uint8_t* output);
void RInverse(const std::uint8_t* input, std::uint8_t* output);

void L(const std::uint8_t* input, std::uint8_t* output);
void LInverse(const std::uint8_t* input, std::uint8_t* output);

// Композиции из основного алгоритма: LS = L после S,
// обратная — S^-1 после L^-1 (порядок важен).
void LS(const std::uint8_t* input, std::uint8_t* output);
void LSInverse(const std::uint8_t* input, std::uint8_t* output);

// Сложение состояния с раундовым ключом.
void XorWithKey(std::uint8_t* state, const std::uint8_t* key);

// Раундовая константа C_index для index = 1..32 (п. 4.4).
void RoundConstant(int index, std::uint8_t* output);

// Разворачивает ключ в 10 раундовых ключей по 16 байт.
void ExpandKey(const std::uint8_t* key, std::uint8_t round_keys[kRoundCount][kBlockSize]);

class BlockCipher {
public:
    // Ключ ровно 32 байта; длину проверяет вызывающая сторона.
    explicit BlockCipher(const std::uint8_t* key);

    void EncryptBlock(const std::uint8_t* input, std::uint8_t* output) const;
    void DecryptBlock(const std::uint8_t* input, std::uint8_t* output) const;

private:
    std::uint8_t round_keys_[kRoundCount][kBlockSize];
};

}  // namespace kuznyechik

#endif  // KUZ_BLOCK_CIPHER_H_
