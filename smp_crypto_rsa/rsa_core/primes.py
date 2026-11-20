"""Случайные кандидаты и проверка простоты для CRYPTO-15 (#149)."""

import secrets
from collections.abc import Callable
from math import isqrt

from .arithmetic import Cancel, check_cancel, gcd, mod_pow

MR_ROUNDS = 64
PUBLIC_EXPONENT = 65537
RandomBits = Callable[[int], int]


def _small_primes(limit: int) -> tuple[int, ...]:
    sieve = bytearray(b"\x01") * (limit + 1)
    sieve[:2] = b"\x00\x00"
    for p in range(2, isqrt(limit) + 1):
        if sieve[p]:
            for multiple in range(p * p, limit + 1, p):
                sieve[multiple] = 0
    return tuple(i for i, prime in enumerate(sieve) if prime)


_SMALL_PRIMES = _small_primes(10_000)


def _draw(bits: int, rng: RandomBits) -> int:
    try:
        value = rng(bits)
    except OSError:
        raise ValueError("Источник случайности недоступен") from None
    if type(value) is not int or value < 0 or value.bit_length() > bits:
        raise ValueError("Источник случайности вернул недопустимое значение")
    return value


def random_bits(bits: int, rng: RandomBits = secrets.randbits) -> int:
    """Нечётный кандидат с установленным старшим битом."""
    if type(bits) is not int or bits < 2:
        raise ValueError("Размер нечётного кандидата должен быть не меньше двух бит")
    return _draw(bits, rng) | (1 << (bits - 1)) | 1


def random_below(bound: int, rng: RandomBits = secrets.randbits, *, cancel: Cancel = None) -> int:
    """Равномерный выбор из [0, bound) через отбрасывание лишних значений."""
    if type(bound) is not int or bound < 1:
        raise ValueError("Верхняя граница должна быть положительным целым числом")
    check_cancel(cancel)
    if bound == 1:
        return 0
    bits = (bound - 1).bit_length()
    while True:
        check_cancel(cancel)
        value = _draw(bits, rng)
        if value < bound:
            return value


def is_probable_prime(n: int, rounds: int = MR_ROUNDS, rng: RandomBits = secrets.randbits,
                      *, cancel: Cancel = None) -> bool:
    """Отсев малыми простыми, затем раунды Миллера — Рабина."""
    if type(n) is not int or type(rounds) is not int or rounds < 1:
        raise ValueError("Ожидается целый кандидат и положительное число раундов")
    check_cancel(cancel)
    if n < 2:
        return False
    for prime in _SMALL_PRIMES:
        if prime * prime > n:
            return True
        if n % prime == 0:
            return n == prime

    # n - 1 = 2^s * d, где d нечётное.
    d, s = n - 1, 0
    while d % 2 == 0:
        d //= 2
        s += 1
    for _ in range(rounds):
        a = 2 + random_below(n - 3, rng, cancel=cancel)
        x = mod_pow(a, d, n, cancel=cancel)
        if x == 1 or x == n - 1:
            continue
        for _ in range(s - 1):
            check_cancel(cancel)
            x = x * x % n
            if x == n - 1:
                break
        else:
            return False
    return True


def generate_prime(bits: int, rounds: int = MR_ROUNDS, rng: RandomBits = secrets.randbits,
                   *, cancel: Cancel = None) -> int:
    """Ищет вероятно простой множитель, совместимый с e = 65537."""
    if type(rounds) is not int or rounds < 1:
        raise ValueError("Число раундов должно быть положительным")
    while True:
        check_cancel(cancel)
        candidate = random_bits(bits, rng)
        if gcd(PUBLIC_EXPONENT, candidate - 1) == 1 and is_probable_prime(
            candidate, rounds, rng, cancel=cancel
        ):
            check_cancel(cancel)
            return candidate
