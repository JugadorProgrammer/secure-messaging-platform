#include "auth/http/recovery_controller.h"

#include <drogon/drogon.h>
#include <trantor/utils/ConcurrentTaskQueue.h>

#include <exception>
#include <functional>
#include <string>
#include <string_view>
#include <utility>

#include "auth/http/error_response.h"
#include "auth/repositories/session_repository.h"
#include "auth/services/login_token.h"
#include "auth/services/recovery_factor.h"

namespace auth::http {

namespace {

drogon::HttpResponsePtr recovery_unavailable() {
    return make_error_response(drogon::k503ServiceUnavailable, "RECOVERY_UNAVAILABLE",
                               "Recovery code verification temporarily unavailable");
}

drogon::HttpResponsePtr recovery_rejected() {
    return make_error_response(drogon::k401Unauthorized, "INVALID_RECOVERY_CODE",
                               "Invalid recovery code or login attempt");
}

drogon::Cookie auth_cookie(std::string_view name, std::string value, int max_age, bool secure) {
    drogon::Cookie cookie(std::string(name), std::move(value));
    cookie.setHttpOnly(true);
    cookie.setSecure(secure);
    cookie.setSameSite(drogon::Cookie::SameSite::kLax);
    cookie.setPath("/");
    cookie.setMaxAge(max_age);
    return cookie;
}

drogon::HttpResponsePtr confirm_recovery_code(
    const drogon::orm::DbClientPtr& database,
    std::string_view pending_token,
    std::string_view code,
    bool secure_cookie
) {
    try {
        auto session = services::complete_recovery_factor(database, pending_token, code);
        if (!session) {
            return recovery_rejected();
        }

        auto response = drogon::HttpResponse::newHttpResponse();
        response->setStatusCode(drogon::k204NoContent);
        response->addHeader("Cache-Control", "no-store");
        response->addCookie(auth_cookie(repositories::kSessionCookie, std::move(*session),
                                       repositories::kSessionLifetimeSeconds, secure_cookie));
        response->addCookie(auth_cookie(services::kPendingRecoveryCookie, "", 0, secure_cookie));
        return response;
    } catch (const std::exception&) {
        LOG_ERROR << "Recovery factor verification failed";
        return recovery_unavailable();
    }
}

}  // namespace

void register_recovery_routes(drogon::orm::DbClientPtr database, bool secure_cookie) {
    constexpr std::size_t kMaxQueuedRequests = 16;
    static trantor::ConcurrentTaskQueue workers(2, "auth-recovery");

    auto handler = [database = std::move(database), secure_cookie](
        const drogon::HttpRequestPtr& request,
        std::function<void(const drogon::HttpResponsePtr&)>&& callback
    ) {
        const auto body = request->getJsonObject();
        if (!body || !body->isObject() || !(*body)["code"].isString()) {
            callback(make_error_response(drogon::k400BadRequest, "INVALID_REQUEST",
                                         "Recovery code is required"));
            return;
        }

        auto pending_token = request->getCookie(std::string(services::kPendingRecoveryCookie));
        if (!services::is_login_token(pending_token)) {
            callback(recovery_rejected());
            return;
        }

        if (workers.getTaskCount() >= kMaxQueuedRequests) {
            callback(recovery_unavailable());
            return;
        }

        workers.runTaskInQueue([database, secure_cookie, pending_token = std::move(pending_token),
                               code = (*body)["code"].asString(), callback = std::move(callback)] {
            callback(confirm_recovery_code(database, pending_token, code, secure_cookie));
        });
    };

    drogon::app().registerHandler("/auth/login/recovery-code", decltype(handler){handler}, {drogon::Post});
    drogon::app().registerHandler("/confirmDisposableCode", std::move(handler), {drogon::Post});
}

}  // namespace auth::http
