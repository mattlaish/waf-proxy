#!/usr/bin/env python3
"""Enterprise package upgrade/rollback qualification for waf-proxy.

This runner is intentionally destructive to the package database on the host on
which --execute is used. It is designed for dedicated disposable qualification
VMs. Preflight mode is non-mutating.

Evidence contains no administrator token or derivative of that secret: the
secret is compared only in memory and reported as a boolean preservation result.
"""
from __future__ import annotations

import argparse
import dataclasses
import hashlib
import json
import os
import pathlib
import platform
import re
import shutil
import subprocess
import sys
import tempfile
import time
from typing import Callable, Iterable

PACKAGE = "waf-proxy"
CONFIG_PATHS = (
    pathlib.Path("/etc/waf/config.json"),
    pathlib.Path("/etc/waf/coraza.conf"),
    pathlib.Path("/etc/waf/waf-tls-frontend.env"),
)
SECRET_PATH = pathlib.Path("/etc/waf/waf-proxy.env")
STATE_MARKER = pathlib.Path("/var/lib/waf-proxy/.package-lifecycle-qualification")
ACK = "I_UNDERSTAND_THIS_MUTATES_A_DEDICATED_HOST"
SUPPORTED = {
    "deb": {"debian", "ubuntu"},
    "rpm": {"rhel", "rocky", "almalinux", "ol"},
}


@dataclasses.dataclass(frozen=True)
class PackageMeta:
    path: str
    version: str
    architecture: str
    sha256: str


class QualificationError(RuntimeError):
    pass


class Blocked(QualificationError):
    pass


def sha256_file(path: pathlib.Path) -> str:
    h = hashlib.sha256()
    with path.open("rb") as f:
        for chunk in iter(lambda: f.read(1024 * 1024), b""):
            h.update(chunk)
    return h.hexdigest()


def run(cmd: list[str], *, check: bool = True, env: dict[str, str] | None = None) -> subprocess.CompletedProcess[str]:
    p = subprocess.run(cmd, text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE, env=env)
    if check and p.returncode != 0:
        raise QualificationError(f"command failed rc={p.returncode}: {cmd[0]} {cmd[1] if len(cmd)>1 else ''}: {p.stderr.strip()[:400]}")
    return p


def os_release(path: pathlib.Path = pathlib.Path("/etc/os-release")) -> dict[str, str]:
    out: dict[str, str] = {}
    if not path.exists():
        return out
    for raw in path.read_text(encoding="utf-8", errors="replace").splitlines():
        if "=" not in raw or raw.lstrip().startswith("#"):
            continue
        k, v = raw.split("=", 1)
        out[k] = v.strip().strip('"')
    return out


def require_commands(names: Iterable[str]) -> None:
    missing = [x for x in names if shutil.which(x) is None]
    if missing:
        raise Blocked("missing required commands: " + ",".join(missing))


def read_secret(path: pathlib.Path = SECRET_PATH) -> bytes:
    if not path.is_file():
        raise QualificationError(f"expected generated secret missing: {path}")
    data = path.read_bytes()
    if b"WAF_ADMIN_TOKEN=" not in data:
        raise QualificationError(f"unexpected secret file shape: {path}")
    return data


def compare_secret(expected: bytes, path: pathlib.Path = SECRET_PATH) -> bool:
    # Never emit or hash the credential into evidence.
    try:
        return path.read_bytes() == expected
    except FileNotFoundError:
        return False


def mutate_operator_configs(root: pathlib.Path = pathlib.Path("/"), run_id: str = "qualification") -> dict[str, str]:
    """Change bytes without changing JSON semantics; comments are safe in text configs."""
    result: dict[str, str] = {}
    for absolute in CONFIG_PATHS:
        path = root / absolute.relative_to("/")
        if not path.is_file():
            raise QualificationError(f"expected packaged config missing: {path}")
        if path.name == "config.json":
            obj = json.loads(path.read_text(encoding="utf-8"))
            # Reformatting changes conffile bytes while retaining exact JSON data.
            text = json.dumps(obj, indent=4, sort_keys=True, ensure_ascii=False) + "\n   \n"
            path.write_text(text, encoding="utf-8")
        else:
            with path.open("a", encoding="utf-8") as f:
                f.write(f"\n# package-lifecycle-qualification {run_id}\n")
        result[str(absolute)] = sha256_file(path)
    return result


