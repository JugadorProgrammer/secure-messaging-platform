#pragma once

#include <drogon/orm/DbClient.h>

#include <memory>
#include <optional>
#include <string>
#include <string_view>

#include "auth/password/i_password_hasher.h"
#include "auth/services/login_token.h"
#include "auth/services/smtp_mailer.h"

namespace auth::services {

inline constexpr std::string_view kPendingLoginCookie = "auth_pending";
inline constexpr int kLoginChallengeLifetimeSeconds = 300;

class FirstFactor {
public:
    FirstFactor(drogon::orm::DbClientPtr database, std::shared_ptr<password::IPasswordHasher> hasher,
                SmtpMailer mailer);

    // Возвращает токен незавершённого входа после отправки письма.
    std::optional<std::string> start(std::string_view login, std::string_view password) const;

private:
    drogon::orm::DbClientPtr database_;
    std::shared_ptr<password::IPasswordHasher> hasher_;
    SmtpMailer mailer_;
    std::string dummy_password_hash_;
};

}  // namespace auth::services
