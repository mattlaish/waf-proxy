#!/usr/bin/env python3
"""Generate deterministic release provenance and SBOM evidence.

This tool creates integrity-bound metadata. It does not by itself authenticate
who produced the evidence; authenticity requires detached signature
verification with an approved public key.
"""
from __future__ import annotations

import argparse
import datetime as dt
import hashlib
import json
import os
import pathlib
import shutil
import subprocess
import uuid

from source_manifest import manifest_text as build_manifest_text, source_files, sha256_file

SCHEMA_VERSION = 2
ALLOWED_GOVULN_STATUS = {"PASS", "FAIL", "BLOCKED", "NOT_RUN"}


def run_version(argv: list[str]) -> dict:
    exe = shutil.which(argv[0])
    if not exe:
        return {"status": "NOT_AVAILABLE", "command": argv[0]}
    try:
        env = os.environ.copy()
        if argv[0] == "go":
            # Never let release evidence trigger Go's auto-toolchain download;
            # network error details (including ephemeral UDP ports) are not
            # reproducible metadata and are not trustworthy toolchain evidence.
            env["GOTOOLCHAIN"] = "local"
        cp = subprocess.run(argv, stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
                            text=True, timeout=8, check=False, env=env)
        text = " ".join(cp.stdout.strip().split())
        return {"status": "DETECTED" if cp.returncode == 0 else "ERROR",
                "command": exe, "exit_code": cp.returncode, "version": text[:1024]}
    except Exception as exc:  # pragma: no cover
        return {"status": "ERROR", "command": exe, "error": type(exc).__name__}


def pkg_config_version(name: str) -> dict:
    exe = shutil.which("pkg-config")
    if not exe:
        return {"status": "NOT_AVAILABLE", "component": name}
    cp = subprocess.run([exe, "--modversion", name], stdout=subprocess.PIPE,
                        stderr=subprocess.PIPE, text=True, check=False)
    if cp.returncode:
        return {"status": "NOT_AVAILABLE", "component": name}
    version = cp.stdout.strip()
    pc = subprocess.run([exe, "--variable=pcfiledir", name], stdout=subprocess.PIPE,
                        stderr=subprocess.DEVNULL, text=True, check=False).stdout.strip()
    return {"status": "DETECTED", "component": name, "version": version,
            "pcfiledir": pc or None}


def parse_go_mod(path: pathlib.Path) -> tuple[str, str, list[dict]]:
    text = path.read_text(encoding="utf-8")
    module = ""
    goversion = ""
    reqs: list[dict] = []
    in_require = False
    for raw in text.splitlines():
        line = raw.strip()
        if line.startswith("module "):
            module = line.split(None, 1)[1]
        elif line.startswith("go "):
            goversion = line.split(None, 1)[1]
        elif line == "require (":
            in_require = True
        elif in_require and line == ")":
            in_require = False
        elif line.startswith("require "):
            fields = line.split()
            if len(fields) >= 3:
                reqs.append({"module": fields[1], "version": fields[2], "indirect": "// indirect" in line})
        elif in_require and line and not line.startswith("//"):
            fields = line.split()
            if len(fields) >= 2:
                reqs.append({"module": fields[0], "version": fields[1], "indirect": "// indirect" in line})
    reqs.sort(key=lambda x: x["module"])
    return module, goversion, reqs


def tree_digest(path: pathlib.Path) -> tuple[str, int]:
    entries: list[str] = []
    count = 0
    if path.is_file():
        return sha256_file(path), 1
    for p in sorted((x for x in path.rglob("*") if x.is_file() and not x.is_symlink()),
                    key=lambda p: p.relative_to(path).as_posix()):
        entries.append(f"{sha256_file(p)}  {p.relative_to(path).as_posix()}\n")
        count += 1
    return hashlib.sha256("".join(entries).encode()).hexdigest(), count


def validate_govuln_evidence(path: pathlib.Path, source_manifest_sha256: str) -> dict:
    try:
        obj = json.loads(path.read_text(encoding="utf-8"))
    except Exception as exc:
        raise SystemExit(f"invalid govulncheck evidence JSON: {type(exc).__name__}")
    required = {"schema_version", "evidence_type", "status", "tool", "tool_version",
                "source_manifest_sha256", "executed_at", "result_digest", "output"}
    missing = sorted(required - set(obj))
    if missing:
        raise SystemExit(f"govulncheck evidence missing fields: {', '.join(missing)}")
    if obj.get("schema_version") != 1 or obj.get("evidence_type") != "govulncheck":
        raise SystemExit("govulncheck evidence schema/type mismatch")
    if obj.get("tool") != "govulncheck" or obj.get("status") not in ALLOWED_GOVULN_STATUS:
        raise SystemExit("govulncheck evidence tool/status invalid")
    if obj.get("source_manifest_sha256") != source_manifest_sha256:
        raise SystemExit("govulncheck evidence is not bound to this source manifest")
    raw = obj.get("output")
    if not isinstance(raw, str):
        raise SystemExit("govulncheck evidence output must be text")
    if hashlib.sha256(raw.encode("utf-8")).hexdigest() != obj.get("result_digest"):
        raise SystemExit("govulncheck evidence result digest mismatch")
    if obj["status"] in {"PASS", "FAIL"} and obj.get("tool_version") in {"", "NOT_AVAILABLE", None}:
        raise SystemExit("executed govulncheck evidence requires tool_version")
    return obj


