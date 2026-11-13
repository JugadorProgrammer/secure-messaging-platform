
#include "auth/validation/registration_validator.h"

#include <regex>

namespace auth::validation {

namespace {

// Предварительная политика
constexpr std::size_t kMinLoginLength = 3;
constexpr std::size_t kMaxLoginLength = 32;
constexpr std::size_t kMaxEmailLength = 254;
constexpr std::size_t kMinPasswordLength = 8;
constexpr std::size_t kMaxPasswordLength = 128;

}  // namespace

bool is_valid_login(std::string_view login) {
    if (login.size() < kMinLoginLength ||
        login.size() > kMaxLoginLength) {
        return false;
    }

    static const std::regex pattern(
        R"(^[A-Za-z0-9_]+$)"
    );

    return std::regex_match(
        login.begin(), login.end(), pattern
    );
}

bool is_valid_email(std::string_view email) {
    if (email.empty() || email.size() > kMaxEmailLength) {
        return false;
    }

    // Упрощённая проверка распространённых email.
    static const std::regex pattern(
        R"(^[A-Za-z0-9._%+-]+@(?:[A-Za-z0-9-]+\.)+[A-Za-z]{2,}$)"
    );

    return std::regex_match(
        email.begin(), email.end(), pattern
    );
}

bool is_valid_password(std::string_view password) {
    if (password.size() < kMinPasswordLength ||
        password.size() > kMaxPasswordLength) {
        return false;
    }

    static const std::regex letter_pattern(R"([A-Za-z])");
    static const std::regex digit_pattern(R"([0-9])");

    return std::regex_search(
               password.begin(),
               password.end(),
               letter_pattern
           ) &&
           std::regex_search(
               password.begin(),
               password.end(),
               digit_pattern
           );
}

ValidationResult validate_registration(
    const RegistrationData& data
) {
    if (!is_valid_login(data.login)) {
        return {ValidationError::InvalidLogin};
    }

    if (!is_valid_email(data.email)) {
        return {ValidationError::InvalidEmail};
    }

    if (!is_valid_password(data.password)) {
        return {ValidationError::InvalidPassword};
    }

    return {};
}

}  // namespace auth::validation
