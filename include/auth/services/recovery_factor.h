#pragma once

#include <drogon/orm/DbClient.h>

#include <memory>
#include <optional>
#include <string>
#include <string_view>

namespace auth::services {

inline constexpr std::string_view kPendingRecoveryCookie = "auth_recovery_pending";
inline constexpr int kRecoveryChallengeLifetimeSeconds = 300;

// AUTH-20 вызывает после проверки email-кода с тем же клиентом транзакции.
[[nodiscard]] std::string create_recovery_challenge(
    const std::shared_ptr<drogon::orm::Transaction>& transaction,
    std::string_view user_id
);

[[nodiscard]] std::optional<std::string> complete_recovery_factor(
    const drogon::orm::DbClientPtr& database,
    std::string_view pending_token,
    std::string_view code
);

}  // namespace auth::services
