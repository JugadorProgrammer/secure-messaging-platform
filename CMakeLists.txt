

find_package(OpenSSL REQUIRED COMPONENTS Crypto)
target_sources(smp_auth PRIVATE src/services/recovery_codes.cpp)
target_sources(smp_auth PRIVATE src/repositories/recovery_codes_repository.cpp)
target_link_libraries(smp_auth PRIVATE OpenSSL::Crypto)

include(CTest)
if(BUILD_TESTING)
    add_executable(auth_recovery_codes_test
        tests/recovery_codes_test.cpp
        src/services/recovery_codes.cpp
    )
    target_include_directories(auth_recovery_codes_test PRIVATE ${CMAKE_CURRENT_SOURCE_DIR}/include)
    target_link_libraries(auth_recovery_codes_test PRIVATE OpenSSL::Crypto)
    add_test(NAME auth_recovery_codes COMMAND auth_recovery_codes_test)
endif()

