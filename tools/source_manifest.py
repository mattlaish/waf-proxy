#!/usr/bin/env python3
"""Canonical source-manifest enumeration for release evidence and scanners."""
from __future__ import annotations

import argparse
import hashlib
import pathlib

GENERATED_ROOTS = {".git", ".cache", "__pycache__", "bin", "release-evidence"}
GENERATED_NAMES = {
    "RELEASE_MANIFEST.txt", "SHA256SUMS.txt", "waf-proxy", "waf-tlsfront",
    "wafctl", "wafqualify", "phase1-qualification.json",
    "BUILD_PROVENANCE.json", "BUILD_SHA256SUMS.txt",
}
GENERATED_SUFFIXES = {".zip", ".patch", ".log", ".tmp"}


def sha256_file(path: pathlib.Path) -> str:
    h = hashlib.sha256()
    with path.open("rb") as f:
        for chunk in iter(lambda: f.read(1024 * 1024), b""):
            h.update(chunk)
    return h.hexdigest()


def ignored(rel: pathlib.PurePath) -> bool:
    if any(part in GENERATED_ROOTS for part in rel.parts):
        return True
    if rel.name in GENERATED_NAMES:
        return True
    if rel.suffix in GENERATED_SUFFIXES:
        return True
    return False


def source_files(root: pathlib.Path) -> list[pathlib.Path]:
    out: list[pathlib.Path] = []
    for p in root.rglob("*"):
        if not p.is_file() or p.is_symlink():
            continue
        rel = p.relative_to(root)
        if ignored(rel):
            continue
        out.append(p)
    return sorted(out, key=lambda p: p.relative_to(root).as_posix())


def manifest_text(root: pathlib.Path) -> str:
    return "".join(
        f"{sha256_file(p)}  {p.relative_to(root).as_posix()}\n"
        for p in source_files(root)
    )


def manifest_digest(root: pathlib.Path) -> str:
    return hashlib.sha256(manifest_text(root).encode("utf-8")).hexdigest()


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--root", required=True)
    ap.add_argument("--output")
    ap.add_argument("--digest-only", action="store_true")
    args = ap.parse_args()
    root = pathlib.Path(args.root).resolve()
    text = manifest_text(root)
    digest = hashlib.sha256(text.encode("utf-8")).hexdigest()
    if args.output:
        pathlib.Path(args.output).write_text(text, encoding="utf-8")
    if args.digest_only:
        print(digest)
    elif not args.output:
        print(text, end="")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
