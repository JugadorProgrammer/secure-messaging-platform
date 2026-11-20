"""Демонстрация, самопроверка и замеры RSA без запуска сервиса SMP."""

import argparse
import json
import platform
import sys
import unittest
from pathlib import Path
from time import perf_counter

from rsa_core.arithmetic import Cancelled
from rsa_core.key_io import hex_to_int, int_to_hex, load_json_key
from rsa_core.rsa import (
    PrivateKey, generate_key_pair, make_private_key, rsadp_crt, rsadp_reference,
    rsaep, rsasp1, rsavp1,
)

ROOT = Path(__file__).resolve().parent
KEY_SIZES = (2048, 4096, 32768)


def _vectors() -> dict:
    return json.loads((ROOT / "test_vectors" / "rsa_demo.json").read_text(encoding="utf-8"))


def _hex(value: int) -> str:
    return "0x" + int_to_hex(value)


def _example(case_id: str, operation: str, value: int, expected: int, actual: int) -> bool:
    ok = expected == actual
    print(f"{case_id} {operation}: вход {_hex(value)}, ожидалось {_hex(expected)}, "
          f"получено {_hex(actual)} [{'OK' if ok else 'FAIL'}]")
    return ok


def _demo(*, full: bool = False) -> bool:
    vectors = _vectors()
    key = make_private_key(**vectors["key"])
    public = key.public_key()
    print("Демонстрация на фиксированном учебном ключе. Значения показаны в hex.")
    checks = []
    for example in vectors["encryption"]:
        if not full and example["id"] != "RSA-03":
            continue
        m, c, case_id = example["m"], example["c"], example["id"]
        checks.extend((
            _example(case_id, "RSAEP", m, c, rsaep(public, m)),
            _example(case_id, "RSADP", c, m, rsadp_reference(key, c)),
            _example(case_id, "RSADP CRT", c, m, rsadp_crt(key, c)),
        ))
    example = vectors["signature"]
    m, s, case_id = example["representative"], example["signature"], example["id"]
    checks.extend((
        _example(case_id, "RSASP1", m, s, rsasp1(key, m)),
        _example(case_id, "RSAVP1", s, m, rsavp1(public, s)),
    ))
    print(f"Контрольные операции: {sum(checks)}/{len(checks)} совпали.", flush=True)
    return all(checks)


def _test() -> bool:
    vectors_ok = _demo(full=True)
    suite = unittest.defaultTestLoader.discover(str(ROOT / "tests"))
    result = unittest.TextTestRunner(verbosity=2).run(suite)
    return vectors_ok and result.wasSuccessful() and result.testsRun > 0


def _peak_rss_mib() -> float | None:
    if sys.platform in ("linux", "darwin"):
        import resource

        divisor = 1024 ** 2 if sys.platform == "darwin" else 1024
        return resource.getrusage(resource.RUSAGE_SELF).ru_maxrss / divisor
    if sys.platform == "win32":
        import ctypes
        from ctypes import wintypes

        class ProcessMemoryCounters(ctypes.Structure):
            _fields_ = [("size", wintypes.DWORD), ("page_faults", wintypes.DWORD)] + [
                (name, ctypes.c_size_t) for name in (
                    "peak_working_set", "working_set", "peak_paged_pool", "paged_pool",
                    "peak_nonpaged_pool", "nonpaged_pool", "pagefile", "peak_pagefile",
                )
            ]

        counters = ProcessMemoryCounters()
        counters.size = ctypes.sizeof(counters)
        get_process = ctypes.WinDLL("kernel32").GetCurrentProcess
        get_process.argtypes = []
        get_process.restype = wintypes.HANDLE
        get_memory = ctypes.WinDLL("psapi").GetProcessMemoryInfo
        get_memory.argtypes = [wintypes.HANDLE, ctypes.POINTER(ProcessMemoryCounters), wintypes.DWORD]
        get_memory.restype = wintypes.BOOL
        if get_memory(get_process(), ctypes.byref(counters), counters.size):
            return counters.peak_working_set / 1024 ** 2
    return None


def _timed(operation, *args):
    start = perf_counter()
    value = operation(*args)
    return value, perf_counter() - start


def _check_operations(key: PrivateKey, *, progress: bool = False) -> dict:
    public = key.public_key()
    message = 65 if key.n > 65 else 2
    times = {}

    def run(label, operation, *args):
        if progress:
            print(f"Выполняется {label}…", file=sys.stderr, flush=True)
        value, times[label] = _timed(operation, *args)
        return value

    ciphertext = run("rsaep", rsaep, public, message)
    reference = run("rsadp_reference", rsadp_reference, key, ciphertext)
    crt = run("rsadp_crt", rsadp_crt, key, ciphertext)
    signature = run("rsasp1", rsasp1, key, message)
    restored = run("rsavp1", rsavp1, public, signature)
    if reference != message or crt != message or restored != message:
        raise ValueError("Контрольные RSA-операции не восстановили исходное число")
    return times


def _load_from_args(args):
    source = "generation" if getattr(args, "generate", False) else "json"
    keygen_time = load_time = None
    if source == "generation":
        print(f"Генерация ключа {args.bits} бит; Ctrl+C для отмены.", file=sys.stderr, flush=True)
        key, keygen_time = _timed(generate_key_pair, args.bits)
        check = "miller_rabin_during_generation"
    else:
        if args.check_primes:
            print("Повторная проверка простоты; Ctrl+C для отмены.", file=sys.stderr, flush=True)
        key, load_time = _timed(lambda: load_json_key(args.key_dir, recheck_primes=args.check_primes))
        check = "miller_rabin_after_import" if args.check_primes else "parameter_relations_only"
    return key, {"key_source": source, "key_check": check,
                 "generation_seconds": keygen_time, "load_seconds": load_time}


