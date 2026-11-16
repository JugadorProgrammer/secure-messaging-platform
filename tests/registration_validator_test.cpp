
#include "auth/validation/registration_validator.h"

#include <iostream>

using auth::validation::RegistrationData;
using auth::validation::ValidationError;
using auth::validation::validate_registration;

int main() {
    int failures = 0;

    auto check = [&](bool condition, const char* name) {
        if (!condition) {
            std::cerr << "FAILED: " << name << '\n';
            ++failures;
        }
    };

    // Корректные данные
    check(
        validate_registration({
            "brilds_123",
            "user@example.com",
            "SecurePass123!"
        }).is_valid(),
        "Valid registration"
    );

    // Логин
    check(
        validate_registration({
            "",
            "user@example.com",
            "SecurePass123!"
        }).error == ValidationError::InvalidLogin,
        "Empty login"
    );

    check(
        validate_registration({
            "ab",
            "user@example.com",
            "SecurePass123!"
        }).error == ValidationError::InvalidLogin,
        "Short login"
    );

    check(
        validate_registration({
            "bad login",
            "user@example.com",
            "SecurePass123!"
        }).error == ValidationError::InvalidLogin,
        "Login with space"
    );

    // Email
    check(
        validate_registration({
            "brilds",
            "incorrect-email",
            "SecurePass123!"
        }).error == ValidationError::InvalidEmail,
        "Invalid email"
    );

    check(
        validate_registration({
            "brilds",
            "user@",
            "SecurePass123!"
        }).error == ValidationError::InvalidEmail,
        "Incomplete email"
    );

    // Пароль
    check(
        validate_registration({
            "brilds",
            "user@example.com",
            "12345"
        }).error == ValidationError::InvalidPassword,
        "Short password"
    );

    check(
        validate_registration({
            "brilds",
            "user@example.com",
            "onlyletters"
        }).error == ValidationError::InvalidPassword,
        "Password without digits"
    );

    check(
        validate_registration({
            "brilds",
            "user@example.com",
            "123456789"
        }).error == ValidationError::InvalidPassword,
        "Password without letters"
    );

    if (failures != 0) {
        std::cerr << failures << " test(s) failed\n";
        return 1;
    }

    std::cout << "Registration validation tests: PASSED\n";
    return 0;
}
