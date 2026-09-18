#!/usr/bin/env python3
from __future__ import annotations

import importlib.util
import json
import os
from pathlib import Path
import stat
import sys
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]
MOD_PATH = ROOT / "tools" / "waf_package_builder.py"
spec = importlib.util.spec_from_file_location("waf_package_builder", MOD_PATH)
assert spec and spec.loader
mod = importlib.util.module_from_spec(spec)
sys.modules[spec.name] = mod
spec.loader.exec_module(mod)


class BuilderTests(unittest.TestCase):
    def test_parse_version(self):
        self.assertEqual(mod.parse_version_tuple("go version go1.25.0 linux/amd64"), (1, 25, 0))
        self.assertGreaterEqual(mod.parse_version_tuple("1.26"), (1, 25, 0))

    def test_project_pins(self):
        go, coraza = mod.read_project_requirements(ROOT)
        self.assertEqual(go, "1.25.0")
        self.assertEqual(coraza, "v3.7.0")

    def test_fingerprint_is_stable_and_ignores_dist(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            (root / "a.txt").write_text("one\n")
            first = mod.source_fingerprint(root)
            (root / "dist").mkdir()
            (root / "dist" / "generated.deb").write_bytes(b"x")
            self.assertEqual(first, mod.source_fingerprint(root))
            (root / "a.txt").write_text("two\n")
            self.assertNotEqual(first, mod.source_fingerprint(root))

    def test_epoch_explicit(self):
        epoch, origin = mod.derive_epoch(ROOT, "1234567890")
        self.assertEqual(epoch, 1234567890)
        self.assertEqual(origin, "explicit/environment")

    def test_default_version_contains_source_identity(self):
        v = mod.default_version("abcdef0123456789", 0)
        self.assertEqual(v, "1970.01.01.srcabcdef01")

    def test_validate_version_is_cross_package_safe(self):
        self.assertEqual(mod.validate_version("2026.09.16.srcdeadbeef"), "2026.09.16.srcdeadbeef")
        with self.assertRaises(mod.BuildError):
            mod.validate_version("1.2.3-rc1")

    def test_package_command_deb(self):
        ns = type("NS", (), {"output_dir": None, "arch": "amd64", "maintainer": None, "rpm_release": "1"})()
        self.assertEqual(
            mod.package_command("deb", ns),
            ["./packaging/deb/build-deb.sh", "--binaries-dir", ".", "--arch", "amd64"],
        )

    def test_package_command_all_uses_format_specific_arches(self):
        ns = type(
            "NS",
            (),
            {
                "output_dir": None,
                "deb_arch": "amd64",
                "rpm_arch": "x86_64",
                "maintainer": None,
                "rpm_release": "2",
            },
        )()
        self.assertEqual(
            mod.package_command("deb", ns),
            ["./packaging/deb/build-deb.sh", "--binaries-dir", ".", "--arch", "amd64"],
        )
        self.assertEqual(
            mod.package_command("rpm", ns),
            ["./packaging/rpm/build-rpm.sh", "--binaries-dir", ".", "--release", "2", "--arch", "x86_64"],
        )

    def test_build_report_has_package_hash(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            pkg = root / "x.deb"
            pkg.write_bytes(b"package")
            report = mod.write_report(
                root, "deb", {"required_go": "1.25.0"}, "f" * 64,
                "1.2.3", "source-abc", 123, "test", {"version": "1.2.3"},
                [pkg], str(root / "out"),
            )
            obj = json.loads(report.read_text())
            self.assertEqual(obj["packages"][0]["sha256"], mod.sha256(pkg))
            self.assertEqual(obj["source_fingerprint_sha256"], "f" * 64)


if __name__ == "__main__":
    unittest.main(verbosity=2)
