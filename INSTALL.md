# waf-proxy — install guide

Coraza-based reverse-proxy WAF with an embedded admin console. Single static
binary; the console is compiled in (`go:embed`), so there are no runtime assets
to deploy and no CDN dependencies.

> **Status — read this first.** This code has been **verified statically only**
> (structure, wiring, imports, cross-file consistency). It has **never been
> compiled or run** by its author. `./build.sh` runs `go vet` and will surface
> anything that doesn't hold. Treat the first deployment as a bring-up, not a
> production cutover: run it in **DetectionOnly** beside real traffic and prove
> it before it's in the path of anything that matters.

---

## 1. Prerequisites

- Debian 12 / Ubuntu 22.04+ (systemd), x86-64 or arm64
- **Go ≥ 1.22** to build (`sudo apt install golang-go`, or the upstream tarball
  if the distro Go is older)
- Network access **once**, for `go mod tidy` (Coraza) and to fetch OWASP CRS
- `openssl` (token generation) and `curl`/`tar` (CRS fetch)
- Root for the install step

Air-gapped? Build on a connected machine and copy `waf-proxy` + this directory
to the target; `install.sh` needs no network.

## 2. Build

```bash
cd go-waf
./build.sh            # go mod tidy → go vet → static binary
```

Produces `./waf-proxy`. Set `VERSION=` / `COMMIT=` to stamp a build.

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
