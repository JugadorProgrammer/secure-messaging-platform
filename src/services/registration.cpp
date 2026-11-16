#include "auth/services/registration.h"

#include <future>
#include <memory>
#include <stdexcept>

#include "auth/repositories/recovery_codes_repository.h"
#include "auth/repositories/user_repository.h"
#include "auth/services/recovery_codes.h"

namespace auth::services {

std::optional<std::vector<std::string>> register_user(
    const drogon::orm::DbClientPtr& database,
    password::IPasswordHasher& hasher,
    const validation::RegistrationData& data
) {
    const auto password_hash = hasher.hash_password(data.password);
    auto recovery_codes = generate_recovery_codes();

    auto commit = std::make_shared<std::promise<bool>>();
    auto commit_result = commit->get_future();
    {
        const auto transaction = database->newTransaction([commit](bool success) {
            commit->set_value(success);
        });
        if (!transaction) {
            throw std::runtime_error("Cannot start registration transaction");
        }

        try {
            const repositories::UserRepository users(transaction);
            const auto user_id = users.create_user(data.login, data.email, password_hash);
            if (!user_id) {
                transaction->rollback();
                return std::nullopt;
            }

            repositories::save_recovery_codes(transaction, *user_id, recovery_codes);
        } catch (...) {
            transaction->rollback();
            throw;
        }
    }

    // Drogon отправляет COMMIT при уничтожении объекта Transaction.
    if (!commit_result.get()) {
        throw std::runtime_error("Registration transaction was not committed");
    }

    return recovery_codes;
}

}  // namespace auth::services
