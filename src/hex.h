#ifndef KUZ_HEX_H_
#define KUZ_HEX_H_

#include <cstdint>
#include <string>
#include <vector>

namespace kuznyechik {

// Переводит hex-строку в байты. Бросает std::invalid_argument на кривой ввод.
std::vector<std::uint8_t> BytesFromHex(const std::string& text);

// Переводит байты в hex-строку маленькими буквами.
std::string HexFromBytes(const std::vector<std::uint8_t>& bytes);

}  // namespace kuznyechik

#endif  // KUZ_HEX_H_
