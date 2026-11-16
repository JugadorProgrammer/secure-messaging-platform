#pragma once

#include <drogon/orm/DbClient.h>

#include <memory>

#include "auth/password/i_password_hasher.h"

namespace auth::http {

void register_signup_routes(
    drogon::orm::DbClientPtr database,
    std::shared_ptr<password::IPasswordHasher> hasher
);

}  // namespace auth::http
