#pragma once

#include <drogon/orm/DbClient.h>

#include <cstdint>
#include <optional>
#include <string>
#include <string_view>

namespace auth::repositories {

inline constexpr std::string_view kSessionCookie = "session";
inline constexpr int kSessionLifetimeSeconds = 8 * 60 * 60;

struct SessionPrincipal {
    std::int64_t user_id;
    std::string login;
    std::string role;
};

class SessionRepository {
public:
    explicit SessionRepository(drogon::orm::DbClientPtr client);

    // Вызывается после прохождения всех факторов. Принимает UUID из AUTH.
    // Для атомарной выдачи сессии передайте в репозиторий клиент транзакции.
    [[nodiscard]] std::string create_full_session(std::string_view user_id) const;

    [[nodiscard]] std::optional<SessionPrincipal> find_active_principal(std::string_view token) const;
    void revoke(std::string_view token) const;

private:
    drogon::orm::DbClientPtr client_;
};

}  // namespace auth::repositories
