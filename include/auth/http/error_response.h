
#pragma once

#include <drogon/HttpResponse.h>

#include <string>

namespace auth::http {

// Создаёт HTTP-ответ об ошибке в едином JSON-формате.
drogon::HttpResponsePtr make_error_response(
    drogon::HttpStatusCode status,
    const std::string& code,
    const std::string& message
);

}  // namespace auth::http
