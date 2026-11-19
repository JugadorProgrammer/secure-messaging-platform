#ifndef KUZ_TABLES_H_
#define KUZ_TABLES_H_

#include <cstdint>

namespace kuznyechik {

// Подстановка Pi из ГОСТ Р 34.12-2015, п. 4.1.
extern const std::uint8_t kSbox[256];
extern const std::uint8_t kSboxInverse[256];

// Коэффициенты линейной функции l: от a_15 к a_0.
extern const std::uint8_t kLinearCoefficients[16];

}  // namespace kuznyechik

#endif  // KUZ_TABLES_H_
