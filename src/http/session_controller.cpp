#include "auth/http/session_controller.h"

#include <drogon/drogon.h>
#include <trantor/utils/ConcurrentTaskQueue.h>

#include <algorithm>
#include <cctype>
#include <exception>
#include <functional>
#include <optional>
#include <string>
#include <string_view>
#include <utility>

#include "auth/http/error_response.h"
#include "auth/repositories/session_repository.h"
#include "auth/services/login_token.h"

namespace auth::http {

namespace {

constexpr std::size_t kMaxQueuedSessionRequests = 64;
using Callback = std::function<void(const drogon::HttpResponsePtr&)>;

trantor::ConcurrentTaskQueue& session_workers() {
    static trantor::ConcurrentTaskQueue workers(2, "auth-session");
    return workers;
}

drogon::HttpResponsePtr sessions_unavailable() {
    return make_error_response(drogon::k503ServiceUnavailable, "SESSIONS_UNAVAILABLE",
                               "Session service temporarily unavailable");
}

std::string_view bearer_token(std::string_view header) {
    const auto split = header.find(' ');
    if (split == std::string_view::npos) {
        return {};
    }

    constexpr std::string_view scheme = "bearer";
    const auto name = header.substr(0, split);
    if (!std::equal(name.begin(), name.end(), scheme.begin(), scheme.end(),
                    [](unsigned char actual, char expected) {
                        return std::tolower(actual) == expected;
                    })) {
        return {};
    }

    const auto first = header.find_first_not_of(" \t", split);
    if (first == std::string_view::npos) {
        return {};
    }
    const auto last = header.find_last_not_of(" \t");
    return header.substr(first, last - first + 1);
}

drogon::HttpResponsePtr introspection_response(
    const std::optional<repositories::SessionPrincipal>& principal
) {
    Json::Value body(Json::objectValue);
    body["active"] = principal.has_value();
    if (principal) {
        body["stage"] = "full";
        body["user_id"] = Json::Int64(principal->user_id);
        body["login"] = principal->login;
        body["role"] = principal->role;
    }

    auto response = drogon::HttpResponse::newHttpJsonResponse(body);
    response->addHeader("Cache-Control", "no-store");
    return response;
}

drogon::HttpResponsePtr logout_response(bool secure_cookie) {
    auto response = drogon::HttpResponse::newHttpResponse();
    response->setStatusCode(drogon::k204NoContent);
    response->addHeader("Cache-Control", "no-store");

    drogon::Cookie cookie(std::string(repositories::kSessionCookie), "");
    cookie.setPath("/");
    cookie.setHttpOnly(true);
    cookie.setSecure(secure_cookie);
    cookie.setSameSite(drogon::Cookie::SameSite::kLax);
    cookie.setMaxAge(0);
    response->addCookie(cookie);
    return response;
}

template <typename Handler>
void queue_session_request(Handler handler, Callback callback) {
    auto& workers = session_workers();
    if (workers.getTaskCount() >= kMaxQueuedSessionRequests) {
        callback(sessions_unavailable());
        return;
    }

    workers.runTaskInQueue([handler = std::move(handler), callback = std::move(callback)] {
        drogon::HttpResponsePtr response;
        try {
            response = handler();
        } catch (const std::exception&) {
            LOG_ERROR << "Session query failed";
            response = sessions_unavailable();
        }
        callback(response);
    });
}

}  // namespace

void register_session_routes(drogon::orm::DbClientPtr database, bool secure_cookie) {
    const repositories::SessionRepository sessions(std::move(database));

    drogon::app().registerHandler("/internal/introspect",
        [sessions](const drogon::HttpRequestPtr& request, Callback&& callback) {
            const auto body = request->getJsonObject();
            if (!body || !body->isObject() || !(*body)["token"].isString()) {
                callback(make_error_response(drogon::k400BadRequest, "INVALID_REQUEST",
                                             "Session token is required"));
                return;
            }

            auto token = (*body)["token"].asString();
            if (!services::is_login_token(token)) {
                callback(introspection_response(std::nullopt));
                return;
            }

            queue_session_request([sessions, token = std::move(token)] {
                return introspection_response(sessions.find_active_principal(token));
            }, std::move(callback));
        }, {drogon::Post});

    drogon::app().registerHandler("/auth/logout",
        [sessions, secure_cookie](const drogon::HttpRequestPtr& request, Callback&& callback) {
            const auto token = bearer_token(request->getHeader("authorization"));
            if (!services::is_login_token(token)) {
                callback(make_error_response(drogon::k401Unauthorized, "INVALID_SESSION",
                                             "A Bearer session token is required"));
                return;
            }

            queue_session_request([sessions, secure_cookie, token = std::string(token)] {
                sessions.revoke(token);
                return logout_response(secure_cookie);
            }, std::move(callback));
        }, {drogon::Post});
}

}  // namespace auth::http
