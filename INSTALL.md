# waf-proxy — install guide

> **Documentation baseline — 2026-09-17.** This file documents a component or qualification path. Repository-wide release truth lives in [`DOCUMENTATION_INDEX.md`](DOCUMENTATION_INDEX.md), [`SOURCE_BASELINE_GATE_RESULT.md`](SOURCE_BASELINE_GATE_RESULT.md), and [`TESTING_RESULTS.md`](TESTING_RESULTS.md). Component PASS evidence must not be promoted into a root-build, runtime, package-lifecycle, clean-host, or release PASS outside its stated scope.

Coraza-based reverse-proxy WAF with an embedded admin console. The console is compiled in (`go:embed`) with no CDN dependency. The portable Coraza-only build is pure Go; optional VectorScan acceleration uses CGO/libhs.

> **Status — read this first.** Production package procedures are implemented,
> but the current repaired root source has not yet passed its mandatory Go 1.25
> buildability gate. Build real packages only from a commit whose exact bytes
> passed the CI tidy/build/vet/test/race/real-Coraza sequence. Package fixtures
> are packaging evidence only. PostgreSQL is **not** currently a WAF runtime
> dependency; persistent state is filesystem-based. See `DOCUMENTATION_INDEX.md`
> and `SOURCE_BASELINE_GATE_RESULT.md`.

---

## Enterprise distribution paths

For production Linux deployments, use the formal package path rather than the
source installer whenever possible:

### Debian / Ubuntu — `.deb`

Build on the qualified release host:

```bash
SOURCE_DATE_EPOCH=<approved-epoch> \
  packaging/deb/build-release-deb.sh --output-dir dist/deb
```

Install locally or from your approved internal repository:

```bash
sudo apt install ./dist/deb/waf-proxy_<version>_<arch>.deb
```

Package behavior is deliberately fail-safe:

- `postinst` performs no network access and never fetches OWASP CRS;
- `/etc/waf/config.json`, `/etc/waf/coraza.conf`, and
  `/etc/waf/waf-tls-frontend.env` are dpkg conffiles;
- an existing `/etc/waf/waf-proxy.env` admin token is never regenerated on
  upgrade; a fresh install creates it once without printing the value to the
  package-manager log;
- `/var/lib/waf-proxy` is persistent state and is preserved across upgrades;
- fresh install does **not** auto-start the WAF before CRS is provisioned;
- package purge removes the generated admin token but intentionally leaves
  operator-managed certificates/CRS and persistent state for explicit admin
  cleanup.

Provision CRS using `/usr/share/doc/waf-proxy/CRS-PROVISIONING.md`, then run:

```bash
sudo waf-doctor --check
sudo systemctl enable --now waf-proxy
```

### RHEL family — `.rpm`

Build on a qualified RHEL-family release host with `rpm-build`, `rpm`,
`rpm2cpio`, `cpio`, `tar`, `python3`, `openssl`, and the normal Go prerequisites:

```bash
SOURCE_DATE_EPOCH=<approved-epoch> \
  packaging/rpm/build-release-rpm.sh --output-dir dist/rpm
```

Install locally or from your approved internal repository:

```bash
sudo dnf install ./dist/rpm/waf-proxy-<version>-1.<arch>.rpm
```

Package behavior mirrors the DEB enterprise contract:

- RPM scriptlets perform no network access and never fetch OWASP CRS;
- `/etc/waf/config.json`, `/etc/waf/coraza.conf`, and
  `/etc/waf/waf-tls-frontend.env` use `%config(noreplace)`;
- an existing `/etc/waf/waf-proxy.env` admin token is never regenerated or
  printed by an upgrade;
- `/var/lib/waf-proxy` and operator-managed certificates/CRS remain persistent;
- fresh install does **not** auto-start or preset the WAF before CRS provision;
- active services are `try-restart`ed on upgrade, while inactive deployments
  remain inactive;
- package scriptlets never disable SELinux or synthesize local policy.

Provision CRS using `/usr/share/doc/waf-proxy/CRS-PROVISIONING.md`, run
`sudo waf-doctor --check`, then explicitly enable/start the service. Clean-host
RHEL-family and enforcing-SELinux qualification remains Distribution Slice D.

### Generic / development / recovery install

The source `install.sh` path below is retained for development, recovery, and
generic environments. It is not the primary enterprise distribution path.

## 1. Prerequisites

- Debian 12 / Ubuntu 22.04+ or supported RHEL-family Linux (systemd), x86-64/x86_64 or arm64/aarch64 as appropriate to the package format
- **Go ≥ 1.25.0** to build. Coraza v3.7.0 declares this minimum; distro Go may be too old, so install an upstream Go 1.25+ toolchain when necessary.
- Network/module-cache access for Coraza v3.7.0 dependencies and to fetch OWASP CRS
- `openssl` (token generation) and `curl`/`tar` (CRS fetch)
- Root for the install step

Air-gapped? Build on a connected machine and copy the required binaries plus deployment files to the target. A native VectorScan build links through CGO/libhs and therefore requires compatible libhs runtime libraries on the target; do not treat it as a self-contained static binary. `install.sh` itself needs no network.

## 2. Build

```bash
cd go-waf
./build.sh            # go mod tidy -diff → go vet/test/race → binaries
```

