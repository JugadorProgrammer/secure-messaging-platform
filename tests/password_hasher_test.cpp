
#include "auth/password/stub_password_hasher.h"

#include <iostream>
#include <stdexcept>

int main() {
    auth::password::StubPasswordHasher hasher;

    try {
        hasher.hash_password("test_password");
    } catch (const std::runtime_error&) {
        std::cout << "Password hasher stub: PASSED\n";
        return 0;
    }

    std::cerr << "Stub unexpectedly returned a hash\n";
    return 1;
}
