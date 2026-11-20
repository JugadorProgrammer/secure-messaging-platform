"""Модель ключа, генерация и числовые примитивы RSA (#146–#150)."""

import secrets
from dataclasses import dataclass

from .arithmetic import Cancel, check_cancel, gcd, lcm, mod_inverse, mod_pow
from .primes import MR_ROUNDS, PUBLIC_EXPONENT, RandomBits, generate_prime, is_probable_prime


@dataclass(frozen=True)
class PublicKey:
    n: int
    e: int


@dataclass(frozen=True, repr=False)
class PrivateKey:
    n: int
    e: int
    d: int
    p: int
    q: int
    d_p: int
    d_q: int
    q_inv: int

    def public_key(self) -> PublicKey:
        return PublicKey(self.n, self.e)


def _validate_public(key: PublicKey) -> None:
    if (type(key.n) is not int or type(key.e) is not int or key.n <= 3 or key.n % 2 == 0
            or not 3 <= key.e < key.n or key.e % 2 == 0):
        raise ValueError("Некорректные параметры открытого ключа")


def _validate_factors(p: int, q: int) -> None:
    if (type(p) is not int or type(q) is not int or min(p, q) <= 2
            or p == q or p % 2 == 0 or q % 2 == 0):
        raise ValueError("Множители p и q должны быть различными нечётными целыми больше 2")


def validate_key(key: PrivateKey, *, recheck_primes: bool = False, cancel: Cancel = None) -> None:
    """Проверяет связи параметров; тест простоты выполняется только по запросу."""
    check_cancel(cancel)
    _validate_public(key.public_key())
    p, q = key.p, key.q
    _validate_factors(p, q)
    if key.n != p * q or type(key.d) is not int or not 0 < key.d < key.n:
        raise ValueError("Не выполняется n = p × q или d вне допустимого диапазона")
    lam = lcm(p - 1, q - 1)
    if gcd(key.e, lam) != 1 or key.e * key.d % lam != 1:
        raise ValueError("Не выполняется e × d ≡ 1 по модулю λ(n)")
    if (any(type(x) is not int for x in (key.d_p, key.d_q, key.q_inv))
            or key.d_p != key.d % (p - 1) or key.d_q != key.d % (q - 1)
            or not 0 < key.q_inv < p or q * key.q_inv % p != 1):
        raise ValueError("Параметры CRT не согласованы")
    if recheck_primes and not (is_probable_prime(p, cancel=cancel) and is_probable_prime(q, cancel=cancel)):
        raise ValueError("Множители не прошли проверку Миллера — Рабина")


def make_private_key(n: int, e: int, d: int, p: int, q: int) -> PrivateKey:
    """Собирает ключ и вычисляет dP, dQ и qInv."""
    _validate_factors(p, q)
    if type(d) is not int:
        raise ValueError("Закрытый показатель d должен быть целым")
    key = PrivateKey(n, e, d, p, q, d % (p - 1), d % (q - 1), mod_inverse(q, p))
    validate_key(key)
    return key


def _check_representative(value: int, n: int) -> None:
    if type(value) is not int or not 0 <= value < n:
        raise ValueError("Требуется целое число x в диапазоне 0 ≤ x < n")


def rsaep(key: PublicKey, message: int, *, cancel: Cancel = None) -> int:
    """Открытая операция: c = m^e mod n."""
    _validate_public(key)
    _check_representative(message, key.n)
    return mod_pow(message, key.e, key.n, cancel=cancel)


def rsadp_reference(key: PrivateKey, ciphertext: int, *, cancel: Cancel = None) -> int:
    """Обычная закрытая операция: m = c^d mod n."""
    validate_key(key, cancel=cancel)
    _check_representative(ciphertext, key.n)
    return mod_pow(ciphertext, key.d, key.n, cancel=cancel)


def _crt_value(key: PrivateKey, x: int, cancel: Cancel) -> int:
    m1 = mod_pow(x, key.d_p, key.p, cancel=cancel)
    m2 = mod_pow(x, key.d_q, key.q, cancel=cancel)
    h = key.q_inv * (m1 - m2) % key.p
    return m2 + key.q * h


def rsadp_crt(key: PrivateKey, ciphertext: int, *, cancel: Cancel = None) -> int:
    """Закрытая операция через CRT с проверкой результата открытым ключом."""
    validate_key(key, cancel=cancel)
    _check_representative(ciphertext, key.n)
    value = _crt_value(key, ciphertext, cancel)
    if not 0 <= value < key.n or rsaep(key.public_key(), value, cancel=cancel) != ciphertext:
        raise ValueError("Контроль результата CRT не пройден")
    return value


def rsasp1(key: PrivateKey, representative: int, *, cancel: Cancel = None) -> int:
    """Примитив подписи заранее подготовленного целого представителя."""
    return rsadp_crt(key, representative, cancel=cancel)


def rsavp1(key: PublicKey, signature: int, *, cancel: Cancel = None) -> int:
    """Восстанавливает целый представитель для сравнения проверяющим."""
    return rsaep(key, signature, cancel=cancel)


def generate_key_pair(bits: int, *, rounds: int = MR_ROUNDS, rng: RandomBits = secrets.randbits,
                      cancel: Cancel = None) -> PrivateKey:
    """Генерирует ключ точной разрядности; малые размеры нужны для тестов."""
    if type(bits) is not int or bits % 2 or not 32 <= bits <= 32768:
        raise ValueError("Разрядность должна быть чётным числом от 32 до 32768")
    if type(rounds) is not int or rounds < 1:
        raise ValueError("Число раундов должно быть положительным")
    while True:
        check_cancel(cancel)
        p = generate_prime(bits // 2, rounds, rng, cancel=cancel)
        q = generate_prime(bits // 2, rounds, rng, cancel=cancel)
        n = p * q
        if p != q and n.bit_length() == bits:
            break
    e = PUBLIC_EXPONENT
    key = make_private_key(n, e, mod_inverse(e, lcm(p - 1, q - 1)), p, q)
    for message in (2, n - 2):
        ciphertext = rsaep(key.public_key(), message, cancel=cancel)
        if rsadp_crt(key, ciphertext, cancel=cancel) != message:
            raise ValueError("Контрольная операция с новым ключом не прошла")
    check_cancel(cancel)
    return key
