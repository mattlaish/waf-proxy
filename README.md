# waf-proxy

## Current canonical status — 2026-09-17

The audited GitHub `main@1d52d65a73a802e32f02994e51f0a07beb240177`
was found non-buildable. A root-build-integrity repair is implemented, but the
exact repaired bytes have not executed the mandatory Go 1.25 dependency-drift
and root-build gates in this environment. Current source state is therefore
`IMPLEMENTED_TESTING_DEFERRED` and the Source Buildability Gate is `BLOCKED`.

Artifact integrity, source reconstruction, package fixtures, and component
source checks retain their separately scoped PASS evidence. They do not promote
root buildability or release readiness. Read `DOCUMENTATION_INDEX.md` and
`SOURCE_BASELINE_GATE_RESULT.md` before relying on older historical sections in
this ledger.

Multi-site, **load-balancing** reverse proxy with an embedded [Coraza](https://github.com/corazawaf/coraza) WAF engine (SecLang-compatible, runs OWASP CRS 4.x) and a built-in admin console. The portable build remains pure Go; optional VectorScan acceleration uses CGO/libhs. The console has no CDN dependency and works on an air-gapped management segment.

```
                          ┌ site blog (:443, blog.example.com) ─┐
client ─▶ listener :443 ──┤                                     ├─▶ pool ─▶ member (node:port)
          (SNI + WAF)      └ site shop (:443, shop.example.com) ┘         └▶ member (node:port)
client ─▶ listener :8443 ─ site api  (:8443, api.example.com) ───▶ pool ─▶ ...
```

## Documentation and current release truth

Start with `DOCUMENTATION_INDEX.md`. For continuation use `AI_HANDOFF.md` and
`HANDOVER_STATUS.md`; `HANDOVER_PROMPT.md` is the copy/paste bootstrap prompt.
The current root-build repair is implemented but **source buildability remains
BLOCKED** until the exact repaired bytes pass Go 1.25 `go mod tidy -diff` and
`go build ./...` plus the downstream CI gates. No production DEB/RPM should be
built from an unproven root source commit.

## Model (F5-style)

| Object | Meaning |
|---|---|
| **node** | one backend server, addressed by IP/hostname (no port). Reusable across pools. |
| **member** | a node **+ port + weight** inside a specific pool |
| **pool** | a set of members **+ load-balancing method + health monitor** |
| **site** | a **listen address** + hostnames + one pool. Its own WAF instance. |

Listen addresses now live **on each site** — different sites can bind different addresses, and sites sharing an address are demultiplexed by `Host` (and SNI for TLS). A request matching no site on its listener gets **421**.

## Load balancing & health

- **LB methods**: `round_robin` (weighted by member weight), `least_conn` (tracks real in-flight connections), `ip_hash` (client affinity), `random`.
- **Health monitor** per pool: `tcp` (port open), `http` (GET a path, match a status), or `none`. Rise/fall hysteresis; only healthy members receive traffic. **Fail-open** — if every member is marked down, the pool still tries them rather than blackholing the site.
- Green/red dots and live connection counts show per member on the **Pools & Nodes** tab.

## Build (Debian)

```bash
# Coraza v3.7.0 requires Go >= 1.25.0. Install a Go 1.25+ toolchain first.
go version

# Portable Coraza-only build
WAF_VECTORSCAN=off ./build.sh

# Native VectorScan build (Debian/Ubuntu: install libvectorscan-dev + pkg-config)
WAF_VECTORSCAN=required ./build.sh

# Release-host truth preflight/core qualification (never substitutes ABI stubs)
./qualify-release-host.sh --preflight
./qualify-release-host.sh --core
```

## Enterprise Linux distribution packages

The formal enterprise Linux distribution paths are:

- Debian / Ubuntu: `.deb` package
- RHEL / Rocky / AlmaLinux / Oracle Linux: `.rpm` package

`install.sh` remains a generic development/recovery path. For Debian/Ubuntu, build
the qualified binaries first and then create the deterministic package:

```bash
SOURCE_DATE_EPOCH=<approved-epoch> \
  packaging/deb/build-release-deb.sh --output-dir dist/deb

sudo apt install ./dist/deb/waf-proxy_<version>_<arch>.deb
```

The `.deb` never downloads OWASP CRS in `postinst`, preserves dpkg conffiles and
`/var/lib/waf-proxy` across upgrades, preserves an existing break-glass admin
token, and does not auto-start a fresh installation before CRS is explicitly
provisioned. See `packaging/deb/README.md` and `INSTALL.md`.

On a qualified RHEL-family release host, the equivalent RPM path is:

```bash
SOURCE_DATE_EPOCH=<approved-epoch> \
  packaging/rpm/build-release-rpm.sh --output-dir dist/rpm

sudo dnf install ./dist/rpm/waf-proxy-<version>-1.<arch>.rpm
```

The RPM uses `%config(noreplace)` for operator configuration, preserves the
existing break-glass token and `/var/lib/waf-proxy`, never fetches CRS/packages
in scriptlets, leaves a fresh install inactive until CRS is explicitly
provisioned, and does not disable or auto-generate SELinux policy. See
`packaging/rpm/README.md`, `packaging/rpm/SELINUX.md`, and `INSTALL.md`.

## Run

```bash
WAF_ADMIN_TOKEN=$(openssl rand -hex 24) ./waf-proxy -config /etc/waf/config.json
```

First run writes a default `config.json` (one node, one pool, one catch-all site on `:8443`). Console at `http://127.0.0.1:9090`.

Example config with two backends load-balanced behind one site:

```json
{
  "rules": "/etc/waf/coraza.conf",
  "engine_mode": "On",
  "read_timeout_sec": 15, "idle_timeout_sec": 60, "backend_timeout_sec": 30,
  "nodes": [
    { "name": "web1", "host": "192.168.1.20" },
    { "name": "web2", "host": "192.168.1.21" }
  ],
  "pools": [
    {
      "name": "blog-pool",
      "scheme": "http",
      "lb_method": "least_conn",
      "monitor": { "type": "http", "path": "/healthz", "expect_status": 200,
                   "interval_sec": 5, "timeout_sec": 2, "rise": 2, "fall": 3 },
      "members": [
        { "node": "web1", "port": 8080, "weight": 1 },
        { "node": "web2", "port": 8080, "weight": 2 }
      ]
    }
  ],
  "sites": [
    {
      "name": "blog",
      "listen": ":443",
      "hostnames": ["blog.example.com", "*.blog.example.com"],
      "pool": "blog-pool",
      "preserve_host": true,
      "tls_cert": "/etc/waf/tls/blog/fullchain.pem",
      "tls_key": "/etc/waf/tls/blog/privkey.pem"
    }
  ]
}
```

## Console tabs

- **Config** — sites (name, listen address, hostnames, **pool selector**, **policy selector**, engine mode, AI mode, Host forwarding, TLS), plus global timeouts and the header engine selector.
- **Pools & Nodes** — define nodes, then pools (scheme, LB method, monitor, members) with live health.
- **Policies** — named rulesets: rules file, paranoia level, body limit, path-scoped exclusions.
- **Site Map** — per-site path tree, learned `seen` (passive) and `crawled` (opt-in polite spider via the site's pool). Mapper, not a scanner.
- **Setup / AI** — connect an LLM and let it analyze traffic, with optional per-site real-time blocking.

## AI-assisted analysis & enforcement (Setup / AI tab)

The WAF has one asynchronous LLM connector with per-site `off`, `advisory`, and
`block` modes. Coraza remains the deterministic/authoritative WAF engine; AI is
an optional second-opinion/enrichment path and is never inserted into the live
request hot path.

### OpenAI Responses API + Structured Outputs

For native OpenAI use, select:

```json
{
  "provider": "openai",
  "api_style": "responses",
  "base_url": "https://api.openai.com/v1",
  "api_key_ref": "env:OPENAI_API_KEY",
  "model": "gpt-4o-mini"
}
```

The connector sends `POST {base_url}/responses`, uses Bearer authentication,
sets `store=false`, maps the configured token cap to `max_output_tokens`, and
requests strict JSON Schema output. Traffic verdicts are constrained to
`verdict`, `score`, `category`, and `reason`; page-profile review uses a separate
strict schema with `agree`, `confidence`, and `reason`.

Refusals, incomplete/non-completed responses, malformed payloads, HTTP failures,
timeouts, and oversized provider responses become connector errors. At the WAF
engine boundary those errors remain **fail-open**: no AI block is created.

For vLLM, Ollama, llama.cpp, or another endpoint implementing the historical
OpenAI Chat Completions contract, select:

```json
"api_style": "chat_completions"
```

That path continues to use `POST {base_url}/chat/completions`. Historical
configs created before `api_style` existed and still carrying a legacy inline
`api_key` retain this Chat Completions wire contract so an upgrade does not
silently change provider protocol. New configuration should not add inline keys.

### Secret references

New AI credentials are references, not secret values in `config.json`:

```text
env:OPENAI_API_KEY
file:/etc/waf/secrets/openai.key
```

The admin API/UI returns neither the secret value nor the stored reference;
blank secret-reference input means “preserve the existing credential”. The
legacy `api_key` field remains read-compatible only for migration and is removed
when an operator supplies `api_key_ref`.

`file:` secrets must be absolute clean paths, regular files with no symlinked
path component, at most 4096 bytes, non-empty, and inaccessible to group/world
(`0600` is the normal mode). See `INSTALL.md` for deployment examples.

### Runtime behavior

- **off** — no AI analysis for that site.
- **advisory** — AI records a verdict but never blocks.
- **block** — only a schema-valid `malicious` verdict meeting `block_threshold`
  adds the source IP to the site-scoped TTL blocklist.
- **Async, never in the hot path.** WAF-flagged requests and an optional sample
  of other requests are queued to workers; inline dataplane cost is the existing
  blocklist lookup.
- **Privacy.** Authorization/cookie/token-like headers are excluded; request body
  is omitted by default; client IP may be HMAC-hashed.
- **Prompt-injection boundary.** Captured request data is untrusted data and only
  validated structured fields can reach enforcement logic.
- **No requested OpenAI response storage.** Native Responses calls explicitly
  send `store=false`.

The console exposes verdicts and the current AI blocklist plus manual unblock.
`POST /api/ai/test` sends a canned SQLi sample through the same configured
provider path before a site is switched to enforcement.

API: `GET /api/ai/verdicts` · `GET /api/ai/blocklist` ·
`POST /api/ai/unblock {ip}` · `POST /api/ai/test`.

Current truth: the Responses/Structured-Outputs/secret-reference implementation
and isolated provider mock suite are implemented, but the repository-root Go
1.25 test/build gate is still `BLOCKED` on this packaging host. See
`OPENAI_INTEGRATION_GATE_RESULT.md` and `SOURCE_BASELINE_GATE_RESULT.md`.

### Backend HTTPS trust

HTTPS pools verify backend certificates using the operating-system public CA
store by default; public roots are not copied into this repository. A pool may
append one or more enterprise PEM CA bundles and may set a certificate
`server_name` when members are addressed by IP. TLS 1.0/1.1 and insecure
skip-verification are not supported. Set `use_system_ca` to `false` only for an
intentionally isolated private trust store containing at least one custom CA.

```json
"backend_tls": {
  "use_system_ca": true,
  "ca_files": ["/etc/waf/pki/company-root.pem"],
  "crl_files": ["/etc/waf/pki/company-root.crl"],
  "server_name": "app.internal.example",
  "revocation_mode": "hard"
}
```

CA files are limited to 8 MiB and strictly parsed as PEM certificate bundles.
Invalid, empty, duplicate, or missing bundles make Apply fail before runtime
swap, so the previous runtime continues serving.

Static CRLs may be PEM (`X509 CRL`) or DER, with at most 32 lists per pool and
the same 8 MiB per-file limit. Apply rejects malformed, not-yet-valid, or expired
lists. During the standard verified TLS handshake, CRL issuer signatures are
checked against the verified chain and both leaf and intermediate serials are
checked. A revoked serial always fails closed. `soft` mode permits an issuer
without a matching CRL; `hard` mode requires valid coverage for every non-root
certificate. CRLs are loaded into an immutable snapshot at Apply—no file I/O is
performed in the handshake path. URL refresh is not implemented yet.

**Binding multiple / unassigned IPs (IP_FREEBIND).** Sites can each bind their own `IP:port`, and listeners are created with `IP_FREEBIND` (Linux), so a site may bind an IP that is **not currently assigned to a local interface** — a floating/VIP address, or one of several service IPs you manage outside the box. Without this the kernel rejects the bind with *"cannot assign requested address."*

**Auto-assigning the IP to the interface (`manage_ip`).** Freebind lets the socket bind, but the host still has to **answer ARP** for the address for traffic to arrive. Tick **"Assign listen IP to interface"** on a site (config: `manage_ip: true`) and, on Apply, the WAF assigns that listen IP with `ip addr add <ip>/<prefix> dev <iface>`. Managed IPs survive routine service restarts and are removed when management is disabled or the site is deleted. This requires **`CAP_NET_ADMIN`** (granted in the shipped unit) and the iproute2 `ip` binary. Leave it **off for a true VIP** that keepalived/VRRP should own. If no explicit interface is selected, legacy same-subnet detection is used; if no match exists, Apply fails with an actionable error.

For multi-NIC systems, set `manage_interface` to the data-plane NIC and
`manage_prefix_len` to the desired CIDR prefix. The console lists live
interfaces and disables the NIC holding the management-console IP. Leaving the
interface empty uses the data NIC selected by `setup-interfaces.sh`
(`WAF_DATA_INTERFACE`), then falls back to legacy same-subnet detection when
no deployment default exists. A managed-IP failure
now fails Apply instead of silently leaving only an IP_FREEBIND listener.

```json
{
  "listen": "192.168.1.80:443",
  "manage_ip": true,
  "manage_interface": "ens37",
  "manage_prefix_len": 21
}
```

## Policies (per-site rulesets)

A **policy** is a named ruleset + tuning that sites reference like they reference a pool:

- **rules file** (SecLang, usually including CRS) — pick it with the file browser
- **paranoia level** (1–4) — injected as CRS `tx.*_paranoia_level` after the rules load
- **request body limit**
- **exclusions** — remove a CRS rule (or a specific target of it), optionally scoped to a URL path prefix

Each site selects a policy; each site still compiles its own engine from that policy, so per-site engine mode and correct per-site match attribution are preserved. Two sites can share one policy (edit once, applies to both) or run different ones. Exclusion values are integer rule IDs / sanitized paths — never raw strings spliced into directives.

## Learned policy fit (Site Map → "Suggest policy fit")

The learner watches per-page signals — request volume, response outcomes, and which CRS rules fire from how many **distinct clients** — and recommends tuning for each page:

- **likely false positive**: a rule firing on a meaningful share of a page's traffic from several distinct clients (legit users tripping it) → suggests a **path-scoped exclusion**.
- **attack**: a rule concentrated on a few sources → flagged as hostile; keep/raise paranoia, don't exclude.
- Each page gets a **risk class** (benign / normal / elevated / hostile); the site rolls up to a **suggested paranoia level**.

True per-*page* engines aren't practical (one Coraza instance per site), so the unit of action is a **page-scoped exclusion written into the site's policy** plus a policy-level paranoia suggestion. Click **Exclude N rule(s)** on a page and it appends the exclusion to that site's policy and applies it live. Heuristics are deliberately transparent — no black box; you review every suggestion before it's applied.

## Page profiles: learn content → suggest → review → apply

The Site Map drives a content- and structure-aware pipeline that picks a **pre-built profile** for each page based on what the page *is*:

1. **Learn** — the crawler (Discover) reads each page's HTML and records content signals: forms, input fields, `type=password`/`file`/`hidden`, method, JSON vs static content-type. Passive traffic contributes weaker signals (observed POST, query-param count).
2. **Suggest** — a classifier maps signals → a pre-built profile with a confidence: a password/form page → **form-sqli-xss** (SQLi 942xxx + XSS 941xxx strict at PL3); a file input → **upload-strict**; JSON under `/api` → **api-json**; query-driven → **query-hardened**; static/no-inputs → **static-lenient**.
3. **Review** — suggestions are shown in the Site Map "Page profiles" panel (path, suggested profile, confidence, rationale). A human accepts per row (with an override dropdown), **or** tick *review with LLM* so the AI confirms each page's purpose before it counts.
4. **Apply** — accept binds the profile as a per-URL page policy. **Auto-apply** button: applies everything at/above a confidence threshold; with *review with LLM* on, only pages the LLM also confirms are applied. Off by default — you press the button.

Profiles are pre-built (editable via `config.profiles`) and express intent as CRS-family tuning (paranoia + relaxing irrelevant families), bound per-URL — not a separate engine per page. Content signals are strongest on **crawled** pages; passively-seen pages are marked `passive` (weaker), so Discover first for best results.

API: `GET /api/profiles` · `GET /api/profiles/suggest?site=` · `POST /api/profiles/apply {site,path,profile}` · `POST /api/profiles/auto {site,threshold,use_llm}`.

## Per-URL policies (Site Map is the editor)



The unit of policy is the **URL**. Each page under a site can carry its own **page policy** bound to it by path (⚙ on any node in the **Site Map**). A page policy is one of two things:

- **Tune** — engine mode, paranoia level, and rule exclusions for that path (what the profiles pipeline binds). A `policy` badge marks bound pages.
- **Block path (virtual patch)** — deny the URL outright before it reaches the backend, for known-vulnerable paths (leftover admin panels, exploitable endpoints, CVE URLs). Choose **exact** or **prefix** (covers sub-paths) and the response: **403** or **404** (hide that it exists). Blocked paths show a red `blocked` badge. The deny forces the engine on for that transaction, so a virtual patch enforces even if the site runs DetectionOnly, and blocks are logged in the match feed.

- **Bound by path, not a separate engine.** A page policy compiles to path-gated SecLang inside the site's single engine: `SecRule REQUEST_FILENAME "@beginsWith /path" "…ctl:ruleEngine=…,ctl:ruleRemoveById=…,setvar:tx.*_paranoia_level=…"`. So each URL effectively has its own policy without the cost of an engine per URL.
- **Generated or designed.** The learner *generates* suggested page policies (rule exclusions for false-positive-prone pages); you can also *design* one by hand in the editor. Both bind to the URL and are reviewed from the Site Map.
- **Per-site, not shared.** Page policies live on the site — tuning one site never leaks into another site that happens to use the same base policy. (The base **policy** on the Policies tab still defines which rules file / CRS and the site-wide defaults; page policies are URL-scoped overrides on top.)

Reliability: `ctl:ruleEngine` and `ctl:ruleRemoveById` apply exactly from phase 1; per-page paranoia reliably affects body (phase-2) rules — for strict phase-1 paranoia, set it on the base policy.

**When it takes effect, and how sub-pages inherit.** Applying a policy or page policy is **immediate, not gradual** — there is no propagation delay and sub-pages do not "catch up over time." On **Apply (go live)** the engine is rebuilt and atomically swapped into the live listener (sub-second); the *next request* to any matching path already uses the new policy. Sub-page coverage is **structural, via prefix matching**, evaluated fresh on every request rather than pushed down to each page:

- A page policy with `Match: prefix` (the default) bound to `/api/` covers `/api/users`, `/api/users/123`, and everything beneath it — because they share the prefix. Nothing propagates; the prefix is tested per request.
- A page policy with `Match: exact` applies **only** to that exact path and does **not** cascade to sub-pages.
- The site's **base policy** (Policies tab) applies to every path; page policies are URL-scoped overrides layered on top. A more specific page policy on a sub-path overrides a broader one on its parent.

So if a sub-page isn't behaving as expected after Apply, it's almost always one of: the page policy is `exact` (won't cascade), a more specific page policy on the sub-path is overriding the parent, or you clicked **Save draft** (persists to disk) instead of **Apply (go live)** (pushes to the running engine). Draft changes never affect live traffic until applied.

API: `GET /api/pagepolicy?site=` · `POST /api/pagepolicy/upsert {site,policy}` · `POST /api/pagepolicy/delete {site,path}`.

### Per-field policies for POST forms

A page policy can also validate individual request fields. Field discovery is
hybrid: **Discover** supplies HTML form metadata, while real POST/PUT/PATCH
traffic supplies field names for login-protected pages, SPAs, and APIs the
crawler cannot reach. Passive discovery reads a bounded request-body prefix and
retains names only—never field values or uploaded content. It is controlled by
`passive_discovery_enabled`; set it to `false` to remove request-body discovery
buffering/parsing from the data path while keeping ordinary path/status learning
and the explicit crawler available. Crawler and passive
results are merged as `crawled`, `passive`, or `both`; a human still reviews and
applies every policy. Open the page-policy editor from Site Map to configure:

- request source (`ARGS_POST` or all `ARGS`) and HTTP methods;
- a safe built-in profile: identifier, password, free text, email, or numeric;
- an optional validated per-field `allow_pattern` regular expression (maximum
  512 bytes; literal double quotes and line breaks in the policy expression are
  rejected before SecLang generation). This restriction protects policy syntax;
  whether submitted field values may contain quotes is decided by the pattern;
- required, minimum length, and maximum length;
- CRS rule IDs excluded only for that field via
  `ctl:ruleRemoveTargetById`—other fields remain fully inspected.

```json
{
  "path": "/register",
  "match": "exact",
  "methods": ["POST"],
  "fields": [
    {
      "name": "user_id",
      "source": "ARGS_POST",
      "profile": "identifier",
      "required": true,
      "min_length": 3,
      "max_length": 64
    },
    {
      "name": "password",
      "source": "ARGS_POST",
      "profile": "password",
      "required": true,
      "min_length": 12,
      "max_length": 256,
      "exclude_rule_ids": [942100]
    }
  ]
}
```

Field names and profiles are allow-listed before SecLang compilation; the UI
does not accept raw regular expressions or directives. Password and free-text
profiles permit punctuation such as apostrophes while their length limits and
all non-excluded CRS targets continue to apply. The application must still use
parameterized database queries and context-appropriate output encoding.

API: `GET /api/forms?site=&path=` returns non-sensitive form shape discovered
from HTML and live write requests (names, types where known, methods, actions,
required flags, and discovery source). The crawler also seeds itself with
previously observed GET paths and follows same-origin form actions using GET;
it never submits a form or authenticates as a user.

## Failure behaviour: fail-open, failover & health

**Software fail-open (process alive but degraded):** built in throughout — a slow/dead SIEM, LLM, or backend never blocks the request path (async queues, drop-on-full, health monitors that fail-open when all members are down).

**Powered-off / crashed box:** software cannot make a dead box pass traffic — there is no process running. That is a *hardware/topology* function. Two supported approaches:

- **Network-level failover (no special hardware).** `/healthz` is a real readiness probe: it returns **503** when the runtime is broken or the process is draining, and 200 with the HA role otherwise. Point keepalived/VRRP or a load-balancer health check at it; a dead or draining node fails the check and the VIP/route moves to the peer. On `SIGTERM` the process enters **draining** (health goes 503) and waits `WAF_DRAIN_SECONDS` (default 3) *before* closing listeners, so upstreams fail over first.
- **Inline bypass hardware (true fail-open when dead).** A fail-open/bypass NIC or external bypass switch whose relay shorts the ports on power loss or loss of heartbeat. The software side is an **opt-in watchdog feeder**: set `WAF_WATCHDOG_DEVICE=/dev/watchdog` (and optionally `WAF_WATCHDOG_INTERVAL_SECONDS`, default 10). It feeds the device while healthy and **stops feeding when unhealthy/draining**, so the hardware acts — a kernel watchdog reboots, a bypass NIC flips to bypass. Set the interval well below the hardware timeout. Off by default.

> **Fail-open vs fail-closed is a real security tradeoff.** A bypass NIC preserves availability by letting traffic reach backends **unprotected** when the WAF dies. For paths carrying PHI you may prefer **fail-closed** (dead WAF = no traffic). Decide per deployment; this tool provides the health signal for either.

## High availability (Setup / AI · HA)



Two instances kept in step, honestly scoped:

- **Config sync** — set a peer admin URL + token; every config apply pushes to the peer's `/api/config` (a loop-guard header stops echo). "Sync now" forces a push.
- **Role** — each node polls the peer's `/healthz` and computes **active / standby / solo**; if the peer goes dark it takes over. `GET /api/ha` exposes the role.
- **What it does NOT do**: move IP addresses. Real packet failover belongs to **keepalived/VRRP** or your load balancer — point its health check at `/healthz` and read role from `/api/ha`. Building an in-process VIP grab would be a dishonest half-solution. Blocklist state is intentionally not shared (each node decides independently).

## Notifications (bell + webhook)

Noteworthy events — new policy-fit suggestions, AI blocks, pool member down, config-sync results, HA peer up/down — surface in the header **bell** and optionally POST to a webhook (Slack/Teams text or generic JSON). Suggestions arrive with a **one-click Apply** (writing the exclusion into the site's policy); nothing auto-applies. State is in-memory (resets on restart). API: `GET /api/notifications` · `POST /api/notifications/{read,dismiss,apply}`.

> **State persistence:** the AI blocklist, learner aggregates, site-map, and notification queue are all in-memory and reset on restart. A JSON-snapshot backer is the clean next step (there's a seam for it); until then, treat these as ephemeral.

## Logs (traffic + WAF events)

The **Logs** tab (between Policies and Setup) shows two live feeds:

- **WAF events** — every rule match: time, site, client, severity, rule ID, phase, URI, message. Filter by free text (IP / path / rule id) and by severity.
- **Access** — every request through the proxy: time, site, client, method, path, and final status (after WAF/AI decisions), colour-coded by status class.

Both update live (3s) with a **pause** toggle for reading, and a text filter. Rings are in-memory and bounded (250 events / 1000 requests); for long-term retention, ship the process logs (structured slog is already emitted) to your SIEM.

## Syslog forwarding to a SIEM (Setup tab)

Forward events to an external SIEM over syslog. Configure host/port, **protocol (UDP / TCP / TCP+TLS)**, **format (RFC 5424 default, or RFC 3164)**, facility, and app-name, then choose which streams to send: **WAF events**, **access log** (higher volume), **audit** (who changed what), and **notifications** (AI blocks, member-down, HA). A **Send test** button emits one message so you can confirm it lands before trusting it.

- **Async + fail-open.** A single background writer owns the connection with a bounded queue and drop-on-full; a slow or unreachable SIEM never blocks the request path or back-pressures the WAF. Reconnects automatically.
- **TLS.** WAF logs can carry URIs, client IPs, and payload fragments — prefer **TCP+TLS** off-box. `tls_skip_verify` is available for private-CA/self-signed collectors.
- **PHI-conservative.** The matched payload fragment is **omitted by default**; include it (truncated to 200 chars) only via the explicit opt-in.
- Messages are structured key=value (`event=waf_match site=… client=… rule_id=… severity=… uri=… msg=…`) for easy SIEM parsing.

Defaults: WAF + audit + notifications on, access log off, payload omitted, TCP+TLS on :6514. API: `POST /api/syslog/test`.

## Users, roles & audit (Users tab)





Multi-user access with role-based permissions, layered on the break-glass startup token (which always works as admin so you can't lock yourself out).

- **Roles:** `admin` (manage users + everything), `operator` (edit config/pools/policies), `reviewer` (apply page policies, profiles & learned suggestions — but not edit global config or users), `viewer` (read-only).
- **Login** swaps the console's bearer to a session token (12h, in-memory — re-login after a restart). Endpoints are role-gated server-side, not just in the UI.
- **Passwords** are stored as **PBKDF2-HMAC-SHA256** (210k iterations, per-user salt) — the correct form of the salted-hash idea, brute-force resistant, stdlib-only. Hashes are masked on read and managed only via the user endpoints (a general config save never touches them).
- **Audit trail** records who did what (logins, config applies, user changes, policy/profile applies) in an in-memory ring, shown in the tab.

> In-memory: sessions and the audit ring reset on restart. Users themselves persist (they live in `config.json`, `0600`). A JSON-snapshot backer for sessions/audit is the clean next step.

API: `POST /api/login` · `POST /api/logout` · `GET /api/whoami` · `GET /api/users` · `POST /api/users/{create,update,password,delete}` · `GET /api/audit`.

## Signed self-update (Setup tab)

Update the running binary (and shipped assets) from a **cryptographically signed** package, built on the shared `sigupdate` engine (`internal/sigupdate/`, copied unchanged from the Signed Update & Publishing Standard).

**Trust model:** the publisher signs packages with an RSA private key held **offline**; the build embeds only the **public** key and verifies. Verification (RSA PKCS#1 v1.5 / SHA-256 over `manifest.json`) happens **before** the manifest is parsed, with per-file SHA-256 checksums and strict path-safety (no absolute paths, no `..`, allow-listed names/extensions). A stolen admin session cannot forge an update — without the private key, nothing installs. If no key is baked in, the whole subsystem is **disabled and 404s** (fail-safe).

**Flow:** upload a `.wafupdate` package (or pull from a signed online catalog if `CatalogURL` is set) → it's verified and **staged** (nothing written) → review the version/notes/file list → **Install** (atomic per-file writes, previous versions backed up) → **Restart** into the new binary (drains first). One-click **Rollback** restores the backup. Admin-only, localhost-only.

**Enabling:** set `PublisherKeyPEM` in `update.go` before `go build`, or point `WAF_PUBLISHER_KEY_FILE` at the public-key PEM at runtime; optionally set `CatalogURL` / `WAF_UPDATE_CATALOG_URL`. Sign packages with the standard's `make-update.sh` (RSA/SHA-256). Run `go test ./...` — the verifier's test vectors are the proof it interoperates.

Endpoints (all admin+localhost, 404 when disabled): `GET /api/update/status` · `GET /api/update/catalog` · `POST /api/update/{stage,download,install,discard,rollback,restart}`.

## Console: single-file vs component build




The shipping console is embedded in the binary as `static/admin.html` plus `static/theme.css` — no external CDN or runtime build step is required, so it remains suitable for an air-gapped management segment. The theme stylesheet is served same-origin at `/theme.css` under the console CSP. A Vite + preact migration scaffold lives in `web/` and imports the same theme tokens; see `web/PORTING.md` for finishing the remaining tabs.

## Admin API

`GET /api/status` · `GET|PUT /api/config` · `POST /api/reload` · `GET /api/matches` · `GET /api/pools` · `GET /api/sitemap` · `POST /api/crawl` · `POST /api/sitemap/clear`. Bearer token, constant-time compare; engine modes allow-listed before SecLang interpolation; config persisted `0600` via atomic rename. Keep the admin listener on loopback (`ssh -L 9090:127.0.0.1:9090 waf-box`).

## systemd

```ini
[Unit]
Description=Coraza WAF reverse proxy
After=network-online.target
Wants=network-online.target

[Service]
User=waf
Group=waf
Environment=WAF_ADMIN_TOKEN=change-me
ExecStart=/usr/local/bin/waf-proxy -config /etc/waf/config.json -admin 127.0.0.1:9090
AmbientCapabilities=CAP_NET_BIND_SERVICE
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/etc/waf /var/log/waf
PrivateTmp=true
Restart=on-failure
RestartSec=2

[Install]
WantedBy=multi-user.target
```

## Smoke tests

```bash
# two backends, one pool — watch requests spread across members on the Pools tab
curl -sk https://waf-box/ -H 'Host: blog.example.com' -o /dev/null -w '%{http_code}\n'
# kill web2, confirm the monitor marks it down and traffic shifts to web1
curl -sk 'https://waf-box/?q=<script>alert(1)</script>' -H 'Host: blog.example.com'  # matches feed
```
## Security scanning & verification

The build gate (`./build.sh`) runs `go mod tidy`, `go vet`, and `go test`. The
checks below go further and are what a release should be held to. Results shown
are from the 2026-08-21 scan of the pre-PKI tree and the 2026-08-24 scan of `76ee868`.

### Static analysis

```bash
go vet ./...                                   # clean
gofmt -l .                                     # style only; see note below
staticcheck ./...                              # honnef.co/go/tools/cmd/staticcheck
golangci-lint run ./...
gosec -severity=low -confidence=low ./...      # github.com/securego/gosec/v2
go mod verify                                  # all modules verified
for f in *.sh benchmark/*.sh; do bash -n "$f"; done  # all release/build scripts syntax-clean
node --check <inline JS from static/admin.html>
```

`go vet` is clean in the external Coraza-stub verification harness. The inherited release-script CRLF/EOF failures have now been repaired: all top-level shell scripts plus `benchmark/build.sh` are LF-normalized, executable, and pass `bash -n`. `release_scripts_test.go` keeps both the line-ending and syntax gates in the repository. The source is now pinned to Coraza v3.7.0 / Go 1.25.0. Real Coraza v3.7.0 and real libvectorscan execution remain release gates on a host with the required toolchain/dependencies; stub/ABI-only gates are not treated as substitutes.

`staticcheck` reports two unused functions
(`ipmanage.go:282`, `profiles.go:410`). `golangci-lint` adds 17 unchecked-error
findings, all `Close()`/`Remove()` in cleanup paths. `gofmt -l` lists ten files
because the codebase uses a deliberately compact single-line style; this is
cosmetic and is not enforced by the build.

**gosec triaged baseline — 27 findings, 8 rated HIGH, none exploitable.** Do not
"fix" these without reading the code first:

| Rule | Location | Why it is not a defect |
|---|---|---|
| G115 ×3 | `pool.go:95`, `pool.go:107`, `users.go:101` | Conversions on a guarded non-empty slice length, a modulo result provably below its divisor, and the PBKDF2 block counter. None can overflow. |
| G404 ×2 | `pool.go:97`, `ai.go:501` | `math/rand` chooses a random pool member and the AI sample rate. Neither is a security decision. |
| G402 | `syslog.go:171` | This is the documented `tls_skip_verify` option for private-CA/self-signed SIEM collectors. Accepted risk, operator-selected. |
| G702 / G703 | `internal/sigupdate` | Taint warnings on the update path-safety logic that the package's own tests already cover. |
| G204 | `ipmanage.go:297` | `exec.Command("ip", …)` runs without a shell; the address passes `net.ParseIP` and the interface is resolved against `net.Interfaces()`. |

**Dependency vulnerability scanning is not covered by the above.** `go mod
verify` proves only that modules match their recorded checksums. Run

```bash
govulncheck ./...
```

on a host that can reach `vuln.go.dev`; it needs that service and fails closed
where egress is restricted. Treat a release as unscanned until this passes.

### Dynamic analysis

Build with the race detector and exercise the running proxy rather than only
its unit tests:

```bash
go test -race -count=1 ./...
go build -race -o /tmp/waf-proxy-race .
GORACE="halt_on_error=0 log_path=/tmp/race" WAF_ADMIN_TOKEN=… \
  /tmp/waf-proxy-race -config /tmp/test-config.json -admin 127.0.0.1:19090
```

Point the site at a throwaway backend on a loopback address **other than** the
admin address — the startup check refuses to run the console and a data-plane
listener on the same IP, and that refusal is itself worth confirming.

Verified behaviours, all passing:

- **No data races or panics** under ~1,440 concurrent requests across 8 workers
  while five live config Applies swapped the runtime mid-traffic.
- **Admin API** returns 401 unauthenticated and with a wrong token on
  `/api/config`, `/api/users`, `/api/audit`, and `/api/ai/blocklist`.
- **Signed update fails safe**: `/api/update/status` returns 404 when no
  publisher key is compiled in.
- **Host routing**: 421 for an undeclared Host and for a missing Host header.
- **WAF enforcement**: SQLi and XSS blocked in both query string and POST body.
- **Draft versus Apply**: a draft save leaves the live runtime untouched, and a
  hostname that exists only in the draft does not serve traffic.
- **Graceful drain**: `SIGTERM` holds `/healthz` at 503 for `WAF_DRAIN_SECONDS`
  before listeners close, then exits cleanly.
- **Request smuggling (CL.TE)**: no desync. The pipelined request appears in the
  WAF's own access log, so it is inspected rather than tunnelled past.
- **Header/URL/method abuse**: oversized headers and URLs, null bytes, CRLF
  injection attempts, and unknown methods are handled without error.
- **X-Forwarded-For spoofing is neutralised**: with no trusted proxies configured,
  client-supplied forwarding headers are ignored and replaced with the immediate
  TCP peer before the request reaches the backend.
- **Trusted upstream proxies are resolved globally**: set
  `trusted_proxy_cidrs` at the config root (or use **Setup / AI → Trusted
  upstream proxy CIDRs**). Only an immediate peer in those networks may supply
  `X-Forwarded-For`; the chain is walked right-to-left and stops at the first
  untrusted hop. A malformed or oversized chain falls back to the TCP peer.
- **One client identity is used everywhere**: Coraza, `ip_hash`, access logs,
  syslog, learner client counts, AI enforcement, `X-Forwarded-For`, and
  `X-Real-IP` all receive the same resolved address.

Trust only CIDRs owned by the CDN or load balancer directly connected to the
WAF. For a single proxy, use a host prefix such as `192.0.2.10/32`; leaving the
list empty preserves edge-deployment behavior and never trusts inbound XFF.

### Fuzzing the certificate parsers

`pki.go` parses operator-supplied CA bundles and CRLs, which is the largest
untrusted-input surface in the tree. Fuzz targets for `parseCRLFile`,
`appendCABundle`, and `validateBackendServerName` were run for the 2026-08-24
scan — roughly 488,000 executions in total, with no crash, panic, or hang. The
targets are not committed; recreate them as `FuzzX` functions in `package main`
and run:

```bash
go test -run=XXX -fuzz=FuzzParseCRLFile -fuzztime=40s .
```

Making these permanent is cheap and worthwhile: they cover the code paths that
consume bytes the proxy did not produce.

When scanning with a reduced ruleset instead of full CRS, remember that CRS
supplies the transformations: a rule written as `@contains ../` will miss
`%2e%2e%2f` without `t:urlDecodeUni`. A miss under a hand-written test ruleset
is usually the ruleset, not the engine.


## Data-plane performance: P0-A response copy path

P0-A removes a steady-state response-copy allocation from the standard-library
reverse proxy. `buildProxy` now supplies a shared fixed 32 KiB
`httputil.BufferPool`, backed by `sync.Pool` of fixed arrays, and leaves
`FlushInterval` at zero for ordinary responses. Go's reverse proxy still selects
immediate flushing for recognized streaming responses (for example SSE); normal
responses no longer opt into a periodic timer/mutex flush writer. The buffer-pool
component benchmark in the offline harness measured about 10.90 ns/op with
0 B/op and 0 allocs/op for Get/Put. This is a component result, not end-to-end
Coraza throughput.

## Data-plane performance: P0-B load-balancer hot path

P0-B removes the remaining per-request load-balancer allocation pair identified
behind the reverse proxy. Pool selection no longer materializes a temporary
`healthyMembers()` slice. Round-robin, least-connections, IP-hash, and random
selection scan the immutable member list directly, preserve health filtering and
all-down fail-open behavior, and allocate zero bytes in the component benchmark.

The selected member is also no longer passed from `Rewrite` to `lbTransport`
through `context.WithValue`. Backend selection now happens inside `lbTransport`,
where the chosen member pointer stays local for active-request accounting until
the response body closes. The transport applies the configured member's
scheme/host immediately before the standard `http.Transport`, while preserving
the original path/query and the site's `preserve_host` behavior. `Rewrite`
continues to own trusted forwarded-header construction.

On the offline AMD EPYC 9V74 harness with an 8-member healthy pool, the previous
selector measured about 42–48 ns/op with 64 B/op and 1 alloc/op depending on the
LB method. P0-B measured roughly 12–28 ns/op with 0 B/op and 0 allocs/op. The
removed request-context handoff independently measured about 32 ns/op, 48 B/op,
and 1 alloc/op in the pre-P0-B tree. These are component benchmarks, not
end-to-end Coraza or network throughput claims.

## Data-plane performance: P0-C bounded observation plane

P0-C removes passive discovery/learning bookkeeping from the synchronous WAF
request path. Host observation, live sitemap updates, learner request/rule
accounting, and request-shape signals now enter one bounded in-memory observation
queue with a non-blocking send. A background worker owns the existing store
updates. Queue saturation is deliberately fail-open for traffic: the observation
is dropped and counted rather than making an HTTP request wait for telemetry.

The queue defaults to 8,192 events. `/api/metrics` now includes an additive
`observations` object with current depth/capacity, processed count, dropped count,
and whether the plane is accepting new events. Drop warnings are aggregated by
the background worker rather than emitted once per dropped event. During graceful
shutdown, listeners are stopped first, accepted observations are drained, and only
then is the final sitemap snapshot persisted. Startup also restores the saved
sitemap before exposing data-plane listeners, avoiding early live observations
being overwritten by a later load.

The live passive-field merge used by `signalStore` was also split from the more
general crawl merge. Repeated request fields are now merged in place instead of
copying the entire field slice, building a temporary map, and constructing string
keys on every request; only genuinely new fields grow the slice.

Offline component benchmarks on the AMD EPYC 9V74 harness:

- original pre-P0-C synchronous request observation (sitemap + learner + signals):
  roughly 2.56–2.63 µs/op, 1,664 B/op, 24 allocs/op;
- the same stores after the live-field merge improvement, if called synchronously:
  roughly 443–445 ns/op, 48 B/op, 3 allocs/op;
- actual P0-C request-path enqueue: roughly 35–36 ns/op, 0 B/op, 0 allocs/op;
- host observation before P0-C: roughly 213–229 ns/op, 48 B/op, 2 allocs/op;
- P0-C host enqueue: roughly 36–38 ns/op, 0 B/op, 0 allocs/op.

Under the harness's parallel benchmark, the optimized stores called directly were
about 0.52–0.66 µs/op with 48 B/op and 3 allocs/op, while the bounded enqueue was
about 0.13–0.15 µs/op with zero allocations. These are component measurements,
not end-to-end Coraza/network throughput claims. The worker intentionally keeps
telemetry eventual: admin discovery/learner views may lag live traffic by the
queue processing interval, and overload may drop observations rather than affect
WAF availability.


## Data-plane performance: P0-D bounded match-log aggregation

P0-D removes the synchronous structured `slog.Warn` call from Coraza's matched-rule
callback. Match enforcement, the in-memory match ring, syslog forwarding, AI match
handling, and learner observation semantics remain unchanged; only the human-oriented
process log moves to a bounded background plane.

The match-log plane uses an 8,192-event non-blocking queue. Events aggregate for five
seconds by `site + rule_id + client`, with at most 1,024 active groups per window. Queue
or group saturation drops logging telemetry rather than delaying a WAF request. The
`/api/metrics` response exposes `match_logging` queue depth/capacity, total/queue/group
drops, processed events, emitted summaries, active/max groups, and accepting state.

Each summary carries one representative URI/message/matched-data sample plus a count and
first/last-seen timestamps. Only the process-log sample is bounded (URI/message 1 KiB,
matched data 2 KiB); the existing match ring and syslog forwarding retain their original
records. Drop telemetry is itself summarized at most once per flush interval.

AMD EPYC 9V74 component benchmark in the external Coraza-stub harness:

- successful P0-D enqueue: roughly 39.7–42.7 ns/op, 0 B/op, 0 allocs/op;
- old per-match JSON `slog.Warn` to `io.Discard`: roughly 1.45–1.51 µs/op,
  232 B/op, 8 allocs/op.

These are component measurements, not end-to-end Coraza throughput claims. Real Coraza v3.7.0 integration/load validation is still required before production release.

## Data-plane performance: P1 response inspection + backend connection pools

P1 makes two previously fixed data-plane choices explicit configuration.

### Response-body inspection policy

Each named WAF policy now accepts:

```json
{
  "response_body_inspection": "inherit",
  "response_body_limit": 0
}
```

`response_body_inspection` is `inherit`, `on`, or `off` (legacy empty string is
also treated as inherit). The override is appended after the policy's SecLang
file, so `on`/`off` can override the shipped `SecResponseBodyAccess` setting
without exposing raw SecLang through the admin API. `response_body_limit` is an
optional byte limit; zero leaves the rules-file value unchanged. Existing
configs therefore retain their prior response-inspection behavior until an
operator explicitly selects a policy override.

Response-body inspection remains a security/performance tradeoff rather than a
global performance switch. Keep it enabled for sites whose outbound content
needs WAF response rules; use a dedicated site/policy with inspection disabled
for static/download/API workloads where those response checks are not required.
The shipping admin console exposes both settings under **Policies**.

### Backend Transport / connection-pool tuning

Each pool now accepts a `transport` object:

```json
{
  "transport": {
    "max_idle_conns": 2048,
    "max_idle_conns_per_host": 256,
    "max_conns_per_host": 0,
    "idle_conn_timeout_sec": 90,
    "dial_timeout_sec": 5,
    "keep_alive_sec": 30
  }
}
```

Zero means the tuned default for every field except `max_conns_per_host`, where
zero deliberately means unlimited (the Go `http.Transport` behavior). Legacy
pool configs that omit `transport` automatically receive the tuned effective
defaults: 2,048 total idle connections, 256 idle connections per backend host,
unlimited total connections per host, a 90-second idle timeout, 5-second dial
timeout, and 30-second TCP keepalive. `backend_timeout_sec` continues to control
the response-header timeout.

The backend `http.Transport` is now **pool-scoped rather than site-scoped**.
Multiple sites that reference the same pool therefore share one backend
connection pool instead of maintaining duplicate keepalive pools. Runtime
replacement and shutdown call `CloseIdleConnections()` on the retired pool
transports; health-monitor transports also close idle connections when their
monitor goroutine exits. Active requests are not interrupted by this cleanup.

`GET /api/pools` now reports the effective transport settings so operators can
confirm which values are actually live. The production console exposes the
connection-pool settings under **Pools & Nodes**. These settings are sizing
controls, not a throughput guarantee: HTTP/2 multiplexing, backend response
latency, upstream connection limits, NAT/conntrack, TLS, and kernel/NIC behavior
still determine real capacity.

## Performance baseline tool: `wafbench`

The repository now includes a repeatable benchmark harness under `cmd/wafbench`.
It is designed for low-production-traffic environments where real traffic is not
large enough to reveal whether the next data-plane optimization should target
Coraza/CRS inspection or the L3/L4 path.

Build it separately from the production WAF binary:

```bash
./benchmark/build.sh
# or: go build -o ./bin/wafbench ./cmd/wafbench
```

The tool provides six commands:

- `backend` — deterministic local HTTP backend with a fixed response size;
- `http` — full running-waf load test with clean GET, 1/16/64/256 KiB JSON,
  SQLi, XSS, and traversal workloads;
- `coraza` — direct Coraza/CRS transactions with no NIC/TCP/TLS/reverse-proxy
  or backend cost, with optional CPU/heap pprof output and response-inspection
  on/off overrides;
- `l4` — legal TCP connect/close or TLS-handshake pressure for connection-rate,
  softirq and PPS baselining (no raw packet spoofing/flooding);
- `compare` — compares JSON HTTP + Coraza results and optional L4 results. The
  sizing heuristic uses clean traffic only for the Coraza-share median and
  surfaces whether regex acceleration, XDP, or more profiling is the stronger
  next experiment.
- `certify` — binds real full-proxy benchmark JSON to an explicit approved
  performance target, records evidence hashes and comparability checks, and
  keeps certification `NOT_RUN` when targets or required runtime evidence are
  absent. VectorScan certification additionally requires the real Phase 1
  zero-false-negative report.

Linux runs can add `--pid <waf-proxy-pid>` to measure WAF process CPU/RSS and
CPU microseconds per operation, plus `--iface <nic>` for RX/TX Mbps and PPS.
The load generator also reports its own CPU so a generator-limited run is not
mistaken for a WAF ceiling. JSON result files include CPU model, kernel, Go
version, architecture, and CPU count for repeatability.

See [`benchmark/README.md`](benchmark/README.md) for the exact baseline flow,
response-inspection comparisons, 1/4/8-core matrix, pprof commands, and the
XDP-vs-regex interpretation rules.


## P2 non-XDP/VectorScan hardening

This slice closes the remaining low-risk performance/release debt that does not depend on
choosing XDP or a regex accelerator.

### AI lazy capture and queue hardening

AI-enabled sites still preserve the rule that a Coraza match can be analyzed even when the
clean-request sample rate is zero. The request path no longer eagerly builds a full
`analysisJob` for every AI-enabled request, though. It now registers a lightweight live
request pointer keyed by `{client, URI}` while the WAF runs. Header filtering/redaction, query
redaction, body-prefix attachment, timestamping, and `analysisJob` construction happen only
when Coraza actually matches or when the request was pre-selected by the clean-traffic sample.
The pending key is a small struct rather than a concatenated string, removing the prior key
allocation.

AI queue saturation is still fail-open, but repeated drops no longer perform one synchronous
warning log per request. Drops are counted and the warning is rate-limited to at most once per
five seconds. `GET /api/metrics` exposes `ai_queue` depth, physical capacity, configured logical
limit, cumulative enqueued count, and cumulative dropped count. AI workers reuse a ticker
instead of allocating `time.After` timers in their polling loop.

For AI `block` mode, the dynamic blocklist is also published as an immutable atomic snapshot.
The request path performs an atomic load plus map lookup and never takes the blocklist write
mutex; add/unblock/expiry cleanup use copy-on-write under the update mutex. Expired entries are
ignored immediately by reads and pruned by the janitor off the request path. In the component
benchmark, an active-block lookup under 8/32-way parallelism measured about 22/21 ns/op versus
about 93/115 ns/op for the former mutex-protected lookup.

On the AMD EPYC 9V74 external stub harness, the unsampled/no-match AI wrapper path measured
about **169–175 ns/op, 0 B/op, 0 allocs/op**, versus the pre-lazy eager-capture model at about
**1.49–1.55 µs/op, 848 B/op, 17 allocs/op**. These are component measurements, not LLM or
end-to-end Coraza throughput claims.

### ResponseWriter correctness

The remaining `statusRecorder` now suppresses duplicate `WriteHeader` calls instead of
forwarding later status codes to the underlying writer. Implicit `Write` records HTTP 200 when
no explicit status was set. `Unwrap()` remains in place so `http.ResponseController` can reach
optional capabilities such as `Hijacker`.

### Release scripts

`build.sh`, `install.sh`, `setup-interfaces.sh`, `uninstall.sh`, `upgrade.sh`,
`waf-doctor.sh`, and `benchmark/build.sh` are LF-normalized and executable. All pass `bash -n`.
A repository test now fails if carriage returns reappear, executable mode is lost, or Bash
syntax regresses. This resolves the inherited installer EOF/CRLF release blocker documented by
P0-B through P1.

### Access-ring sharding decision

A 16-shard prototype was benchmarked before changing production code and was rejected. The
current fixed circular ring is ~9.5–10.2 ns/op serial; under the test host it was ~26–36 ns/op
at 8-way parallelism and ~43–49 ns/op at 32-way parallelism. The sharded prototype regressed to
~14.6–15.1 ns/op serial, ~42 ns/op at 8-way and ~61 ns/op at 32-way because the global sequence
atomic plus shard lock cost more than the existing short critical section. Production therefore
keeps the simpler fixed ring until a real profile proves it is a bottleneck.

## TLS acceleration frontend (C1-C3)

`tls_acceleration` is optional and defaults to `mode: "go"`, preserving the
existing Go `crypto/tls` path. `mode: "frontend"` moves public TLS termination
to the companion `waf-tlsfront`, which renders and supervises NGINX/OpenSSL and
forwards decrypted HTTP over private Unix-domain sockets back to waf-proxy.
Coraza, policy evaluation, backend selection, and logging remain in waf-proxy.

```json
"tls_acceleration": {
  "mode": "frontend",
  "ktls": "auto",
  "qat": "auto",
  "worker_processes": 0,
  "worker_connections": 4096,
  "http2": true
}
```

`ktls` and `qat` accept `off`, `auto`, or `required`. `auto` safely falls back
to ordinary OpenSSL software TLS when the capability is absent; `required`
rejects Apply during preflight instead of silently degrading. QAT uses the
OpenSSL 3 `qatprovider` path and also loads the default provider. Real QAT
hardware/provider validation remains deployment-specific and is not claimed by
this package.

HTTP/2 syntax is selected from the detected NGINX version. NGINX 1.25.1 and
newer receive the modern `listen ... ssl;` plus `http2 on;` form; older NGINX
receives the compatible legacy `listen ... ssl http2;` form. This keeps one
package compatible with both generations while avoiding the deprecation warning
on current NGINX.

The public client cannot supply trusted forwarding metadata: the frontend
clears Internet-provided forwarding headers, writes private client IP/port/proto
headers itself, and waf-proxy accepts those headers only on its private Unix
listener. Before switching modes, waf-proxy performs host capability checks and
an `nginx -t` preflight, then publishes the live control file consumed by the
companion. The Setup tab exposes configuration and live probe/resolution state.



## External HSM / PKCS#11 TLS keys — Phase 5 Slice G

The built-in Go TLS listener can keep a site's private key inside an external
PKCS#11 token/HSM. The certificate chain remains a normal PEM file, while the
private key is represented by a Go `crypto.Signer` backed by an exact PKCS#11
slot/token + key label/ID lookup. The WAF never exports or persists HSM private
key material.

PKCS#11 is an explicit build capability. To build an HSM-capable Coraza-only
binary without requiring VectorScan:

```bash
WAF_VECTORSCAN=off WAF_HSM_PKCS11=required ./build.sh
```

To combine native VectorScan and PKCS#11:

```bash
WAF_VECTORSCAN=required WAF_HSM_PKCS11=required ./build.sh
```

`WAF_HSM_PKCS11=off` is the default so the pre-existing portable Coraza build
remains CGO-free. `auto` enables the provider only when Linux + a CGO compiler
are available. The PKCS#11 module itself is loaded at runtime with `dlopen`; it
must be an absolute, regular, non-symlink, root-owned file under an approved
`hsm.allowed_module_dirs` tree, with no group/world write permission.

Example site configuration:

```json
{
  "hsm": {
    "allowed_module_dirs": ["/usr/lib", "/usr/local/lib", "/opt/vendor"]
  },
  "tls_acceleration": {"mode": "go"},
  "sites": [{
    "name": "payments",
    "listen": ":443",
    "hostnames": ["payments.example.com"],
    "tls_cert": "/etc/waf/certs/payments-chain.pem",
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
  }]
}
```

`pin_secret_ref` accepts only `env:NAME` or `file:/absolute/path`; inline PINs
are rejected. File-backed PINs must be regular, non-symlink files with no
group/world permissions and are preferred over environment secrets for
production. Admin config responses redact the secret reference, HSM audit
records contain only provider / slot / key reference / operation / result, and
qualification reports intentionally omit the PIN, secret reference, module
path, and token label.

HSM-backed sites cannot also configure `tls_key`. There is **no automatic
filesystem-key fallback**. The external NGINX/OpenSSL TLS frontend is also
rejected for HSM-backed sites; use `tls_acceleration.mode=go` so TLS handshakes
call the HSM-backed `crypto.Signer` directly. Apply verifies the certificate
public key by signing and verifying a challenge before runtime swap. HSM
initialization, login, key lookup, certificate association, or signing failure
therefore fails closed.

Runtime health is available at `GET /api/hsm/status`; restricted audit evidence
is available at `GET /api/hsm/audit`. SoftHSM and real-vendor qualification
instructions live in `qualification/hsm/`. A SoftHSM PASS proves the software
PKCS#11 path only and must never be promoted to vendor-HSM qualification.

## VectorScan Learning Accelerator — 2026-09-04

The optional VectorScan path is a **learning accelerator**, not a replacement WAF engine. Coraza v3.7.0 remains authoritative. Eligible standalone positive `@rx` rules are conservatively grouped only when the request data source and transformation semantics can be reproduced exactly; unsupported rules remain `CORAZA_ONLY`. Initial supported sources are `REQUEST_URI`, `REQUEST_FILENAME`, `REQUEST_METHOD`, `REQUEST_PROTOCOL`, and fixed-name `REQUEST_HEADERS:name`, with explicit `t:none` and optional `t:lowercase`. Chains, negated regex, aggregate/multi-variable selectors, ARGS/body rules, and unsupported transforms are not accelerated.

Each group moves through `LEARNING -> VALIDATED -> ACCELERATED`. Learning compares VectorScan candidates with the transaction-final `tx.MatchedRules()` set after Coraza `ProcessLogging()`. The wrapper uses Coraza v3.7's context-aware transaction creation so concurrent HTTP/2 requests are correlated by request context rather than client/URI heuristics. A false negative or native scan error immediately puts that group into `FAILSAFE`; Coraza-only processing continues. While accelerated, no-hit groups are periodically fully verified according to `verification_sample_rate`. Rule/adapter/Coraza/native-version fingerprints invalidate stale learning state automatically.

Config:

```json
{
  "vector_acceleration": {
    "mode": "off",
    "state_path": "/var/lib/waf-proxy/vector-learning.json",
    "min_samples": 100000,
    "min_coraza_matches": 10,
    "min_learning_sec": 3600,
    "verification_sample_rate": 0.01
  }
}
```

`mode` is `off`, `auto`, or `required`. `auto` falls back to Coraza-only if libhs is unavailable; `required` fails the runtime build if native VectorScan is missing. `/api/vector-acceleration` and `/api/metrics.vector_acceleration` expose group state, samples, Coraza matches, false negatives, skips and scan errors. Reviewer-or-higher can reset a site from FAILSAFE back to Learning through `POST /api/vector-acceleration/reset`.

Build policy is controlled by `WAF_VECTORSCAN=off|auto|required`. Coraza v3.7.0 declares Go 1.25.0, and `build.sh` now checks the installed local toolchain with `GOTOOLCHAIN=local` so an old host is rejected deterministically instead of triggering an implicit toolchain download. `qualify-release-host.sh --preflight` checks the release-host prerequisites and real VectorScan provenance; exit code 3 means **BLOCKED**, not a WAF correctness failure. `--core` then runs the required native build/test/race path, the real-Coraza DetectionOnly+nolog `MatchedRules()` gate, and a native multi-pattern compile/scan semantic gate. Historical earlier-baseline evidence includes portable Coraza-API-stub regression/vet/race and a native libhs ABI compile/vet/race gate. The exact current Phase 3 source baseline has **not** rerun the canonical Go 1.25 suite on this packaging host because only Go 1.23.2 is installed. **Real Go 1.25 + Coraza v3.7.0 execution and real libvectorscan matching remain NOT_RUN until `--core` passes on a qualified release/target host**; historical stub/ABI evidence is not current-baseline or production qualification evidence.

## Operator diagnostics and Phase 1 qualification — 2026-09-13

`wafctl` is the installed operator/support CLI. API-oriented commands use the authenticated admin API; local commands such as `coverage analyze`, local support collection, and release-signature verification operate on local files/tools and do not imply an admin-API call.

```bash
wafctl doctor
wafctl debug capture --tenant SITE --duration 5m
wafctl debug list --tenant SITE
wafctl debug export --tenant SITE --transaction-id ID --output incident.zip
wafctl debug stop --tenant SITE
wafctl support bundle --output waf-support.zip
```

Debug capture is disabled by default. When enabled for one site, the WAF generates a transaction ID and correlates request/response metadata, TLS, selected upstream/LB evidence, Coraza's final `MatchedRules()` and any VectorScan candidate/FN evidence. Request/response bodies are not captured by this path; only an allow-listed request-header subset is recorded, sensitive keys are masked, retention is bounded by `WAF_DEBUG_TTL` (default 24h) and `WAF_DEBUG_MAX_ENTRIES` (default 1000), and expired evidence is cleaned periodically. Debug evidence is support-only and never changes a Coraza verdict.

`wafctl support bundle` collects sanitized config/status/metrics, bounded logs when locally readable, runtime/dependency versions, SPDX-formatted SBOM evidence and optional exact-transaction incident evidence. It writes a manifest and SHA-256 list and rejects obvious secret/private-key material.

The real Phase 1 differential runner is:

```bash
./run-phase1-qualification.sh --rules /etc/waf/coraza.conf \
  --corpus qualification/corpus/default.jsonl \
  --output phase1-qualification.json
```

It must run with Go >=1.25 and real `libvectorscan`/libhs. It executes real Coraza in DetectionOnly and the actual native VectorScan scanner, compares only currently eligible accelerated rules, and requires zero observed false negatives plus sufficient eligible-match evidence. Exit 3 is BLOCKED/NOT_RUN; it is never a PASS substitute.

### CRS VectorScan coverage analysis

The operator CLI can inspect a Coraza/OWASP CRS configuration without enabling acceleration:

```bash
wafctl coverage analyze --rules /etc/waf/coraza.conf \
  --report phase2-coverage-report-v2.json \
  --inventory coverage-inventory.json
```

The report is conservative and mirrors the current VectorScan runtime classifier. `ELIGIBLE` is capability inventory only; it does **not** promote a rule to LEARNING, VALIDATED or ACCELERATED. Real Coraza/VectorScan differential qualification with zero observed false negatives remains mandatory.

## Phase 3 release hardening

Release engineering treats provenance and packaging as security controls. The source ZIP builder now identifies its output only as `SOURCE_ARCHIVE`; portable/native **binary** identity is recorded only for actual binaries produced by `build.sh`. Source releases generate schema-v2 `RELEASE_EVIDENCE.json`, `PROVENANCE.json`, SPDX 2.3 and CycloneDX 1.5 SBOMs, a complete source SHA-256 manifest, and deterministic metadata when `SOURCE_DATE_EPOCH` is supplied. `release-security-scan.sh` records source-bound `govulncheck` PASS/FAIL/BLOCKED/NOT_RUN evidence; missing prerequisites are never converted to PASS. The artifact verifier requires complete source/release manifests, provenance cross-digests, SBOM↔`go.mod` parity, and truthful source-artifact identity. Detached minisign signing remains optional; authenticity exists only after `verify-release-signature.sh` or `wafctl release verify-signature` validates the detached signature with an approved public key. Unsigned provenance is integrity-bound metadata, not authenticated producer evidence.


## Enterprise package upgrade / rollback qualification — Distribution Slice C

The formal `.deb` and `.rpm` paths now share one package-lifecycle qualification
harness under `packaging/qualification/`. The runner is non-mutating by default;
real execution requires root on a dedicated disposable host plus the exact
`I_UNDERSTAND_THIS_MUTATES_A_DEDICATED_HOST` acknowledgement. It does not call
`apt`, `dnf`, `yum`, `curl`, or `wget`, so lifecycle qualification preserves the
offline-safe distribution contract.

Version N, N+1 and an optional intentional-failure package are bound into the
JSON evidence by native package metadata and SHA-256. A real run verifies local
configuration bytes, the generated admin secret, `/var/lib/waf-proxy` state and
service active-state across upgrade, failed upgrade/recovery and rollback. The
secret is compared only in memory and neither its value nor a digest is written
to evidence. Changed packaged defaults can exercise `.dpkg-dist` or `.rpmnew`;
`.dpkg-old` / `.rpmsave` are recorded only when the native package manager emits
them.

`build-lifecycle-fixtures.sh` can create deterministic semantic fixtures using
`/bin/true`. A fixture or preflight PASS proves packaging mechanics only. Real
package-manager execution and the Slice D clean-host distro matrix remain
separate gates.

## Clean-host distribution qualification — Distribution Slice D

The formal DEB/RPM distribution paths now include a dedicated clean-machine
acceptance harness under `packaging/cleanhost/`. It is **not** a container or
source-install smoke test. A PASS requires a real dedicated host matching one of
these exact targets: Debian 12, Ubuntu 22.04/24.04, RHEL 9, Rocky 9, AlmaLinux 9,
or Oracle Linux 9. RHEL-family runs additionally require SELinux `Enforcing`.

The runner uses only local packages and a local approved CRS directory. It never
calls apt/dnf/yum/curl/wget/git. It validates fresh-install no-autostart, explicit
CRS provisioning, `waf-doctor --check`, explicit systemd start, `/healthz`,
break-glass authenticated admin API access, real reverse-proxy traffic, Version
N→N+1 upgrade preservation, and non-purge package removal with persistent state
retained. Preflight/source/container results remain `NOT_RUN` and cannot promote
a distro in the checked-in matrix.

## Project-local DEB/RPM builder — `waf-package`

The repository now has one project-specific packaging entry point:

```bash
./waf-package doctor --format deb
./waf-package deb --version 2026.09.16.1
./waf-package doctor --format rpm
./waf-package rpm --version 2026.09.16.1 --rpm-release 1
```

`./waf-package all` builds the canonical binaries once and packages the exact
same provenance-bound bytes as both DEB and RPM. It reads the Go/Coraza pins
from `go.mod`, forces `GOTOOLCHAIN=local`, runs the existing `build.sh` gates,
and then invokes the existing format-specific builders/verifiers. Missing prerequisites fail closed. Go module access is offline by default and
requires explicit `--allow-module-network`; the utility never bootstraps a Go
toolchain, OS packages, CRS, or native libraries. See `PACKAGING_TOOL.md` for
host requirements and examples.


## API Security Roadmap

Future API-aware WAF evolution is documented in `API_SECURITY_ROADMAP.md`.
