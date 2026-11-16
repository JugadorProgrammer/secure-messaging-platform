
#pragma once

#include <drogon/orm/DbClient.h>

#include <optional>
#include <string>
#include <string_view>

namespace auth::services {

// Проверяет второй фактор.
// Возвращает токен третьего фактора при успехе.
// nullopt означает неверный код, истёкшую или
// отсутствующую попытку входа.
//
// Ошибки PostgreSQL передаются через исключения.
[[nodiscard]] std::optional<std::string>
complete_email_factor(
    const drogon::orm::DbClientPtr& database,
    std::string_view pending_token,
    std::string_view email_code
);

}  // namespace auth::services
