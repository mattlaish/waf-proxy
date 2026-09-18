#!/usr/bin/env python3
"""Project-specific DEB/RPM build orchestrator for WAF Reverse Proxy.

The tool deliberately composes the repository's existing canonical build and
packaging scripts instead of re-implementing them. It verifies the source pin,
Go toolchain, packaging host prerequisites and reproducibility inputs, builds
one exact set of provenance-bound binaries, and then packages those bytes as a
DEB, RPM, or both.

It does not silently install toolchains/dependencies or weaken build policy.
"""
from __future__ import annotations

import argparse
import datetime as dt
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import sys
from typing import Iterable, Mapping

MIN_PYTHON = (3, 10)
REQUIRED_CORAZA_MODULE = "github.com/corazawaf/coraza/v3"
GENERATED_NAMES = {
    "waf-proxy",
    "wafctl",
    "waf-tlsfront",
    "wafqualify",
    "BUILD_PROVENANCE.json",
    "BUILD_SHA256SUMS.txt",
    "PACKAGE_BUILD_REPORT.json",
    "RELEASE_MANIFEST.txt",
    "SHA256SUMS.txt",
}
EXCLUDED_DIRS = {
    ".git",
    ".cache",
    "__pycache__",
    "release-evidence",
    "dist",
    "bin",
}
EXCLUDED_SUFFIXES = {".zip", ".patch", ".deb", ".rpm", ".log", ".tmp", ".pyc"}


class BuildError(RuntimeError):
    pass


def project_root() -> Path:
    return Path(__file__).resolve().parent.parent


def parse_version_tuple(text: str) -> tuple[int, ...]:
    m = re.search(r"(\d+)\.(\d+)(?:\.(\d+))?", text)
    if not m:
        raise BuildError(f"unable to parse version from: {text!r}")
    return tuple(int(x or 0) for x in m.groups())


def read_project_requirements(root: Path) -> tuple[str, str]:
    go_mod = (root / "go.mod").read_text(encoding="utf-8")
    gm = re.search(r"(?m)^go\s+([0-9]+(?:\.[0-9]+){1,2})\s*$", go_mod)
    if not gm:
        raise BuildError("go.mod does not declare a Go version")
    cm = re.search(
        rf"(?m)^\s*(?:require\s+)?{re.escape(REQUIRED_CORAZA_MODULE)}\s+(v[^\s]+)\s*$",
        go_mod,
    )
    if not cm:
        raise BuildError(f"go.mod does not pin {REQUIRED_CORAZA_MODULE}")
    return gm.group(1), cm.group(1)


def source_files(root: Path) -> Iterable[Path]:
    for path in sorted(root.rglob("*"), key=lambda p: p.relative_to(root).as_posix()):
        if not path.is_file() or path.is_symlink():
            continue
        rel = path.relative_to(root)
        if any(part in EXCLUDED_DIRS for part in rel.parts):
            continue
        if rel.name in GENERATED_NAMES or rel.suffix.lower() in EXCLUDED_SUFFIXES:
            continue
        yield path


def source_fingerprint(root: Path) -> str:
    h = hashlib.sha256()
    for path in source_files(root):
        rel = path.relative_to(root).as_posix().encode()
        mode = path.stat().st_mode & 0o777
        h.update(len(rel).to_bytes(4, "big"))
        h.update(rel)
        h.update(f"{mode:o}".encode() + b"\0")
        with path.open("rb") as f:
            for chunk in iter(lambda: f.read(1024 * 1024), b""):
                h.update(chunk)
    return h.hexdigest()


def git_value(root: Path, args: list[str]) -> str | None:
    if not (root / ".git").exists() or not shutil.which("git"):
        return None
    cp = subprocess.run(
        ["git", *args], cwd=root, text=True, stdout=subprocess.PIPE,
        stderr=subprocess.DEVNULL, check=False,
    )
    value = cp.stdout.strip()
    return value if cp.returncode == 0 and value else None


