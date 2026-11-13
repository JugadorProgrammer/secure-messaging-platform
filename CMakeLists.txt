
cmake_minimum_required(VERSION 3.20)

project(smp_auth
    VERSION 0.1.0
    LANGUAGES CXX
)

# Используем C++20
set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)

# Поиск установленного фреймворка Drogon
find_package(Drogon CONFIG REQUIRED)

# Исполняемый файл сервиса AUTH
add_executable(smp_auth
    src/main.cpp
    src/http/health_controller.cpp
    src/http/error_response.cpp
    src/repositories/database.cpp
    src/password/stub_password_hasher.cpp
    src/validation/registration_validator.cpp
    src/repositories/user_repository.cpp
)

# Путь для поиска собственных заголовочных файлов
target_include_directories(smp_auth PRIVATE
    ${CMAKE_CURRENT_SOURCE_DIR}/include
)

# Подключение Drogon
target_link_libraries(smp_auth PRIVATE
    Drogon::Drogon
)

# Предупреждения компилятора
if(MSVC)
    target_compile_options(smp_auth PRIVATE /W4)
else()
    target_compile_options(smp_auth PRIVATE
        -Wall
        -Wextra
        -Wpedantic
    )
endif()

include(CTest)

if(BUILD_TESTING)
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

    option(
    AUTH_ENABLE_DB_INTEGRATION_TESTS
    "Enable PostgreSQL integration tests"
    OFF
)

if(AUTH_ENABLE_DB_INTEGRATION_TESTS)
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
endif()
endif()