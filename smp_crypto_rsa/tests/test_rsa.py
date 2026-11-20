import json
import math
import random
import subprocess
import sys
import unittest
from dataclasses import replace
from pathlib import Path
from tempfile import TemporaryDirectory
from unittest.mock import Mock, patch

from rsa_core.arithmetic import Cancelled, mod_pow
from rsa_core.key_io import hex_to_int, int_to_hex, load_json_key
from rsa_core.primes import PUBLIC_EXPONENT, generate_prime, is_probable_prime, random_below, random_bits
from rsa_core.rsa import (
    PublicKey, generate_key_pair, make_private_key, rsadp_crt, rsadp_reference,
    rsaep, rsasp1, rsavp1, validate_key,
)

ROOT = Path(__file__).resolve().parents[1]
VECTORS = json.loads((ROOT / "test_vectors" / "rsa_demo.json").read_text(encoding="utf-8"))
TOY_KEY = make_private_key(**VECTORS["key"])


class RSAPrimitiveTests(unittest.TestCase):
    def test_known_ciphertexts_and_ranges(self):
        key = TOY_KEY
        for case in VECTORS["encryption"]:
            with self.subTest(case=case["id"]):
                self.assertEqual(rsaep(key.public_key(), case["m"]), case["c"])
                self.assertEqual(rsadp_reference(key, case["c"]), case["m"])
                self.assertEqual(rsadp_crt(key, case["c"]), case["m"])

        for func, given_key in ((rsaep, key.public_key()), (rsadp_reference, key),
                                (rsadp_crt, key), (rsasp1, key), (rsavp1, key.public_key())):
            for value in (-1, key.n, 0.0, True):
                with self.subTest(operation=func.__name__, value=value):
                    with self.assertRaises(ValueError):
                        func(given_key, value)
        with self.assertRaises(ValueError):
            rsaep(PublicKey(key.n, 2), 65)

    def test_crt_intermediates(self):
        key = TOY_KEY
        case = VECTORS["crt"]
        c = case["ciphertext"]
        m1 = mod_pow(c, key.d_p, key.p)
        m2 = mod_pow(c, key.d_q, key.q)
        h = key.q_inv * (m1 - m2) % key.p
        self.assertEqual((m1, m2, h), (case["m1"], case["m2"], case["h"]))
        self.assertEqual(rsadp_crt(key, c), case["m"])

    def test_wrong_key_and_faulty_crt_result_are_rejected(self):
        with self.assertRaisesRegex(ValueError, "CRT"):
            rsadp_crt(replace(TOY_KEY, d_p=TOY_KEY.d_p + 1), 2790)
        with self.assertRaises(ValueError):
            rsadp_reference(replace(TOY_KEY, d=2), 2790)
        for wrong in (66, TOY_KEY.n):
            with patch("rsa_core.rsa._crt_value", return_value=wrong):
                with self.assertRaisesRegex(ValueError, "Контроль результата CRT"):
                    rsadp_crt(TOY_KEY, 2790)

    def test_signature_primitive_known_vector(self):
        m = VECTORS["signature"]["representative"]
        s = VECTORS["signature"]["signature"]
        self.assertEqual(rsasp1(TOY_KEY, m), s)
        self.assertEqual(rsavp1(TOY_KEY.public_key(), s), m)
        self.assertNotEqual(rsavp1(TOY_KEY.public_key(), s + 1), m)


class KeyImportTests(unittest.TestCase):
    def setUp(self):
        temporary = TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        self.folder = Path(temporary.name)
        self._write_key(TOY_KEY)

    def _write(self, name, fields):
        (self.folder / name).write_text(json.dumps(fields), encoding="utf-8")

    def _write_key(self, key):
        self.private = {name: format(getattr(key, name), "x") for name in ("n", "e", "d", "p", "q")}
        self.public = {name: f"  0X{getattr(key, name):X}  " for name in ("n", "e")}
        self._write("private_key.json", self.private)
        self._write("public_key.json", self.public)
        self._write("p.json", {"prime": format(key.p, "x")})
        self._write("q.json", {"prime": format(key.q, "x")})

    def test_load_and_validate_optional_crt_fields(self):
        self.assertEqual(load_json_key(self.folder), TOY_KEY)
        self._write("private_key.json", dict(self.private, dP="35", dQ="31", qInv="26"))
        self.assertEqual(load_json_key(self.folder, recheck_primes=True), TOY_KEY)

    def test_reject_missing_malformed_or_disagreeing_input(self):
        bad_json = ('{"n":"ca1","n":"ca1","e":"11"}', '{', '[]', '{"n":NaN}', '{"e":"11"}')
        for bad in bad_json:
            with self.subTest(bad=bad):
                (self.folder / "public_key.json").write_text(bad, encoding="utf-8")
                with self.assertRaisesRegex(ValueError, "public_key.json"):
                    load_json_key(self.folder)
        changes = (
            ("public_key.json", dict(self.public, e="13"), "расходятся"),
            ("p.json", {"prime": "35"}, "не совпадает"),
            ("private_key.json", dict(self.private, dP="0"), "dP"),
            ("private_key.json", dict(self.private, d="0xNOTHEXSECRET"), "d"),
        )
        for filename, value, reason in changes:
            with self.subTest(filename=filename, reason=reason):
                self._write_key(TOY_KEY)
                self._write(filename, value)
                with self.assertRaisesRegex(ValueError, reason) as error:
                    load_json_key(self.folder)
                self.assertNotIn("NOTHEXSECRET", str(error.exception))
        self._write_key(TOY_KEY)
        (self.folder / "q.json").unlink()
        with self.assertRaisesRegex(ValueError, "q.json"):
            load_json_key(self.folder)

    def test_primality_check_is_separate_from_parameter_relations(self):
        composite = make_private_key(n=225, e=5, d=5, p=9, q=25)
        self._write_key(composite)
        self.assertEqual(load_json_key(self.folder), composite)
        with self.assertRaisesRegex(ValueError, "Миллера"):
            load_json_key(self.folder, recheck_primes=True)

    def test_cli_bench_with_imported_toy_key(self):
        command = [sys.executable, "-B", str(ROOT / "rsa_demo.py"), "bench", "--key-dir", str(self.folder)]
        output = subprocess.run(command, cwd=self.folder, capture_output=True, text=True, check=True, timeout=15)
        report = json.loads(output.stdout)
        self.assertEqual(report["key_source"], "json")
        self.assertEqual(report["bits"], 12)
        self.assertIsNone(report["generation_seconds"])
        self.assertEqual(set(report["operations_seconds"]),
                         {"rsaep", "rsadp_reference", "rsadp_crt", "rsasp1", "rsavp1"})
        self.assertEqual(report["key_check"], "parameter_relations_only")

    def test_hex_format(self):
        self.assertEqual(hex_to_int("  0X00aF  "), 175)
        self.assertEqual(int_to_hex(175), "af")
        for value in (None, 42, "0x", "+1", "-1", "0x1 2", "0x4_1"):
            with self.subTest(value=value):
                with self.assertRaises(ValueError):
                    hex_to_int(value)