def config_hashes(root: pathlib.Path = pathlib.Path("/")) -> dict[str, str]:
    out: dict[str, str] = {}
    for absolute in CONFIG_PATHS:
        p = root / absolute.relative_to("/")
        if not p.is_file():
            raise QualificationError(f"expected config missing: {p}")
        out[str(absolute)] = sha256_file(p)
    return out


def write_state_marker(root: pathlib.Path = pathlib.Path("/"), run_id: str = "qualification") -> str:
    p = root / STATE_MARKER.relative_to("/")
    p.parent.mkdir(parents=True, exist_ok=True)
    data = (f"waf-package-lifecycle:{run_id}\n").encode()
    p.write_bytes(data)
    return hashlib.sha256(data).hexdigest()


def state_marker_preserved(expected_digest: str, root: pathlib.Path = pathlib.Path("/")) -> bool:
    p = root / STATE_MARKER.relative_to("/")
    return p.is_file() and sha256_file(p) == expected_digest


def service_states() -> dict[str, bool | str]:
    if shutil.which("systemctl") is None:
        return {"available": False, "waf-proxy.service": False, "waf-tls-frontend.service": False}
    out: dict[str, bool | str] = {"available": True}
    for name in ("waf-proxy.service", "waf-tls-frontend.service"):
        p = run(["systemctl", "is-active", "--quiet", name], check=False)
        out[name] = p.returncode == 0
    return out


def service_state_preserved(before: dict[str, bool | str], after: dict[str, bool | str]) -> bool:
    if before.get("available") is not True or after.get("available") is not True:
        return True
    return all(before.get(k) == after.get(k) for k in ("waf-proxy.service", "waf-tls-frontend.service"))


def package_config_hashes(fmt: str, package: pathlib.Path) -> dict[str, str]:
    with tempfile.TemporaryDirectory() as td:
        root = pathlib.Path(td)
        if fmt == "deb":
            require_commands(["dpkg-deb"])
            run(["dpkg-deb", "-x", str(package), str(root)])
        else:
            require_commands(["rpm2cpio", "cpio"])
            # shell is used only for a fixed local extraction pipeline; package path is argv.
            p1 = subprocess.Popen(["rpm2cpio", str(package)], stdout=subprocess.PIPE, stderr=subprocess.PIPE)
            p2 = subprocess.run(["cpio", "-idm", "--quiet"], cwd=root, stdin=p1.stdout, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=False)
            assert p1.stdout is not None
            p1.stdout.close()
            _, err1 = p1.communicate()
            if p1.returncode != 0 or p2.returncode != 0:
                raise QualificationError(f"RPM extraction failed: {err1.decode(errors='replace')[:300]} {p2.stderr.decode(errors='replace')[:300]}")
        out: dict[str, str] = {}
        for absolute in CONFIG_PATHS:
            p = root / absolute.relative_to("/")
            if p.is_file():
                out[str(absolute)] = sha256_file(p)
        return out


def conflict_paths(fmt: str) -> list[pathlib.Path]:
    suffixes = (".dpkg-dist", ".dpkg-old") if fmt == "deb" else (".rpmnew", ".rpmsave")
    return [pathlib.Path(str(p) + s) for p in CONFIG_PATHS for s in suffixes]


def capture_conflicts(fmt: str) -> list[str]:
    return [str(p) for p in conflict_paths(fmt) if p.exists()]


class Backend:
    fmt: str

    def metadata(self, path: pathlib.Path) -> PackageMeta:
        raise NotImplementedError

    def installed_version(self) -> str | None:
        raise NotImplementedError

    def install_baseline(self, path: pathlib.Path) -> None:
        raise NotImplementedError

    def upgrade(self, path: pathlib.Path, *, expect_failure: bool = False) -> int:
        raise NotImplementedError

    def rollback(self, path: pathlib.Path) -> None:
        raise NotImplementedError