def derive_commit(root: Path, fingerprint: str) -> str:
    commit = git_value(root, ["rev-parse", "--short=12", "HEAD"])
    if commit:
        status = subprocess.run(
            ["git", "status", "--porcelain", "--untracked-files=normal"],
            cwd=root,
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.DEVNULL,
            check=False,
        )
        dirty = status.returncode != 0 or bool(status.stdout.strip())
        return f"{commit}-dirty-{fingerprint[:8]}" if dirty else commit
    return f"source-{fingerprint[:12]}"


def derive_epoch(root: Path, explicit: str | None) -> tuple[int, str]:
    raw = explicit or os.environ.get("SOURCE_DATE_EPOCH")
    if raw:
        if not raw.isdigit():
            raise BuildError("SOURCE_DATE_EPOCH must be a non-negative integer")
        return int(raw), "explicit/environment"
    git_epoch = git_value(root, ["log", "-1", "--format=%ct"])
    if git_epoch and git_epoch.isdigit():
        return int(git_epoch), "git-commit"
    mtimes = [int(p.stat().st_mtime) for p in source_files(root)]
    if not mtimes:
        raise BuildError("unable to derive SOURCE_DATE_EPOCH from an empty source tree")
    return max(mtimes), "newest-source-mtime"


def default_version(fingerprint: str, epoch: int | None = None) -> str:
    when = (
        dt.datetime.fromtimestamp(epoch, tz=dt.timezone.utc)
        if epoch is not None
        else dt.datetime.now(dt.timezone.utc)
    )
    return f"{when.strftime('%Y.%m.%d')}.src{fingerprint[:8]}"


def validate_version(version: str) -> str:
    """Keep one portable version spelling across DEB and RPM outputs.

    The lower-level package builders have format-specific normalizers. This
    orchestrator is intentionally stricter so a single `all` invocation cannot
    silently produce DEB/RPM packages with different normalized versions.
    """
    if not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9._+~]*", version):
        raise BuildError(
            "package version must match [A-Za-z0-9][A-Za-z0-9._+~]* "
            "so DEB and RPM keep the same version identity"
        )
    return version


def selected_go(go_bin: str | None) -> str:
    if go_bin:
        path = Path(go_bin).expanduser().resolve()
        if not path.is_file() or not os.access(path, os.X_OK):
            raise BuildError(f"--go-bin is not executable: {path}")
        return str(path)
    found = shutil.which("go")
    if not found:
        raise BuildError("Go is not installed; provide --go-bin /path/to/go")
    return found


def go_version(go: str) -> str:
    cp = subprocess.run(
        [go, "version"], text=True, stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT, env={**os.environ, "GOTOOLCHAIN": "local"}, check=False,
    )
    if cp.returncode != 0:
        raise BuildError(f"unable to execute selected Go toolchain: {cp.stdout.strip()}")
    return cp.stdout.strip()


def command_missing(names: Iterable[str]) -> list[str]:
    return [name for name in names if shutil.which(name) is None]


def preflight(root: Path, target: str, go: str, vectorscan: str, hsm: str) -> dict:
    required_go, coraza = read_project_requirements(root)
    got_text = go_version(go)
    got = parse_version_tuple(got_text)
    required = parse_version_tuple(required_go)
    if got < required:
        raise BuildError(
            f"Go >= {required_go} is required by go.mod/Coraza; selected toolchain is {got_text}"
        )

    base_tools = ["bash", "python3", "sha256sum", "gcc"]
    target_tools: list[str] = []
    if target in {"deb", "all"}:
        target_tools += ["dpkg-deb", "dpkg"]
    if target in {"rpm", "all"}:
        target_tools += ["rpmbuild", "rpm", "rpm2cpio", "cpio", "tar"]
    missing = command_missing([*base_tools, *target_tools])
    if missing:
        raise BuildError("missing required host tools: " + ", ".join(missing))

    if vectorscan == "required":
        if shutil.which("pkg-config") is None:
            raise BuildError("VectorScan required but pkg-config is unavailable")
        cp = subprocess.run(["pkg-config", "--exists", "libhs"], check=False)
        if cp.returncode != 0:
            raise BuildError("VectorScan required but libhs is unavailable to pkg-config")
    if hsm == "required" and shutil.which(os.environ.get("CC", "cc")) is None:
        raise BuildError("PKCS#11 HSM build required but no C compiler is available")

    for required_path in (
        "build.sh",
        "packaging/deb/build-deb.sh",
        "packaging/rpm/build-rpm.sh",
        "packaging/deb/verify-deb.sh",
        "packaging/rpm/verify-rpm.sh",
    ):
        if not (root / required_path).is_file():
            raise BuildError(f"project packaging component missing: {required_path}")

    return {
        "required_go": required_go,
        "selected_go": got_text,
        "coraza_version": coraza,
        "target": target,
        "vectorscan_policy": vectorscan,
        "hsm_policy": hsm,
    }


