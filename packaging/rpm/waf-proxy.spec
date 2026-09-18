# WAF Reverse Proxy enterprise RHEL-family binary package.
# Build with packaging/rpm/build-rpm.sh; do not invoke this spec directly because
# the wrapper binds the package to BUILD_PROVENANCE.json / BUILD_SHA256SUMS.txt.

%global debug_package %{nil}
%global _build_id_links none
%global __os_install_post %{nil}

Name:           waf-proxy
Version:        %{waf_version}
Release:        %{waf_release}
Summary:        Coraza reverse-proxy WAF with enterprise operations tooling
License:        Proprietary
URL:            https://example.invalid/waf-proxy
Source0:        waf-proxy-payload.tar
ExclusiveArch:  x86_64 aarch64

Requires:       openssl
Requires:       ca-certificates
Requires:       systemd
Requires(pre):  shadow-utils
Requires(post): systemd
Requires(preun): systemd
Requires(postun): systemd
%{?waf_extra_requires:Requires: %{waf_extra_requires}}

%description
Multi-site reverse-proxy WAF using Coraza and OWASP CRS. The package installs
prebuilt runtime binaries, CLI tooling, hardened systemd units and local
configuration skeletons. OWASP CRS content is intentionally not downloaded or
bundled by RPM scriptlets; provision an approved CRS explicitly before first
service start.

%prep
rm -rf payload
mkdir -p payload
tar -xf %{SOURCE0} -C payload

%build
# Binary-only package: compilation happens on the qualified release host before
# rpmbuild and is bound to this package by provenance/checksum evidence.
:

%install
rm -rf %{buildroot}
mkdir -p %{buildroot}
cp -a payload/rootfs/. %{buildroot}/

%pre
getent group waf >/dev/null 2>&1 || groupadd --system waf >/dev/null 2>&1 || exit 1
id waf >/dev/null 2>&1 || useradd --system --gid waf --home-dir /nonexistent --shell /sbin/nologin --comment 'WAF Reverse Proxy service account' waf >/dev/null 2>&1 || exit 1
exit 0

%post
%if 0%{?waf_qualification_fail_post}
echo 'waf-proxy qualification: intentional RPM %post failure' >&2
exit 42
%endif
install -d -o root -g waf -m 0770 /etc/waf
install -d -o root -g waf -m 0750 /etc/waf/certs /etc/waf/crs
install -d -o waf -g waf -m 0750 /var/lib/waf-proxy /var/log/waf
[ -e /var/log/waf/audit.log ] || install -o waf -g waf -m 0640 /dev/null /var/log/waf/audit.log

if [ -e /etc/waf/config.json ]; then chown waf:waf /etc/waf/config.json; chmod 0600 /etc/waf/config.json; fi
if [ -e /etc/waf/coraza.conf ]; then chown root:waf /etc/waf/coraza.conf; chmod 0640 /etc/waf/coraza.conf; fi
if [ -e /etc/waf/waf-tls-frontend.env ]; then chown root:root /etc/waf/waf-tls-frontend.env; chmod 0644 /etc/waf/waf-tls-frontend.env; fi

# Create the break-glass admin credential only once. Upgrades preserve the
# existing file and never print the token into RPM/DNF logs.
if [ ! -e /etc/waf/waf-proxy.env ]; then
  umask 077
  token="$(openssl rand -hex 24)" || exit 1
  {
    echo '# waf-proxy environment. Keep 0600 — this is a break-glass admin credential.'
    printf 'WAF_ADMIN_TOKEN=%s\n' "$token"
  } > /etc/waf/waf-proxy.env
  chown root:root /etc/waf/waf-proxy.env
  chmod 0600 /etc/waf/waf-proxy.env
  echo 'waf-proxy: created initial admin token in /etc/waf/waf-proxy.env (value not printed)'
else
  chown root:root /etc/waf/waf-proxy.env
  chmod 0600 /etc/waf/waf-proxy.env
  echo 'waf-proxy: preserved existing /etc/waf/waf-proxy.env'
fi

if command -v systemctl >/dev/null 2>&1; then
  main_was_active=0
  tls_was_active=0
  systemctl is-active --quiet waf-proxy.service >/dev/null 2>&1 && main_was_active=1 || :
  systemctl is-active --quiet waf-tls-frontend.service >/dev/null 2>&1 && tls_was_active=1 || :
  systemctl daemon-reload >/dev/null 2>&1 || :
  # Fresh installs stay inactive. An upgrade replaces files while the old
  # process remains active, so try-restart activates the new binary only for
  # services that were already running.
  if [ "$main_was_active" -eq 1 ]; then systemctl try-restart waf-proxy.service || exit 1; fi
  if [ "$tls_was_active" -eq 1 ]; then systemctl try-restart waf-tls-frontend.service || exit 1; fi
fi

if [ ! -s /etc/waf/crs/crs-setup.conf ]; then
  echo 'waf-proxy: OWASP CRS is not provisioned; service was not auto-started on fresh install.'
  echo 'waf-proxy: see /usr/share/doc/waf-proxy/CRS-PROVISIONING.md before systemctl enable --now waf-proxy.'
fi
exit 0

%preun
# $1 == 0 on final erase; $1 >= 1 on upgrade/replacement.
if [ "$1" -eq 0 ] && command -v systemctl >/dev/null 2>&1; then
  systemctl stop waf-tls-frontend.service >/dev/null 2>&1 || :
  systemctl stop waf-proxy.service >/dev/null 2>&1 || :
fi
exit 0

%postun
if command -v systemctl >/dev/null 2>&1; then
  systemctl daemon-reload >/dev/null 2>&1 || :
fi
if [ "$1" -eq 0 ]; then
  echo 'waf-proxy: preserved /etc/waf/waf-proxy.env, /var/lib/waf-proxy and operator-managed /etc/waf/{certs,crs}'
fi
exit 0

%files
%dir %attr(0770,root,waf) /etc/waf
%dir %attr(0750,root,waf) /etc/waf/certs
%dir %attr(0750,root,waf) /etc/waf/crs
%config(noreplace) %attr(0600,waf,waf) /etc/waf/config.json
%config(noreplace) %attr(0640,root,waf) /etc/waf/coraza.conf
%config(noreplace) %attr(0644,root,root) /etc/waf/waf-tls-frontend.env
/usr/bin/waf-proxy
/usr/bin/wafctl
/usr/bin/waf-tlsfront
/usr/sbin/waf-doctor
/usr/sbin/waf-setup-interfaces
/usr/lib/systemd/system/waf-proxy.service
/usr/lib/systemd/system/waf-tls-frontend.service
/usr/share/doc/waf-proxy/README.md
/usr/share/doc/waf-proxy/INSTALL.md
/usr/share/doc/waf-proxy/CRS-PROVISIONING.md
/usr/share/doc/waf-proxy/SELINUX.md
/usr/share/doc/waf-proxy/BUILD_PROVENANCE.json
/usr/share/doc/waf-proxy/BUILD_SHA256SUMS.txt
/usr/share/doc/waf-proxy/PACKAGE_BUILD.json
%dir %attr(0750,waf,waf) /var/lib/waf-proxy
%dir %attr(0750,waf,waf) /var/log/waf
%ghost %attr(0640,waf,waf) /var/log/waf/audit.log

%changelog
* Wed Sep 16 2026 WAF Reverse Proxy Project <noreply@example.invalid> - %{waf_version}-%{waf_release}
- Enterprise Linux Distribution Packaging Slice B