Produces `./waf-proxy` and `./waf-tlsfront`. Set `VERSION=` / `COMMIT=` to stamp a build. `WAF_VECTORSCAN=auto` is the default; use `off` for portable Coraza-only, or install `pkg-config` + `libvectorscan-dev` and use `required` to fail closed unless native libhs is present.


## 2a. Optional VectorScan Learning Accelerator

VectorScan is never authoritative; Coraza v3.7.0 remains the final rule engine. The accelerator begins in Learning and only skips an eligible regex group after its learning thresholds have been met with zero observed false negatives. Any false negative or native scan error moves the group to `FAILSAFE` and Coraza-only processing continues.

On Debian/Ubuntu, install the native build dependency before a required build:

```bash
sudo apt install pkg-config libvectorscan-dev
WAF_VECTORSCAN=required ./build.sh
```

For a portable build with no CGO/libhs:

```bash
WAF_VECTORSCAN=off ./build.sh
```

Production release gate: run the repository tests with real Coraza v3.7.0 and real libvectorscan on the target/release architecture. Use `./qualify-release-host.sh --preflight` first; it exits 3 when prerequisites are **BLOCKED** (for example, Go <1.25 or no verified real libvectorscan provenance). On a qualified host, run `./qualify-release-host.sh --core`; it requires native VectorScan rather than silently falling back, executes the real-Coraza DetectionOnly+nolog `MatchedRules()` gate and the native `hs_compile_multi`/`hs_scan` semantic gate, and checks that qualification did not drift `go.mod`/`go.sum`. A source-installed VectorScan build requires an explicit `WAF_VECTORSCAN_PROVENANCE_ACK` string so ABI-only shims cannot be mislabeled as real release evidence. The package's stub and ABI-only verification are compile/regression aids only. Learning state defaults to `/var/lib/waf-proxy/vector-learning.json`; preserve that path across restarts but expect rule/Coraza/VectorScan semantic fingerprint changes to force re-learning.


## 2b. Optional external HSM / PKCS#11 TLS keys

PKCS#11 support is independent from VectorScan and is opt-in at build time:

```bash
# Coraza + PKCS#11, no VectorScan dependency
WAF_VECTORSCAN=off WAF_HSM_PKCS11=required ./build.sh

# Or enable both native capabilities
WAF_VECTORSCAN=required WAF_HSM_PKCS11=required ./build.sh
```

`WAF_HSM_PKCS11=off` is the default. `auto` enables the provider only when the
host is Linux and a CGO compiler is available. A configured HSM site must use
the built-in Go TLS path (`tls_acceleration.mode=go`); the external TLS frontend
is rejected rather than falling back to a filesystem key.

Keep the certificate chain on disk but leave `tls_key` empty. Configure
`tls_key_provider` with the approved PKCS#11 module path, an exact slot and/or
token label, an exact key label and/or CKA_ID, and a PIN **secret reference**.
Never put the PIN itself in JSON or a command line. For production, prefer a
root-protected `0600` secret file:

```json
"tls_cert": "/etc/waf/certs/site-chain.pem",
"tls_key": "",
"tls_key_provider": {
  "provider": "pkcs11",
  "module_path": "/opt/vendor/lib/libpkcs11.so",
  "slot_id": 7,
  "token_label": "prod-token",
  "key_label": "waf-tls",
  "key_id": "01",
  "pin_secret_ref": "file:/run/secrets/waf-hsm-pin"
}
```

Also restrict `hsm.allowed_module_dirs` to directories owned and managed by the
operator. The module and its approved parent path must be root-owned and not
group/world writable. Apply opens the token/session, resolves the secret,
performs exact private-key lookup, and verifies certificate↔key association via
a real signature before the runtime swap.

Qualification paths:

```bash
# Software PKCS#11 path only; does not qualify production vendor hardware
./qualification/hsm/run-softhsm-qualification.sh

# Real vendor HSM/service only; run only on the actual production-class provider
./qualification/hsm/run-vendor-hsm-qualification.sh
```

Both runners require `WAF_HSM_PIN_SECRET_REF=env:NAME` or
`file:/absolute/path`; neither accepts a raw PIN argument. Until actually run,
the checked-in SoftHSM and vendor reports remain `NOT_RUN`.

## 3. Install

```bash
sudo ./install.sh
```

It is idempotent (safe for upgrades) and will:

- create the `waf` system user
- install the binary to `/usr/local/bin/waf-proxy`
- create `/etc/waf` (`0770 root:waf` — group-writable so the console can persist
  config) and `/etc/waf/{certs,crs}` (`0750 root:waf`)
- install `coraza.conf` and `config.json` **only if absent** (never clobbers yours)
- generate an **admin token** into `/etc/waf/waf-proxy.env` (0600) and print it
- install the hardened systemd unit

**Save the admin token it prints** — it's the first-time login.

## 4. Get the OWASP CRS (required)

`coraza.conf` includes CRS from `/etc/waf/crs`; without it the engine has no rules.

```bash
cd /etc/waf/crs
sudo curl -fsSL -o crs.tar.gz \
  https://github.com/coreruleset/coreruleset/archive/refs/tags/v4.7.0.tar.gz
sudo tar xzf crs.tar.gz --strip-components=1
sudo cp crs-setup.conf.example crs-setup.conf
sudo chown -R root:waf /etc/waf/crs && sudo chmod -R g+rX /etc/waf/crs
```

In `crs-setup.conf`, start with anomaly thresholds inbound **5** / outbound **4**.