def build_env(go: str, args: argparse.Namespace, version: str, commit: str, epoch: int) -> dict[str, str]:
    env = dict(os.environ)
    env.update(
        {
            "PATH": str(Path(go).parent) + os.pathsep + env.get("PATH", ""),
            "GOTOOLCHAIN": "local",
            "VERSION": version,
            "COMMIT": commit,
            "SOURCE_DATE_EPOCH": str(epoch),
            "WAF_VECTORSCAN": args.vectorscan,
            "WAF_HSM_PKCS11": args.hsm,
        }
    )
    if args.offline:
        env["GOPROXY"] = "off"
    if getattr(args, "deb_extra_dep", None):
        env["WAF_DEB_EXTRA_DEPENDS"] = args.deb_extra_dep
    if getattr(args, "rpm_extra_require", None):
        env["WAF_RPM_EXTRA_REQUIRES"] = args.rpm_extra_require
    if getattr(args, "maintainer", None):
        env["WAF_PACKAGE_MAINTAINER"] = args.maintainer
    return env


def run_stream(cmd: list[str], root: Path, env: Mapping[str, str]) -> str:
    print("+", " ".join(cmd), flush=True)
    cp = subprocess.Popen(
        cmd, cwd=root, env=dict(env), text=True,
        stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
    )
    assert cp.stdout is not None
    lines: list[str] = []
    for line in cp.stdout:
        sys.stdout.write(line)
        lines.append(line)
    rc = cp.wait()
    output = "".join(lines)
    if rc != 0:
        raise BuildError(f"command failed ({rc}): {' '.join(cmd)}")
    return output


def parse_key(output: str, key: str) -> str | None:
    prefix = key + "="
    for line in reversed(output.splitlines()):
        if line.startswith(prefix):
            return line[len(prefix):].strip()
    return None


def sha256(path: Path) -> str:
    h = hashlib.sha256()
    with path.open("rb") as f:
        for chunk in iter(lambda: f.read(1024 * 1024), b""):
            h.update(chunk)
    return h.hexdigest()


def load_provenance(root: Path) -> dict:
    p = root / "BUILD_PROVENANCE.json"
    if not p.is_file():
        raise BuildError("canonical build did not produce BUILD_PROVENANCE.json")
    obj = json.loads(p.read_text(encoding="utf-8"))
    if obj.get("coraza_version") != "v3.7.0":
        raise BuildError(f"unexpected built Coraza version: {obj.get('coraza_version')!r}")
    return obj


def package_command(target: str, args: argparse.Namespace) -> list[str]:
    if target == "deb":
        cmd = ["./packaging/deb/build-deb.sh", "--binaries-dir", "."]
        if args.output_dir:
            cmd += ["--output-dir", str(Path(args.output_dir).resolve())]
        deb_arch = getattr(args, "deb_arch", None) or getattr(args, "arch", None)
        if deb_arch:
            cmd += ["--arch", deb_arch]
        maintainer = getattr(args, "maintainer", None)
        if maintainer:
            cmd += ["--maintainer", maintainer]
        return cmd
    rpm_release = getattr(args, "rpm_release", "1")
    cmd = ["./packaging/rpm/build-rpm.sh", "--binaries-dir", ".", "--release", rpm_release]
    if args.output_dir:
        cmd += ["--output-dir", str(Path(args.output_dir).resolve())]
    rpm_arch = getattr(args, "rpm_arch", None) or getattr(args, "arch", None)
    if rpm_arch:
        cmd += ["--arch", rpm_arch]
    return cmd


