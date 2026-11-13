
#pragma once

#include <drogon/orm/DbClient.h>

#include <optional>
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

private:
    drogon::orm::DbClientPtr client_;
};

}  // namespace auth::repositories