## 4a. Separate management & data planes (recommended)

The admin console (management plane) and the traffic listeners (data plane) are
independent sockets. On a two-NIC appliance you pin the console to a management
IP and keep traffic on the data NIC. Run the interactive helper:

```bash
sudo ./setup-interfaces.sh
```

It lists the machine's interfaces, asks which IP is **management** and which NIC
is **data**, generates a self-signed TLS cert for the console (or takes one you
supply), writes a systemd drop-in pinning the console to `MGMT_IP:port` over
HTTPS, and **validates** that no configured site listens on the management IP.
It does **not** change your site listen addresses — you set those to data-NIC
IPs yourself (in the console or `config.json`), one per site as needed.

Guarantees enforced (fail-closed, at both setup time and every startup):

- the admin console never shares an IP with a data-plane site,
- an off-loopback console must have TLS,
- a data-plane site on the management IP (or all-interfaces on the admin port)
  refuses to start.

Single-NIC / lab installs can skip this — the console defaults to
`127.0.0.1:9090` (reach it via `ssh -L`, see §6). To expose it later without the
script, set `WAF_ADMIN_ADDR` / `WAF_ADMIN_TLS_CERT` / `WAF_ADMIN_TLS_KEY` (flags
`-admin`, `-admin-cert`, `-admin-key`). Re-run `setup-interfaces.sh` any time to
change the pinning; it just rewrites the drop-in.

## 5. Configure & start

Point the default pool at a real backend, and set each site's `listen` to its
data-NIC IP:port (`/etc/waf/config.json` or via the console), then:

```bash
sudo systemctl enable --now waf-proxy
systemctl status waf-proxy
journalctl -u waf-proxy -f
```

> **Debugging tip + footgun.** To see startup errors inline you can run the
> binary in the foreground: `sudo /usr/local/bin/waf-proxy -config
> /etc/waf/config.json`. This runs as **root**, so anything it creates —
> notably `/var/log/waf/audit.log` — is left **root-owned**. The service runs as
> the `waf` user and then **can't write that root-owned file** (`permission
> denied`), even though the directory is writable. After any foreground/root
> run, clean up before starting the service:
> ```bash
> sudo rm -f /var/log/waf/audit.log      # waf recreates it with correct owner
> sudo chown -R waf:waf /var/log/waf
> # check nothing else got left root-owned:
> find /etc/waf /var/log/waf ! -user waf -ls 2>/dev/null
> ```

If it refuses to start citing **plane separation**, a site is bound to the
management IP (or all-interfaces on the admin port) — fix that site's `listen`
and restart.

## 6. First login

**Reach the console:**

- If you ran §4a: browse to `https://MGMT_IP:9090` from the management network.
  The self-signed cert warns in the browser (expected on an internal VLAN) —
  verify the fingerprint on first connect.
- Otherwise the console is on **127.0.0.1:9090** — tunnel in:

  ```bash
  ssh -L 9090:127.0.0.1:9090 user@waf-host
  # → http://127.0.0.1:9090
  ```

Then:

1. Paste the **admin token** on the login screen (leave user/pass blank).
   There is **no default password** — a fresh install has zero accounts.
2. **Users tab → Add account** → create your first `admin`.
3. Use that account daily; keep the token offline for break-glass only.

**Where's the admin token?** `install.sh` prints it once and stores it here:

```bash
sudo cat /etc/waf/waf-proxy.env      # WAF_ADMIN_TOKEN=…
```

It is **not** in the service logs (deliberately — it would leak into journald).
Lost it entirely? Set a new one and restart:

```bash
sudo sh -c 'echo "WAF_ADMIN_TOKEN=$(openssl rand -hex 24)" > /etc/waf/waf-proxy.env'
sudo chown root:root /etc/waf/waf-proxy.env && sudo chmod 0600 /etc/waf/waf-proxy.env
sudo systemctl restart waf-proxy
sudo cat /etc/waf/waf-proxy.env
```

The file must be `root:root 0600` — it's a full-admin break-glass credential.

> Signed-update endpoints stay localhost-only even when the console is on the
> management network — run those from an `ssh -L` session.

## 7. Verify (do this before trusting it)

```bash
# health (drives keepalived/LB failover)
curl -s http://127.0.0.1:9090/healthz          # {"status":"ok","role":"solo"}

# traffic passes
curl -k https://waf-host:8443/ -H 'Host: your-site'

# WAF sees an attack (DetectionOnly ⇒ logged, not blocked)
curl -k "https://waf-host:8443/?id=1'+OR+1=1--" -H 'Host: your-site'
# → appears in the console landing feed + Logs tab within seconds
```

Then, per feature you enable: **Send test** for syslog, **Test connection** for
the LLM, and confirm a virtual-patch path returns your chosen 403/404.

## 8. Rollout (DetectionOnly → Block)

1. Run **DetectionOnly** against real traffic for days, not minutes.
2. Watch **Logs** and **Site Map → Suggest policy fit** for false positives.
3. Tune via per-URL page policies; run **Discover** so content-based profile
   suggestions have real signal.
4. Flip **one** site to Block, watch, then proceed.

## 9. Hardening notes

- `/etc/waf/config.json` holds the **LLM API key and HA peer token in
  cleartext** (0600, `waf`-owned). `/etc/waf/waf-proxy.env` holds the admin
  token. Treat both as secrets at rest; they're why `/etc/waf` is 0750.