def spdx(release: str, module: str, deps: list[dict], manifest_sha: str, variant: str,
         artifact: pathlib.Path | None) -> dict:
    ns = uuid.uuid5(uuid.NAMESPACE_URL, f"{release}\0{module}\0{manifest_sha}\0{variant}")
    packages = [{
        "SPDXID": "SPDXRef-Package-WAF", "name": module or "waf-proxy",
        "downloadLocation": "NOASSERTION", "filesAnalyzed": False,
        "licenseConcluded": "NOASSERTION", "licenseDeclared": "NOASSERTION",
        "versionInfo": release,
        "externalRefs": [{"referenceCategory": "OTHER", "referenceType": "build-variant", "referenceLocator": variant}],
    }]
    relationships = []
    for i, dep in enumerate(deps, 1):
        sid = f"SPDXRef-GoModule-{i}"
        purl = f"pkg:golang/{dep['module']}@{dep['version']}"
        packages.append({
            "SPDXID": sid, "name": dep["module"], "versionInfo": dep["version"],
            "downloadLocation": "NOASSERTION", "filesAnalyzed": False,
            "licenseConcluded": "NOASSERTION", "licenseDeclared": "NOASSERTION",
            "externalRefs": [{"referenceCategory": "PACKAGE-MANAGER", "referenceType": "purl", "referenceLocator": purl}],
        })
        relationships.append({"spdxElementId": "SPDXRef-Package-WAF", "relationshipType": "DEPENDS_ON", "relatedSpdxElement": sid})
    if artifact and artifact.is_file():
        packages[0]["checksums"] = [{"algorithm": "SHA256", "checksumValue": sha256_file(artifact)}]
    return {
        "spdxVersion": "SPDX-2.3", "dataLicense": "CC0-1.0", "SPDXID": "SPDXRef-DOCUMENT",
        "name": f"{release}-sbom", "documentNamespace": f"https://waf-proxy.invalid/spdx/{ns}",
        "creationInfo": {"creators": ["Tool: waf-proxy/tools/release_evidence.py"], "created": "1970-01-01T00:00:00Z"},
        "packages": packages, "relationships": relationships,
    }


