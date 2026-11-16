
#pragma once

#include <drogon/orm/DbClient.h>

namespace auth::http {

void register_email_factor_routes(
    drogon::orm::DbClientPtr database,
    bool secure_cookie
);

}  // namespace auth::http