- The unit runs unprivileged with `CAP_NET_BIND_SERVICE`, `ProtectSystem=strict`,
  a syscall filter, and `MemoryDenyWriteExecute`. `/etc/waf` is the only
  writable path (the console saves config there).
- Keep the admin listener on localhost. If you must expose it, put it behind
  mTLS/VPN — it is a full-control surface.
- **Do not** enable `include_match_data` (syslog) or `include_body` (AI) without
  deciding they're acceptable for your data: both can carry PHI off-box.

## 9a. File ownership & permissions

`install.sh` and `setup-interfaces.sh` set these for you. Use this to verify, or
to fix ownership after manually replacing a file. Governing rule: **root owns
everything; the `waf` service account gets read-only access, and write access
only where it must; "other" gets nothing** — `/etc/waf` holds the admin token,
LLM API key, and HA peer token.

| Path | Owner:Group | Mode | Notes |
|------|-------------|------|-------|
| `/usr/local/bin/waf-proxy` | `root:root` | `0755` | binary; writable only by root |
| `/etc/systemd/system/waf-proxy.service` | `root:root` | `0644` | unit |
| `…/waf-proxy.service.d/interfaces.conf` | `root:root` | `0644` | drop-in from setup-interfaces.sh |
| `/etc/waf/` | `root:waf` | **`0770`** | waf must **create** `config.json.tmp` here for atomic saves → group-writable |
| `/etc/waf/coraza.conf` | `root:waf` | `0640` | rules; waf reads |
| `/etc/waf/crs/` and contents | `root:waf` | dirs `0750`, files `0640` | `chmod -R g+rX` yields this |
| `/etc/waf/config.json` | **`waf:waf`** | `0600` | the console **writes** it on Save → waf must own it; holds secrets |
| `/var/log/waf/` | `waf:waf` | `0750` | audit log dir (also set by systemd `LogsDirectory`) |
| `/etc/waf/waf-proxy.env` | **`root:root`** | `0600` | admin token; root-only — systemd reads it before dropping to waf |
| `/etc/waf/certs/` | `root:waf` | `0750` | dir |
| `/etc/waf/certs/admin.crt` | `root:waf` | `0640` | cert (not secret, but group-scoped) |
| `/etc/waf/certs/admin.key` | `root:waf` | `0640` | private key — waf must read to serve TLS; never world-readable |

Three easy-to-get-wrong points:

- **`/etc/waf` must be group-writable (`0770 root:waf`), not `0750`.** The console
  saves config by writing `config.json.tmp` in this directory and atomically
  renaming it over `config.json` — which needs the `waf` group to **create files
  in the directory**, not just read them. With `0750` you get *"applied but not
  persisted: open /etc/waf/config.json.tmp: permission denied"* — the change
  applies live but is lost on restart. Fix: `sudo chmod 0770 /etc/waf`.
- `config.json` is `waf:waf 0600` (not `root:…`). It's the one file the process
  rewrites, so waf must **own** it — and it holds secrets, so `0600`. Make it
  root-owned and the console's Save fails with a permission error.
- `waf-proxy.env` is `root:root 0600` (not group-waf). systemd reads
  `EnvironmentFile` as root **before** dropping to the waf user, so the service
  account never needs it — keeping it root-only means a compromise of the waf
  process doesn't expose the break-glass admin token. (It correctly shows up in
  a `find /etc/waf ! -group waf` check — that's expected, leave it as-is.)

Fix them all at once:

```bash
# binary + break-glass token: root-only
sudo chown root:root /usr/local/bin/waf-proxy /etc/waf/waf-proxy.env
sudo chmod 0755 /usr/local/bin/waf-proxy
sudo chmod 0600 /etc/waf/waf-proxy.env

# rules, CRS, certs: root-owned, waf-group readable
sudo chown -R root:waf /etc/waf/coraza.conf /etc/waf/crs /etc/waf/certs
sudo chmod 0640 /etc/waf/coraza.conf
sudo chmod -R g+rX /etc/waf/crs
sudo find /etc/waf/certs -type f -exec chmod 0640 {} \;
sudo chmod 0750 /etc/waf/crs /etc/waf/certs

# /etc/waf itself must be GROUP-WRITABLE so the console can persist config
sudo chown root:waf /etc/waf
sudo chmod 0770 /etc/waf

# config + audit dir: waf-owned
sudo chown waf:waf /etc/waf/config.json /var/log/waf
sudo chmod 0600 /etc/waf/config.json
sudo chmod 0750 /var/log/waf
```

Verify:

```bash
sudo -u waf test -w /etc/waf && echo "waf can persist config: ok"        # the save test
sudo -u waf test -r /etc/waf/config.json && echo "config readable by waf: ok"
sudo -u waf test -r /etc/waf/waf-proxy.env || echo "env correctly denied to waf"
```

## 9b. Troubleshooting startup

**First resort — run the doctor.** One script repairs and verifies every
ownership/permission/unit/capability invariant at once (config persistence,
audit-log ownership, stray root-owned files, the port-binding capability, the
`LogsDirectory` sandbox line):

```bash
sudo ./waf-doctor.sh          # repair + verify, prints PASS/FAIL per check
sudo ./waf-doctor.sh --check  # read-only: report without changing anything
sudo systemctl restart waf-proxy
```