def _menu_key(key: PrivateKey | None) -> PrivateKey | None:
    print("1 — учебный ключ; 2 — загрузить JSON; 3 — сгенерировать ключ.")
    choice = input("Режим (Enter — назад): ").strip()
    if choice == "1":
        return make_private_key(**_vectors()["key"])
    if choice == "2":
        folder = input("Каталог с четырьмя JSON (Enter — назад): ").strip()
        return load_json_key(folder) if folder else key
    if choice == "3":
        bits = input(f"Разрядность {KEY_SIZES} (Enter — назад): ").strip()
        if not bits:
            return key
        if bits not in tuple(map(str, KEY_SIZES)):
            raise ValueError("Укажите разрядность 2048, 4096 или 32768")
        print("Генерация ключа; Ctrl+C для отмены.", flush=True)
        return generate_key_pair(int(bits))
    if choice:
        raise ValueError("Неизвестный режим выбора ключа")
    return key


def _menu() -> int:
    key = None
    while True:
        print("\n1 — ключи; 2 — шифрование; 3 — расшифрование; 4 — подпись; "
              "5 — проверка подписи; 6 — тесты; 0 — выход.")
        choice = input("Действие: ").strip()
        if choice == "0":
            return 0
        try:
            if choice == "1":
                key = _menu_key(key)
                if key is not None:
                    print(f"Выбран ключ: {key.n.bit_length()} бит.")
            elif choice == "6":
                _test()
            elif choice in ("2", "3", "4", "5"):
                if key is None:
                    raise ValueError("Сначала выберите ключ (пункт 1)")
                label = {"2": "Сообщение", "3": "Шифртекст", "4": "Представитель", "5": "Подпись"}[choice]
                entry = input(f"{label} в hex (Enter — назад): ").strip()
                if not entry:
                    continue
                value = hex_to_int(entry)
                if choice == "2":
                    print("Шифртекст:", _hex(rsaep(key.public_key(), value)))
                elif choice == "3":
                    first, second = rsadp_reference(key, value), rsadp_crt(key, value)
                    if first != second:
                        raise ValueError("Результаты обычного расшифрования и CRT разошлись")
                    print(f"Обычное: {_hex(first)}; CRT: {_hex(second)}")
                elif choice == "4":
                    print("Подпись:", _hex(rsasp1(key, value)))
                else:
                    entry = input("Ожидаемый представитель в hex (Enter — назад): ").strip()
                    if not entry:
                        continue
                    expected = hex_to_int(entry)
                    if expected >= key.n:
                        raise ValueError("Ожидаемый представитель должен быть меньше n")
                    actual = rsavp1(key.public_key(), value)
                    print("Подпись верна" if actual == expected else "Подпись не прошла проверку")
            elif choice:
                raise ValueError("Неизвестное действие")
        except (ValueError, OSError) as error:
            print(f"Ошибка: {error}", file=sys.stderr)
        except (KeyboardInterrupt, Cancelled):
            print("\nОперация отменена.", file=sys.stderr)


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    actions = parser.add_subparsers(dest="action")
    actions.add_parser("demo", help="показать учебный пример")
    actions.add_parser("test", help="выполнить автоматические проверки")
    check = actions.add_parser("key-check", help="проверить операции на готовом ключе")
    check.add_argument("--key-dir", type=Path, required=True)
    bench = actions.add_parser("bench", help="измерить время операций")
    source = bench.add_mutually_exclusive_group(required=True)
    source.add_argument("--key-dir", type=Path)
    source.add_argument("--generate", action="store_true")
    bench.add_argument("--bits", type=int, choices=KEY_SIZES)
    for command in (check, bench):
        command.add_argument("--check-primes", action="store_true", help="повторно проверить p и q при импорте")
    args = parser.parse_args()
    if args.action == "bench":
        if args.generate and (args.bits is None or args.check_primes):
            bench.error("--generate требует --bits; --check-primes применяется только при загрузке JSON")
        if not args.generate and args.bits is not None:
            bench.error("--bits применяется только с --generate")

    try:
        if args.action is None:
            return _menu()
        if args.action == "demo":
            return int(not _demo())
        if args.action == "test":
            return int(not _test())
        key, report = _load_from_args(args)
        times = _check_operations(key, progress=True)
        if args.action == "key-check":
            primes = "выполнена" if args.check_primes else "не запрошена"
            print(f"Ключ: {key.n.bit_length()} бит. Связи параметров верны; проверка простоты: {primes}.")
            print("Обычное расшифрование, CRT и проверка подписи восстановили исходное число.")
        else:
            report.update({"bits": key.n.bit_length(), "operations_seconds": times,
                           "peak_rss_mib": _peak_rss_mib(), "python": platform.python_version(),
                           "platform": platform.platform()})
            print(json.dumps(report, indent=2))
    except (ValueError, OSError) as error:
        print(f"Ошибка: {error}", file=sys.stderr)
        return 1
    except (KeyboardInterrupt, Cancelled):
        print("\nОперация отменена.", file=sys.stderr)
        return 130
    except EOFError:
        return 0
    return 0


if __name__ == "__main__":
    sys.exit(main())
