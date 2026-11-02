
#include "auth/http/error_response.h"

#include <drogon/drogon.h>

#include <iostream>
#include <string>

bool check_error_response(
    drogon::HttpStatusCode status,
    const std::string& code,
    const std::string& message
) {
    auto response =
        auth::http::make_error_response(status, code, message);

    if (response->statusCode() != status) {
        std::cerr << "Incorrect HTTP status\n";
        return false;
    }

    const auto json = response->getJsonObject();

    if (!json || !json->isObject() ||
        !json->isMember("error")) {
        std::cerr << "Missing error object\n";
        return false;
    }

    const auto& error = (*json)["error"];

    if (!error.isObject() ||
        !error["code"].isString() ||
        !error["message"].isString()) {
        std::cerr << "Invalid error structure\n";
        return false;
    }

    if (error["code"].asString() != code ||
        error["message"].asString() != message) {
        std::cerr << "Incorrect error values\n";
        return false;
    }

    return true;
}

int main() {
    if (!check_error_response(
            drogon::k400BadRequest,
            "INVALID_REQUEST",
            "Invalid request parameters")) {
        return 1;
    }

    if (!check_error_response(
            drogon::k401Unauthorized,
            "UNAUTHORIZED",
            "Authentication required")) {
        return 1;
    }

    if (!check_error_response(
            drogon::k500InternalServerError,
            "INTERNAL_ERROR",
            "Internal server error")) {
        return 1;
    }

    std::cout << "Error response tests: PASSED\n";
    return 0;
}
