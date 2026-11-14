#pragma once

#include <drogon/orm/DbClient.h>

#include <string>
#include <string_view>
#include <vector>

namespace auth::repositories {

void save_recovery_codes(const drogon::orm::DbClientPtr& db, const std::string& user_id, const std::vector<std::string>& codes);
bool consume_recovery_code(const drogon::orm::DbClientPtr& db, const std::string& user_id, std::string_view code);

}
