#ifndef KUZ_CBC_MODE_H_
#define KUZ_CBC_MODE_H_

#include <cstdint>
#include <stdexcept>
#include <vector>

namespace kuznyechik {

// Дополнение неверно: испорчен шифртекст, чужой ключ или IV.
class PaddingError : public std::runtime_error {
public:
    explicit PaddingError(const char* message) : std::runtime_error(message) {}
};

// Дополнение по PKCS#7: блок добивается байтами, равными их количеству.
std::vector<std::uint8_t> AddPadding(const std::vector<std::uint8_t>& data);

// Снимает дополнение на месте. Бросает PaddingError, если оно неверно.
void RemovePadding(std::vector<std::uint8_t>& data);

// Шифрование и расшифровка в режиме CBC (ГОСТ Р 34.13-2015, п. 5.3).
// Ключ 32 байта, IV 16 байт; длину проверяет вызывающая сторона.
std::vector<std::uint8_t> EncryptCbc(const std::vector<std::uint8_t>& key,
                                     const std::vector<std::uint8_t>& iv,
                                     const std::vector<std::uint8_t>& data);

std::vector<std::uint8_t> DecryptCbc(const std::vector<std::uint8_t>& key,
                                     const std::vector<std::uint8_t>& iv,
                                     const std::vector<std::uint8_t>& ciphertext);

}  // namespace kuznyechik

#endif  // KUZ_CBC_MODE_H_
