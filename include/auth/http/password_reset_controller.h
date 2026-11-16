#pragma once

#include <memory>

#include "auth/services/password_reset.h"

namespace auth::http {

void register_password_reset_routes(std::shared_ptr<const services::PasswordReset> reset, bool secure_cookie);

}  // namespace auth::http
