
#include "auth/http/error_response.h"

#include <drogon/drogon.h>

namespace auth::http {

drogon::HttpResponsePtr make_error_response(
    drogon::HttpStatusCode status,
    const std::string& code,
    const std::string& message
) {
    Json::Value body;
    body["error"]["code"] = code;
    body["error"]["message"] = message;

    auto response = drogon::HttpResponse::newHttpJsonResponse(body);
    response->setStatusCode(status);

    return response;
}

}  // namespace auth::http
