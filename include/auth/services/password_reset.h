#pragma once

#include <drogon/orm/DbClient.h>

#include <memory>
#include <optional>
#include <string>
#include <string_view>

#include "auth/password/i_password_hasher.h"
#include "auth/services/smtp_mailer.h"

namespace auth::services {

inline constexpr int kPasswordResetLifetimeSeconds = 300;
inline constexpr std::string_view kPasswordResetPendingCookie = "auth_reset_pending";
inline constexpr std::string_view kPasswordResetGrantCookie = "auth_reset_grant";

struct PasswordResetDelivery {
    std::string email;
    std::string code;
};

struct PasswordResetRequest {
    std::string pending_token;
    std::optional<PasswordResetDelivery> delivery;
};

class PasswordReset {
public:
    PasswordReset(drogon::orm::DbClientPtr database,
                  std::shared_ptr<password::IPasswordHasher> hasher,
                  SmtpMailer mailer);

    PasswordResetRequest request_code(std::string_view email, std::string_view previous_token) const;
    void send_code(const PasswordResetRequest& request) const;
    std::optional<std::string> confirm_code(std::string_view pending_token, std::string_view code) const;
    bool change_password(std::string_view grant_token, std::string_view password) const;

private:
    drogon::orm::DbClientPtr database_;
    std::shared_ptr<password::IPasswordHasher> hasher_;
    SmtpMailer mailer_;
};

}  // namespace auth::services