`waf-doctor` is idempotent and safe to run anytime the service misbehaves after
a manual file copy or a foreground/root run. It's also run automatically at the
end of `install.sh`. The table below explains the individual causes it fixes.

The service crash-loops on any config error, so `systemctl status` may only show
`Start request repeated too quickly`. Get the **real** reason, then match it below.

```bash
sudo systemctl stop waf-proxy
sudo systemctl reset-failed waf-proxy
# the actual error line:
journalctl -u waf-proxy --no-pager | grep '"err"' | tail -1
```

`status=1/FAILURE` = the app exited on a config error (fix below). A status
**≥ 200** instead means a systemd sandbox failure (a path the unit doesn't
permit) — reinstall the unit from the package and `daemon-reload`.

| `err` contains | Cause | Fix |
|----------------|-------|-----|
| `readfile /etc/waf/coraza.conf: invalid argument` | stale binary (pre-fix) | rebuild + reinstall the binary (§11) |
| `unsupported Perl syntax` / `(?!` | stale `coraza.conf` (PCRE lookahead; Coraza uses RE2) | `sudo install -o root -g waf -m 0640 ~/waf-proxy/coraza.conf /etc/waf/coraza.conf` |
| `open /var/log/waf/audit.log: permission denied` | audit log left **root-owned** by a foreground/root run | `sudo rm -f /var/log/waf/audit.log; sudo chown -R waf:waf /var/log/waf`, then restart |
| `applied but not persisted: open /etc/waf/config.json.tmp: permission denied` | `/etc/waf` not group-writable; console can't create the temp file for atomic save | `sudo chown root:waf /etc/waf && sudo chmod 0770 /etc/waf` |
| `/etc/waf/crs/...` no such file / include failed | OWASP CRS not installed | do §4 |
| `refusing to start: … plane separation` | admin IP collides with a site, or off-loopback without TLS | give the site a data-NIC IP; re-run §4a |
| `bind: address already in use` | port already taken | `sudo ss -tlnp \| grep <port>` |
| `bind: cannot assign requested address` | bound to an IP this host doesn't have | fix the IP in `config.json` / the drop-in |
| a site on port 80/443 never binds, no error shown | installed unit is stale, lacks `CAP_NET_BIND_SERVICE` | reinstall the unit: `sudo install -o root -g root -m 0644 ~/waf-proxy/waf-proxy.service /etc/systemd/system/waf-proxy.service && sudo systemctl daemon-reload && sudo systemctl restart waf-proxy` |

**Golden rule for this whole class of problem:** after any `sudo` foreground
run, check for root-owned files before starting the service —
`find /etc/waf /var/log/waf ! -user waf -ls`. Root-owned artifacts are the most
common reason "it runs by hand but fails as a service".

**Runs foreground but fails under systemd?** That's the sandbox or ownership, not
the app. Verify the unit is current and the paths are `waf`-owned:

```bash
grep LogsDirectory /etc/systemd/system/waf-proxy.service   # expect: LogsDirectory=waf
ls -ld /var/log/waf                                        # expect: waf waf
ls -l  /etc/waf/config.json                                # expect: -rw------- waf waf
find /etc/waf /var/log/waf ! -user waf -ls                 # expect: nothing
```

**Console not reachable?** Check where it actually bound — after §4a it's on the
**management IP**, not localhost:

```bash
sudo ss -tlnp | grep waf-proxy    # data listener + MGMT_IP:9090 (admin)
```

Nothing listed = the process isn't running (it crashed — see above). Listed on
an IP you didn't expect = it's up, you're looking at the wrong address.

## 10. Known limits (as of this package)

- **In-memory state**, reset on restart: AI blocklist, learner aggregates,
  content signals, notification queue, sessions, audit ring, log rings.
  Users/config persist in `config.json`; the **site content map now persists**
  to `/etc/waf/sitemap.json` (autosaved every 60s + on shutdown, reloaded at
  startup), so it survives restarts and upgrades. Persisting the remaining state
  is the top open item.
- **Signed self-update** exists (Setup → Signed self-update), admin+localhost
  only, and **disabled unless a publisher public key is baked in** (`go build`
  with `PublisherKeyPEM`, or `WAF_PUBLISHER_KEY_FILE`). Packages are verified
  against that key before anything is written; the private key stays offline.
  You can still upgrade manually: rebuild, re-run `install.sh`, restart.
- **OS patching is `apt`'s job**, not this tool's.
- **HA syncs config and computes role only** — it does not move IPs. Real
  failover is keepalived/VRRP or your LB reading `/healthz`.
- **Fail-open when powered off requires bypass hardware.** Software can't do it;
  `WAF_WATCHDOG_DEVICE` only feeds/withholds a heartbeat for such hardware.
- The `web/` Vite console is a **scaffold**, not a replacement; the shipping
  console is the embedded single file.

## 11. Upgrading / replacing files

**Routine update (the common case — only code/console changed):** one command.

```bash
cd ~ && rm -rf waf-proxy && tar xzf waf-proxy-install.tar.gz && cd waf-proxy
sudo ./upgrade.sh
```

`upgrade.sh` rebuilds, swaps just the binary (which embeds the console), and
reinstalls the unit or `coraza.conf` **only if they actually changed** — so you
don't have to track which files differ. It never touches your `config.json`,
`certs/`, `crs/`, or `interfaces.conf`. `enable` and permissions are one-time
setup, not per-upgrade: once done they persist across rebuilds, so the routine
path doesn't repeat them.

