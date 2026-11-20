"""Импорт учебной пары ключей из четырёх JSON-файлов (#146)."""

import json
import re
from pathlib import Path

from .arithmetic import Cancel, check_cancel
from .rsa import PrivateKey, make_private_key, validate_key

_HEX = re.compile(r"(?:0[xX])?[0-9a-fA-F]{1,16384}")
_MAX_BYTES = 128 * 1024


def hex_to_int(value: str) -> int:
    if not isinstance(value, str) or not _HEX.fullmatch(value.strip()):
        raise ValueError("Ожидается hex-строка без знака и внутренних пробелов")
    return int(value.strip(), 16)


def int_to_hex(value: int) -> str:
    if type(value) is not int or value < 0:
        raise ValueError("Ожидается неотрицательное целое число")
    return format(value, "x")


def _unique_fields(pairs):
    result = {}
    for name, value in pairs:
        if name in result:
            raise ValueError("Повторяющееся поле JSON")
        result[name] = value
    return result


def _reject_constant(_token):
    raise ValueError("Недопустимая константа JSON")


def _read(path: Path) -> dict:
    try:
        with path.open("rb") as source:
            raw = source.read(_MAX_BYTES + 1)
    except OSError:
        raise ValueError(f"Не удалось прочитать {path.name}") from None
    if len(raw) > _MAX_BYTES:
        raise ValueError(f"Файл {path.name} превышает 128 КиБ")
    try:
        parsed = json.loads(raw.decode("utf-8-sig"), object_pairs_hook=_unique_fields,
                            parse_constant=_reject_constant)
    except (ValueError, RecursionError):
        raise ValueError(f"Некорректный JSON в {path.name}") from None
    if not isinstance(parsed, dict):
        raise ValueError(f"В файле {path.name} ожидается JSON-объект")
    return parsed


def _field(record: dict, name: str, filename: str) -> int:
    if name not in record:
        raise ValueError(f"В {filename} отсутствует поле {name}")
    try:
        return hex_to_int(record[name])
    except ValueError:
        raise ValueError(f"Поле {name} в {filename} должно быть строкой hex") from None


def load_json_key(directory: str | Path, *, recheck_primes: bool = False,
                  cancel: Cancel = None) -> PrivateKey:
    """Читает согласованный комплект; закрытые значения в ошибки не включаются."""
    check_cancel(cancel)
    folder = Path(directory)
    private_name, public_name = "private_key.json", "public_key.json"
    private, public = _read(folder / private_name), _read(folder / public_name)
    n, e, d, p, q = (_field(private, f, private_name) for f in ("n", "e", "d", "p", "q"))
    if (n, e) != tuple(_field(public, f, public_name) for f in ("n", "e")):
        raise ValueError("Параметры n или e расходятся между открытым и закрытым ключом")
    for name, expected in (("p.json", p), ("q.json", q)):
        if _field(_read(folder / name), "prime", name) != expected:
            raise ValueError(f"Множитель из {name} не совпадает с закрытым ключом")

    key = make_private_key(n, e, d, p, q)
    for name, expected in (("dP", key.d_p), ("dQ", key.d_q), ("qInv", key.q_inv)):
        if name in private and _field(private, name, private_name) != expected:
            raise ValueError(f"Сохранённый параметр {name} не совпадает с вычисленным")
    if recheck_primes:
        validate_key(key, recheck_primes=True, cancel=cancel)
    check_cancel(cancel)
    return key
