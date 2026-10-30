
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
    src/repositories/database.cpp
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
