#pragma once

#include <string>
#include <string_view>

namespace auth::services {

struct SmtpSettings {
    std::string url;
    std::string from;
    std::string username;
    std::string password;
    bool require_tls = true;
};

SmtpSettings smtp_settings_from_environment();

class SmtpMailer {
public:
    explicit SmtpMailer(SmtpSettings settings);

    // Отправляет шестизначный код. При ошибке выбрасывает исключение.
    void send_verification_code(std::string_view recipient, std::string_view code) const;
    void send_password_reset_code(std::string_view recipient, std::string_view code) const;

private:
    void send_code(std::string_view recipient, std::string_view code, std::string_view purpose) const;
    SmtpSettings settings_;
};

}  // namespace auth::services
