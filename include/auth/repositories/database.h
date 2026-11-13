
#pragma once

#include <drogon/orm/DbClient.h>

namespace auth::repositories {

// Создаёт клиент PostgreSQL по переменным окружения.
drogon::orm::DbClientPtr create_database_client();

// Проверяет соединение с PostgreSQL.
bool check_database_connection();

}  // namespace auth::repositories
