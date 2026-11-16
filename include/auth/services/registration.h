#pragma once

#include <drogon/orm/DbClient.h>

#include <optional>
#include <string>
#include <string_view>
#include <vector>

#include "auth/password/i_password_hasher.h"
#include "auth/validation/registration_validator.h"

namespace auth::services {

// Принимает проверенные данные. Пустой результат означает конфликт логина или email.
[[nodiscard]] std::optional<std::vector<std::string>> register_user(
    const drogon::orm::DbClientPtr& database,
    password::IPasswordHasher& hasher,
    const validation::RegistrationData& data,
    std::string_view role = "user"
);

}  // namespace auth::services
