
#include "auth/http/health_controller.h"

#include <drogon/drogon.h>

#include <functional>

namespace auth::http {

void register_health_routes() {
    drogon::app().registerHandler(
        "/health",

        [](const drogon::HttpRequestPtr&,
           std::function<void(const drogon::HttpResponsePtr&)>&& callback) {

            Json::Value body;
            body["status"] = "ok";
            body["service"] = "auth";

            auto response =
                drogon::HttpResponse::newHttpJsonResponse(body);

            callback(response);
        },

        {drogon::Get}
    );
}

}  // namespace auth::http
