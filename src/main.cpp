#include <drogon/drogon.h>

#include <cstdlib>
#include <exception>
#include <iostream>
#include <memory>
#include <string>
#include <string_view>

#include "auth/http/auth_controller.h"
#include "auth/http/health_controller.h"
#include "auth/http/recovery_controller.h"
#include "auth/http/session_controller.h"
#include "auth/http/signup_controller.h"
#include "auth/password/pbkdf2_password_hasher.h"
#include "auth/repositories/database.h"
#include "auth/services/first_factor.h"
#include "auth/services/smtp_mailer.h"

int main(int argc, char* argv[]) {
    auto hasher = std::make_shared<auth::password::Pbkdf2PasswordHasher>();

    try {
        if (argc == 2 && std::string_view(argv[1]) == "--hash-password") {
            std::string password;
            if (!std::getline(std::cin, password) || password.empty()) {
                std::cerr << "Read a non-empty password from stdin\n";
                return 1;
            }
            std::cout << hasher->hash_password(password) << '\n';
            return 0;
        }

        const char* config_env = std::getenv("AUTH_CONFIG_PATH");
        const std::string config_path =
            config_env != nullptr && config_env[0] != '\0'
                ? config_env
                : "config/config.example.json";

        drogon::app().loadConfigFile(config_path);

        if (!auth::repositories::check_database_connection()) {
            return 1;
        }

        auto database = auth::repositories::create_database_client();
        auto first_factor = std::make_shared<auth::services::FirstFactor>(
            database, hasher,
            auth::services::SmtpMailer(auth::services::smtp_settings_from_environment())
        );
        const char* cookie_config = std::getenv("AUTH_COOKIE_SECURE");
        const bool secure_cookie = cookie_config == nullptr ||
            std::string_view(cookie_config) != "false";
        auth::http::register_health_routes();
        auth::http::register_signup_routes(database, hasher);
        auth::http::register_login_routes(first_factor, secure_cookie);
        auth::http::register_recovery_routes(database, secure_cookie);
        auth::http::register_session_routes(database, secure_cookie);
        drogon::app().run();
    } catch (const std::exception&) {
        std::cerr << "AUTH initialization failed\n";
        return 1;
    }

    return 0;
}
