
#include "auth/password/stub_password_hasher.h"

#include <iostream>
#include <stdexcept>

int main() {
    auth::password::StubPasswordHasher hasher;

    bool hash_failed = false;
    bool verify_failed = false;

    // Проверяем, что заглушка не создаёт фиктивный хэш.
    try {
        hasher.hash_password("test_password");
    } catch (const std::runtime_error&) {
        hash_failed = true;
    }

    // Проверяем, что заглушка не подтверждает пароль.
    try {
        hasher.verify_password(
            "test_password",
            "stored_hash"
        );
    } catch (const std::runtime_error&) {
        verify_failed = true;
    }

    if (!hash_failed || !verify_failed) {
        std::cerr
            << "Password provider stub test: FAILED\n";
        return 1;
    }

    std::cout
        << "Password provider stub test: PASSED\n";

    return 0;
}
