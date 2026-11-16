
#pragma once

#include <drogon/orm/DbClient.h>

#include <optional>
#include <string>
#include <string_view>

#include "auth/models/user.h"

namespace auth::repositories {

class UserRepository {
public:
    explicit UserRepository(
        drogon::orm::DbClientPtr client
    );

    // Возвращает пользователя либо std::nullopt.
    // При ошибке БД выбрасывает исключение.
    [[nodiscard]] std::optional<models::User> find_by_login(
        std::string_view login
    ) const;

    // Поиск пользователя по email без учёта регистра.
    // Возвращает std::nullopt, если пользователь не найден.
    [[nodiscard]] std::optional<models::User> find_by_email(
        std::string_view email
    ) const;

    // Создаёт обычного пользователя и возвращает его UUID. Конфликт логина
    // или email возвращается как std::nullopt.
    [[nodiscard]] std::optional<std::string> create_user(
        std::string_view login,
        std::string_view email,
        std::string_view password_hash
    ) const;

private:
    drogon::orm::DbClientPtr client_;
};

}  // namespace auth::repositories
