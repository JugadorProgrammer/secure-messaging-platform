"""Целочисленная арифметика для задачи CRYPTO-11 (#145)."""

from collections.abc import Callable

Cancel = Callable[[], bool] | None


class Cancelled(Exception):
    """Вычисление прервано до выдачи результата."""


def check_cancel(cancel: Cancel) -> None:
    if cancel is not None and cancel():
        raise Cancelled("Операция отменена")


def _nonnegative(value: int, name: str) -> None:
    if type(value) is not int or value < 0:
        raise ValueError(f"{name}: ожидается неотрицательное целое число")


def gcd(a: int, b: int) -> int:
    """НОД неотрицательных целых по алгоритму Евклида; НОД(0, 0) = 0."""
    _nonnegative(a, "a")
    _nonnegative(b, "b")
    while b:
        a, b = b, a % b
    return a


def lcm(a: int, b: int) -> int:
    """НОК неотрицательных целых; если один аргумент нулевой, НОК равен 0."""
    divisor = gcd(a, b)
    return (a // divisor) * b if divisor else 0


def mod_inverse(a: int, modulus: int) -> int:
    """Обратное число по модулю > 1, вычисленное расширенным алгоритмом Евклида."""
    _nonnegative(a, "a")
    _nonnegative(modulus, "modulus")
    if modulus <= 1:
        raise ValueError("Модуль для обратного числа должен быть больше 1")

    r, next_r = modulus, a % modulus
    coefficient, next_coefficient = 0, 1
    while next_r:
        quotient, remainder = divmod(r, next_r)
        r, next_r = next_r, remainder
        coefficient, next_coefficient = next_coefficient, coefficient - quotient * next_coefficient

    if r != 1:
        raise ValueError("Обратного числа по указанному модулю не существует")
    return coefficient % modulus


class _Montgomery:
    """Операции над остатками в представлении xR mod n для нечётного n > 1."""

    def __init__(self, modulus: int) -> None:
        self.modulus = modulus
        self.shift = modulus.bit_length()
        self.radix = 1 << self.shift
        self.mask = self.radix - 1
        self.factor = (-mod_inverse(modulus, self.radix)) & self.mask

    def encode(self, value: int) -> int:
        return ((value % self.modulus) << self.shift) % self.modulus

    def reduce(self, value: int) -> int:
        # Младшие биты суммы обнуляются, поэтому деление на R — это сдвиг.
        correction = (value * self.factor) & self.mask
        result = (value + correction * self.modulus) >> self.shift
        return result - self.modulus if result >= self.modulus else result

    def multiply(self, a: int, b: int) -> int:
        return self.reduce(a * b)


def _binary_mod_pow(base: int, exponent: int, modulus: int, cancel: Cancel) -> int:
    result = 1
    base %= modulus
    while exponent:
        if cancel is not None:
            check_cancel(cancel)
        if exponent & 1:
            result = (result * base) % modulus
        base = (base * base) % modulus
        exponent >>= 1
    return result


def mod_pow(base: int, exponent: int, modulus: int, *, cancel: Cancel = None) -> int:
    """Степень по модулю: Монтгомери с окном 4 бита или двоичный метод."""
    _nonnegative(base, "base")
    _nonnegative(exponent, "exponent")
    _nonnegative(modulus, "modulus")
    if modulus == 0:
        raise ValueError("Модуль должен быть положительным")
    check_cancel(cancel)
    if modulus == 1:
        return 0
    if exponent == 0:
        return 1
    if modulus % 2 == 0:
        return _binary_mod_pow(base, exponent, modulus, cancel)

    mont = _Montgomery(modulus)
    one, base = mont.encode(1), mont.encode(base)
    powers = [one, base]
    for _ in range(2, 16):
        powers.append(mont.multiply(powers[-1], base))

    # Первая группа может содержать меньше четырёх бит; ведущие нули безопасны.
    first_shift = ((exponent.bit_length() - 1) // 4) * 4
    result = powers[exponent >> first_shift]
    for shift in range(first_shift - 4, -1, -4):
        if cancel is not None:
            check_cancel(cancel)
        for _ in range(4):
            result = mont.multiply(result, result)
        digit = (exponent >> shift) & 15
        if digit:
            result = mont.multiply(result, powers[digit])
    return mont.reduce(result)
