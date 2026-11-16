
cmake_minimum_required(VERSION 3.20)

project(smp_auth
    VERSION 0.1.0
    LANGUAGES CXX
)

# ============================================================
# Настройки C++
# ============================================================

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)

# ============================================================
# Зависимости
# ============================================================

find_package(Drogon CONFIG REQUIRED)
find_package(CURL 7.85 REQUIRED)
find_package(OpenSSL REQUIRED COMPONENTS Crypto)

# ============================================================
# Основной исполняемый файл AUTH
# ============================================================

add_executable(smp_auth
    src/main.cpp

    # HTTP-контроллеры
    src/http/health_controller.cpp
    src/http/auth_controller.cpp
    src/http/email_factor_controller.cpp
    src/http/password_reset_controller.cpp
    src/http/recovery_controller.cpp
    src/http/session_controller.cpp
    src/http/signup_controller.cpp
    src/http/users_controller.cpp
    src/http/error_response.cpp

    # Сервисы
    src/services/smtp_mailer.cpp
    src/services/first_factor.cpp
    src/services/email_factor.cpp
    src/services/login_token.cpp
    src/services/password_reset.cpp
    src/services/registration.cpp
    src/services/recovery_factor.cpp
    src/services/recovery_codes.cpp

    # Репозитории
    src/repositories/database.cpp
    src/repositories/user_repository.cpp
    src/repositories/session_repository.cpp
    src/repositories/recovery_codes_repository.cpp

    # Криптография
    src/password/pbkdf2_password_hasher.cpp

    # Валидация
    src/validation/registration_validator.cpp
)

target_include_directories(smp_auth PRIVATE
    ${CMAKE_CURRENT_SOURCE_DIR}/include
)

target_link_libraries(smp_auth PRIVATE
    Drogon::Drogon
    CURL::libcurl
    OpenSSL::Crypto
)

# ============================================================
# Предупреждения компилятора
# ============================================================

if(MSVC)
    target_compile_options(smp_auth PRIVATE /W4)
else()
    target_compile_options(smp_auth PRIVATE
        -Wall
        -Wextra
        -Wpedantic
    )
endif()

# ============================================================
# Настройки тестирования
# ============================================================

include(CTest)

# Интеграционные тесты требуют запущенной PostgreSQL.
#
# OFF — Docker-сборка и обычные модульные тесты.
# ON  — локальная сборка с подключением к PostgreSQL.

option(
    AUTH_ENABLE_DB_INTEGRATION_TESTS
    "Enable PostgreSQL integration tests"
    OFF
)

