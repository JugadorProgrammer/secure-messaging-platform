#include "auth/services/recovery_codes.h"
#include <set>
#include <stdexcept>
#include <string>
#include <iostream>
int main() {
    const auto codes = auth::services::generate_recovery_codes();
    if (codes.size() != 10 || std::set<std::string>(codes.begin(), codes.end()).size() != 10) return 1;
    for (const auto& code : codes) {
        if (code.size() != 35 || code[8] != '-' || code[17] != '-' || code[26] != '-') return 2;
        for (size_t i = 0; i < code.size(); ++i) {
            if (i == 8 || i == 17 || i == 26) continue;
            if (!(code[i] >= '0' && code[i] <= '9') && !(code[i] >= 'A' && code[i] <= 'F')) return 3;
        }
    }
    try { auth::services::generate_recovery_codes(0); return 4; } catch (const std::invalid_argument&) {}
    std::cout << "AUTH-14: OK\n";
}
