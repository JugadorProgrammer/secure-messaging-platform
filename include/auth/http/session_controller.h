#pragma once

#include <drogon/orm/DbClient.h>

namespace auth::http {

void register_session_routes(drogon::orm::DbClientPtr database, bool secure_cookie);

}  // namespace auth::http
