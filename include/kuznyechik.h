#ifndef KUZNYECHIK_H_
#define KUZNYECHIK_H_

/*
 * Публичный контракт DLL "Кузнечик".
 *
 * Блочный шифр по ГОСТ Р 34.12-2015 (блок 128 бит, ключ 256 бит)
 * в режиме CBC с дополнением PKCS#7.
 *
 * Единицы измерения — байты: ключ 32, IV 16, блок 16.
 * Память выделяет вызывающая сторона, библиотека только пишет в буфер.
 * Ни одно исключение не выходит за границу DLL: наружу отдаются коды.
 */

#include <stddef.h>
#include <stdint.h>

/*
 * KUZ_BUILD_DLL — определён при сборке самой библиотеки,
 * KUZ_STATIC    — для статической сборки (тесты, утилиты),
 * иначе          — обычное использование готовой DLL.
 */
#if defined(KUZ_STATIC)
#  define KUZ_API
#elif defined(_WIN32)
#  if defined(KUZ_BUILD_DLL)
#    define KUZ_API __declspec(dllexport)
#  else
#    define KUZ_API __declspec(dllimport)
#  endif
#else
#  define KUZ_API __attribute__((visibility("default")))
#endif

#ifdef __cplusplus
extern "C" {
#endif

/* Код возврата: 0 — успех, остальное — ошибка. */
enum KuzStatus {
    KUZ_STATUS_OK = 0,
    KUZ_STATUS_INVALID_KEY = 1,
    KUZ_STATUS_INVALID_IV = 2,
    KUZ_STATUS_INVALID_ARGUMENT = 3,
    KUZ_STATUS_BUFFER_TOO_SMALL = 4,
    KUZ_STATUS_INVALID_PADDING = 5,
    KUZ_STATUS_INTERNAL_ERROR = 6
};

/* Размеры в байтах. */
enum {
    KUZ_KEY_SIZE = 32,
    KUZ_IV_SIZE = 16,
    KUZ_BLOCK_SIZE = 16
};

/* Версия библиотеки, например "1.0.0". Строка живёт, пока загружена DLL. */
KUZ_API const char* KuzVersion(void);

/* Текст ошибки для кода возврата. */
KUZ_API const char* KuzStatusMessage(enum KuzStatus status);

/* Прогоняет встроенные векторы ГОСТ Р 34.12-2015 внутри библиотеки. */
KUZ_API enum KuzStatus KuzSelfTest(void);

/*
 * Сколько байт нужно под шифртекст для data_size байт открытого текста.
 * Возвращает 0, если размер слишком велик (переполнение).
 */
KUZ_API size_t KuzCiphertextSize(size_t data_size);

/*
 * Шифрует data_size байт в режиме CBC.
 *
 * key_size должен быть 32, iv_size — 16.
 * out — буфер вызывающей стороны на out_capacity байт.
 *
 * В *out_size всегда пишется нужный размер результата: при нехватке места
 * функция возвращает KUZ_STATUS_BUFFER_TOO_SMALL, и повторный вызов с буфером
 * нужного размера завершится успешно.
 */
KUZ_API enum KuzStatus KuzEncrypt(const uint8_t* key, size_t key_size,
                                  const uint8_t* iv, size_t iv_size,
                                  const uint8_t* data, size_t data_size,
                                  uint8_t* out, size_t out_capacity,
                                  size_t* out_size);

/*
 * Расшифровывает data_size байт шифртекста в режиме CBC.
 *
 * data_size должен быть положительным и кратным 16. Буфер out должен быть
 * не меньше data_size: открытый текст никогда не длиннее шифртекста.
 * Если дополнение после расшифровки неверно (чужой ключ, IV или испорченный
 * шифртекст), возвращается KUZ_STATUS_INVALID_PADDING.
 */
KUZ_API enum KuzStatus KuzDecrypt(const uint8_t* key, size_t key_size,
                                  const uint8_t* iv, size_t iv_size,
                                  const uint8_t* data, size_t data_size,
                                  uint8_t* out, size_t out_capacity,
                                  size_t* out_size);

#ifdef __cplusplus
}
#endif

#endif  // KUZNYECHIK_H_
