#include "auth/services/smtp_mailer.h"

#include <curl/curl.h>

#include <algorithm>
#include <cstdlib>
#include <cstring>
#include <ctime>
#include <iomanip>
#include <locale>
#include <memory>
#include <regex>
#include <sstream>
#include <stdexcept>
#include <utility>

namespace auth::services {

namespace {

std::string read_env(const char* variable) {
    const char* value = std::getenv(variable);
    return value == nullptr ? std::string{} : std::string(value);
}

bool is_mailbox(std::string_view value) {
    static const std::regex mailbox(R"([A-Za-z0-9.!#$%&'*+/=?^_`{|}~-]+@[A-Za-z0-9.-]+)");
    return value.size() <= 254 && std::regex_match(value.begin(), value.end(), mailbox);
}

std::string smtp_date() {
    const auto now = std::time(nullptr);
    std::tm utc{};
#ifdef _WIN32
    if (gmtime_s(&utc, &now) != 0) {
#else
    if (gmtime_r(&now, &utc) == nullptr) {
#endif
        throw std::runtime_error("Cannot format SMTP date");
    }

    std::ostringstream date;
    date.imbue(std::locale::classic());
    date << std::put_time(&utc, "%a, %d %b %Y %H:%M:%S +0000");
    return date.str();
}

template <typename T>
void set_option(CURL* client, CURLoption option, T value) {
    if (curl_easy_setopt(client, option, value) != CURLE_OK) {
        throw std::runtime_error("Cannot configure SMTP transfer");
    }
}

std::size_t read_payload(char* buffer, std::size_t size, std::size_t count, void* data) {
    auto& remaining = *static_cast<std::string_view*>(data);
    const auto bytes = std::min(size * count, remaining.size());
    if (bytes != 0) {
        std::memcpy(buffer, remaining.data(), bytes);
        remaining.remove_prefix(bytes);
    }
    return bytes;
}

}  // namespace

SmtpSettings smtp_settings_from_environment() {
    const auto security = read_env("AUTH_SMTP_SECURITY");
    if (!security.empty() && security != "tls" && security != "none") {
        throw std::invalid_argument("AUTH_SMTP_SECURITY must be tls or none");
    }

    return {
        read_env("AUTH_SMTP_URL"),
        read_env("AUTH_SMTP_FROM"),
        read_env("AUTH_SMTP_USERNAME"),
        read_env("AUTH_SMTP_PASSWORD"),
        security != "none"
    };
}

SmtpMailer::SmtpMailer(SmtpSettings settings) : settings_(std::move(settings)) {
    if (settings_.url.empty()) {
        return;
    }

    if ((!settings_.url.starts_with("smtp://") && !settings_.url.starts_with("smtps://")) ||
        !is_mailbox(settings_.from) ||
        settings_.username.empty() != settings_.password.empty() ||
        (!settings_.require_tls && !settings_.username.empty())) {
        throw std::invalid_argument("Invalid SMTP settings");
    }

    static const auto initialized = curl_global_init(CURL_GLOBAL_DEFAULT);
    if (initialized != CURLE_OK) {
        throw std::runtime_error("Cannot initialize SMTP client");
    }
}

void SmtpMailer::send_verification_code(std::string_view recipient, std::string_view code) const {
    if (settings_.url.empty()) {
        throw std::runtime_error("SMTP is not configured");
    }
    if (!is_mailbox(recipient) || code.size() != 6 ||
        !std::all_of(code.begin(), code.end(), [](char ch) { return ch >= '0' && ch <= '9'; })) {
        throw std::invalid_argument("Invalid verification email data");
    }

    const auto from = "<" + settings_.from + ">";
    const auto to = "<" + std::string(recipient) + ">";
    const auto payload = "Date: " + smtp_date() + "\r\nFrom: " + from + "\r\nTo: " + to +
        "\r\nSubject: Secure Messaging Platform verification code"
        "\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n"
        "\r\nYour verification code is " + std::string(code) + ".\r\n";
    std::string_view payload_remaining(payload);

    std::unique_ptr<curl_slist, decltype(&curl_slist_free_all)> recipients(
        curl_slist_append(nullptr, to.c_str()), curl_slist_free_all
    );
    std::unique_ptr<CURL, decltype(&curl_easy_cleanup)> client(curl_easy_init(), curl_easy_cleanup);
    if (!client || !recipients) {
        throw std::runtime_error("Cannot create SMTP transfer");
    }

    set_option(client.get(), CURLOPT_URL, settings_.url.c_str());
    set_option(client.get(), CURLOPT_PROTOCOLS_STR, "smtp,smtps");
    set_option(client.get(), CURLOPT_DISALLOW_USERNAME_IN_URL, 1L);
    set_option(client.get(), CURLOPT_PROXY, "");
    set_option(client.get(), CURLOPT_USE_SSL,
               settings_.require_tls ? static_cast<long>(CURLUSESSL_ALL) : static_cast<long>(CURLUSESSL_NONE));
    set_option(client.get(), CURLOPT_NOSIGNAL, 1L);
    set_option(client.get(), CURLOPT_CONNECTTIMEOUT, 3L);
    set_option(client.get(), CURLOPT_TIMEOUT, 8L);
    if (!settings_.username.empty()) {
        set_option(client.get(), CURLOPT_USERNAME, settings_.username.c_str());
        set_option(client.get(), CURLOPT_PASSWORD, settings_.password.c_str());
    }
    set_option(client.get(), CURLOPT_MAIL_FROM, from.c_str());
    set_option(client.get(), CURLOPT_MAIL_RCPT, recipients.get());
    set_option(client.get(), CURLOPT_READFUNCTION, read_payload);
    set_option(client.get(), CURLOPT_READDATA, &payload_remaining);
    set_option(client.get(), CURLOPT_UPLOAD, 1L);

    const auto result = curl_easy_perform(client.get());
    if (result != CURLE_OK) {
        throw std::runtime_error(std::string("SMTP delivery failed: ") + curl_easy_strerror(result));
    }
}

}  // namespace auth::services
