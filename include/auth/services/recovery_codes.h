#pragma once
#include <cstddef>
#include <string>
#include <vector>
namespace auth::services {
std::vector<std::string> generate_recovery_codes(std::size_t count = 10);
}