The rest of this section is the **manual / first-time** detail.

Go source must be **rebuilt** — you can't copy `.go` files over a running
install. Flow:

```bash
tar xzf waf-proxy-install.tar.gz && cd waf-proxy
./build.sh                                   # produces ./waf-proxy
sudo systemctl stop waf-proxy
sudo install -o root -g root -m 0755 ./waf-proxy /usr/local/bin/waf-proxy
```

Replace config-side files **selectively** — ours to update, yours to keep:

| File | Replace on upgrade? |
|------|---------------------|
| `coraza.conf` | Yes — `sudo install -o root -g waf -m 0640 ./coraza.conf /etc/waf/coraza.conf` |
| `waf-proxy.service` | Yes — `sudo install -o root -g root -m 0644 ./waf-proxy.service /etc/systemd/system/waf-proxy.service && sudo systemctl daemon-reload` |
| `config.json` | **No** — your sites/users/pools |
| `/etc/waf/crs/*` | No — your downloaded rules |
| `/etc/waf/certs/*` | No — your certs |

```bash
sudo systemctl start waf-proxy && systemctl status waf-proxy
```

> Note: re-running `install.sh` is idempotent but **won't** overwrite an existing
> `config.json` or `coraza.conf` (it never clobbers your config). So to pick up a
> shipped `coraza.conf` change, run the `install` line above by hand. Set
> permissions per §9a if anything looks off after a manual copy.

Once publisher signing is set up, the **Signed self-update** feature (Setup tab)
is the intended upgrade path — it verifies, stages, installs, and restarts with
rollback. Until then, use the build-and-install flow above.

## 12. Uninstall

```bash
sudo ./uninstall.sh          # keeps /etc/waf by default
sudo ./uninstall.sh --purge  # also removes /etc/waf and the waf user
```

## TLS acceleration frontend (optional)

The installer now installs both `waf-proxy` and the optional `waf-tlsfront`
companion plus their systemd units. Go TLS remains the default and does not
require NGINX. To enable frontend TLS termination, install NGINX and OpenSSL on
the host, then choose **Setup → TLS acceleration → frontend**.

Recommended baseline is NGINX >= 1.25.1 with OpenSSL 3.x. The companion probes
the actual host: NGINX >= 1.25.1 gets the standalone `http2 on;` directive,
while older NGINX gets the legacy compatible listen parameter. `ktls:auto` and
`qat:auto` fall back to software TLS; `required` fails the Apply preflight if
the requested capability is unavailable.

```bash
sudo systemctl enable --now waf-proxy waf-tls-frontend
systemctl status waf-proxy waf-tls-frontend
journalctl -u waf-tls-frontend -n 50 --no-pager
```

For QAT, install the vendor driver/provider separately and set
`OPENSSL_MODULES` in `/etc/waf/waf-tls-frontend.env` only when the provider is
outside OpenSSL's default module path. waf-proxy does not store or load QAT
private material and does not use CGO for TLS acceleration.

## Operator support CLI

`build.sh` now produces `waf-proxy`, `waf-tlsfront`, and `wafctl`; `install.sh`/`upgrade.sh` install `wafctl` as `/usr/local/bin/wafctl`, and `waf-doctor.sh` verifies that binary.

After installation, set `WAF_ADMIN_TOKEN` (or use the protected token file) and run:

```bash
sudo wafctl doctor
```

For a bounded incident capture, use the exact configured site name:

```bash
sudo wafctl debug capture --tenant SITE --duration 5m
# reproduce the request and read X-WAF-Request-ID or list captures
sudo wafctl debug list --tenant SITE
sudo wafctl debug export --tenant SITE --transaction-id ID --output incident.zip
sudo wafctl debug stop --tenant SITE
sudo wafctl support bundle --output waf-support.zip
```

Do not leave debug capture enabled unnecessarily. The server enforces a one-hour maximum capture window and bounded retention; diagnostic artifacts should still be handled as sensitive operational data.


## Enterprise package upgrade / rollback qualification

For release qualification, use a dedicated disposable VM rather than a live
production node. Build or provide three artifacts of the same package format:
Version N, Version N+1, and an intentional-failure fixture. The fixture builder
creates package-semantics payloads only; production qualification should rerun
the same harness with qualified Go 1.25 WAF packages.

```bash
# Debian / Ubuntu semantic fixtures
SOURCE_DATE_EPOCH=1700000000 \
  ./packaging/qualification/build-lifecycle-fixtures.sh \
  --format deb --output-dir /tmp/waf-lifecycle-deb

# Non-mutating package metadata/default-conflict preflight
./packaging/qualification/run-package-lifecycle-qualification.sh \
  --format deb \
  --baseline /tmp/waf-lifecycle-deb/baseline/waf-proxy_*.deb \
  --candidate /tmp/waf-lifecycle-deb/candidate/waf-proxy_*.deb \
  --failure-fixture /tmp/waf-lifecycle-deb/failure/waf-proxy_*.deb \
  --output /tmp/waf-deb-lifecycle.json
```

On a dedicated qualification VM, rerun with:

```bash
sudo ./packaging/qualification/run-package-lifecycle-qualification.sh ... \
  --execute \
  --ack I_UNDERSTAND_THIS_MUTATES_A_DEDICATED_HOST
```