def cyclone(release: str, module: str, deps: list[dict], manifest_sha: str, variant: str,
            artifact: pathlib.Path | None) -> dict:
    serial = uuid.uuid5(uuid.NAMESPACE_URL, f"{release}\0{module}\0{manifest_sha}\0{variant}")
    root_comp = {
        "type": "application", "bom-ref": "pkg:waf-proxy", "name": module or "waf-proxy",
        "version": release, "properties": [{"name": "waf.build.variant", "value": variant}],
    }
    if artifact and artifact.is_file():
        root_comp["hashes"] = [{"alg": "SHA-256", "content": sha256_file(artifact)}]
    comps, refs = [], []
    for dep in deps:
        purl = f"pkg:golang/{dep['module']}@{dep['version']}"
        comps.append({"type": "library", "bom-ref": purl, "name": dep["module"],
                      "version": dep["version"], "purl": purl, "scope": "required"})
        refs.append(purl)
    return {
        "bomFormat": "CycloneDX", "specVersion": "1.5", "serialNumber": f"urn:uuid:{serial}", "version": 1,
        "metadata": {"component": root_comp, "tools": {"components": [{"type": "application", "name": "waf-proxy release_evidence.py", "version": "2"}]}},
        "components": comps, "dependencies": [{"ref": "pkg:waf-proxy", "dependsOn": refs}],
    }


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--root", required=True)
    ap.add_argument("--out", required=True)
    ap.add_argument("--release", required=True)
    ap.add_argument("--variant", choices=["source", "portable-coraza", "native-vectorscan"], required=True)
    ap.add_argument("--artifact")
    ap.add_argument("--crs")
    ap.add_argument("--source-date-epoch", type=int)
    ap.add_argument("--govulncheck-json")
    args = ap.parse_args()

    root = pathlib.Path(args.root).resolve()
    out = pathlib.Path(args.out).resolve()
    artifact = pathlib.Path(args.artifact).resolve() if args.artifact else None
    if not (root / "go.mod").is_file():
        raise SystemExit("go.mod missing from release root")
    if artifact is None and args.variant != "source":
        raise SystemExit("source archive evidence must use --variant source; binary identity requires --artifact")
    out.mkdir(parents=True, exist_ok=True)

    module, go_directive, deps = parse_go_mod(root / "go.mod")
    manifest = build_manifest_text(root)
    manifest_sha = hashlib.sha256(manifest.encode()).hexdigest()
    files = source_files(root)
    (out / "SOURCE_MANIFEST.sha256").write_text(manifest, encoding="utf-8")

    coraza = next((x for x in deps if x["module"] == "github.com/corazawaf/coraza/v3"), None)
    versions = {
        "go_directive": go_directive,
        "go_runtime": run_version(["go", "version"]),
        "coraza": {"status": "PINNED" if coraza else "NOT_FOUND", "version": coraza["version"] if coraza else None},
        "vectorscan": pkg_config_version("libhs"),
        "nginx": run_version(["nginx", "-v"]),
        "openssl": run_version(["openssl", "version"]),
    }
    crs = {"status": "NOT_PROVIDED"}
    if args.crs:
        cp = pathlib.Path(args.crs).resolve()
        if cp.exists():
            digest, count = tree_digest(cp)
            crs = {"status": "HASHED", "path": cp.name, "sha256_tree": digest, "file_count": count}
        else:
            crs = {"status": "NOT_FOUND", "path": str(cp)}
    versions["crs"] = crs

    govuln = {
        "schema_version": 1, "evidence_type": "govulncheck", "status": "NOT_RUN",
        "tool": "govulncheck", "tool_version": "NOT_AVAILABLE",
        "source_manifest_sha256": manifest_sha, "executed_at": None,
        "result_digest": hashlib.sha256(b"").hexdigest(), "output": "",
        "reason": "no govulncheck evidence supplied",
    }
    if args.govulncheck_json:
        govuln = validate_govuln_evidence(pathlib.Path(args.govulncheck_json), manifest_sha)

    epoch = args.source_date_epoch
    generated = dt.datetime.fromtimestamp(epoch, tz=dt.timezone.utc).isoformat().replace("+00:00", "Z") if epoch is not None else None
    artifact_type = "SOURCE_ARCHIVE"
    if artifact is not None:
        artifact_type = "NATIVE_BINARY" if args.variant == "native-vectorscan" else "PORTABLE_BINARY"
    native_required = artifact_type == "NATIVE_BINARY"
    native_status = versions["vectorscan"]["status"] if native_required else "NOT_REQUIRED_FOR_ARTIFACT_TYPE"
    if native_required and native_status != "DETECTED":
        raise SystemExit("native binary evidence requires detected libhs provenance")

    evidence = {
        "schema_version": SCHEMA_VERSION,
        "release": args.release,
        "artifact_type": artifact_type,
        "build_variant": args.variant,
        "source_date_epoch": epoch,
        "generated_from_source_date_epoch": generated,
        "reproducible_metadata": epoch is not None,
        "source_manifest_sha256": manifest_sha,
        "source_file_count": len(files),
        "module": module,
        "versions": versions,
        "govulncheck": govuln,
        "native_vectorscan_provenance": {"required": native_required, "status": native_status},
        "authenticity": "UNAUTHENTICATED_UNLESS_DETACHED_SIGNATURE_VERIFIED",
        "truth_boundary": "Integrity-bound metadata does not authenticate its producer and does not qualify Coraza, VectorScan, CRS, kTLS, QAT, or target-host runtime behavior.",
    }
    if artifact and artifact.is_file():
        evidence["artifact"] = {"path": artifact.name, "sha256": sha256_file(artifact), "size": artifact.stat().st_size}

    evidence_path = out / "RELEASE_EVIDENCE.json"
    spdx_path = out / "sbom.spdx.json"
    cdx_path = out / "sbom.cyclonedx.json"
    evidence_path.write_text(json.dumps(evidence, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    spdx_path.write_text(json.dumps(spdx(args.release, module, deps, manifest_sha, args.variant, artifact), indent=2, sort_keys=True) + "\n", encoding="utf-8")
    cdx_path.write_text(json.dumps(cyclone(args.release, module, deps, manifest_sha, args.variant, artifact), indent=2, sort_keys=True) + "\n", encoding="utf-8")

    provenance = {
        "schema_version": 1,
        "artifact_type": artifact_type,
        "build_variant": args.variant,
        "source_manifest_sha256": manifest_sha,
        "release_evidence_sha256": sha256_file(evidence_path),
        "spdx_sha256": sha256_file(spdx_path),
        "cyclonedx_sha256": sha256_file(cdx_path),
        "signature_state": "NOT_VERIFIED_IN_ARCHIVE",
        "truth_boundary": "Cross-digest binding proves internal consistency only. Producer authenticity requires detached signature verification with an approved public key.",
    }
    (out / "PROVENANCE.json").write_text(json.dumps(provenance, indent=2, sort_keys=True) + "\n", encoding="utf-8")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
