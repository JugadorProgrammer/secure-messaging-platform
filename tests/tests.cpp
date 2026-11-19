#include <cstdio>
#include <cstring>
#include <stdexcept>
#include <string>
#include <vector>

#include "block_cipher.h"
#include "cbc_mode.h"
#include "hex.h"
#include "kuz_tables.h"
#include "kuznyechik.h"

namespace {

// Вектор из ГОСТ Р 34.12-2015, приложение А.1.
const char* const kKeyHex =
    "8899aabbccddeeff0011223344556677fedcba98765432100123456789abcdef";
const char* const kIvHex = "00000000000000000000000000000000";

// Раундовые константы C_1..C_8, пример 5.4 стандарта (RFC 7801).
const char* const kRoundConstants[8] = {
    "6ea276726c487ab85d27bd10dd849401", "dc87ece4d890f4b3ba4eb92079cbeb02",
    "b2259a96b4d88e0be7690430a44f7f03", "7bcd1b0b73e32ba5b79cb140f2551504",
    "156f6d791fab511deabb0c502fd18105", "a74af7efab73df160dd208608b9efe06",
    "c9e8819dc73ba5ae50f5b570561a6a07", "f6593616e6055689adfba18027aa2a08",
};

// Раундовые ключи K_1..K_10 для того же ключа, пример 5.4.
const char* const kRoundKeys[10] = {
    "8899aabbccddeeff0011223344556677", "fedcba98765432100123456789abcdef",
    "db31485315694343228d6aef8cc78c44", "3d4553d8e9cfec6815ebadc40a9ffd04",
    "57646468c44a5e28d3e59246f429f1ac", "bd079435165c6432b532e82834da581b",
    "51e640757e8745de705727265a0098b1", "5a7925017b9fdd3ed72a91a22286f984",
    "bb44e25378c73123a5f32f73cdb6e517", "72e9dd7416bcf45b755dbaa88e4a4043",
};

int g_checks = 0;
int g_failures = 0;

// Единая строка успеха на весь проект: так договорились в ПЗ № 4.
const char* const kSuccessMessage = "All checks passed";

void Check(bool condition, const std::string& name) {
    ++g_checks;
    if (!condition) {
        ++g_failures;
        std::printf("FAILED: %s\n", name.c_str());
    }
}

void CheckHex(const std::uint8_t* data, std::size_t size, const std::string& expected,
              const std::string& name) {
    const std::vector<std::uint8_t> bytes(data, data + size);
    const std::string actual = kuznyechik::HexFromBytes(bytes);
    ++g_checks;
    if (actual != expected) {
        ++g_failures;
        std::printf("FAILED: %s\n  got  %s\n  want %s\n", name.c_str(), actual.c_str(),
                    expected.c_str());
    }
}

// Сравнивает код возврата и печатает, что пришло вместо ожидаемого.
void CheckStatus(KuzStatus actual, KuzStatus expected, const std::string& name) {
    ++g_checks;
    if (actual != expected) {
        ++g_failures;
        std::printf("FAILED: %s\n  got  %s\n  want %s\n", name.c_str(),
                    KuzStatusMessage(actual), KuzStatusMessage(expected));
    }
}

// Возвращает true, если действие бросило исключение.
template <typename Action>
bool Throws(Action action) {
    try {
        action();
    } catch (const std::exception&) {
        return true;
    }
    return false;
}

void TestHex() {
    const std::vector<std::uint8_t> bytes = kuznyechik::BytesFromHex("00ff10");
    Check(bytes.size() == 3, "hex: length");
    Check(bytes[0] == 0 && bytes[1] == 255 && bytes[2] == 16, "hex: values");
    Check(kuznyechik::HexFromBytes({0xAB, 0x01}) == "ab01", "hex: lowercase output");
    Check(Throws([] { kuznyechik::BytesFromHex("0"); }), "hex: odd length throws");
    Check(Throws([] { kuznyechik::BytesFromHex("zz"); }), "hex: non-hex throws");
}

void TestSubstitution() {
    const std::vector<std::uint8_t> input =
        kuznyechik::BytesFromHex("ffeeddccbbaa99881122334455667700");
    std::uint8_t output[16];
    kuznyechik::Substitute(input.data(), output);
    CheckHex(output, 16, "b66cd8887d38e8d77765aeea0c9a7efc", "S: vector from RFC 7801");

    bool consistent = true;
    for (int value = 0; value < 256; ++value) {
        if (kuznyechik::kSboxInverse[kuznyechik::kSbox[value]] != value) {
            consistent = false;
        }
    }
    Check(consistent, "S: Pi and Pi^-1 are mutually inverse");
}

void TestLinearTransform() {
    std::uint8_t block[16];
    std::uint8_t next[16];

    // Пример 5.2: четыре шага R подряд.
    std::memcpy(block, kuznyechik::BytesFromHex("00000000000000000000000000000100").data(), 16);
    kuznyechik::R(block, next);
    CheckHex(next, 16, "94000000000000000000000000000001", "R: step 1");

    std::memcpy(block, next, 16);
    kuznyechik::R(block, next);
    CheckHex(next, 16, "a5940000000000000000000000000000", "R: step 2");

    std::memcpy(block, next, 16);
    kuznyechik::R(block, next);
    CheckHex(next, 16, "64a59400000000000000000000000000", "R: step 3");

    std::memcpy(block, next, 16);
    kuznyechik::R(block, next);
    CheckHex(next, 16, "0d64a594000000000000000000000000", "R: step 4");

    // Пример 5.3: L от последнего значения.
    std::memcpy(block, kuznyechik::BytesFromHex("64a59400000000000000000000000000").data(), 16);
    kuznyechik::L(block, next);
    CheckHex(next, 16, "d456584dd0e3e84cc3166e4b7fa2890d", "L: step 1");

    std::memcpy(block, next, 16);
    kuznyechik::L(block, next);
    CheckHex(next, 16, "79d26221b87b584cd42fbc4ffea5de9a", "L: step 2");

    std::memcpy(block, next, 16);
    kuznyechik::L(block, next);
    CheckHex(next, 16, "0e93691a0cfc60408b7b68f66b513c13", "L: step 3");

    std::memcpy(block, next, 16);
    kuznyechik::L(block, next);
    CheckHex(next, 16, "e6a8094fee0aa204fd97bcb0b44b8580", "L: step 4");
}

void TestInverseLinearTransform() {
    // Пример 5.6: L^-1 от состояния после X[K_10].
    const std::vector<std::uint8_t> state =
        kuznyechik::BytesFromHex("0d8e40e4a800d06b2f1b37ea379ead8e");
    std::uint8_t back[16];
    kuznyechik::LInverse(state.data(), back);
    CheckHex(back, 16, "8a6b930a52211b45c5baa43ff8b91319", "L^-1: vector from RFC 7801");

    std::uint8_t round_trip[16];
    kuznyechik::LInverse(back, round_trip);
    kuznyechik::L(round_trip, round_trip);
    CheckHex(round_trip, 16, "8a6b930a52211b45c5baa43ff8b91319", "L after L^-1 returns the input");
}

void TestRoundConstants() {
    for (int index = 1; index <= 8; ++index) {
        std::uint8_t constant[16];
        kuznyechik::RoundConstant(index, constant);
        CheckHex(constant, 16, kRoundConstants[index - 1],
                 "C_" + std::to_string(index) + " from RFC 7801");
    }
}

void TestKeySchedule() {
    const std::vector<std::uint8_t> key = kuznyechik::BytesFromHex(kKeyHex);
    std::uint8_t round_keys[10][16];
    kuznyechik::ExpandKey(key.data(), round_keys);

    for (int index = 0; index < 10; ++index) {
        CheckHex(round_keys[index], 16, kRoundKeys[index],
                 "K_" + std::to_string(index + 1) + " from RFC 7801");
    }
}

void TestBlockEncryption() {
    const std::vector<std::uint8_t> key = kuznyechik::BytesFromHex(kKeyHex);
    const std::vector<std::uint8_t> plaintext =
        kuznyechik::BytesFromHex("1122334455667700ffeeddccbbaa9988");

    const kuznyechik::BlockCipher cipher(key.data());
    std::uint8_t ciphertext[16];
    cipher.EncryptBlock(plaintext.data(), ciphertext);
    CheckHex(ciphertext, 16, "7f679d90bebc24305a468d42b9d4edcd",
             "encryption of the test block");
}

void TestBlockDecryption() {
    const std::vector<std::uint8_t> key = kuznyechik::BytesFromHex(kKeyHex);
    const std::vector<std::uint8_t> ciphertext =
        kuznyechik::BytesFromHex("7f679d90bebc24305a468d42b9d4edcd");

    const kuznyechik::BlockCipher cipher(key.data());
    std::uint8_t plaintext[16];
    cipher.DecryptBlock(ciphertext.data(), plaintext);
    CheckHex(plaintext, 16, "1122334455667700ffeeddccbbaa9988",
             "decryption of the test block");
}

void TestPadding() {
    const std::vector<std::uint8_t> empty;
    const std::vector<std::uint8_t> padded = kuznyechik::AddPadding(empty);
    Check(padded.size() == 16, "padding: empty input becomes one block");
    Check(padded[0] == 0x10 && padded[15] == 0x10, "padding: empty input filled with 0x10");

    const std::vector<std::uint8_t> fifteen(15, 0xAA);
    const std::vector<std::uint8_t> padded_fifteen = kuznyechik::AddPadding(fifteen);
    Check(padded_fifteen.size() == 16, "padding: 15 bytes become one block");
    Check(padded_fifteen[15] == 0x01, "padding: 15 bytes add one 0x01");

    std::vector<std::uint8_t> round_trip = padded;
    kuznyechik::RemovePadding(round_trip);
    Check(round_trip.empty(), "padding: remove restores empty input");

    std::vector<std::uint8_t> broken(16, 0);
    Check(Throws([&broken] { kuznyechik::RemovePadding(broken); }),
          "padding: zero length rejected");

    broken.assign(16, 0x11);
    Check(Throws([&broken] { kuznyechik::RemovePadding(broken); }),
          "padding: 0x11 length rejected");

    broken.assign(16, 0x02);
    broken[14] = 0x03;
    Check(Throws([&broken] { kuznyechik::RemovePadding(broken); }),
          "padding: inconsistent bytes rejected");
}

void TestCiphertextSize() {
    Check(KuzCiphertextSize(0) == 16, "size: 0 bytes -> 16");
    Check(KuzCiphertextSize(1) == 16, "size: 1 byte -> 16");
    Check(KuzCiphertextSize(15) == 16, "size: 15 bytes -> 16");
    Check(KuzCiphertextSize(16) == 32, "size: 16 bytes -> 32");
    Check(KuzCiphertextSize(17) == 32, "size: 17 bytes -> 32");
}

void TestCbcRoundTrip() {
    const std::vector<std::uint8_t> key = kuznyechik::BytesFromHex(kKeyHex);
    const std::vector<std::uint8_t> iv = kuznyechik::BytesFromHex(kIvHex);

    for (const std::size_t length : {0, 1, 15, 16, 17, 31, 32, 33, 1000}) {
        std::vector<std::uint8_t> data(length);
        for (std::size_t index = 0; index < length; ++index) {
            data[index] = static_cast<std::uint8_t>(index * 7);
        }

        const std::vector<std::uint8_t> ciphertext = kuznyechik::EncryptCbc(key, iv, data);
        const std::string tail = std::to_string(length);
        Check(ciphertext.size() == length + (16 - length % 16), "cbc: size " + tail);

        const std::vector<std::uint8_t> back = kuznyechik::DecryptCbc(key, iv, ciphertext);
        Check(back == data, "cbc: round trip " + tail);
    }
}

void TestApiArguments() {
    const std::vector<std::uint8_t> key = kuznyechik::BytesFromHex(kKeyHex);
    const std::vector<std::uint8_t> iv = kuznyechik::BytesFromHex(kIvHex);
    const std::vector<std::uint8_t> data = kuznyechik::BytesFromHex("00112233");

    std::uint8_t output[64];
    std::size_t written = 0;

    CheckStatus(KuzEncrypt(nullptr, 32, iv.data(), 16, data.data(), data.size(), output, 64,
                           &written),
                KUZ_STATUS_INVALID_ARGUMENT, "api: null key");
    CheckStatus(KuzEncrypt(key.data(), 31, iv.data(), 16, data.data(), data.size(), output, 64,
                           &written),
                KUZ_STATUS_INVALID_KEY, "api: wrong key size");
    CheckStatus(KuzEncrypt(key.data(), 32, iv.data(), 15, data.data(), data.size(), output, 64,
                           &written),
                KUZ_STATUS_INVALID_IV, "api: wrong iv size");
    CheckStatus(KuzEncrypt(key.data(), 32, iv.data(), 16, data.data(), data.size(), output, 64,
                           nullptr),
                KUZ_STATUS_INVALID_ARGUMENT, "api: null out size");
    CheckStatus(KuzEncrypt(key.data(), 32, iv.data(), 16, data.data(), data.size(), output, 4,
                           &written),
                KUZ_STATUS_BUFFER_TOO_SMALL, "api: small buffer");
    Check(written == 16, "api: required size is reported");
    CheckStatus(KuzEncrypt(key.data(), 32, iv.data(), 16, data.data(), data.size(), nullptr, 64,
                           &written),
                KUZ_STATUS_INVALID_ARGUMENT, "api: null output");

    CheckStatus(KuzEncrypt(key.data(), 32, iv.data(), 16, data.data(), data.size(), output, 64,
                           &written),
                KUZ_STATUS_OK, "api: encryption succeeds");
    Check(written == 16, "api: ciphertext takes one block");

    std::uint8_t plaintext[64];
    std::size_t plain_size = 0;
    CheckStatus(KuzDecrypt(key.data(), 32, iv.data(), 16, output, 15, plaintext, 64, &plain_size),
                KUZ_STATUS_INVALID_ARGUMENT, "api: ciphertext is not a block multiple");
    CheckStatus(KuzDecrypt(key.data(), 32, iv.data(), 16, output, 16, plaintext, 64, &plain_size),
                KUZ_STATUS_OK, "api: decryption succeeds");
    Check(plain_size == data.size() && std::memcmp(plaintext, data.data(), data.size()) == 0,
          "api: decrypted bytes match the input");

    // Пустой ввод: шифртекст — один блок дополнения, расшифровка пуста.
    CheckStatus(KuzEncrypt(key.data(), 32, iv.data(), 16, nullptr, 0, output, 64, &written),
                KUZ_STATUS_OK, "api: empty input encrypts");
    Check(written == 16, "api: empty input gives one block");
    CheckStatus(KuzDecrypt(key.data(), 32, iv.data(), 16, output, written, plaintext, 64,
                          &plain_size),
                KUZ_STATUS_OK, "api: empty input decrypts");
    Check(plain_size == 0, "api: empty plaintext has zero length");
}

void TestApiInvalidPadding() {
    const std::vector<std::uint8_t> key = kuznyechik::BytesFromHex(kKeyHex);
    const std::vector<std::uint8_t> iv(16, 0);

    // C2 = E(0), C1 = 0 => P2 = D(C2) xor C1 = 0: последний байт нулевой,
    // значит дополнение неверно и должно вернуться INVALID_PADDING.
    std::uint8_t zero[16] = {0};
    std::uint8_t encrypted_zero[16];
    const kuznyechik::BlockCipher cipher(key.data());
    cipher.EncryptBlock(zero, encrypted_zero);

    std::vector<std::uint8_t> ciphertext(32, 0);
    std::memcpy(&ciphertext[16], encrypted_zero, 16);

    std::uint8_t plaintext[32];
    std::size_t plain_size = 0;
    CheckStatus(KuzDecrypt(key.data(), 32, iv.data(), 16, ciphertext.data(), ciphertext.size(),
                           plaintext, 32, &plain_size),
                KUZ_STATUS_INVALID_PADDING, "api: invalid padding is reported");
}

void TestVersionAndMessages() {
    CheckStatus(KuzSelfTest(), KUZ_STATUS_OK, "selftest");
    Check(std::string(KuzVersion()) == "1.0.0", "version string");

    for (const KuzStatus status :
         {KUZ_STATUS_OK, KUZ_STATUS_INVALID_KEY, KUZ_STATUS_INVALID_IV,
          KUZ_STATUS_INVALID_ARGUMENT, KUZ_STATUS_BUFFER_TOO_SMALL,
          KUZ_STATUS_INVALID_PADDING, KUZ_STATUS_INTERNAL_ERROR}) {
        Check(KuzStatusMessage(status) != nullptr, "status message is not null");
    }
}

}  // namespace

int main() {
    TestHex();
    TestSubstitution();
    TestLinearTransform();
    TestInverseLinearTransform();
    TestRoundConstants();
    TestKeySchedule();
    TestBlockEncryption();
    TestBlockDecryption();
    TestPadding();
    TestCiphertextSize();
    TestCbcRoundTrip();
    TestApiArguments();
    TestApiInvalidPadding();
    TestVersionAndMessages();

    if (g_failures != 0) {
        std::printf("%d of %d checks failed\n", g_failures, g_checks);
        return 1;
    }
    std::printf("%s\n", kSuccessMessage);
    return 0;
}