class DebBackend(Backend):
    fmt = "deb"

    def __init__(self) -> None:
        require_commands(["dpkg", "dpkg-deb", "dpkg-query"])

    def metadata(self, path: pathlib.Path) -> PackageMeta:
        fields = [run(["dpkg-deb", "-f", str(path), field]).stdout.strip() for field in ("Package", "Version", "Architecture")]
        if len(fields) < 3 or fields[0] != PACKAGE:
            raise QualificationError(f"unexpected DEB metadata: {fields}")
        return PackageMeta(str(path), fields[1], fields[2], sha256_file(path))

    def installed_version(self) -> str | None:
        p = run(["dpkg-query", "-W", "-f=${Status}\n${Version}\n", PACKAGE], check=False)
        if p.returncode != 0:
            return None
        lines = p.stdout.splitlines()
        if not lines or "install ok installed" not in lines[0]:
            return None
        return lines[1].strip() if len(lines) > 1 else None

    @staticmethod
    def _install(path: pathlib.Path, check: bool) -> subprocess.CompletedProcess[str]:
        env = os.environ.copy()
        env["DEBIAN_FRONTEND"] = "noninteractive"
        # confold is an explicit qualification policy: operator config must win.
        return run(["dpkg", "--force-confdef", "--force-confold", "--install", str(path)], check=check, env=env)

    def install_baseline(self, path: pathlib.Path) -> None:
        self._install(path, True)

    def upgrade(self, path: pathlib.Path, *, expect_failure: bool = False) -> int:
        p = self._install(path, not expect_failure)
        if expect_failure and p.returncode == 0:
            raise QualificationError("failure fixture unexpectedly upgraded successfully")
        return p.returncode

    def rollback(self, path: pathlib.Path) -> None:
        env = os.environ.copy(); env["DEBIAN_FRONTEND"] = "noninteractive"
        run(["dpkg", "--force-downgrade", "--force-confdef", "--force-confold", "--install", str(path)], env=env)


class RpmBackend(Backend):
    fmt = "rpm"

    def __init__(self) -> None:
        require_commands(["rpm", "rpm2cpio", "cpio"])

    def metadata(self, path: pathlib.Path) -> PackageMeta:
        p = run(["rpm", "-qp", "--qf", "%{NAME}\n%{VERSION}-%{RELEASE}\n%{ARCH}\n", str(path)])
        lines = [x.strip() for x in p.stdout.splitlines() if x.strip()]
        if len(lines) < 3 or lines[0] != PACKAGE:
            raise QualificationError(f"unexpected RPM metadata: {lines}")
        return PackageMeta(str(path), lines[1], lines[2], sha256_file(path))

    def installed_version(self) -> str | None:
        p = run(["rpm", "-q", "--qf", "%{VERSION}-%{RELEASE}\n", PACKAGE], check=False)
        return p.stdout.strip() if p.returncode == 0 else None

    @staticmethod
    def _install(path: pathlib.Path, *, old: bool, check: bool) -> subprocess.CompletedProcess[str]:
        args = ["rpm", "-Uvh", "--replacepkgs"]
        if old:
            args.append("--oldpackage")
        args.append(str(path))
        return run(args, check=check)

    def install_baseline(self, path: pathlib.Path) -> None:
        self._install(path, old=False, check=True)

    def upgrade(self, path: pathlib.Path, *, expect_failure: bool = False) -> int:
        p = self._install(path, old=False, check=not expect_failure)
        if expect_failure and p.returncode == 0:
            raise QualificationError("failure fixture unexpectedly upgraded successfully")
        return p.returncode

    def rollback(self, path: pathlib.Path) -> None:
        self._install(path, old=True, check=True)


def backend_for(fmt: str) -> Backend:
    return DebBackend() if fmt == "deb" else RpmBackend()


def criterion(name: str, status: str, detail: str = "") -> dict[str, str]:
    return {"name": name, "status": status, "detail": detail}


def validate_distro(fmt: str) -> dict[str, str]:
    info = os_release()
    distro = info.get("ID", "unknown").lower()
    if distro not in SUPPORTED[fmt]:
        raise Blocked(f"{fmt} lifecycle qualification requires supported dedicated distro; detected ID={distro}")
    return {"id": distro, "version_id": info.get("VERSION_ID", "unknown"), "pretty_name": info.get("PRETTY_NAME", distro)}


def package_set(backend: Backend, baseline: pathlib.Path, candidate: pathlib.Path, failure: pathlib.Path | None) -> dict[str, PackageMeta | None]:
    metas = {
        "baseline": backend.metadata(baseline),
        "candidate": backend.metadata(candidate),
        "failure": backend.metadata(failure) if failure else None,
    }
    if metas["baseline"].version == metas["candidate"].version:  # type: ignore[union-attr]
        raise QualificationError("baseline and candidate package versions must differ")
    if failure and metas["failure"].version in {metas["baseline"].version, metas["candidate"].version}:  # type: ignore[union-attr]
        raise QualificationError("failure fixture must use a distinct package version")
    return metas


def meta_json(meta: PackageMeta | None) -> dict[str, str] | None:
    return dataclasses.asdict(meta) if meta is not None else None


