#include "auth/password/pbkdf2_password_hasher.h"

#include <iostream>

int main() {
    auth::password::Pbkdf2PasswordHasher hasher;
    const auto first_hash = hasher.hash_password("test password");
    const auto second_hash = hasher.hash_password("test password");

    if (first_hash == second_hash || !hasher.verify_password("test password", first_hash) ||
        !hasher.verify_password("test password", second_hash) ||
        hasher.verify_password("wrong password", first_hash) ||
        hasher.verify_password("test password", first_hash + "$") ||
        hasher.verify_password("plaintext", "plaintext")) {
        std::cerr << "Unexpected result verifying password hashes\n";
        return 1;
    }
}
