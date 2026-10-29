
#pragma once

namespace auth::repositories {

// Проверяет доступность PostgreSQL.
// Возвращает true, если SELECT 1 выполнен успешно.
bool check_database_connection();

}  // namespace auth::repositories