class PrimeSearchTests(unittest.TestCase):
    def test_random_candidate_and_rejection_sampling(self):
        candidate = random_bits(16, lambda _: 0)
        self.assertEqual(candidate.bit_length(), 16)
        self.assertEqual(candidate % 2, 1)
        draws = Mock(side_effect=[7, 3])
        self.assertEqual(random_below(5, draws), 3)
        self.assertEqual(draws.call_count, 2)
        self.assertEqual(random_below(1, draws), 0)
        for bound in (0, -4):
            with self.assertRaises(ValueError):
                random_below(bound)
        with self.assertRaises(ValueError):
            random_bits(16, lambda _: 2 ** 16)

    def test_small_numbers_and_real_miller_rabin_rounds(self):
        for n, expected in ((0, False), (1, False), (2, True), (3, True), (561, False)):
            with self.subTest(n=n):
                self.assertEqual(is_probable_prime(n), expected)
        draws = Mock(return_value=0)  # Основание a = 2, без случайного выбора в тесте.
        self.assertFalse(is_probable_prime(10007 * 10009, rng=draws))
        self.assertEqual(draws.call_count, 1)
        draws.reset_mock()
        self.assertTrue(is_probable_prime(2 ** 61 - 1, rng=draws))
        self.assertEqual(draws.call_count, 64)

    def test_primes_must_allow_public_exponent(self):
        # 917519 простое, но 917519 − 1 делится на 65537.
        draws = Mock(side_effect=[917519, 917549])
        p = generate_prime(20, rng=draws)
        self.assertEqual(p, 917549)
        self.assertEqual(math.gcd(PUBLIC_EXPONENT, p - 1), 1)

    def test_cancel_during_search(self):
        cancel = Mock(side_effect=[False, True])
        with self.assertRaises(Cancelled):
            generate_prime(16, rng=lambda _: 0, cancel=cancel)


class KeyGenerationTests(unittest.TestCase):
    def test_small_key_and_both_operation_paths(self):
        key = generate_key_pair(64, rng=random.Random(150).getrandbits)
        validate_key(key, recheck_primes=True)
        self.assertEqual(key.n.bit_length(), 64)
        self.assertNotEqual(key.p, key.q)
        self.assertEqual(key.n, key.p * key.q)
        self.assertEqual(key.e, PUBLIC_EXPONENT)
        self.assertEqual(key.e * key.d % math.lcm(key.p - 1, key.q - 1), 1)
        ciphertext = rsaep(key.public_key(), 65)
        self.assertEqual(rsadp_reference(key, ciphertext), 65)
        self.assertEqual(rsadp_crt(key, ciphertext), 65)

    def test_retries_equal_factors_and_short_modulus(self):
        small_1, small_2 = 2147483659, 2147483693
        large_1, large_2 = 4294967291, 4294967279
        p_and_q = [large_1, large_1, small_1, small_2, large_1, large_2]
        with patch("rsa_core.rsa.generate_prime", side_effect=p_and_q) as generate:
            key = generate_key_pair(64)
        self.assertEqual(generate.call_count, len(p_and_q))
        self.assertEqual(key.n, large_1 * large_2)
        self.assertEqual(key.n.bit_length(), 64)

    def test_invalid_size_cancellation_and_entropy_failure(self):
        for size in (16, 33, 65536):
            with self.assertRaises(ValueError):
                generate_key_pair(size)
        with self.assertRaises(Cancelled):
            generate_key_pair(64, cancel=lambda: True)
        with self.assertRaisesRegex(ValueError, "Источник случайности недоступен"):
            generate_key_pair(64, rng=Mock(side_effect=OSError("source failure")))
