#pragma once

#include <string>
#include <string_view>

namespace auth::services {

std::string generate_login_token();
std::string generate_email_code();
bool is_login_token(std::string_view token);
std::string login_token_hash(std::string_view token);
std::string login_email_code_hash(std::string_view token, std::string_view code);

}  // namespace auth::services
