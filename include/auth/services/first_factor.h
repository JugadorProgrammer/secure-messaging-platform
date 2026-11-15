#pragma once

#include <drogon/orm/DbClient.h>

#include <memory>
#include <optional>
#include <string>
#include <string_view>

#include "auth/password/i_password_hasher.h"
#include "auth/repositories/user_repository.h"
#include "auth/services/smtp_mailer.h"

namespace auth::services {

inline constexpr std::string_view kPendingLoginCookie = "auth_pending";
inline constexpr int kLoginChallengeLifetimeSeconds = 300;

// Эти два преобразования также используются при подтверждении email-кода.
std::string login_token_hash(std::string_view token);
std::string login_email_code_hash(std::string_view token, std::string_view code);

class FirstFactor {
public:
    FirstFactor(drogon::orm::DbClientPtr database, std::shared_ptr<password::IPasswordHasher> hasher,
                SmtpMailer mailer);

    // Возвращает токен незавершённого входа после отправки письма.
    std::optional<std::string> start(std::string_view login, std::string_view password) const;

private:
    drogon::orm::DbClientPtr database_;
    repositories::UserRepository users_;
    std::shared_ptr<password::IPasswordHasher> hasher_;
    SmtpMailer mailer_;
    std::string dummy_password_hash_;
};

}  // namespace auth::services
