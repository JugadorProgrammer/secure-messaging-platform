#pragma once

#include <string>
#include <vector>

namespace auth::models {

struct User {
    std::string id;
    std::string login;
    std::string email;
    std::string password_hash;
    std::vector<std::string> roles;
};

}