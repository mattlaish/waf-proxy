#!/usr/bin/env python3
"""End-to-end clean-host package qualification for waf-proxy.

PASS is only possible on an explicitly acknowledged dedicated clean host after
real native package-manager install/upgrade/remove transactions, systemd start,
authenticated admin API access and reverse-proxy traffic. Preflight is NOT_RUN.
"""
from __future__ import annotations

import argparse
import dataclasses
import hashlib
import http.client
import json
import os
import pathlib
import platform
import re
import shutil
import signal
import subprocess
import sys
import tempfile
import time
from typing import Iterable

PACKAGE = "waf-proxy"
ACK = "I_UNDERSTAND_THIS_MUTATES_A_DEDICATED_CLEAN_HOST"
ADMIN_ADDR = ("127.0.0.1", 19090)
DATA_ADDR = ("127.0.0.1", 18081)
BACKEND_ADDR = ("127.0.0.1", 18080)
STATE_MARKER = pathlib.Path("/var/lib/waf-proxy/.clean-host-qualification")
SECRET_PATH = pathlib.Path("/etc/waf/waf-proxy.env")
CONFIG_PATH = pathlib.Path("/etc/waf/config.json")
CRS_PATH = pathlib.Path("/etc/waf/crs")
DROPIN_DIR = pathlib.Path("/etc/systemd/system/waf-proxy.service.d")
DROPIN_PATH = DROPIN_DIR / "90-clean-host-qualification.conf"

PLATFORMS = {
    "debian-12": {"format": "deb", "id": "debian", "version": r"^12(?:\.|$)", "selinux": False},
    "ubuntu-22.04": {"format": "deb", "id": "ubuntu", "version": r"^22\.04(?:\.|$)", "selinux": False},
    "ubuntu-24.04": {"format": "deb", "id": "ubuntu", "version": r"^24\.04(?:\.|$)", "selinux": False},
    "rhel-9": {"format": "rpm", "id": "rhel", "version": r"^9(?:\.|$)", "selinux": True},
    "rocky-9": {"format": "rpm", "id": "rocky", "version": r"^9(?:\.|$)", "selinux": True},
    "almalinux-9": {"format": "rpm", "id": "almalinux", "version": r"^9(?:\.|$)", "selinux": True},
    "oraclelinux-9": {"format": "rpm", "id": "ol", "version": r"^9(?:\.|$)", "selinux": True},
}

FORBIDDEN_FETCHERS = {"apt", "apt-get", "dnf", "yum", "curl", "wget", "git"}


class QualificationError(RuntimeError):
    pass


class Blocked(QualificationError):
    pass


@dataclasses.dataclass(frozen=True)
class PackageMeta:
    path: str
    version: str
    architecture: str
    sha256: str


def run(cmd: list[str], *, check: bool = True, env: dict[str, str] | None = None) -> subprocess.CompletedProcess[str]:
    if pathlib.Path(cmd[0]).name in FORBIDDEN_FETCHERS:
        raise QualificationError(f"network/dependency fetcher is forbidden in clean-host qualification: {cmd[0]}")
    p = subprocess.run(cmd, text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE, env=env)
    if check and p.returncode != 0:
        raise QualificationError(f"command failed rc={p.returncode}: {' '.join(cmd[:3])}: {p.stderr.strip()[:500]}")
    return p


def require_commands(names: Iterable[str]) -> None:
    missing = [n for n in names if shutil.which(n) is None]
    if missing:
        raise Blocked("missing required commands: " + ",".join(missing))


def sha256_file(path: pathlib.Path) -> str:
    h = hashlib.sha256()
    with path.open("rb") as f:
        for chunk in iter(lambda: f.read(1024 * 1024), b""):
            h.update(chunk)
    return h.hexdigest()


def tree_digest(root: pathlib.Path) -> tuple[str, int]:
    files = sorted(p for p in root.rglob("*") if p.is_file() and not p.is_symlink())
    h = hashlib.sha256()
    for p in files:
        rel = p.relative_to(root).as_posix().encode()
        h.update(len(rel).to_bytes(4, "big")); h.update(rel)
        digest = bytes.fromhex(sha256_file(p)); h.update(digest)
    return h.hexdigest(), len(files)


