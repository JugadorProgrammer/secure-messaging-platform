#include "auth/http/signup_controller.h"

#include <drogon/drogon.h>
#include <trantor/utils/ConcurrentTaskQueue.h>

#include <exception>
#include <functional>
#include <utility>

#include "auth/http/error_response.h"
#include "auth/services/registration.h"

namespace auth::http {

namespace {

drogon::HttpResponsePtr signup_unavailable() {
    return make_error_response(drogon::k503ServiceUnavailable, "SIGNUP_UNAVAILABLE",
                               "Registration temporarily unavailable");
}

drogon::HttpResponsePtr sign_up(
    const drogon::orm::DbClientPtr& database,
    password::IPasswordHasher& hasher,
    const validation::RegistrationData& data
) {
    try {
        const auto codes = services::register_user(database, hasher, data);
        if (!codes) {
            return make_error_response(drogon::k409Conflict, "ACCOUNT_ALREADY_EXISTS",
                                       "Login or email is already in use");
        }

        Json::Value body(Json::arrayValue);
        for (const auto& code : *codes) {
            body.append(code);
        }

        auto response = drogon::HttpResponse::newHttpJsonResponse(body);
        response->setStatusCode(drogon::k201Created);
        response->addHeader("Cache-Control", "no-store");
        return response;
    } catch (const std::exception&) {
        LOG_ERROR << "Registration failed";
        return signup_unavailable();
    }
}

}  // namespace

void register_signup_routes(
    drogon::orm::DbClientPtr database,
    std::shared_ptr<password::IPasswordHasher> hasher
) {
    constexpr std::size_t kMaxQueuedSignups = 16;
    // Хэширование и синхронные запросы к БД выполняются вне потоков HTTP.
    static trantor::ConcurrentTaskQueue workers(2, "auth-signup");

    auto handler = [database = std::move(database), hasher = std::move(hasher)](
        const drogon::HttpRequestPtr& request,
        std::function<void(const drogon::HttpResponsePtr&)>&& callback
    ) {
        const auto body = request->getJsonObject();
        if (!body || !body->isObject() || !(*body)["login"].isString() ||
            !(*body)["email"].isString() || !(*body)["password"].isString()) {
            callback(make_error_response(drogon::k400BadRequest, "INVALID_REQUEST",
                                         "Login, email and password are required"));
            return;
        }

        validation::RegistrationData data{
            (*body)["login"].asString(),
            (*body)["email"].asString(),
            (*body)["password"].asString()
        };
        if (!validation::validate_registration(data).is_valid()) {
            callback(make_error_response(drogon::k400BadRequest, "INVALID_REQUEST",
                                         "Invalid login, email or password"));
            return;
        }

        if (workers.getTaskCount() >= kMaxQueuedSignups) {
            callback(signup_unavailable());
            return;
        }

        workers.runTaskInQueue([database, hasher, data = std::move(data),
                               callback = std::move(callback)] {
            callback(sign_up(database, *hasher, data));
        });
    };

    // Каждый маршрут владеет своим экземпляром обработчика.
    drogon::app().registerHandler("/auth/signup", decltype(handler){handler}, {drogon::Post});
    drogon::app().registerHandler("/signup", std::move(handler), {drogon::Post});
}

}  // namespace auth::http
