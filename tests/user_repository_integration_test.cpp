
#include "auth/repositories/database.h"
#include "auth/repositories/user_repository.h"

#include <chrono>
#include <iostream>
#include <stdexcept>
#include <string>

int main() {
    using namespace auth::repositories;

    try {
        auto client = create_database_client();
        UserRepository repository(client);

        const auto stamp =
            std::chrono::steady_clock::now()
                .time_since_epoch().count();

        const std::string login =
            "auth8_test_" + std::to_string(stamp);
        const std::string email =
            login + "@example.test";

        // Создаём тестовую запись.
        const auto inserted = client->execSqlSync(
            R"(
                INSERT INTO users
                    (login, email, password_hash)
                VALUES ($1, $2, $3)
                RETURNING id::text AS id
            )",
            login,
            email,
            "test-only-not-a-password-hash"
        );

        const std::string id =
            inserted[0]["id"].as<std::string>();

        // Важно: запись будет удалена даже при ошибке
        // в одной из последующих проверок.
        try {
            client->execSqlSync(
                R"(
                    INSERT INTO user_roles (user_id, role_id)
                    SELECT $1::uuid, id
                    FROM roles
                    WHERE code = 'user'
                )",
                id
            );

            const auto user =
                repository.find_by_login(login);

            if (!user ||
                user->id != id ||
                user->login != login ||
                user->email != email ||
                user->roles.size() != 1 ||
                user->roles[0] != "user") {
                throw std::runtime_error(
                    "User lookup failed"
                );
            }

            // Проверяем поиск без учёта регистра.
            std::string upper_login = login;
            for (char& c : upper_login) {
                if (c >= 'a' && c <= 'z') {
                    c = static_cast<char>(c - 'a' + 'A');
                }
            }

            if (!repository.find_by_login(upper_login)) {
                throw std::runtime_error(
                    "Case insensitive lookup failed"
                );
            }

            // Пользователь отсутствует.
            if (repository.find_by_login(
                    login + "_missing")) {
                throw std::runtime_error(
                    "Unexpected user found"
                );
            }

            // Проверка параметризации SQL.
            if (repository.find_by_login(
                    "' OR 1=1 --")) {
                throw std::runtime_error(
                    "SQL injection test failed"
                );
            }

            const auto found_by_email =
                repository.find_by_email(email);

            if (!found_by_email ||
                found_by_email->id != id ||
                found_by_email->login != login ||
                found_by_email->email != email ||
                found_by_email->password_hash !=
                    "test-only-not-a-password-hash" ||
                found_by_email->roles.size() != 1 ||
                found_by_email->roles[0] != "user") {
                throw std::runtime_error(
                    "User lookup by email failed"
                );
            }

            // Проверка поиска без учёта регистра.
            std::string upper_email = email;

            for (char& c : upper_email) {
                if (c >= 'a' && c <= 'z') {
                    c = static_cast<char>(c - 'a' + 'A');
                }
            }

            const auto case_insensitive =
                repository.find_by_email(upper_email);

            if (!case_insensitive ||
                case_insensitive->id != id) {
                throw std::runtime_error(
                    "Case insensitive email lookup failed"
                );
            }

            // Несуществующий email.
            if (repository.find_by_email(
                    "missing_" + email)) {
                throw std::runtime_error(
                    "Unexpected user found by email"
                );
            }

            // Проверка на SQL-инъекцию.
            if (repository.find_by_email("' OR 1=1 --")) {
                throw std::runtime_error(
                    "Email lookup SQL injection test failed"
                );
            }


        } catch (...) {
            client->execSqlSync(
                "DELETE FROM users WHERE id = $1::uuid",
                id
            );
            throw;
        }

        client->execSqlSync(
            "DELETE FROM users WHERE id = $1::uuid",
            id
        );

        std::cout
            << "User repository integration test: PASSED\n";

        return 0;

    } catch (const std::exception& e) {
        std::cerr
            << "User repository integration test: FAILED: "
            << e.what() << '\n';

        return 1;
    }
}
