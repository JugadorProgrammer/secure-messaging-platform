#include "auth/http/password_reset_controller.h"

#include <drogon/drogon.h>
#include <trantor/utils/ConcurrentTaskQueue.h>

#include <algorithm>
#include <exception>
#include <functional>
#include <utility>

#include "auth/http/error_response.h"
#include "auth/validation/registration_validator.h"

namespace auth::http {

namespace {

using Callback = std::function<void(const drogon::HttpResponsePtr&)>;
using services::kPasswordResetGrantCookie;
using services::kPasswordResetLifetimeSeconds;
using services::kPasswordResetPendingCookie;

enum class ResetStep { confirm, code, password };

drogon::HttpResponsePtr unavailable() {
    return make_error_response(drogon::k503ServiceUnavailable, "PASSWORD_RESET_UNAVAILABLE",
                               "Password recovery temporarily unavailable");
}

drogon::HttpResponsePtr invalid_request(const char* message) {
    return make_error_response(drogon::k400BadRequest, "INVALID_REQUEST", message);
}

drogon::HttpResponsePtr rejected() {
    return make_error_response(drogon::k401Unauthorized, "INVALID_PASSWORD_RESET",
                               "Invalid code or password recovery attempt");
}

drogon::HttpResponsePtr no_content() {
    auto response = drogon::HttpResponse::newHttpResponse();
    response->setStatusCode(drogon::k204NoContent);
    response->addHeader("Cache-Control", "no-store");
    return response;
}

void set_cookie(const drogon::HttpResponsePtr& response, std::string_view name,
                const std::string& value, int max_age, bool secure_cookie) {
    drogon::Cookie cookie(std::string(name), value);
    cookie.setPath("/");
    cookie.setHttpOnly(true);
    cookie.setSecure(secure_cookie);
    cookie.setSameSite(drogon::Cookie::SameSite::kLax);
    cookie.setMaxAge(max_age);
    response->addCookie(cookie);
}

auto& reset_workers() {
    static trantor::ConcurrentTaskQueue workers(2, "auth-password-reset");
    return workers;
}

bool queue_full() {
    constexpr std::size_t kMaxQueuedResets = 16;
    return reset_workers().getTaskCount() >= kMaxQueuedResets;
}

drogon::HttpResponsePtr confirm_reset(const services::PasswordReset& reset,
                                      const drogon::HttpRequestPtr& request,
                                      ResetStep step, bool secure_cookie) {
    const auto body = request->getJsonObject();
    if (!body || !body->isObject()) {
        return invalid_request("A JSON object is required");
    }

    const auto& fields = *body;
    const bool has_code = fields.isMember("code");
    const bool has_password = fields.isMember("password");
    if (has_code == has_password || (step == ResetStep::code && !has_code) ||
        (step == ResetStep::password && !has_password)) {
        return invalid_request("Supply either the email code or the new password");
    }

    if (has_code) {
        const auto code = fields["code"].isString() ? fields["code"].asString() : std::string{};
        if (code.size() != 6 || !std::all_of(code.begin(), code.end(), [](char ch) {
                return ch >= '0' && ch <= '9';
            })) {
            return invalid_request("A six-digit email code is required");
        }

        const auto grant = reset.confirm_code(request->getCookie(std::string(kPasswordResetPendingCookie)), code);
        if (!grant) {
            return rejected();
        }

        auto response = no_content();
        set_cookie(response, kPasswordResetPendingCookie, "", 0, secure_cookie);
        set_cookie(response, kPasswordResetGrantCookie, *grant, kPasswordResetLifetimeSeconds, secure_cookie);
        return response;
    }

    const auto password = fields["password"].isString() ? fields["password"].asString() : std::string{};
    if (!validation::is_valid_password(password)) {
        return invalid_request("Invalid new password");
    }

    if (!reset.change_password(request->getCookie(std::string(kPasswordResetGrantCookie)), password)) {
        return rejected();
    }

    auto response = no_content();
    set_cookie(response, kPasswordResetGrantCookie, "", 0, secure_cookie);
    return response;
}

}  // namespace

void register_password_reset_routes(std::shared_ptr<const services::PasswordReset> reset, bool secure_cookie) {
    auto request_code = [reset, secure_cookie](const drogon::HttpRequestPtr& request, Callback&& callback) {
        const auto body = request->getJsonObject();
        if (!body || !body->isObject() || !std::as_const(*body)["email"].isString()) {
            callback(invalid_request("Email is required"));
            return;
        }

        auto email = std::as_const(*body)["email"].asString();
        if (!validation::is_valid_email(email)) {
            callback(invalid_request("Invalid email"));
            return;
        }
        if (queue_full()) {
            callback(unavailable());
            return;
        }

        auto previous = request->getCookie(std::string(kPasswordResetPendingCookie));
        reset_workers().runTaskInQueue([reset, secure_cookie, email = std::move(email),
                                        previous = std::move(previous), callback = std::move(callback)] {
            services::PasswordResetRequest pending;
            try {
                pending = reset->request_code(email, previous);
            } catch (const std::exception&) {
                LOG_ERROR << "Password reset request failed";
                callback(unavailable());
                return;
            }

            auto response = no_content();
            set_cookie(response, kPasswordResetPendingCookie, pending.pending_token,
                       kPasswordResetLifetimeSeconds, secure_cookie);
            // Reply before SMTP delivery for both registered and unknown addresses.
            callback(response);

            try {
                reset->send_code(pending);
            } catch (const std::exception&) {
                LOG_ERROR << "Password reset email delivery failed";
            }
        });
    };
    drogon::app().registerHandler("/auth/password/restore", decltype(request_code){request_code}, {drogon::Post});
    drogon::app().registerHandler("/restorePassword", std::move(request_code), {drogon::Post});

    auto confirm = [reset, secure_cookie](ResetStep step) {
        return [reset, secure_cookie, step](const drogon::HttpRequestPtr& request, Callback&& callback) {
            if (queue_full()) {
                callback(unavailable());
                return;
            }
            reset_workers().runTaskInQueue([reset, request, step, secure_cookie, callback = std::move(callback)] {
                drogon::HttpResponsePtr response;
                try {
                    response = confirm_reset(*reset, request, step, secure_cookie);
                } catch (const std::exception&) {
                    LOG_ERROR << "Password reset confirmation failed";
                    response = unavailable();
                }
                callback(response);
            });
        };
    };

    drogon::app().registerHandler("/auth/password/confirm", confirm(ResetStep::confirm), {drogon::Post});
    drogon::app().registerHandler("/confirmRestoreCode", confirm(ResetStep::code), {drogon::Post});
    drogon::app().registerHandler("/confirmNewPassword", confirm(ResetStep::password), {drogon::Post});
}

}  // namespace auth::http
