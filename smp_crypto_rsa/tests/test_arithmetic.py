import json
import math
import random
import unittest
from pathlib import Path

from rsa_core.arithmetic import _Montgomery, gcd, lcm, mod_inverse, mod_pow


class ArithmeticTests(unittest.TestCase):
    def test_control_vectors(self):
        path = Path(__file__).resolve().parents[1] / "test_vectors" / "arithmetic.json"
        vectors = json.loads(path.read_text(encoding="utf-8"))
        self.assertTrue(vectors)
        operations = {func.__name__: func for func in (gcd, lcm, mod_inverse, mod_pow)}
        for case in vectors:
            with self.subTest(operation=case["op"], args=case["args"]):
                self.assertEqual(operations[case["op"]](*case["args"]), case["expected"])

    def test_gcd_lcm_and_inverse_against_standard_library(self):
        rng = random.Random(145)
        for _ in range(64):
            a, b = rng.getrandbits(256), rng.getrandbits(256)
            self.assertEqual(gcd(a, b), math.gcd(a, b))
            self.assertEqual(lcm(a, b), math.lcm(a, b))
            if b > 1 and math.gcd(a, b) == 1:
                self.assertEqual(mod_inverse(a, b), pow(a, -1, b))

    def test_mod_pow_against_builtin_for_both_parities(self):
        rng = random.Random(146)
        for size in (8, 64, 256, 2048):
            for odd in (0, 1):
                for _ in range(8):
                    modulus = (1 << (size - 1)) | (rng.getrandbits(size - 2) << 1) | odd
                    base, exponent = rng.getrandbits(size * 2), rng.getrandbits(size)
                    with self.subTest(bits=size, odd=odd):
                        self.assertEqual(mod_pow(base, exponent, modulus), pow(base, exponent, modulus))

    def test_32768_bit_modulus_without_key_generation(self):
        rng = random.Random(147)
        modulus = (1 << 32767) | rng.getrandbits(32767) | 1
        base = modulus - 12345
        self.assertEqual(mod_pow(base, 65537, modulus), pow(base, 65537, modulus))

    def test_montgomery_conversion_and_product(self):
        for modulus in (3, 17, 3233, 65537):
            mont = _Montgomery(modulus)
            for a in (0, 1, modulus - 1, modulus + 2):
                with self.subTest(modulus=modulus, a=a):
                    encoded = mont.encode(a)
                    self.assertEqual(mont.reduce(encoded), a % modulus)
                    self.assertEqual(mont.reduce(mont.multiply(encoded, mont.encode(7))), (a * 7) % modulus)

    def test_invalid_arguments_and_impossible_inverse(self):
        for func, args in (
            (gcd, (-1, 2)), (gcd, (True, 2)), (lcm, (2, -1)),
            (mod_inverse, (-1, 5)), (mod_inverse, (1, 0)),
            (mod_inverse, (1, 1)), (mod_inverse, (6, 9)),
            (mod_pow, (-1, 1, 5)), (mod_pow, (1, -1, 5)),
            (mod_pow, (1, 1, 0)), (mod_pow, (1, 1, -1)),
            (mod_pow, (1, 1.0, 5)),
        ):
            with self.subTest(function=func.__name__, args=args):
                with self.assertRaises(ValueError):
                    func(*args)
