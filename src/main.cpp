
#include <drogon/drogon.h>

#include <cstdlib>
#include <string>

#include "auth/http/health_controller.h"
#include "auth/repositories/database.h"

int main() {
    const char* config_env = std::getenv("AUTH_CONFIG_PATH");

    const std::string config_path =
        config_env != nullptr && config_env[0] != '\0'
            ? config_env
            : "config/config.example.json";

    drogon::app().loadConfigFile(config_path);

    if (!auth::repositories::check_database_connection()) {
        return 1;
    }

    auth::http::register_health_routes();

    drogon::app().run();

    return 0;
}
