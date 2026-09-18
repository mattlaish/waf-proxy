#!/usr/bin/env bash
set -euo pipefail
usage(){ cat <<'USAGE'
Usage: ./verify-release-artifact.sh <release.zip> [source-tree]

Validates a SOURCE_ARCHIVE after packaging: archive safety, required files,
script modes/syntax, source-to-package byte/mode equality, complete source and
release manifests, source-bound release evidence/provenance, and SBOM/module
consistency. This proves artifact integrity, not producer authenticity; use
verify-release-signature.sh with an approved public key for authenticity.
USAGE
}
[[ $# -ge 1 && $# -le 2 ]] || { usage >&2; exit 2; }
archive=$1; source_tree=${2:-.}
[[ -f "$archive" ]] || { echo "ERROR: archive not found: $archive" >&2; exit 1; }
[[ -d "$source_tree" ]] || { echo "ERROR: source tree not found: $source_tree" >&2; exit 1; }
command -v python3 >/dev/null 2>&1 || { echo "ERROR: python3 is required" >&2; exit 1; }
command -v bash >/dev/null 2>&1 || { echo "ERROR: bash is required" >&2; exit 1; }
archive=$(python3 -c 'import os,sys; print(os.path.abspath(sys.argv[1]))' "$archive")
source_tree=$(python3 -c 'import os,sys; print(os.path.abspath(sys.argv[1]))' "$source_tree")
tmp=$(mktemp -d); trap 'rm -rf "$tmp"' EXIT

python3 - "$archive" "$tmp/extracted" <<'PY'
import collections,os,pathlib,stat,sys,zipfile
archive,out=sys.argv[1:]; root=pathlib.Path(out); root.mkdir(parents=True,exist_ok=True)
secret_names={'.env','.npmrc','.pypirc','id_rsa','id_ed25519'}; secret_suffixes={'.key','.p12','.pfx','.jks','.keystore'}
max_file=128*1024*1024; max_total=512*1024*1024
with zipfile.ZipFile(archive) as zf:
 bad=zf.testzip()
 if bad: raise SystemExit(f"ERROR: ZIP CRC failure: {bad}")
 infos=zf.infolist()
 if not infos: raise SystemExit('ERROR: empty ZIP')
 names=[i.filename for i in infos]; dup=[n for n,c in collections.Counter(names).items() if c>1]
 if dup: raise SystemExit(f"ERROR: duplicate archive entries: {dup[:10]}")
 total=0
 for info in infos:
  name=info.filename; p=pathlib.PurePosixPath(name)
  if info.flag_bits & 0x1: raise SystemExit(f"ERROR: encrypted entry not allowed: {name}")
  if name.startswith(('/', '\\')) or p.is_absolute() or '..' in p.parts or '\\' in name: raise SystemExit(f"ERROR: unsafe archive path: {name}")
  if p.name in secret_names or p.suffix.lower() in secret_suffixes: raise SystemExit(f"ERROR: secret/private-key-like path not allowed: {name}")
  mode=(info.external_attr>>16)&0xFFFF
  if stat.S_ISLNK(mode): raise SystemExit(f"ERROR: symbolic link not allowed: {name}")
  if mode and not (stat.S_ISREG(mode) or stat.S_ISDIR(mode)): raise SystemExit(f"ERROR: special file not allowed: {name}")
  if stat.S_IMODE(mode)&0o022: raise SystemExit(f"ERROR: group/world-writable entry not allowed: {name}")
  if info.file_size>max_file: raise SystemExit(f"ERROR: entry too large: {name}")
  total+=info.file_size
  if total>max_total: raise SystemExit('ERROR: archive uncompressed size exceeds limit')
 zf.extractall(root)
 for info in infos:
  target=root/pathlib.PurePosixPath(info.filename); mode=(info.external_attr>>16)&0xFFFF
  if target.exists() and mode: os.chmod(target,stat.S_IMODE(mode))
PY
extracted="$tmp/extracted"
required=(
 AGENTS.md README.md AI_HANDOFF.md DEVELOPMENT_ROADMAP.md TESTING_RESULTS.md MANIFEST.md patch.md INSTALL.md PACKAGING_TOOL.md RELEASE_PROCESS.md DEVELOPMENT.md TESTING.md DEB_PACKAGE_GATE_RESULT.md RPM_PACKAGE_GATE_RESULT.md PACKAGE_LIFECYCLE_GATE_RESULT.md CLEAN_HOST_DISTRIBUTION_GATE_RESULT.md PACKAGE_TOOL_GATE_RESULT.md OPENAI_INTEGRATION_GATE_RESULT.md
 build.sh build-release-artifact.sh verify-release-artifact.sh verify-release-signature.sh release-security-scan.sh release-artifact-negative-tests.sh verify-reproducible-source-release.sh
 tools/release_evidence.py tools/source_manifest.py tools/waf_package_builder.py tools/tests/test_waf_package_builder.py tools/tests/test-waf-package-source.sh tools/tests/test-openai-integration-source.sh internal/openaiapi/responses.go internal/openaiapi/responses_test.go internal/secretref/secretref.go internal/secretref/secretref_test.go ai_openai_integration_test.go waf-package qualify-release-host.sh run-phase1-qualification.sh install.sh go.mod go.sum
 packaging/deb/build-deb.sh packaging/deb/build-release-deb.sh packaging/deb/verify-deb.sh packaging/deb/tests/test-deb-packaging.sh
 packaging/rpm/waf-proxy.spec packaging/rpm/build-rpm.sh packaging/rpm/build-release-rpm.sh packaging/rpm/verify-rpm.sh packaging/rpm/validate-rpm-source.py packaging/rpm/tests/test-rpm-source.sh packaging/rpm/tests/test-rpm-packaging.sh
 packaging/qualification/README.md packaging/qualification/package_lifecycle_qualify.py packaging/qualification/build-lifecycle-fixtures.sh packaging/qualification/run-package-lifecycle-qualification.sh packaging/qualification/tests/test_package_lifecycle.py packaging/qualification/tests/test-qualification-source.sh
 qualification/package-lifecycle/deb-upgrade-rollback-NOT_RUN.json qualification/package-lifecycle/rpm-upgrade-rollback-NOT_RUN.json
 packaging/cleanhost/README.md packaging/cleanhost/clean_host_qualify.py packaging/cleanhost/run-clean-host-qualification.sh packaging/cleanhost/tests/test_clean_host_qualify.py packaging/cleanhost/tests/test-clean-host-source.sh
 qualification/clean-host/matrix.json qualification/clean-host/debian-12-NOT_RUN.json qualification/clean-host/ubuntu-22.04-NOT_RUN.json qualification/clean-host/ubuntu-24.04-NOT_RUN.json qualification/clean-host/rhel-9-NOT_RUN.json qualification/clean-host/rocky-9-NOT_RUN.json qualification/clean-host/almalinux-9-NOT_RUN.json qualification/clean-host/oraclelinux-9-NOT_RUN.json
 cmd/wafctl/main.go cmd/wafqualify/main.go cmd/hsmqualify/main.go qualification/corpus/default.jsonl
 internal/hsm/config.go internal/hsm/provider.go internal/hsm/pkcs11.go internal/hsm/pkcs11_linux_cgo.go internal/hsm/pkcs11_stub.go internal/hsm/secret.go hsm_integration.go
 qualification/hsm/README.md qualification/hsm/run-softhsm-qualification.sh qualification/hsm/run-vendor-hsm-qualification.sh qualification/hsm/softhsm-qualification-report.json qualification/hsm/vendor-hsm-qualification.json
 release-evidence/RELEASE_EVIDENCE.json release-evidence/PROVENANCE.json release-evidence/SOURCE_MANIFEST.sha256 release-evidence/sbom.spdx.json release-evidence/sbom.cyclonedx.json
 RELEASE_MANIFEST.txt
)
for path in "${required[@]}"; do [[ -f "$extracted/$path" ]] || { echo "ERROR: required file missing: $path" >&2; exit 1; }; done

for path in waf-package tools/tests/test-waf-package-source.sh tools/tests/test-openai-integration-source.sh build.sh build-release-artifact.sh verify-release-artifact.sh verify-release-signature.sh release-security-scan.sh release-artifact-negative-tests.sh verify-reproducible-source-release.sh qualify-release-host.sh run-phase1-qualification.sh install.sh upgrade.sh uninstall.sh waf-doctor.sh setup-interfaces.sh qualification/hsm/run-softhsm-qualification.sh qualification/hsm/run-vendor-hsm-qualification.sh packaging/deb/build-deb.sh packaging/deb/build-release-deb.sh packaging/deb/verify-deb.sh packaging/deb/tests/test-deb-packaging.sh packaging/rpm/build-rpm.sh packaging/rpm/build-release-rpm.sh packaging/rpm/verify-rpm.sh packaging/rpm/tests/test-rpm-source.sh packaging/rpm/tests/test-rpm-packaging.sh packaging/qualification/build-lifecycle-fixtures.sh packaging/qualification/run-package-lifecycle-qualification.sh packaging/qualification/tests/test-qualification-source.sh packaging/cleanhost/run-clean-host-qualification.sh packaging/cleanhost/tests/test-clean-host-source.sh; do
 [[ -f "$extracted/$path" ]] || continue
 size=$(wc -c < "$extracted/$path"); (( size>=200 )) || { echo "ERROR: critical script implausibly small: $path" >&2; exit 1; }
 head -n1 "$extracted/$path" | grep -Eq '^#!/(usr/bin/env bash|bin/bash)' || { echo "ERROR: unexpected shell shebang: $path" >&2; exit 1; }
 [[ -x "$extracted/$path" ]] || { echo "ERROR: executable mode lost: $path" >&2; exit 1; }
 bash -n "$extracted/$path"
done
for path in tools/release_evidence.py tools/source_manifest.py tools/waf_package_builder.py tools/tests/test_waf_package_builder.py packaging/rpm/validate-rpm-source.py packaging/qualification/package_lifecycle_qualify.py packaging/qualification/tests/test_package_lifecycle.py packaging/cleanhost/clean_host_qualify.py packaging/cleanhost/tests/test_clean_host_qualify.py; do
 [[ -x "$extracted/$path" ]] || { echo "ERROR: executable mode lost: $path" >&2; exit 1; }
 python3 - "$extracted/$path" <<'PYCOMPILE'
import pathlib,sys
src=pathlib.Path(sys.argv[1]).read_text(encoding='utf-8'); compile(src,sys.argv[1],'exec')
PYCOMPILE
done
[[ ! -f "$extracted/benchmark/build.sh" || -x "$extracted/benchmark/build.sh" ]] || { echo "ERROR: benchmark/build.sh mode lost" >&2; exit 1; }
[[ ! -f "$extracted/benchmark/build.sh" ]] || bash -n "$extracted/benchmark/build.sh"

python3 - "$source_tree" "$extracted" <<'PY'
import hashlib,json,pathlib,re,stat,sys
src,dst=map(pathlib.Path,sys.argv[1:])
EX_ROOTS={'.git','bin','.cache','__pycache__','release-evidence'}
EX_NAMES={'RELEASE_MANIFEST.txt','SHA256SUMS.txt','waf-proxy','waf-tlsfront','wafctl','wafqualify','phase1-qualification.json','BUILD_PROVENANCE.json','BUILD_SHA256SUMS.txt'}
EX_SUFFIX={'.zip','.patch','.log','.tmp'}
def ignored(rel): return any(x in EX_ROOTS for x in rel.parts) or rel.name in EX_NAMES or rel.suffix in EX_SUFFIX
def source_files(root):
 out={}
 for p in root.rglob('*'):
  if not p.is_file() or p.is_symlink(): continue
  rel=p.relative_to(root)
  if ignored(rel): continue
  out[rel.as_posix()]=p
 return out
def sha(p):
 h=hashlib.sha256()
 with p.open('rb') as f:
  for c in iter(lambda:f.read(1024*1024),b''): h.update(c)
 return h.hexdigest()
def parse_hash_manifest(path):
 out={}
 for n,line in enumerate(path.read_text(encoding='utf-8').splitlines(),1):
  if not line.strip(): continue
  try: digest,rel=line.split('  ',1)
  except ValueError: raise SystemExit(f'ERROR: malformed manifest line {path.name}:{n}')
  if not re.fullmatch(r'[0-9a-f]{64}',digest): raise SystemExit(f'ERROR: invalid SHA-256 in {path.name}:{n}')
  if rel in out: raise SystemExit(f'ERROR: duplicate manifest path: {rel}')
  out[rel]=digest
 return out

# Exact source/package parity for all non-generated source files.
sfiles=source_files(src); dfiles=source_files(dst)
if set(sfiles)!=set(dfiles):
 raise SystemExit(f"ERROR: source/package file-set mismatch; missing={sorted(set(sfiles)-set(dfiles))[:20]} extra={sorted(set(dfiles)-set(sfiles))[:20]}")
for rel in sorted(sfiles):
 a,b=sfiles[rel],dfiles[rel]
 if sha(a)!=sha(b): raise SystemExit(f'ERROR: source/package SHA-256 mismatch: {rel}')
 if stat.S_IMODE(a.stat().st_mode)!=stat.S_IMODE(b.stat().st_mode): raise SystemExit(f'ERROR: source/package mode mismatch: {rel}')

# SOURCE_MANIFEST must be complete, not merely internally hash-correct.
sm_path=dst/'release-evidence/SOURCE_MANIFEST.sha256'; sm=parse_hash_manifest(sm_path)
expected_source=set(dfiles)
if set(sm)!=expected_source:
 raise SystemExit(f"ERROR: SOURCE_MANIFEST completeness mismatch; missing={sorted(expected_source-set(sm))[:20]} extra={sorted(set(sm)-expected_source)[:20]}")
for rel,digest in sm.items():
 if sha(dst/rel)!=digest: raise SystemExit(f'ERROR: source evidence manifest mismatch: {rel}')
sm_text=sm_path.read_text(encoding='utf-8')
sm_sha=hashlib.sha256(sm_text.encode()).hexdigest()

# RELEASE_MANIFEST must cover every packaged file except itself exactly once.
rm=dst/'RELEASE_MANIFEST.txt'; text=rm.read_text(encoding='utf-8')
if 'Version:' not in text or 'Build Variant:' not in text or '\nFiles:\n' not in text: raise SystemExit('ERROR: malformed RELEASE_MANIFEST.txt')
release_entries={}
for n,line in enumerate(text.split('\nFiles:\n',1)[1].splitlines(),1):
 if not line.strip(): continue
 try: digest,rel=line.split('  ',1)
 except ValueError: raise SystemExit(f'ERROR: malformed RELEASE_MANIFEST file line {n}')
 if rel in release_entries: raise SystemExit(f'ERROR: duplicate RELEASE_MANIFEST path: {rel}')
 release_entries[rel]=digest
packaged={p.relative_to(dst).as_posix() for p in dst.rglob('*') if p.is_file()}
expected_release=packaged-{'RELEASE_MANIFEST.txt'}
if set(release_entries)!=expected_release:
 raise SystemExit(f"ERROR: RELEASE_MANIFEST completeness mismatch; missing={sorted(expected_release-set(release_entries))[:20]} extra={sorted(set(release_entries)-expected_release)[:20]}")
for rel,digest in release_entries.items():
 if not re.fullmatch(r'[0-9a-f]{64}',digest) or sha(dst/rel)!=digest: raise SystemExit(f'ERROR: RELEASE_MANIFEST hash mismatch: {rel}')

# Evidence schema, source binding, and truth-boundary semantics.
ev_path=dst/'release-evidence/RELEASE_EVIDENCE.json'; ev=json.loads(ev_path.read_text())
if ev.get('schema_version')!=2 or ev.get('artifact_type')!='SOURCE_ARCHIVE' or ev.get('build_variant')!='source':
 raise SystemExit('ERROR: release evidence must identify this package as SOURCE_ARCHIVE/source')
if ev.get('source_manifest_sha256')!=sm_sha or ev.get('source_file_count')!=len(expected_source):
 raise SystemExit('ERROR: release evidence source binding mismatch')
native=ev.get('native_vectorscan_provenance',{})
if native.get('required') is not False or native.get('status')!='NOT_REQUIRED_FOR_ARTIFACT_TYPE':
 raise SystemExit('ERROR: misleading native VectorScan provenance state for source archive')
if ev.get('authenticity')!='UNAUTHENTICATED_UNLESS_DETACHED_SIGNATURE_VERIFIED':
 raise SystemExit('ERROR: release evidence authenticity boundary missing')
gov=ev.get('govulncheck',{})
need={'schema_version','evidence_type','status','tool','tool_version','source_manifest_sha256','executed_at','result_digest','output'}
if not need.issubset(gov): raise SystemExit('ERROR: incomplete govulncheck evidence schema')
if gov.get('schema_version')!=1 or gov.get('evidence_type')!='govulncheck' or gov.get('tool')!='govulncheck': raise SystemExit('ERROR: invalid govulncheck identity')
if gov.get('status') not in {'PASS','FAIL','BLOCKED','NOT_RUN'}: raise SystemExit('ERROR: invalid govulncheck status')
if gov.get('source_manifest_sha256')!=sm_sha: raise SystemExit('ERROR: govulncheck evidence source binding mismatch')
raw=gov.get('output');
if not isinstance(raw,str) or hashlib.sha256(raw.encode()).hexdigest()!=gov.get('result_digest'): raise SystemExit('ERROR: govulncheck evidence digest mismatch')
if gov.get('status') in {'PASS','FAIL'} and gov.get('tool_version') in {None,'','NOT_AVAILABLE'}: raise SystemExit('ERROR: executed govulncheck evidence missing tool version')

# Cross-digest provenance binding. This is integrity, not producer authenticity.
prov=json.loads((dst/'release-evidence/PROVENANCE.json').read_text())
if prov.get('schema_version')!=1 or prov.get('artifact_type')!='SOURCE_ARCHIVE' or prov.get('build_variant')!='source': raise SystemExit('ERROR: invalid provenance identity')
if prov.get('source_manifest_sha256')!=sm_sha: raise SystemExit('ERROR: provenance source binding mismatch')
for field,name in [('release_evidence_sha256','RELEASE_EVIDENCE.json'),('spdx_sha256','sbom.spdx.json'),('cyclonedx_sha256','sbom.cyclonedx.json')]:
 if prov.get(field)!=sha(dst/'release-evidence'/name): raise SystemExit(f'ERROR: provenance digest mismatch: {name}')

# SBOM format and dependency-set parity with go.mod.
spdx=json.loads((dst/'release-evidence/sbom.spdx.json').read_text()); cdx=json.loads((dst/'release-evidence/sbom.cyclonedx.json').read_text())
if spdx.get('spdxVersion')!='SPDX-2.3': raise SystemExit('ERROR: invalid SPDX document')
if cdx.get('bomFormat')!='CycloneDX' or cdx.get('specVersion')!='1.5': raise SystemExit('ERROR: invalid CycloneDX document')
def go_deps(path):
 deps=set(); in_req=False
 for raw in path.read_text().splitlines():
  line=raw.strip()
  if line=='require (': in_req=True; continue
  if in_req and line==')': in_req=False; continue
  if line.startswith('require '):
   f=line.split();
   if len(f)>=3: deps.add((f[1],f[2]))
  elif in_req and line and not line.startswith('//'):
   f=line.split();
   if len(f)>=2: deps.add((f[0],f[1]))
 return deps
expected_deps=go_deps(dst/'go.mod')
spdx_deps={(p.get('name'),p.get('versionInfo')) for p in spdx.get('packages',[]) if p.get('SPDXID')!='SPDXRef-Package-WAF'}
cdx_deps={(c.get('name'),c.get('version')) for c in cdx.get('components',[])}
if spdx_deps!=expected_deps: raise SystemExit('ERROR: SPDX dependency set does not match go.mod')
if cdx_deps!=expected_deps: raise SystemExit('ERROR: CycloneDX dependency set does not match go.mod')
print(f'ARTIFACT_INTEGRITY_PASS source_files={len(sfiles)} packaged_files={len(packaged)}')
PY
