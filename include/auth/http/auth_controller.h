#pragma once

#include <memory>

namespace auth::services {
class FirstFactor;
}

namespace auth::http {

void register_login_routes(std::shared_ptr<const services::FirstFactor> first_factor, bool secure_cookie);

}  // namespace auth::http
