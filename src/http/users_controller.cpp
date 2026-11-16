#include "auth/http/users_controller.h"

#include <drogon/drogon.h>
#include <trantor/utils/ConcurrentTaskQueue.h>

#include <exception>
#include <functional>
#include <string>
#include <utility>

#include "auth/http/error_response.h"
#include "auth/http/session_controller.h"
#include "auth/repositories/session_repository.h"
#include "auth/repositories/user_repository.h"
#include "auth/services/login_token.h"
#include "auth/services/registration.h"

namespace auth::http {

namespace {

using Callback = std::function<void(const drogon::HttpResponsePtr&)>;

drogon::HttpResponsePtr users_unavailable() {
    return make_error_response(drogon::k503ServiceUnavailable, "USERS_UNAVAILABLE",
                               "User management temporarily unavailable");
}

drogon::HttpResponsePtr invalid_session() {
    return make_error_response(drogon::k401Unauthorized, "INVALID_SESSION",
                               "An active Bearer session token is required");
}

drogon::HttpResponsePtr list_users(const drogon::orm::DbClientPtr& database) {
    const repositories::UserRepository users(database);
    Json::Value body(Json::arrayValue);
    for (const auto& user : users.list_users()) {
        Json::Value entry(Json::objectValue);
        entry["login"] = user.login;
        entry["email"] = user.email;
        entry["role"] = user.role ? Json::Value(*user.role) : Json::Value();
        body.append(std::move(entry));
    }

    auto response = drogon::HttpResponse::newHttpJsonResponse(body);
    response->addHeader("Cache-Control", "no-store");
    return response;
}

drogon::HttpResponsePtr create_user(
    const drogon::HttpRequestPtr& request,
    const drogon::orm::DbClientPtr& database,
    password::IPasswordHasher& hasher
) {
    const auto body = request->getJsonObject();
    if (!body || !body->isObject()) {
        return make_error_response(drogon::k400BadRequest, "INVALID_REQUEST",
                                   "A JSON object is required");
    }

    const auto& fields = *body;
    if (!fields["login"].isString() || !fields["email"].isString() ||
        !fields["password"].isString() || !fields["role"].isString()) {
        return make_error_response(drogon::k400BadRequest, "INVALID_REQUEST",
                                   "Login, email, password and role are required");
    }

    const auto role = fields["role"].asString();
    const validation::RegistrationData data{
        fields["login"].asString(), fields["email"].asString(), fields["password"].asString()
    };
    if ((role != "user" && role != "admin") || !validation::validate_registration(data).is_valid()) {
        return make_error_response(drogon::k400BadRequest, "INVALID_REQUEST",
                                   "Invalid login, email, password or role");
    }

    const auto codes = services::register_user(database, hasher, data, role);
    if (!codes) {
        return make_error_response(drogon::k409Conflict, "ACCOUNT_ALREADY_EXISTS",
                                   "Login or email is already in use");
    }

    Json::Value result(Json::arrayValue);
    for (const auto& code : *codes) {
        result.append(code);
    }
    auto response = drogon::HttpResponse::newHttpJsonResponse(result);
    response->setStatusCode(drogon::k201Created);
    response->addHeader("Cache-Control", "no-store");
    return response;
}

}  // namespace

void register_users_routes(
    drogon::orm::DbClientPtr database,
    std::shared_ptr<password::IPasswordHasher> hasher
) {
    constexpr std::size_t kMaxQueuedUserRequests = 16;
    static trantor::ConcurrentTaskQueue workers(2, "auth-users");
    const repositories::SessionRepository sessions(database);

    drogon::app().registerHandler("/users",
        [database = std::move(database), hasher = std::move(hasher), sessions](
            const drogon::HttpRequestPtr& request, Callback&& callback
        ) {
            const auto token = bearer_token(request->getHeader("authorization"));
            if (!services::is_login_token(token)) {
                callback(invalid_session());
                return;
            }
            if (workers.getTaskCount() >= kMaxQueuedUserRequests) {
                callback(users_unavailable());
                return;
            }

            workers.runTaskInQueue([database, hasher, sessions, request,
                                    token = std::string(token), callback = std::move(callback)] {
                drogon::HttpResponsePtr response;
                try {
                    const auto principal = sessions.find_active_principal(token);
                    if (!principal) {
                        response = invalid_session();
                    } else if (principal->role != "admin") {
                        response = make_error_response(drogon::k403Forbidden, "FORBIDDEN",
                                                       "Administrator role is required");
                    } else if (request->method() == drogon::Get) {
                        response = list_users(database);
                    } else {
                        response = create_user(request, database, *hasher);
                    }
                } catch (const std::exception&) {
                    LOG_ERROR << "User management request failed";
                    response = users_unavailable();
                }
                callback(response);
            });
        }, {drogon::Get, drogon::Post});
}

}  // namespace auth::http