if(BUILD_TESTING)

    # ========================================================
    # AUTH-17: Единый формат HTTP-ошибок
    # ========================================================

    add_executable(auth_error_response_test
        tests/error_response_test.cpp
        src/http/error_response.cpp
    )

    target_include_directories(auth_error_response_test PRIVATE
        ${CMAKE_CURRENT_SOURCE_DIR}/include
    )

    target_link_libraries(auth_error_response_test PRIVATE
        Drogon::Drogon
    )

    add_test(
        NAME auth_error_response
        COMMAND auth_error_response_test
    )

    # ========================================================
    # AUTH-10 / AUTH-11: Интерфейс работы с паролями
    # ========================================================

    add_executable(auth_password_hasher_test
        tests/password_hasher_test.cpp
        src/password/stub_password_hasher.cpp
    )

    target_include_directories(auth_password_hasher_test PRIVATE
        ${CMAKE_CURRENT_SOURCE_DIR}/include
    )

    add_test(
        NAME auth_password_hasher
        COMMAND auth_password_hasher_test
    )

    # ========================================================
    # AUTH-12: Валидация регистрации
    # ========================================================

    add_executable(auth_registration_validator_test
        tests/registration_validator_test.cpp
        src/validation/registration_validator.cpp
    )

    target_include_directories(auth_registration_validator_test PRIVATE
        ${CMAKE_CURRENT_SOURCE_DIR}/include
    )

    add_test(
        NAME auth_registration_validator
        COMMAND auth_registration_validator_test
    )

    # ========================================================
    # AUTH-14: Генерация кодов восстановления
    # ========================================================

    add_executable(auth_recovery_codes_test
        tests/recovery_codes_test.cpp
        src/services/recovery_codes.cpp
    )

    target_include_directories(auth_recovery_codes_test PRIVATE
        ${CMAKE_CURRENT_SOURCE_DIR}/include
    )

    target_link_libraries(auth_recovery_codes_test PRIVATE
        OpenSSL::Crypto
    )

    add_test(
        NAME auth_recovery_codes
        COMMAND auth_recovery_codes_test
    )

    # ========================================================
    # PBKDF2: Хэширование и проверка паролей
    # ========================================================

    add_executable(auth_pbkdf2_password_hasher_test
        tests/pbkdf2_password_hasher_test.cpp
        src/password/pbkdf2_password_hasher.cpp
    )

    target_include_directories(
        auth_pbkdf2_password_hasher_test PRIVATE
        ${CMAKE_CURRENT_SOURCE_DIR}/include
    )

    target_link_libraries(
        auth_pbkdf2_password_hasher_test PRIVATE
        OpenSSL::Crypto
    )

    add_test(
        NAME auth_pbkdf2_password_hasher
        COMMAND auth_pbkdf2_password_hasher_test
    )

    # ========================================================
    # Интеграционные тесты PostgreSQL
    # ========================================================

    if(AUTH_ENABLE_DB_INTEGRATION_TESTS)

        # ----------------------------------------------------
        # AUTH-8 / AUTH-9: Поиск пользователей
        # ----------------------------------------------------

        add_executable(auth_user_repository_test
            tests/user_repository_integration_test.cpp
            src/repositories/database.cpp
            src/repositories/user_repository.cpp
        )

        target_include_directories(auth_user_repository_test PRIVATE
            ${CMAKE_CURRENT_SOURCE_DIR}/include
        )

        target_link_libraries(auth_user_repository_test PRIVATE
            Drogon::Drogon
        )

        add_test(
            NAME auth_user_repository
            COMMAND auth_user_repository_test
        )

        # ----------------------------------------------------
        # AUTH-21: Работа с сессиями
        # ----------------------------------------------------

        add_executable(auth_session_repository_test
            tests/session_repository_integration_test.cpp
            src/repositories/database.cpp
            src/repositories/session_repository.cpp
            src/services/login_token.cpp
        )

        target_include_directories(auth_session_repository_test PRIVATE
            ${CMAKE_CURRENT_SOURCE_DIR}/include
        )

        target_link_libraries(auth_session_repository_test PRIVATE
            Drogon::Drogon
            OpenSSL::Crypto
        )

        add_test(
            NAME auth_session_repository
            COMMAND auth_session_repository_test
        )

        # ----------------------------------------------------
        # AUTH-22: Третий фактор аутентификации
        # ----------------------------------------------------

        add_executable(auth_recovery_factor_test
            tests/recovery_factor_integration_test.cpp
            src/repositories/database.cpp
            src/repositories/recovery_codes_repository.cpp
            src/repositories/session_repository.cpp
            src/services/login_token.cpp
            src/services/recovery_codes.cpp
            src/services/recovery_factor.cpp
        )

        target_include_directories(auth_recovery_factor_test PRIVATE
            ${CMAKE_CURRENT_SOURCE_DIR}/include
        )

        target_link_libraries(auth_recovery_factor_test PRIVATE
            Drogon::Drogon
            OpenSSL::Crypto
        )

        add_test(
            NAME auth_recovery_factor
            COMMAND auth_recovery_factor_test
        )

        # ----------------------------------------------------
        # AUTH-20: Второй фактор аутентификации (email)
        # ----------------------------------------------------

        add_executable(auth_email_factor_test
            tests/email_factor_integration_test.cpp
            src/repositories/database.cpp
            src/repositories/recovery_codes_repository.cpp
            src/repositories/session_repository.cpp
            src/services/login_token.cpp
            src/services/recovery_codes.cpp
            src/services/recovery_factor.cpp
            src/services/email_factor.cpp
        )

        target_include_directories(auth_email_factor_test PRIVATE
            ${CMAKE_CURRENT_SOURCE_DIR}/include
        )

        target_link_libraries(auth_email_factor_test PRIVATE
            Drogon::Drogon
            OpenSSL::Crypto
        )

        add_test(
            NAME auth_email_factor
            COMMAND auth_email_factor_test
        )

    endif()

    # ========================================================
    # Ограничение времени выполнения тестов
    # ========================================================

    # Защита от бесконечного ожидания при ошибках PostgreSQL.
    set_property(
        DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR}
        PROPERTY TEST_TIMEOUT 30
    )

endif()