def os_release(path: pathlib.Path = pathlib.Path("/etc/os-release")) -> dict[str, str]:
    out: dict[str, str] = {}
    if not path.exists():
        return out
    for line in path.read_text(encoding="utf-8", errors="replace").splitlines():
        if "=" not in line or line.lstrip().startswith("#"):
            continue
        k, v = line.split("=", 1)
        out[k] = v.strip().strip('"')
    return out


def validate_platform(name: str, release: dict[str, str] | None = None) -> dict[str, str]:
    spec = PLATFORMS[name]
    release = release or os_release()
    got_id = release.get("ID", "unknown").lower()
    got_ver = release.get("VERSION_ID", "unknown")
    if got_id != spec["id"] or re.match(str(spec["version"]), got_ver) is None:
        raise Blocked(f"platform {name} requires ID={spec['id']} VERSION_ID~={spec['version']}; detected {got_id} {got_ver}")
    return {"id": got_id, "version_id": got_ver, "pretty_name": release.get("PRETTY_NAME", f"{got_id} {got_ver}")}


def selinux_state(required: bool) -> str:
    if not required:
        return "NOT_REQUIRED"
    require_commands(["getenforce"])
    state = run(["getenforce"]).stdout.strip()
    if state != "Enforcing":
        raise Blocked(f"RHEL-family clean-host acceptance requires SELinux Enforcing; detected {state or 'unknown'}")
    return state


def package_meta(fmt: str, path: pathlib.Path) -> PackageMeta:
    if not path.is_file():
        raise Blocked(f"package not found: {path}")
    if fmt == "deb":
        require_commands(["dpkg-deb"])
        p = run(["dpkg-deb", "-f", str(path), "Package", "Version", "Architecture"])
        lines = [x.strip() for x in p.stdout.splitlines() if x.strip()]
        if len(lines) != 3 or lines[0] != PACKAGE:
            raise QualificationError(f"unexpected DEB metadata: {lines}")
        return PackageMeta(str(path.resolve()), lines[1], lines[2], sha256_file(path))
    require_commands(["rpm"])
    p = run(["rpm", "-qp", "--qf", "%{NAME}\n%{VERSION}-%{RELEASE}\n%{ARCH}\n", str(path)])
    lines = [x.strip() for x in p.stdout.splitlines() if x.strip()]
    if len(lines) != 3 or lines[0] != PACKAGE:
        raise QualificationError(f"unexpected RPM metadata: {lines}")
    return PackageMeta(str(path.resolve()), lines[1], lines[2], sha256_file(path))


def package_installed(fmt: str) -> bool:
    if fmt == "deb":
        p = run(["dpkg-query", "-W", "-f=${Status}", PACKAGE], check=False)
        return p.returncode == 0 and "install ok installed" in p.stdout
    return run(["rpm", "-q", PACKAGE], check=False).returncode == 0


def install_or_upgrade(fmt: str, path: pathlib.Path) -> None:
    if fmt == "deb":
        run(["dpkg", "-i", "--force-confold", str(path)])
    else:
        run(["rpm", "-Uvh", "--replacepkgs", str(path)])


def remove_package(fmt: str) -> None:
    if fmt == "deb":
        run(["dpkg", "-r", PACKAGE])
    else:
        run(["rpm", "-e", PACKAGE])


def validate_crs(root: pathlib.Path) -> tuple[str, int]:
    if not root.is_dir():
        raise Blocked(f"CRS directory not found: {root}")
    if not (root / "crs-setup.conf").is_file():
        raise Blocked("CRS directory must contain crs-setup.conf")
    if not list((root / "rules").glob("*.conf")):
        raise Blocked("CRS directory must contain rules/*.conf")
    return tree_digest(root)


def copy_crs(src: pathlib.Path, *, restore_selinux: bool = False) -> None:
    CRS_PATH.mkdir(parents=True, exist_ok=True)
    for child in CRS_PATH.iterdir():
        if child.is_dir() and not child.is_symlink():
            shutil.rmtree(child)
        else:
            child.unlink()
    for child in src.iterdir():
        dest = CRS_PATH / child.name
        if child.is_symlink():
            raise QualificationError(f"CRS symlink rejected: {child}")
        if child.is_dir():
            shutil.copytree(child, dest, symlinks=False)
        elif child.is_file():
            shutil.copy2(child, dest)
    run(["chown", "-R", "root:waf", str(CRS_PATH)])
    run(["chmod", "-R", "g+rX", str(CRS_PATH)])
    if restore_selinux:
        require_commands(["restorecon"])
        run(["restorecon", "-R", str(CRS_PATH)])