def write_report(
    root: Path,
    target: str,
    pre: dict,
    fingerprint: str,
    version: str,
    commit: str,
    epoch: int,
    epoch_source: str,
    provenance: dict,
    packages: list[Path],
    output_dir: str | None,
) -> Path:
    if output_dir:
        report_dir = Path(output_dir).resolve()
    elif target == "deb":
        report_dir = root / "dist" / "deb"
    elif target == "rpm":
        report_dir = root / "dist" / "rpm"
    else:
        report_dir = root / "dist"
    report_dir.mkdir(parents=True, exist_ok=True)
    report = report_dir / "PACKAGE_BUILD_REPORT.json"
    obj = {
        "schema_version": 1,
        "project": "waf-proxy",
        "builder": "waf-package",
        "target": target,
        "version": version,
        "source_identity": commit,
        "source_fingerprint_sha256": fingerprint,
        "source_date_epoch": epoch,
        "source_date_epoch_origin": epoch_source,
        "toolchain": pre,
        "build_provenance": provenance,
        "packages": [
            {"path": str(p), "sha256": sha256(p), "size": p.stat().st_size}
            for p in packages
        ],
    }
    report.write_text(json.dumps(obj, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    return report


def add_common(p: argparse.ArgumentParser) -> None:
    p.add_argument("--version", help="package version; default is YYYY.MM.DD.src<sourcehash>")
    p.add_argument("--commit", help="build source identity; default git SHA or source fingerprint")
    p.add_argument("--source-date-epoch", help="reproducible epoch; default env, git commit, then newest source mtime")
    p.add_argument("--go-bin", help="Go binary to use (must satisfy go.mod, currently >=1.25.0)")
    p.add_argument("--vectorscan", choices=("off", "auto", "required"), default="off",
                   help="native VectorScan build policy (default: off for portable Coraza package)")
    p.add_argument("--hsm", choices=("off", "auto", "required"), default="off",
                   help="PKCS#11 provider build policy (default: off)")
    network = p.add_mutually_exclusive_group()
    network.add_argument(
        "--offline",
        dest="offline",
        action="store_true",
        default=True,
        help="require a complete local module cache (default; sets GOPROXY=off)",
    )
    network.add_argument(
        "--allow-module-network",
        dest="offline",
        action="store_false",
        help="allow canonical Go commands to use the configured GOPROXY for missing modules",
    )


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        prog="waf-package",
        description="Build verified WAF Reverse Proxy DEB/RPM packages from this source tree.",
    )
    sub = parser.add_subparsers(dest="command", required=True)

    doctor = sub.add_parser("doctor", help="check source/toolchain/package-host readiness without building")
    doctor.add_argument("--format", choices=("deb", "rpm", "all"), default="deb")
    doctor.add_argument("--go-bin")
    doctor.add_argument("--vectorscan", choices=("off", "auto", "required"), default="off")
    doctor.add_argument("--hsm", choices=("off", "auto", "required"), default="off")

    for name in ("deb", "rpm", "all"):
        p = sub.add_parser(name, help=f"build verified {name.upper()} package(s)")
        add_common(p)
        p.add_argument("--output-dir", help="override package output directory")
        if name == "deb":
            p.add_argument("--arch", choices=("amd64", "arm64"),
                           help="Debian architecture override (must match ELF architecture)")
            p.add_argument("--deb-extra-dep", help="exact extra Debian runtime dependency, e.g. native VectorScan package")
            p.add_argument("--maintainer", help="Debian Maintainer field override")
        elif name == "rpm":
            p.add_argument("--arch", choices=("x86_64", "aarch64"),
                           help="RPM architecture override (must match ELF architecture)")
            p.add_argument("--rpm-extra-require", help="exact extra RPM runtime requirement, e.g. native VectorScan package")
            p.add_argument("--rpm-release", default="1", help="RPM Release field (default: 1)")
        else:
            p.add_argument("--deb-arch", choices=("amd64", "arm64"),
                           help="DEB architecture override")
            p.add_argument("--rpm-arch", choices=("x86_64", "aarch64"),
                           help="RPM architecture override")
            p.add_argument("--deb-extra-dep", help="exact extra Debian runtime dependency, e.g. native VectorScan package")
            p.add_argument("--rpm-extra-require", help="exact extra RPM runtime requirement, e.g. native VectorScan package")
            p.add_argument("--rpm-release", default="1", help="RPM Release field (default: 1)")
            p.add_argument("--maintainer", help="Debian Maintainer field override")
    return parser


def doctor_main(args: argparse.Namespace) -> int:
    root = project_root()
    go = selected_go(args.go_bin)
    pre = preflight(root, args.format, go, args.vectorscan, args.hsm)
    fingerprint = source_fingerprint(root)
    print(json.dumps({**pre, "source_fingerprint_sha256": fingerprint}, indent=2, sort_keys=True))
    print("PACKAGE_HOST_READY")
    return 0


def build_main(args: argparse.Namespace) -> int:
    root = project_root()
    fingerprint = source_fingerprint(root)
    epoch, epoch_source = derive_epoch(root, args.source_date_epoch)
    version = validate_version(
        args.version or os.environ.get("VERSION") or default_version(fingerprint, epoch)
    )
    commit = args.commit or os.environ.get("COMMIT") or derive_commit(root, fingerprint)
    go = selected_go(args.go_bin)
    pre = preflight(root, args.command, go, args.vectorscan, args.hsm)
    env = build_env(go, args, version, commit, epoch)

    print("==> WAF package build plan")
    print(f"target={args.command}")
    print(f"version={version}")
    print(f"source_identity={commit}")
    print(f"source_fingerprint_sha256={fingerprint}")
    print(f"source_date_epoch={epoch} ({epoch_source})")
    print(f"go={pre['selected_go']}")
    print(f"coraza={pre['coraza_version']}")
    print(f"vectorscan={args.vectorscan}")
    print(f"hsm={args.hsm}")
    print(f"offline={args.offline}")

    run_stream(["./build.sh"], root, env)
    provenance = load_provenance(root)

    packages: list[Path] = []
    targets = [args.command] if args.command != "all" else ["deb", "rpm"]
    for target in targets:
        output = run_stream(package_command(target, args), root, env)
        key = "DEB_PACKAGE" if target == "deb" else "RPM_PACKAGE"
        value = parse_key(output, key)
        if not value:
            raise BuildError(f"{target} builder did not report {key}")
        pkg = Path(value)
        if not pkg.is_absolute():
            pkg = (root / pkg).resolve()
        if not pkg.is_file():
            raise BuildError(f"reported package does not exist: {pkg}")
        packages.append(pkg)

    report = write_report(
        root, args.command, pre, fingerprint, version, commit, epoch,
        epoch_source, provenance, packages, args.output_dir,
    )
    print("\nPACKAGE_BUILD_PASS")
    for pkg in packages:
        print(f"package={pkg}")
        print(f"sha256={sha256(pkg)}")
    print(f"report={report}")
    return 0


def main() -> int:
    if sys.version_info < MIN_PYTHON:
        raise BuildError(f"Python {MIN_PYTHON[0]}.{MIN_PYTHON[1]}+ is required")
    parser = build_parser()
    args = parser.parse_args()
    return doctor_main(args) if args.command == "doctor" else build_main(args)


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except BuildError as exc:
        print(f"PACKAGE_BUILD_BLOCKED: {exc}", file=sys.stderr)
        raise SystemExit(2)
