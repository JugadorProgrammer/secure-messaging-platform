
#pragma once

#include "auth/password/i_password_hasher.h"

namespace auth::password {

class StubPasswordHasher final : public IPasswordHasher {
public:
    std::string hash_password(
        std::string_view password
    ) override;

    bool verify_password(
        std::string_view password,
        std::string_view stored_hash
    ) override;
};

}  // namespace auth::password
