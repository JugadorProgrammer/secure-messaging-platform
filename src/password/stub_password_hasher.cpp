
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

}  // namespace auth::password
