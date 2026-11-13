
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

private:
    drogon::orm::DbClientPtr client_;
};

}  // namespace auth::repositories
