#!/usr/bin/env python3
import pathlib,re,sys
root=pathlib.Path(__file__).resolve().parents[2]
spec=root/'packaging/rpm/waf-proxy.spec'
text=spec.read_text(encoding='utf-8')
errors=[]

def need(pattern,msg,flags=0):
    if not re.search(pattern,text,flags): errors.append(msg)

def forbid(pattern,msg,flags=0):
    if re.search(pattern,text,flags): errors.append(msg)

need(r'^Name:\s+waf-proxy\s*$', 'Name must be waf-proxy', re.M)
need(r'^License:\s+Proprietary\s*$', 'RPM must retain proprietary/all-rights-reserved license positioning', re.M)
need(r'^ExclusiveArch:\s+x86_64 aarch64\s*$', 'ExclusiveArch must be x86_64 aarch64', re.M)
for dep in ('openssl','ca-certificates','systemd'):
    need(r'^Requires:\s+'+re.escape(dep)+r'\s*$', f'missing runtime requirement: {dep}', re.M)
need(r'^Requires\(pre\):\s+shadow-utils\s*$', 'missing shadow-utils pre dependency', re.M)
for path,mode in [
    ('/etc/waf/config.json','0600,waf,waf'),
    ('/etc/waf/coraza.conf','0640,root,waf'),
    ('/etc/waf/waf-tls-frontend.env','0644,root,root')]:
    need(r'^%config\(noreplace\)\s+%attr\('+re.escape(mode)+r'\)\s+'+re.escape(path)+r'\s*$', f'missing config(noreplace) policy for {path}', re.M)
need(r'^%dir\s+%attr\(0750,waf,waf\)\s+/var/lib/waf-proxy\s*$', 'persistent state directory missing', re.M)
need(r'^/usr/lib/systemd/system/waf-proxy\.service\s*$', 'main systemd unit not packaged under /usr/lib/systemd/system', re.M)
need(r'^/usr/lib/systemd/system/waf-tls-frontend\.service\s*$', 'TLS systemd unit not packaged under /usr/lib/systemd/system', re.M)
need(r'if \[ ! -e /etc/waf/waf-proxy\.env \]; then', 'one-time admin secret creation missing')
need(r'try-restart waf-proxy\.service', 'controlled main-service upgrade restart missing')
need(r'if \[ "\$1" -eq 0 \].*systemctl', 'erase-only stop semantics missing', re.S)
need(r'preserved /etc/waf/waf-proxy\.env', 'explicit secret/state preservation notice missing')

# Extract only lifecycle scriptlet bodies for network/SELinux weakening checks.
sections={}
markers=list(re.finditer(r'^(%(?:pre|post|preun|postun))\s*$',text,re.M))
for i,m in enumerate(markers):
    end=markers[i+1].start() if i+1<len(markers) else text.find('\n%files',m.end())
    if end < 0: end=len(text)
    sections[m.group(1)]=text[m.end():end]
joined='\n'.join(sections.values())
for cmd in ('curl','wget','dnf','yum'):
    if re.search(r'(^|[;&|\s])'+cmd+r'([\s]|$)',joined,re.M|re.I): errors.append(f'network/package fetch command forbidden in scriptlets: {cmd}')
if re.search(r'git\s+clone',joined,re.I): errors.append('git clone forbidden in scriptlets')
for cmd in ('setenforce','semanage','audit2allow'):
    if re.search(r'(^|[;&|\s])'+cmd+r'([\s]|$)',joined,re.M|re.I): errors.append(f'SELinux mutation forbidden in scriptlets: {cmd}')
if re.search(r'^\s*systemctl\s+(enable|preset|start)\b',joined,re.I|re.M): errors.append('fresh-install service enable/start/preset forbidden in scriptlets')

# Generated admin token is deliberately unowned by RPM so erase/reinstall does
# not silently delete/recreate it.
files=text.split('\n%files\n',1)[1] if '\n%files\n' in text else ''
if re.search(r'^.*?/etc/waf/waf-proxy\.env\s*$', files, re.M): errors.append('generated admin secret must not be RPM-owned')


# Builder/verifier release invariants that can be checked without an RPM toolchain.
build=(root/'packaging/rpm/build-rpm.sh').read_text(encoding='utf-8')
verify=(root/'packaging/rpm/verify-rpm.sh').read_text(encoding='utf-8')
for token,label in [
    ('SOURCE_DATE_EPOCH','deterministic source epoch'),
    ('use_source_date_epoch_as_buildtime','RPM build-time normalization'),
    ('clamp_mtime_to_source_date_epoch','RPM mtime clamping'),
    ('_buildhost reproducible.invalid','fixed RPM build host'),
    ('--sort=name','deterministic payload ordering'),
    ('WAF_RPM_EXTRA_REQUIRES','explicit native runtime dependency'),
    ('native VectorScan build detected','native dependency fail-closed message'),
    ('BUILD_SHA256SUMS.txt','binary checksum binding'),
    ('BUILD_PROVENANCE.json','build provenance binding')]:
    if token not in build: errors.append(f'missing builder invariant: {label}')
for token,label in [
    ('%{FILEFLAGS}','RPM config flag verification'),
    ('try-restart waf-proxy.service','upgrade restart verification'),
    ('StateDirectory=waf-proxy','persistent state verification'),
    ('admin secret must not be packaged','secret exclusion verification')]:
    if token not in verify: errors.append(f'missing verifier invariant: {label}')

if '%if 0%{?waf_qualification_fail_post}' not in text:
    errors.append('qualification-only RPM post-failure macro missing')
if '--qualification-fail-post' not in build or 'restricted to qualification fixture versions' not in build:
    errors.append('qualification RPM failpoint is missing its explicit fixture-only guard')

for rel in ['packaging/rpm/README.md','packaging/rpm/CRS-PROVISIONING.md','packaging/rpm/SELINUX.md','packaging/rpm/config/waf-tls-frontend.env']:
    if not (root/rel).is_file(): errors.append(f'missing RPM packaging file: {rel}')

if errors:
    for e in errors: print('RPM_SOURCE_VALIDATE_FAIL',e,file=sys.stderr)
    raise SystemExit(1)
print('RPM_SOURCE_VALIDATE_PASS')