Use `--format rpm` on RHEL, Rocky Linux, AlmaLinux or Oracle Linux. Dependencies
must already be installed; the qualification runner intentionally does not use
network dependency resolvers. A PASS from this runner covers package lifecycle
only. Clean-host OS acceptance remains Distribution Slice D.

## Clean-host distribution qualification

Use a dedicated disposable VM/host for final distro acceptance. Do not run the
mutating flow on a shared build machine or production node. Supply the real
Version N and N+1 package files plus an already-approved **local** CRS tree; the
runner never downloads dependencies or CRS.

```bash
# Non-mutating preflight
./packaging/cleanhost/run-clean-host-qualification.sh \
  --platform debian-12 \
  --baseline /qualification/waf-proxy_N_amd64.deb \
  --candidate /qualification/waf-proxy_Nplus1_amd64.deb \
  --crs-dir /qualification/coreruleset \
  --output /qualification/debian-12-clean-host.json

# Real dedicated-host execution
sudo ./packaging/cleanhost/run-clean-host-qualification.sh ... \
  --execute \
  --ack I_UNDERSTAND_THIS_MUTATES_A_DEDICATED_CLEAN_HOST
```

Supported platform IDs are `debian-12`, `ubuntu-22.04`, `ubuntu-24.04`,
`rhel-9`, `rocky-9`, `almalinux-9`, and `oraclelinux-9`. RHEL-family acceptance
requires SELinux to be `Enforcing`. The host must start with no installed
`waf-proxy`, no `/etc/waf`, and no `/var/lib/waf-proxy`; otherwise the run is
BLOCKED rather than pretending to be a clean-host result.

## Project-local package build tool

After modifying the WAF source, use the repository-local package builder rather
than invoking low-level package scripts manually:

```bash
./waf-package doctor --format deb
./waf-package deb --version 2026.09.16.1
```

For RHEL-family RPM:

```bash
./waf-package doctor --format rpm
./waf-package rpm --version 2026.09.16.1 --rpm-release 1
```

To package one canonical build in both formats:

```bash
./waf-package all \
  --version 2026.09.16.1 \
  --deb-arch amd64 \
  --rpm-arch x86_64
```

The tool does not install Go, RPM/DEB tooling, CRS, or native libraries. Go
module access is offline by default (`GOPROXY=off`); use
`--allow-module-network` only on a build host where module-proxy access is
explicitly permitted. Complete operator documentation is in `PACKAGING_TOOL.md`.
## 13. Production package operations checklist

This section is the operator path for package-managed deployments. It complements
the earlier source-install sections; production Debian/Ubuntu and RHEL-family
installations should prefer DEB/RPM.

### 13.1 Clean install and preflight

1. Verify the package SHA-256 and native package metadata.
2. Confirm the host is an intended/supported distro and architecture.
3. Confirm ports/listener addresses are available and DNS/NTP are correct.
4. Install the local package. Fresh package install must remain inactive before
   CRS provisioning.
5. Provision approved CRS content locally; package scriptlets must not fetch it.
6. Install TLS certificates or configure the approved PKCS#11 provider.
7. Configure management/data-plane binding and backend pools/sites.
8. Run `sudo waf-doctor --check`. Resolve every blocking error before start.
9. Start in `DetectionOnly`, then explicitly enable/start `waf-proxy`.
10. Validate `/healthz`, backend traffic, WAF event generation, and first login.

Debian/Ubuntu example:

```bash
sudo apt install ./waf-proxy_<version>_<arch>.deb
sudo waf-doctor --check
sudo systemctl enable --now waf-proxy
```

RHEL-family example:

```bash
sudo dnf install ./waf-proxy-<version>-1.<arch>.rpm
sudo restorecon -RFv /etc/waf /var/lib/waf-proxy /var/log/waf 2>/dev/null || true
sudo waf-doctor --check
sudo systemctl enable --now waf-proxy
```

### 13.2 First setup

The package owns the executable/service layout; the operator owns site/pool/TLS
configuration. Preserve `/etc/waf/waf-proxy.env` as a root-only break-glass
secret. Create normal named administrative users after the first authenticated
login and retain the break-glass token outside normal daily workflows. Keep the
WAF in `DetectionOnly` until representative traffic has been observed and false
positives are tuned.

### 13.3 PostgreSQL boundary

There is currently **no PostgreSQL persistence backend** in waf-proxy. Do not
create a WAF database, `DATABASE_URL`, `PGHOST`, `PGPASSWORD`, or PostgreSQL
migrations as part of installation. Current authoritative state is:

```text
/etc/waf/            configuration, CRS and certificates
/var/lib/waf-proxy/  persistent learner/security/runtime state
/var/log/waf/        logs/audit data
```

A future PostgreSQL backend would require an explicit architecture/migration
slice and cannot be created by documentation alone.

### 13.4 TLS

Admin access should remain loopback-only or on a dedicated management address.
If the admin listener is exposed off-loopback, use TLS. Data-plane sites should
use approved certificate/key paths or the PKCS#11 provider. HSM-backed sites
must fail closed and must not silently fall back to filesystem keys. Go TLS is
the default; the optional external TLS frontend is separately qualified.

### 13.5 systemd

Use package-installed units and local drop-ins rather than editing vendor unit
files in place:

```bash
sudo systemctl daemon-reload
sudo systemctl status waf-proxy --no-pager
sudo journalctl -u waf-proxy -n 200 --no-pager
```

