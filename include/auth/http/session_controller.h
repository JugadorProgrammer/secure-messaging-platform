#pragma once

#include <drogon/orm/DbClient.h>

#include <string_view>

namespace auth::http {

[[nodiscard]] std::string_view bearer_token(std::string_view header);

void register_session_routes(drogon::orm::DbClientPtr database, bool secure_cookie);

}  // namespace auth::http
