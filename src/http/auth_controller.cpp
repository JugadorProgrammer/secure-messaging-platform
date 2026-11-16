#include "auth/http/auth_controller.h"

#include <drogon/drogon.h>
#include <trantor/utils/ConcurrentTaskQueue.h>

#include <atomic>
#include <functional>
#include <utility>

#include "auth/http/error_response.h"
#include "auth/services/first_factor.h"

namespace auth::http {

namespace {

bool valid_field(const std::string& value, std::size_t max_size) {
    return value.size() <= max_size && value.find('\0') == std::string::npos &&
        value.find_first_not_of(" \t\r\n") != std::string::npos;
}

drogon::HttpResponsePtr start_login(const services::FirstFactor& first_factor,
                                  const std::string& login, const std::string& password,
                                  bool secure_cookie) {
    try {
        const auto token = first_factor.start(login, password);
        if (!token) {
            return make_error_response(drogon::k401Unauthorized, "INVALID_CREDENTIALS",
                                       "Invalid login or password");
        }

        drogon::Cookie cookie(std::string(services::kPendingLoginCookie), *token);
        cookie.setHttpOnly(true);
        cookie.setSecure(secure_cookie);
        cookie.setSameSite(drogon::Cookie::SameSite::kLax);
        cookie.setPath("/");
        cookie.setMaxAge(services::kLoginChallengeLifetimeSeconds);

        auto response = drogon::HttpResponse::newHttpResponse();
        response->setStatusCode(drogon::k204NoContent);
        response->addHeader("Cache-Control", "no-store");
        response->addCookie(cookie);
        return response;
    } catch (const std::exception&) {
        LOG_ERROR << "First-factor processing failed";
        return make_error_response(drogon::k503ServiceUnavailable, "LOGIN_UNAVAILABLE",
                                   "Login temporarily unavailable");
    }
}

}  // namespace

void register_login_routes(std::shared_ptr<const services::FirstFactor> first_factor, bool secure_cookie) {
    constexpr std::size_t max_pending_requests = 16;
    // Блокирующие запросы к PostgreSQL и SMTP не занимают потоки HTTP.
    static trantor::ConcurrentTaskQueue workers(2, "auth-login");
    static std::atomic_size_t pending_requests{0};

    auto handler = [first_factor = std::move(first_factor), secure_cookie](
        const drogon::HttpRequestPtr& request,
        std::function<void(const drogon::HttpResponsePtr&)>&& callback
    ) {
        const auto body = request->getJsonObject();
        if (!body || !body->isObject() || !(*body)["login"].isString() ||
            !(*body)["password"].isString()) {
            callback(make_error_response(drogon::k400BadRequest, "INVALID_REQUEST",
                                         "Login and password are required"));
            return;
        }

        auto login = (*body)["login"].asString();
        auto password = (*body)["password"].asString();
        if (!valid_field(login, 256) || !valid_field(password, 4096)) {
            callback(make_error_response(drogon::k400BadRequest, "INVALID_REQUEST",
                                         "Invalid login or password input"));
            return;
        }

        if (pending_requests.fetch_add(1) >= max_pending_requests) {
            pending_requests.fetch_sub(1);
            callback(make_error_response(drogon::k503ServiceUnavailable, "LOGIN_UNAVAILABLE",
                                         "Login temporarily unavailable"));
            return;
        }

        workers.runTaskInQueue(
            [first_factor, secure_cookie, login = std::move(login), password = std::move(password),
             callback = std::move(callback)]() {
                auto response = start_login(*first_factor, login, password, secure_cookie);
                pending_requests.fetch_sub(1);
                callback(response);
            }
        );
    };
    // Каждый маршрут владеет своим экземпляром обработчика.
    drogon::app().registerHandler("/auth/login", decltype(handler){handler}, {drogon::Post});
    drogon::app().registerHandler("/login", std::move(handler), {drogon::Post});
}

}  // namespace auth::http
