
#pragma once

#include <string>
#include <string_view>

namespace auth::password {

class IPasswordHasher {
public:
    virtual ~IPasswordHasher() = default;

    // Создать хэш пароля для хранения в БД.
    virtual std::string hash_password(
        std::string_view password
    ) = 0;

    // Проверить пароль относительно сохранённого хэша.
    virtual bool verify_password(
        std::string_view password,
        std::string_view stored_hash
    ) = 0;
};

}  // namespace auth::password
