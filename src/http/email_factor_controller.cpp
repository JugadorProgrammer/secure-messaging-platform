
#include "auth/http/email_factor_controller.h"

#include <drogon/drogon.h>
#include <trantor/utils/ConcurrentTaskQueue.h>

#include <exception>
#include <functional>
#include <string>
#include <utility>

#include "auth/http/error_response.h"
#include "auth/services/email_factor.h"
#include "auth/services/first_factor.h"
#include "auth/services/login_token.h"
#include "auth/services/recovery_factor.h"

namespace auth::http {

namespace {

using Callback =
    std::function<void(const drogon::HttpResponsePtr&)>;

drogon::HttpResponsePtr invalid_code() {
    return make_error_response(
        drogon::k401Unauthorized,
        "INVALID_EMAIL_CODE",
        "Invalid email code or login attempt"
    );
}

drogon::HttpResponsePtr unavailable() {
    return make_error_response(
        drogon::k503ServiceUnavailable,
        "EMAIL_FACTOR_UNAVAILABLE",
        "Email verification temporarily unavailable"
    );
}

drogon::Cookie make_cookie(
    std::string name,
    std::string value,
    int max_age,
    bool secure
) {
    drogon::Cookie cookie(
        std::move(name),
        std::move(value)
    );

    cookie.setPath("/");
    cookie.setHttpOnly(true);
    cookie.setSecure(secure);
    cookie.setSameSite(
        drogon::Cookie::SameSite::kLax
    );
    cookie.setMaxAge(max_age);

    return cookie;
}

drogon::HttpResponsePtr verify_code(
    const drogon::orm::DbClientPtr& database,
    const std::string& pending_token,
    const std::string& code,
    bool secure_cookie
) {
    try {
        const auto recovery_token =
            services::complete_email_factor(
                database,
                pending_token,
                code
            );

        if (!recovery_token) {
            return invalid_code();
        }

        auto response =
            drogon::HttpResponse::newHttpResponse();

        response->setStatusCode(
            drogon::k204NoContent
        );

        response->addHeader(
            "Cache-Control", "no-store"
        );

        // Выдаём cookie для третьего фактора.
        response->addCookie(make_cookie(
            std::string(services::kPendingRecoveryCookie),
            *recovery_token,
            services::kRecoveryChallengeLifetimeSeconds,
            secure_cookie
        ));

        // Удаляем cookie предыдущего этапа.
        response->addCookie(make_cookie(
            std::string(services::kPendingLoginCookie),
            "",
            0,
            secure_cookie
        ));

        return response;

    } catch (const std::exception&) {
        LOG_ERROR << "Email factor verification failed";
        return unavailable();
    }
}

}  // namespace

void register_email_factor_routes(
    drogon::orm::DbClientPtr database,
    bool secure_cookie
) {
    constexpr std::size_t kMaxQueuedRequests = 16;

    // Отдельные потоки для синхронных запросов к БД.
    static trantor::ConcurrentTaskQueue workers(
        2, "auth-email-factor"
    );

    auto handler = [database = std::move(database),
                    secure_cookie](
        const drogon::HttpRequestPtr& request,
        Callback&& callback
    ) {
        const auto body = request->getJsonObject();

        if (!body || !body->isObject() ||
            !(*body)["code"].isString()) {
            callback(make_error_response(
                drogon::k400BadRequest,
                "INVALID_REQUEST",
                "Email code is required"
            ));
            return;
        }

        const std::string code =
            (*body)["code"].asString();

        if (code.size() != 6 ||
            code.find_first_not_of("0123456789")
                != std::string::npos) {
            callback(make_error_response(
                drogon::k400BadRequest,
                "INVALID_REQUEST",
                "A six-digit email code is required"
            ));
            return;
        }

        auto pending_token = request->getCookie(
            std::string(services::kPendingLoginCookie)
        );

        if (!services::is_login_token(pending_token)) {
            callback(invalid_code());
            return;
        }

        if (workers.getTaskCount() >= kMaxQueuedRequests) {
            callback(unavailable());
            return;
        }

        workers.runTaskInQueue(
            [database, secure_cookie,
             pending_token = std::move(pending_token),
             code,
             callback = std::move(callback)] {
                callback(verify_code(
                    database,
                    pending_token,
                    code,
                    secure_cookie
                ));
            }
        );
    };

    drogon::app().registerHandler(
        "/auth/login/email-code",
        decltype(handler){handler},
        {drogon::Post}
    );

    // Совместимый маршрут для FRONT.
    drogon::app().registerHandler(
        "/confirmEmailCode",
        std::move(handler),
        {drogon::Post}
    );
}

}  // namespace auth::http