Custom overrides belong under `/etc/systemd/system/waf-proxy.service.d/`. The
persistent state contract requires `/var/lib/waf-proxy`; do not move learner or
security state into transient directories.

### 13.6 Host firewall

Package installation deliberately does not rewrite the host firewall. Open only
the required data listeners and restrict SSH/admin access to management sources.
If the admin listener remains on loopback, do not expose its port at all. Always
inspect the active ruleset before changing a remote host:

```bash
sudo ss -tlnp
sudo nft list ruleset
```

Do not apply an unreviewed default-drop policy over a remote SSH session.

### 13.7 SELinux on RHEL-family systems

RHEL/Rocky/AlmaLinux/Oracle Linux clean-host acceptance requires SELinux
`Enforcing`. RPM scriptlets must never call `setenforce 0`, synthesize
`audit2allow` policy, or mutate permanent SELinux policy to make the product
appear to work. Use standard paths and restore existing labels after manually
copying operator files:

```bash
getenforce
sudo restorecon -RFv /etc/waf /var/lib/waf-proxy /var/log/waf
sudo ausearch -m AVC -ts recent
```

If a product-owned SELinux policy is truly required, implement, version, review,
and qualify it explicitly rather than generating policy from transient AVCs.

### 13.8 Backup and restore

Before upgrade or rollback, capture `/etc/waf`, `/var/lib/waf-proxy`, relevant
logs/evidence, systemd drop-ins, installed package version, and the exact old
package artifact. Backups containing admin tokens, TLS keys, or API secrets must
be access-controlled and encrypted. For a consistent filesystem snapshot, stop
traffic/service first when the operational window permits.

Example backup:

```bash
sudo install -d -m 0700 /var/backups/waf-proxy
sudo systemctl stop waf-tls-frontend 2>/dev/null || true
sudo systemctl stop waf-proxy
sudo tar --acls --xattrs --numeric-owner -C / -czf \
  /var/backups/waf-proxy/waf-$(date +%Y%m%d-%H%M%S).tar.gz \
  etc/waf var/lib/waf-proxy var/log/waf
sudo sha256sum /var/backups/waf-proxy/waf-*.tar.gz
```

Restore only to a compatible package/application version, then run
`waf-doctor --check` before starting services. HSM private keys are outside this
filesystem backup and must follow the HSM vendor backup/recovery procedure.

### 13.9 Upgrade

Use the native package manager for package deployments. Preserve operator
configuration, generated admin secret, certificates/CRS, and
`/var/lib/waf-proxy`. After upgrade run doctor, health, auth, TLS/backend traffic,
and WAF event smoke tests. Inspect `.dpkg-dist/.dpkg-old` or `.rpmnew/.rpmsave`
files rather than blindly replacing production configuration.

### 13.10 Rollback

Keep the previous signed/hashed package and a pre-upgrade filesystem backup. A
package downgrade is not automatically an application-state rollback: if a new
version changes persistent-state format, restore only through a version-qualified
procedure. Real lifecycle behavior remains a Slice C qualification gate until
executed on dedicated supported hosts.

### 13.11 Uninstall

Package removal should remove package-owned executables/units while preserving
operator state according to DEB/RPM semantics. Never automate destructive
removal of `/etc/waf/certs`, `/etc/waf/crs`, or `/var/lib/waf-proxy` without an
explicit backup/cleanup decision. Source installs use the existing
`uninstall.sh` path and are not the primary enterprise package workflow.

### 13.12 Troubleshooting

Start with:

```bash
sudo waf-doctor --check
sudo systemctl status waf-proxy --no-pager
sudo journalctl -u waf-proxy -n 200 --no-pager
sudo ss -tlnp
sudo ls -la /etc/waf /var/lib/waf-proxy /var/log/waf
```

Common classes are missing/unapproved CRS, listener conflicts, config/TLS
validation errors, backend health failures, filesystem ownership, stale systemd
drop-ins, and on RHEL-family systems SELinux AVCs. Do not "fix" startup by
disabling SELinux, weakening file permissions, bypassing TLS validation, or
silently falling back from HSM to a filesystem key.\n\n## 14. OpenAI Responses API credentials\n\nFor native OpenAI, configure `provider=openai`, `api_style=responses`, and use a secret reference rather than putting the key in `config.json`:\n\n```json\n"api_key_ref": "env:OPENAI_API_KEY"\n```\n\nThe packaged systemd unit already reads `/etc/waf/waf-proxy.env` before dropping privileges, so an operator may add the variable to that root-owned `0600` file without exposing it to the admin API:\n\n```bash\nsudo sh -c 'printf "\\nOPENAI_API_KEY=%s\\n" "YOUR_KEY" >> /etc/waf/waf-proxy.env'\nsudo chown root:root /etc/waf/waf-proxy.env\nsudo chmod 0600 /etc/waf/waf-proxy.env\nsudo systemctl restart waf-proxy\n```\n\nFor file references, use a dedicated absolute regular file such as `/etc/waf/secrets/openai.key`, mode `0600`, with no symlinked path component. The admin API never returns the key or stored reference. Existing historical inline `api_key` configs remain migration-compatible but should be replaced with `api_key_ref`. Native Responses requires HTTPS; self-hosted/OpenAI-compatible endpoints that still implement Chat Completions should use `api_style=chat_completions`.\n
