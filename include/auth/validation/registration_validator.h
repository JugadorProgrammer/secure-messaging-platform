
#pragma once

#include <string>
#include <string_view>

namespace auth::validation {

struct RegistrationData {
    std::string login;
    std::string email;
    std::string password;
};

enum class ValidationError {
    None,
    InvalidLogin,
    InvalidEmail,
    InvalidPassword
};

struct ValidationResult {
    ValidationError error = ValidationError::None;

    [[nodiscard]] bool is_valid() const {
        return error == ValidationError::None;
    }
};

// Проверка отдельных полей регистрации.
[[nodiscard]] bool is_valid_login(std::string_view login);
[[nodiscard]] bool is_valid_email(std::string_view email);
[[nodiscard]] bool is_valid_password(std::string_view password);

// Комплексная проверка данных регистрации.
[[nodiscard]] ValidationResult validate_registration(
    const RegistrationData& data
);

}  // namespace auth::validation
