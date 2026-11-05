
#pragma once

#include <string>
#include <string_view>

namespace auth::password {

class IPasswordHasher {
public:
    virtual ~IPasswordHasher() = default;

    // Возвращает значение, пригодное для хранения в БД.
    // Конкретный формат определяет реализация CRYPTO.
    virtual std::string hash_password(
        std::string_view password
    ) = 0;
};

}  // namespace auth::password
