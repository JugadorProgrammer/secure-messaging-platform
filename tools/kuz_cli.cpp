#include <fstream>
#include <iostream>
#include <iterator>
#include <string>
#include <vector>

#include "hex.h"
#include "kuznyechik.h"

namespace {

void PrintUsage() {
    std::cout << "usage: kuz_cli selftest\n"
                 "       kuz_cli encrypt <key-hex> <iv-hex> <input-file> <output-hex-file>\n"
                 "       kuz_cli decrypt <key-hex> <iv-hex> <input-hex-file> <output-file>\n";
}

bool ReadBinary(const std::string& path, std::vector<std::uint8_t>& data) {
    std::ifstream input(path, std::ios::binary);
    if (!input.is_open()) {
        return false;
    }
    data.assign(std::istreambuf_iterator<char>(input), std::istreambuf_iterator<char>());
    return true;
}

bool WriteBinary(const std::string& path, const std::vector<std::uint8_t>& data) {
    std::ofstream output(path, std::ios::binary);
    if (!output.is_open()) {
        return false;
    }
    output.write(reinterpret_cast<const char*>(data.data()),
                 static_cast<std::streamsize>(data.size()));
    return output.good();
}

bool WriteText(const std::string& path, const std::string& text) {
    std::ofstream output(path, std::ios::binary);
    if (!output.is_open()) {
        return false;
    }
    output << text;
    return output.good();
}

// Шифртекст лежит в файле как hex-текст: так удобно смотреть глазами.
int EncryptFile(const std::vector<std::uint8_t>& key, const std::vector<std::uint8_t>& iv,
                const std::vector<std::uint8_t>& plaintext, const std::string& path) {
    std::vector<std::uint8_t> ciphertext(KuzCiphertextSize(plaintext.size()));
    std::size_t written = 0;
    const KuzStatus status =
        KuzEncrypt(key.data(), key.size(), iv.data(), iv.size(), plaintext.data(),
                   plaintext.size(), ciphertext.data(), ciphertext.size(), &written);
    if (status != KUZ_STATUS_OK) {
        std::cerr << "error: " << KuzStatusMessage(status) << '\n';
        return 1;
    }

    ciphertext.resize(written);
    if (!WriteText(path, kuznyechik::HexFromBytes(ciphertext))) {
        std::cerr << "error: cannot write " << path << '\n';
        return 2;
    }
    return 0;
}

int DecryptFile(const std::vector<std::uint8_t>& key, const std::vector<std::uint8_t>& iv,
                const std::vector<std::uint8_t>& ciphertext, const std::string& path) {
    std::vector<std::uint8_t> plaintext(ciphertext.size());
    std::size_t written = 0;
    const KuzStatus status =
        KuzDecrypt(key.data(), key.size(), iv.data(), iv.size(), ciphertext.data(),
                   ciphertext.size(), plaintext.data(), plaintext.size(), &written);
    if (status != KUZ_STATUS_OK) {
        std::cerr << "error: " << KuzStatusMessage(status) << '\n';
        return 1;
    }

    plaintext.resize(written);
    if (!WriteBinary(path, plaintext)) {
        std::cerr << "error: cannot write " << path << '\n';
        return 2;
    }
    return 0;
}

// Hex-файл может заканчиваться переводом строки — отбрасываем пробелы по краям.
std::string ToText(const std::vector<std::uint8_t>& data) {
    std::string text(data.begin(), data.end());
    while (!text.empty() && (text.back() == '\n' || text.back() == '\r' || text.back() == ' ')) {
        text.pop_back();
    }
    return text;
}

}  // namespace

int main(int argc, char** argv) {
    try {
        if (argc == 2 && std::string(argv[1]) == "selftest") {
            const KuzStatus status = KuzSelfTest();
            std::cout << "version: " << KuzVersion() << '\n';
            std::cout << "selftest: " << KuzStatusMessage(status) << '\n';
            return status == KUZ_STATUS_OK ? 0 : 1;
        }

        if (argc != 6) {
            PrintUsage();
            return 2;
        }

        const std::string command = argv[1];
        const std::vector<std::uint8_t> key = kuznyechik::BytesFromHex(argv[2]);
        const std::vector<std::uint8_t> iv = kuznyechik::BytesFromHex(argv[3]);

        std::vector<std::uint8_t> input;
        if (!ReadBinary(argv[4], input)) {
            std::cerr << "error: cannot read " << argv[4] << '\n';
            return 2;
        }

        if (command == "encrypt") {
            return EncryptFile(key, iv, input, argv[5]);
        }
        if (command == "decrypt") {
            const std::vector<std::uint8_t> ciphertext =
                kuznyechik::BytesFromHex(ToText(input));
            return DecryptFile(key, iv, ciphertext, argv[5]);
        }

        PrintUsage();
        return 2;
    } catch (const std::exception& error) {
        std::cerr << "error: " << error.what() << '\n';
        return 2;
    }
}
