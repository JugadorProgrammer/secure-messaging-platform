#pragma once

#include <cstdint>
#include <string>

namespace auth::models {

struct Role {
    std::int64_t id = 0;
    std::string code;
    std::string description;
};

}