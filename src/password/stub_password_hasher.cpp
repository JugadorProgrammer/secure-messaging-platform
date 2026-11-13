
#include "auth/password/stub_password_hasher.h"

#include <stdexcept>

namespace auth::password {

std::string StubPasswordHasher::hash_password(
    std::string_view password
) {
    (void)password;

    throw std::runtime_error(
        "Password hashing provider is not configured"
    );
}

bool StubPasswordHasher::verify_password(
    std::string_view password,
    std::string_view stored_hash
) {
    (void)password;
    (void)stored_hash;

    throw std::runtime_error(
        "Password verification provider is not configured"
    );
}

}  // namespace auth::password
