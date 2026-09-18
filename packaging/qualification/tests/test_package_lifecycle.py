#!/usr/bin/env python3
import importlib.util
import json
import pathlib
import tempfile
import unittest

ROOT = pathlib.Path(__file__).resolve().parents[3]
MODPATH = ROOT / "packaging/qualification/package_lifecycle_qualify.py"
spec = importlib.util.spec_from_file_location("package_lifecycle_qualify", MODPATH)
mod = importlib.util.module_from_spec(spec)
assert spec and spec.loader
import sys
sys.modules[spec.name] = mod
spec.loader.exec_module(mod)


class LifecycleHelpersTest(unittest.TestCase):
    def make_root(self):
        td = tempfile.TemporaryDirectory()
        root = pathlib.Path(td.name)
        (root / "etc/waf").mkdir(parents=True)
        (root / "var/lib/waf-proxy").mkdir(parents=True)
        (root / "etc/waf/config.json").write_text(json.dumps({"listen": ":8443", "sites": []}) + "\n")
        (root / "etc/waf/coraza.conf").write_text("SecRuleEngine On\n")
        (root / "etc/waf/waf-tls-frontend.env").write_text("WAF_TLS_FRONTEND_ENABLED=0\n")
        (root / "etc/waf/waf-proxy.env").write_bytes(b"WAF_ADMIN_TOKEN=high-entropy-test-only-token\n")
        return td, root

    def test_mutation_changes_bytes_but_keeps_json_data(self):
        td, root = self.make_root()
        try:
            before = mod.config_hashes(root)
            hashes = mod.mutate_operator_configs(root, "abc")
            self.assertNotEqual(before, hashes)
            obj = json.loads((root / "etc/waf/config.json").read_text())
            self.assertEqual(obj, {"listen": ":8443", "sites": []})
            self.assertIn("package-lifecycle-qualification abc", (root / "etc/waf/coraza.conf").read_text())
        finally:
            td.cleanup()

    def test_secret_compare_is_boolean_only(self):
        td, root = self.make_root()
        try:
            p = root / "etc/waf/waf-proxy.env"
            secret = mod.read_secret(p)
            self.assertTrue(mod.compare_secret(secret, p))
            p.write_text("WAF_ADMIN_TOKEN=changed\n")
            self.assertFalse(mod.compare_secret(secret, p))
        finally:
            td.cleanup()

    def test_state_marker_preservation(self):
        td, root = self.make_root()
        try:
            digest = mod.write_state_marker(root, "run-1")
            self.assertTrue(mod.state_marker_preserved(digest, root))
            (root / "var/lib/waf-proxy/.package-lifecycle-qualification").write_text("changed\n")
            self.assertFalse(mod.state_marker_preserved(digest, root))
        finally:
            td.cleanup()

    def test_conflict_path_sets(self):
        deb = {str(x) for x in mod.conflict_paths("deb")}
        rpm = {str(x) for x in mod.conflict_paths("rpm")}
        self.assertTrue(any(x.endswith(".dpkg-dist") for x in deb))
        self.assertTrue(any(x.endswith(".dpkg-old") for x in deb))
        self.assertTrue(any(x.endswith(".rpmnew") for x in rpm))
        self.assertTrue(any(x.endswith(".rpmsave") for x in rpm))

    def test_supported_distro_matrix_is_exact_enterprise_scope(self):
        self.assertEqual(mod.SUPPORTED["deb"], {"debian", "ubuntu"})
        self.assertEqual(mod.SUPPORTED["rpm"], {"rhel", "rocky", "almalinux", "ol"})

    def test_ack_is_explicit(self):
        self.assertEqual(mod.ACK, "I_UNDERSTAND_THIS_MUTATES_A_DEDICATED_HOST")

    def test_service_state_comparison(self):
        before = {"available": True, "waf-proxy.service": False, "waf-tls-frontend.service": True}
        self.assertTrue(mod.service_state_preserved(before, dict(before)))
        after = dict(before); after["waf-proxy.service"] = True
        self.assertFalse(mod.service_state_preserved(before, after))
        self.assertTrue(mod.service_state_preserved({"available": False}, {"available": False}))


if __name__ == "__main__":
    unittest.main(verbosity=2)
