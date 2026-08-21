# waf-proxy

Multi-site, **load-balancing** reverse proxy with an embedded [Coraza](https://github.com/corazawaf/coraza) WAF engine (SecLang-compatible, runs OWASP CRS 4.x) and a built-in admin console. No CGO, no CDN dependencies — the console works on an air-gapped management segment.

```
                          ┌ site blog (:443, blog.example.com) ─┐
client ─▶ listener :443 ──┤                                     ├─▶ pool ─▶ member (node:port)
          (SNI + WAF)      └ site shop (:443, shop.example.com) ┘         └▶ member (node:port)
client ─▶ listener :8443 ─ site api  (:8443, api.example.com) ───▶ pool ─▶ ...
```

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
sudo apt install golang-go        # want Go ≥ 1.22
go mod init waf-proxy
go get github.com/corazawaf/coraza/v3@latest
go build -o waf-proxy .
```

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

Connect an LLM (OpenAI, or any OpenAI-compatible endpoint — vLLM, Ollama, llama.cpp, a local gateway — or Anthropic) and have it judge traffic. Configure the connector (provider, base URL, model, API key, timeout), an analysis policy, and redaction, then set each site's **AI enforcement** mode:

- **off** — no AI.
- **advisory** — AI analyzes and logs a verdict; never blocks.
- **block** — a high-confidence malicious verdict adds the source IP to a **dynamic blocklist** (with TTL); subsequent requests are dropped inline.

**How it stays fast and safe:**

- **Async, never in the hot path.** WAF-flagged requests (any engine mode) and a configurable sample of others are queued; a small worker pool calls the LLM. The only inline cost on the data path is a blocklist map lookup. LLM latency never touches live requests.
- **Fail-open, always.** Disabled/slow/errored/garbage output ⇒ nothing blocked. The AI can only ever *add* a time-boxed block via a schema-valid, high-score verdict.
- **Privacy/PHI.** Credentials (`Authorization`, `Cookie`, `X-Api-Key`, …) are **never** sent, regardless of settings. Redaction is on by default (only a safe header subset leaves; body omitted). Optional client-IP hashing. Point it at a self-hosted model to keep data on-prem. **The config file stores the API key in cleartext at `0600` — treat it as a secret at rest.**
- **Prompt injection.** The request is wrapped in delimiters and the model is told to treat it as untrusted data; only a parsed, validated JSON verdict (`{verdict, score, category, reason}`) can affect control flow — never the model's prose. This is defense-in-depth, not a guarantee: keep CRS doing the deterministic blocking and use the AI as a second opinion.

The tab shows a live **AI blocklist** (with one-click unblock) and a **verdicts** feed (verdict, score, category, action, reason). **Test connection** runs a canned SQLi sample end-to-end so you can validate the connector before enabling enforcement.

**Prompts (system + user).** The analyzer sends the model two parts. The **system prompt** is fixed and injection-hardened — it instructs the model to act as a WAF analyst, to treat everything inside the request delimiters as untrusted attacker-controlled data, and to ignore any instructions embedded in that data. It is compiled into the binary and is **not** editable from the console, on purpose: an edit that weakened it would let hostile traffic prompt-inject the very analyzer inspecting it. The **user prompt** is built per-request from the captured HTTP data (method, path, safe header subset, optionally body) inside `<request>…</request>` delimiters. Only a parsed, schema-valid JSON verdict (`{verdict, score, category, reason}`) can affect control flow — never the model's free-text output. "Analysis policy" (threshold, only-on-match, sample rate) is part of the **global LLM connector** config; whether a given site *uses* the analyzer, and whether a verdict may block, is the **per-site AI mode** above — the two are deliberately separate (one connection, per-site policy).

API: `GET /api/ai/verdicts` · `GET /api/ai/blocklist` · `POST /api/ai/unblock {ip}` · `POST /api/ai/test`. The API key is masked on read and preserved when the form submits it blank.

**Save draft vs Apply.** The console separates persisting your work from making it live. **Save draft** (`PUT /api/config?draft=1`) writes your work-in-progress to disk with only light structural checks — so you can build a config incrementally (add a node, save; add a pool, save) without every half-finished state being fully consistent, and the running WAF is untouched. **Apply** (`PUT /api/config`) validates the whole config (cross-references, rules files, everything) and swaps it into the live engine. Full validation errors — an unknown pool, a member with no node — surface at Apply, which is the correct place for them, not while you're still wiring things up.

**Everything applies live.** On Apply: pools, members, monitors, LB methods, engine modes, certificates, **and listen addresses** all reconcile via atomic swap plus live listener management — a bad config leaves the old runtime serving. Adding or removing a listen address opens or closes **only that socket**; unchanged listeners and their in-flight connections are untouched, so building a new site never interrupts existing traffic. No process restart is needed for any config change.

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
retains names only—never field values or uploaded content. Crawler and passive
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




The shipping console is `static/admin.html` — one self-contained file, no build step (ideal air-gapped). A Vite + preact migration scaffold lives in `web/` with the shell, API client, config store, notification bell, and the HA/Notifications tab fully ported as the pattern; see `web/PORTING.md` for finishing the remaining tabs.

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
are from the 2026-08-21 scan of commit `aac312f`.

### Static analysis

```bash
go vet ./...                                   # clean
gofmt -l .                                     # style only; see note below
staticcheck ./...                              # honnef.co/go/tools/cmd/staticcheck
golangci-lint run ./...
gosec -severity=low -confidence=low ./...      # github.com/securego/gosec/v2
go mod verify                                  # all modules verified
bash -n *.sh                                   # all six scripts parse
node --check <inline JS from static/admin.html>
```

`go vet` is clean. `staticcheck` reports two unused functions
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
- **X-Forwarded-For spoofing is neutralised**: client-supplied `X-Forwarded-For`
  and `X-Real-IP` are replaced before the request reaches the backend.

> **Corollary worth knowing before deployment.** Because inbound
> `X-Forwarded-For` is replaced rather than appended, the address the backend
> receives — and the one used for `ip_hash`, the access log, syslog, and the
> learner's distinct-client signal — is always the immediate TCP peer. Deploy
> this at the edge and that is correct. Deploy it behind a CDN or another load
> balancer and every one of those consumers sees the upstream proxy instead of
> the real client. Nothing errors; the data is just wrong.

When scanning with a reduced ruleset instead of full CRS, remember that CRS
supplies the transformations: a rule written as `@contains ../` will miss
`%2e%2e%2f` without `t:urlDecodeUni`. A miss under a hand-written test ruleset
is usually the ruleset, not the engine.