def read_secret() -> bytes:
    if not SECRET_PATH.is_file():
        raise QualificationError("package did not create /etc/waf/waf-proxy.env")
    data = SECRET_PATH.read_bytes()
    if b"WAF_ADMIN_TOKEN=" not in data:
        raise QualificationError("unexpected admin secret file shape")
    return data


def token_from_secret(data: bytes) -> str:
    for line in data.decode("utf-8", errors="strict").splitlines():
        if line.startswith("WAF_ADMIN_TOKEN="):
            token = line.split("=", 1)[1].strip()
            if token:
                return token
    raise QualificationError("admin token missing")


def configure_for_smoke() -> str:
    obj = json.loads(CONFIG_PATH.read_text(encoding="utf-8"))
    obj["passive_discovery_enabled"] = False
    obj.setdefault("vector_acceleration", {})["mode"] = "off"
    obj.setdefault("ai", {})["enabled"] = False
    obj["nodes"] = [{"name": "clean-host-backend", "host": BACKEND_ADDR[0]}]
    policy = (obj.get("policies") or [{"name": "default", "rules_path": "/etc/waf/coraza.conf", "paranoia_level": 1, "request_body_limit": 0, "exclusions": [], "response_body_inspection": "on", "response_body_limit": 1048576}])[0]
    policy["name"] = "default"; policy["rules_path"] = "/etc/waf/coraza.conf"
    obj["policies"] = [policy]
    pool = (obj.get("pools") or [{}])[0]
    pool.update({"name": "clean-host-pool", "scheme": "http", "lb_method": "round_robin", "backend_tls": {}, "members": [{"node": "clean-host-backend", "port": BACKEND_ADDR[1], "weight": 1}]})
    pool["monitor"] = {"type": "none", "interval_sec": 5, "timeout_sec": 2, "rise": 1, "fall": 1}
    obj["pools"] = [pool]
    site = (obj.get("sites") or [{}])[0]
    site.update({"name": "clean-host", "listen": f"{DATA_ADDR[0]}:{DATA_ADDR[1]}", "hostnames": ["qualification.local"], "pool": "clean-host-pool", "policy": "default", "preserve_host": True, "manage_ip": False, "engine_mode": "DetectionOnly", "ai_mode": "off", "tls_cert": "", "tls_key": "", "page_policies": [], "tls_key_provider": {}})
    obj["sites"] = [site]
    CONFIG_PATH.write_text(json.dumps(obj, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    run(["chown", "waf:waf", str(CONFIG_PATH)]); run(["chmod", "0600", str(CONFIG_PATH)])
    return sha256_file(CONFIG_PATH)


def write_dropin() -> None:
    DROPIN_DIR.mkdir(parents=True, exist_ok=True)
    DROPIN_PATH.write_text(f"[Service]\nEnvironment=WAF_ADMIN_ADDR={ADMIN_ADDR[0]}:{ADMIN_ADDR[1]}\n", encoding="utf-8")
    os.chmod(DROPIN_PATH, 0o644)
    run(["systemctl", "daemon-reload"])


def start_backend() -> subprocess.Popen[str]:
    code = r'''import http.server, socketserver
class H(http.server.BaseHTTPRequestHandler):
 def do_GET(self):
  body=b"waf-clean-host-backend-ok\n"
  self.send_response(200); self.send_header("Content-Type","text/plain"); self.send_header("Content-Length",str(len(body))); self.end_headers(); self.wfile.write(body)
 def log_message(self, fmt, *args): pass
class S(socketserver.TCPServer): allow_reuse_address=True
with S(("127.0.0.1",18080),H) as s: s.serve_forever()
'''
    p = subprocess.Popen([sys.executable, "-c", code], text=True, stdout=subprocess.DEVNULL, stderr=subprocess.PIPE)
    deadline = time.time() + 5
    while time.time() < deadline:
        if p.poll() is not None:
            err = p.stderr.read() if p.stderr else ""
            raise QualificationError(f"qualification backend exited: {err[:300]}")
        try:
            status, body = http_get(BACKEND_ADDR[0], BACKEND_ADDR[1], "/", {})
            if status == 200 and b"waf-clean-host-backend-ok" in body:
                return p
        except OSError:
            pass
        time.sleep(0.1)
    p.terminate()
    raise QualificationError("qualification backend did not become ready")


def http_get(host: str, port: int, path: str, headers: dict[str, str]) -> tuple[int, bytes]:
    conn = http.client.HTTPConnection(host, port, timeout=3)
    try:
        conn.request("GET", path, headers=headers)
        resp = conn.getresponse(); body = resp.read(1024 * 1024)
        return resp.status, body
    finally:
        conn.close()


def wait_health() -> dict[str, object]:
    deadline = time.time() + 25
    last = ""
    while time.time() < deadline:
        try:
            status, body = http_get(ADMIN_ADDR[0], ADMIN_ADDR[1], "/healthz", {})
            last = body.decode(errors="replace")[:300]
            if status == 200:
                obj = json.loads(body)
                if obj.get("status") == "ok":
                    return obj
        except Exception as e:
            last = str(e)
        time.sleep(0.5)
    raise QualificationError(f"health endpoint did not become ready: {last}")


def first_login(token: str) -> bool:
    status, body = http_get(ADMIN_ADDR[0], ADMIN_ADDR[1], "/api/whoami", {"Authorization": "Bearer " + token})
    if status != 200:
        return False
    obj = json.loads(body)
    return obj.get("role") == "admin" and obj.get("user") == "(token)"


def traffic_smoke() -> bool:
    status, body = http_get(DATA_ADDR[0], DATA_ADDR[1], "/distribution-qualification", {"Host": "qualification.local"})
    return status == 200 and b"waf-clean-host-backend-ok" in body


def doctor_check() -> str:
    doctor = shutil.which("waf-doctor") or "/usr/sbin/waf-doctor"
    if not pathlib.Path(doctor).is_file():
        raise QualificationError("waf-doctor is missing after package install")
    p = run([doctor, "--check"])
    return hashlib.sha256(p.stdout.encode()).hexdigest()


def systemctl_active() -> bool:
    return run(["systemctl", "is-active", "--quiet", "waf-proxy.service"], check=False).returncode == 0


def unit_exists() -> bool:
    p = run(["systemctl", "show", "-p", "LoadState", "--value", "waf-proxy.service"], check=False)
    return p.returncode == 0 and p.stdout.strip() not in {"", "not-found"}


def preserved_config_after_remove(fmt: str, expected: str) -> tuple[bool, str]:
    if CONFIG_PATH.is_file() and sha256_file(CONFIG_PATH) == expected:
        return True, str(CONFIG_PATH)
    if fmt == "rpm":
        saved = pathlib.Path(str(CONFIG_PATH) + ".rpmsave")
        if saved.is_file() and sha256_file(saved) == expected:
            return True, str(saved)
    return False, ""


def criterion(name: str, status: str, detail: str = "") -> dict[str, str]:
    return {"name": name, "status": status, "detail": detail}


def report_base(args: argparse.Namespace, platform_info: dict[str, str], fmt: str, base: PackageMeta, cand: PackageMeta, crs_digest: str, crs_files: int, selinux: str) -> dict[str, object]:
    return {
        "schema_version": 1,
        "evidence_type": "clean-host-distribution-qualification",
        "platform": args.platform,
        "package_format": fmt,
        "status": "NOT_RUN",
        "executed": False,
        "host": {"os": platform_info, "kernel": platform.release(), "machine": platform.machine(), "selinux": selinux},
        "packages": {"baseline": dataclasses.asdict(base), "candidate": dataclasses.asdict(cand)},
        "crs": {"tree_sha256": crs_digest, "file_count": crs_files},
        "truth_boundary": "PASS requires real dedicated clean-host install/start/authenticated-admin/proxy-traffic/upgrade/remove execution; preflight/container/source simulation is NOT_RUN",
        "criteria": [],
    }


def clean_host_preconditions(fmt: str) -> None:
    if package_installed(fmt):
        raise Blocked("waf-proxy is already installed; Slice D requires a clean host")
    leftovers = [str(p) for p in (pathlib.Path("/etc/waf"), pathlib.Path("/var/lib/waf-proxy")) if p.exists()]
    if leftovers:
        raise Blocked("clean-host paths already exist: " + ",".join(leftovers))


def execute(args: argparse.Namespace, fmt: str, report: dict[str, object]) -> None:
    if os.geteuid() != 0:
        raise Blocked("--execute requires root")
    if args.ack != ACK:
        raise Blocked("--execute requires exact clean-host acknowledgement")
    require_commands(["systemctl", "python3", "chown", "chmod"] + (["dpkg", "dpkg-query"] if fmt == "deb" else ["rpm"]))
    clean_host_preconditions(fmt)
    criteria: list[dict[str, str]] = report["criteria"]  # type: ignore[assignment]
    backend: subprocess.Popen[str] | None = None
    secret_initial: bytes | None = None
    try:
        install_or_upgrade(fmt, pathlib.Path(args.baseline))
        criteria.append(criterion("baseline_install", "PASS"))
        if systemctl_active():
            raise QualificationError("fresh package install unexpectedly auto-started waf-proxy")
        criteria.append(criterion("fresh_install_no_autostart", "PASS"))

        secret_initial = read_secret(); token = token_from_secret(secret_initial)
        copy_crs(pathlib.Path(args.crs_dir), restore_selinux=bool(PLATFORMS[args.platform]["selinux"]))
        config_digest = configure_for_smoke()
        write_dropin()
        doctor_digest = doctor_check()
        criteria.append(criterion("package_doctor_check", "PASS", f"stdout_sha256={doctor_digest}"))

        backend = start_backend()
        run(["systemctl", "enable", "--now", "waf-proxy.service"])
        if not systemctl_active():
            raise QualificationError("waf-proxy service did not become active")
        criteria.append(criterion("explicit_service_start", "PASS"))
        wait_health(); criteria.append(criterion("healthz", "PASS"))
        if not first_login(token):
            raise QualificationError("break-glass first-login API authentication failed")
        criteria.append(criterion("first_login_break_glass", "PASS"))
        if not traffic_smoke():
            raise QualificationError("reverse-proxy traffic smoke failed")
        criteria.append(criterion("reverse_proxy_traffic", "PASS"))

        STATE_MARKER.parent.mkdir(parents=True, exist_ok=True)
        marker = os.urandom(32).hex().encode() + b"\n"
        STATE_MARKER.write_bytes(marker); run(["chown", "waf:waf", str(STATE_MARKER)])
        marker_digest = hashlib.sha256(marker).hexdigest()

        install_or_upgrade(fmt, pathlib.Path(args.candidate))
        criteria.append(criterion("candidate_upgrade", "PASS"))
        if not systemctl_active():
            raise QualificationError("service not active after upgrade")
        if SECRET_PATH.read_bytes() != secret_initial:
            raise QualificationError("admin secret changed during upgrade")
        criteria.append(criterion("admin_secret_preserved_on_upgrade", "PASS"))
        if sha256_file(CONFIG_PATH) != config_digest:
            raise QualificationError("operator config changed during upgrade")
        criteria.append(criterion("operator_config_preserved_on_upgrade", "PASS"))
        if not STATE_MARKER.is_file() or sha256_file(STATE_MARKER) != marker_digest:
            raise QualificationError("persistent state marker changed during upgrade")
        criteria.append(criterion("persistent_state_preserved_on_upgrade", "PASS"))
        wait_health(); criteria.append(criterion("healthz_after_upgrade", "PASS"))
        if not first_login(token):
            raise QualificationError("first-login token failed after upgrade")
        criteria.append(criterion("admin_auth_after_upgrade", "PASS"))
        if not traffic_smoke():
            raise QualificationError("traffic smoke failed after upgrade")
        criteria.append(criterion("traffic_after_upgrade", "PASS"))

        run(["systemctl", "disable", "--now", "waf-proxy.service"], check=False)
        remove_package(fmt)
        criteria.append(criterion("package_remove", "PASS"))
        if package_installed(fmt):
            raise QualificationError("package still installed after removal")
        if systemctl_active() or unit_exists():
            raise QualificationError("service still active/loadable after package removal")
        criteria.append(criterion("service_removed", "PASS"))
        if not SECRET_PATH.is_file() or SECRET_PATH.read_bytes() != secret_initial:
            raise QualificationError("break-glass secret was not preserved by non-purge removal")
        criteria.append(criterion("admin_secret_preserved_on_remove", "PASS"))
        if not STATE_MARKER.is_file() or sha256_file(STATE_MARKER) != marker_digest:
            raise QualificationError("persistent state was not preserved by package removal")
        criteria.append(criterion("persistent_state_preserved_on_remove", "PASS"))
        if not (CRS_PATH / "crs-setup.conf").is_file():
            raise QualificationError("operator-provisioned CRS was removed")
        criteria.append(criterion("crs_preserved_on_remove", "PASS"))
        cfg_ok, cfg_path = preserved_config_after_remove(fmt, config_digest)
        if not cfg_ok:
            raise QualificationError("operator config not preserved by package removal")
        criteria.append(criterion("operator_config_preserved_on_remove", "PASS", cfg_path))

        report["status"] = "PASS"
        report["executed"] = True
    finally:
        if backend is not None:
            backend.terminate()
            try: backend.wait(timeout=3)
            except subprocess.TimeoutExpired: backend.kill()
        DROPIN_PATH.unlink(missing_ok=True)
        if DROPIN_DIR.exists():
            try: DROPIN_DIR.rmdir()
            except OSError: pass
        if shutil.which("systemctl"):
            run(["systemctl", "daemon-reload"], check=False)


def parse_args(argv: list[str]) -> argparse.Namespace:
    p = argparse.ArgumentParser()
    p.add_argument("--platform", required=True, choices=sorted(PLATFORMS))
    p.add_argument("--baseline", required=True)
    p.add_argument("--candidate", required=True)
    p.add_argument("--crs-dir", required=True)
    p.add_argument("--output", required=True)
    p.add_argument("--execute", action="store_true")
    p.add_argument("--ack", default="")
    return p.parse_args(argv)


def main(argv: list[str]) -> int:
    args = parse_args(argv)
    spec = PLATFORMS[args.platform]; fmt = str(spec["format"])
    out = pathlib.Path(args.output)
    try:
        platform_info = validate_platform(args.platform)
        selinux = selinux_state(bool(spec["selinux"]))
        base = package_meta(fmt, pathlib.Path(args.baseline))
        cand = package_meta(fmt, pathlib.Path(args.candidate))
        if base.version == cand.version:
            raise Blocked("baseline and candidate package versions must differ")
        crs_digest, crs_files = validate_crs(pathlib.Path(args.crs_dir))
        report = report_base(args, platform_info, fmt, base, cand, crs_digest, crs_files, selinux)
        if args.execute:
            execute(args, fmt, report)
        else:
            report["criteria"] = [criterion("preflight", "PASS", "inputs/platform validated; no host mutation executed")]
        out.parent.mkdir(parents=True, exist_ok=True)
        out.write_text(json.dumps(report, indent=2, sort_keys=True) + "\n", encoding="utf-8")
        print(f"{report['status']} {out}")
        return 0 if report["status"] == "PASS" else 3
    except Blocked as e:
        report = {"schema_version": 1, "evidence_type": "clean-host-distribution-qualification", "platform": args.platform, "status": "BLOCKED", "executed": False, "reason": str(e), "truth_boundary": "BLOCKED is not PASS"}
        out.parent.mkdir(parents=True, exist_ok=True); out.write_text(json.dumps(report, indent=2, sort_keys=True) + "\n")
        print(f"BLOCKED: {e}", file=sys.stderr); return 3
    except Exception as e:
        report = {"schema_version": 1, "evidence_type": "clean-host-distribution-qualification", "platform": args.platform, "status": "FAIL", "executed": bool(args.execute), "reason": str(e), "truth_boundary": "FAIL is not PASS"}
        out.parent.mkdir(parents=True, exist_ok=True); out.write_text(json.dumps(report, indent=2, sort_keys=True) + "\n")
        print(f"FAIL: {e}", file=sys.stderr); return 1


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