def report_skeleton(args: argparse.Namespace) -> dict:
    return {
        "schema_version": 1,
        "evidence_type": "waf-package-upgrade-rollback-qualification",
        "status": "NOT_RUN",
        "package_format": args.format,
        "real_package_manager_execution": False,
        "started_at_unix": int(time.time()),
        "host": {"kernel": platform.release(), "machine": platform.machine()},
        "packages": {},
        "criteria": [],
        "truth_boundary": {
            "fixture_build_is_not_distro_qualification": True,
            "preflight_is_not_lifecycle_pass": True,
            "secret_material_in_evidence": False,
            "clean_host_matrix_is_slice_d": True,
        },
    }


def write_report(path: pathlib.Path, report: dict) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(report, indent=2, sort_keys=True) + "\n", encoding="utf-8")


def execute(args: argparse.Namespace) -> int:
    report = report_skeleton(args)
    out = pathlib.Path(args.output).resolve()
    try:
        backend = backend_for(args.format)
        baseline = pathlib.Path(args.baseline).resolve(); candidate = pathlib.Path(args.candidate).resolve()
        failure = pathlib.Path(args.failure_fixture).resolve() if args.failure_fixture else None
        for p in [baseline, candidate] + ([failure] if failure else []):
            if p is None or not p.is_file():
                raise QualificationError(f"package artifact not found: {p}")
        metas = package_set(backend, baseline, candidate, failure)
        report["packages"] = {k: meta_json(v) for k, v in metas.items()}

        base_cfg = package_config_hashes(args.format, baseline)
        cand_cfg = package_config_hashes(args.format, candidate)
        changed_defaults = sorted(k for k in set(base_cfg) & set(cand_cfg) if base_cfg[k] != cand_cfg[k])
        report["packaged_config_defaults_changed"] = changed_defaults
        report["criteria"].append(criterion("package_metadata_preflight", "PASS"))
        if changed_defaults:
            report["criteria"].append(criterion("config_conflict_fixture_precondition", "PASS", ",".join(changed_defaults)))
        else:
            report["criteria"].append(criterion("config_conflict_fixture_precondition", "BLOCKED", "baseline/candidate packaged defaults are identical"))

        if not args.execute:
            report["status"] = "NOT_RUN"
            report["criteria"].append(criterion("real_upgrade_rollback_execution", "NOT_RUN", "rerun with --execute on a dedicated supported host"))
            write_report(out, report)
            print(f"PACKAGE_LIFECYCLE_PREFLIGHT_PASS report={out}")
            return 0

        if args.ack != ACK:
            raise Blocked(f"--execute requires --ack {ACK}")
        if os.geteuid() != 0:
            raise Blocked("real package lifecycle qualification requires root on a disposable host")
        report["host"]["os_release"] = validate_distro(args.format)
        current = backend.installed_version()
        if current is not None:
            if not args.allow_installed_baseline or current != metas["baseline"].version:  # type: ignore[union-attr]
                raise Blocked(f"{PACKAGE} already installed as {current}; expected clean host or --allow-installed-baseline matching baseline")
        else:
            backend.install_baseline(baseline)
        report["criteria"].append(criterion("baseline_install", "PASS"))
        report["real_package_manager_execution"] = True

        before_services = service_states()
        run_id = f"{int(time.time())}-{os.getpid()}"
        wanted_cfg = mutate_operator_configs(run_id=run_id)
        secret = read_secret()
        state_digest = write_state_marker(run_id=run_id)

        backend.upgrade(candidate)
        report["criteria"].append(criterion("candidate_upgrade", "PASS"))
        cfg_ok = config_hashes() == wanted_cfg
        secret_ok = compare_secret(secret)
        state_ok = state_marker_preserved(state_digest)
        service_ok = service_state_preserved(before_services, service_states())
        report["criteria"].extend([
            criterion("operator_config_preserved_on_upgrade", "PASS" if cfg_ok else "FAIL"),
            criterion("admin_secret_preserved_on_upgrade", "PASS" if secret_ok else "FAIL"),
            criterion("persistent_state_preserved_on_upgrade", "PASS" if state_ok else "FAIL"),
            criterion("service_active_state_preserved_on_upgrade", "PASS" if service_ok else "FAIL"),
        ])
        if not all((cfg_ok, secret_ok, state_ok, service_ok)):
            raise QualificationError("upgrade preservation invariant failed")

        conflicts = capture_conflicts(args.format)
        report["upgrade_conflict_artifacts"] = conflicts
        if changed_defaults:
            expected_suffix = ".dpkg-dist" if args.format == "deb" else ".rpmnew"
            conflict_ok = any(x.endswith(expected_suffix) for x in conflicts)
            report["criteria"].append(criterion(
                "native_config_conflict_semantics",
                "PASS" if conflict_ok else "FAIL",
                f"expected at least one {expected_suffix} artifact because packaged defaults changed",
            ))
            if not conflict_ok:
                raise QualificationError("native config conflict artifact not observed")
        else:
            report["criteria"].append(criterion("native_config_conflict_semantics", "NOT_RUN", "requires intentionally changed packaged default"))

        if failure is not None:
            rc = backend.upgrade(failure, expect_failure=True)
            report["failed_upgrade_return_code"] = rc
            fail_cfg = config_hashes() == wanted_cfg
            fail_secret = compare_secret(secret)
            fail_state = state_marker_preserved(state_digest)
            report["criteria"].extend([
                criterion("failure_fixture_rejected", "PASS"),
                criterion("operator_config_preserved_after_failed_upgrade", "PASS" if fail_cfg else "FAIL"),
                criterion("admin_secret_preserved_after_failed_upgrade", "PASS" if fail_secret else "FAIL"),
                criterion("persistent_state_preserved_after_failed_upgrade", "PASS" if fail_state else "FAIL"),
            ])
            if not all((fail_cfg, fail_secret, fail_state)):
                raise QualificationError("failed-upgrade preservation invariant failed")
            # Recover to the known candidate before the rollback phase. Both backends
            # may have advanced package metadata/payload before a post-install failure.
            backend.rollback(candidate)
            report["criteria"].append(criterion("failed_upgrade_recovery_to_candidate", "PASS"))
        else:
            report["criteria"].append(criterion("failed_upgrade_behavior", "NOT_RUN", "no --failure-fixture supplied"))

        backend.rollback(baseline)
        rollback_cfg = config_hashes() == wanted_cfg
        rollback_secret = compare_secret(secret)
        rollback_state = state_marker_preserved(state_digest)
        final_version = backend.installed_version()
        version_ok = final_version == metas["baseline"].version  # type: ignore[union-attr]
        report["final_installed_version"] = final_version
        report["rollback_conflict_artifacts"] = capture_conflicts(args.format)
        report["criteria"].extend([
            criterion("rollback_to_baseline", "PASS" if version_ok else "FAIL"),
            criterion("operator_config_preserved_on_rollback", "PASS" if rollback_cfg else "FAIL"),
            criterion("admin_secret_preserved_on_rollback", "PASS" if rollback_secret else "FAIL"),
            criterion("persistent_state_preserved_on_rollback", "PASS" if rollback_state else "FAIL"),
        ])
        if not all((version_ok, rollback_cfg, rollback_secret, rollback_state)):
            raise QualificationError("rollback invariant failed")

        report["status"] = "PASS"
        report["completed_at_unix"] = int(time.time())
        write_report(out, report)
        print(f"PACKAGE_LIFECYCLE_QUALIFICATION_PASS format={args.format} report={out}")
        return 0
    except Blocked as e:
        report["status"] = "BLOCKED"
        report["criteria"].append(criterion("execution_environment", "BLOCKED", str(e)))
        report["completed_at_unix"] = int(time.time())
        write_report(out, report)
        print(f"PACKAGE_LIFECYCLE_QUALIFICATION_BLOCKED reason={e} report={out}")
        return 77
    except Exception as e:
        report["status"] = "FAIL"
        report["criteria"].append(criterion("qualification_exception", "FAIL", str(e)))
        report["completed_at_unix"] = int(time.time())
        write_report(out, report)
        print(f"PACKAGE_LIFECYCLE_QUALIFICATION_FAIL reason={e} report={out}", file=sys.stderr)
        return 1


def parser() -> argparse.ArgumentParser:
    p = argparse.ArgumentParser(description="Qualify waf-proxy DEB/RPM upgrade and rollback lifecycle")
    p.add_argument("--format", choices=("deb", "rpm"), required=True)
    p.add_argument("--baseline", required=True, help="Version N package")
    p.add_argument("--candidate", required=True, help="Version N+1 package")
    p.add_argument("--failure-fixture", help="higher-version package whose post-install is designed to fail")
    p.add_argument("--output", required=True)
    p.add_argument("--execute", action="store_true", help="mutate the dedicated host package database")
    p.add_argument("--ack", default="", help="required destructive-execution acknowledgement")
    p.add_argument("--allow-installed-baseline", action="store_true", help="permit starting when exactly the baseline version is already installed")
    return p


if __name__ == "__main__":
    raise SystemExit(execute(parser().parse_args()))
