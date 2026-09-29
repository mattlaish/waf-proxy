# Candidate native → OWI-1.0 mapping (not active)

| Native authority | Wire kind/field | Treatment and revision trigger |
|---|---|---|
| Runtime `Config.Nodes`, `Config.Pools`, `Config.Sites`, source instance | `ASSET`: distinct WAF node, origin node, origin pool, site, virtual service IDs | Stable hashed IDs; names, listen IPs, raw config and secrets excluded. UPSERT on runtime projection change; DELETE on disappearance. |
| Coraza match callback (not yet wired) | `DETECTION`: category, rule ID, site ref; rule_version null | Selected rule match only; no URI/query/body/cookie/header/IP. Durable sequence on append. Rule match alone does not create HIGH_SIGNAL_ATTACK attention. HISTORY target 30 days is not load-qualified. |
| Active runtime per-site mode and build time | `POLICY`: `active_generation`, enforcement_state, site refs | Active generation derives from runtime build; desired generation and content digest null. Never exports rule text. |
| Runtime origin-pool monitor and healthy member count (not yet scheduled) | `HEALTH`: state, metrics with unit/window/threshold | UNKNOWN if monitor disabled; all-monitored-members-down may yield PROTECTION_DEGRADED. Capacity and latency claims omitted without instrumentation. |
| Admin/session/HA bearer | No wire authority | Explicitly rejected; distinct machine credential only. |

The JSONL state is a proposed reader cache, not an independent WAF execution
authority. `ASSET`/`POLICY`/`HEALTH` are STATE; `DETECTION` is HISTORY. Current
64 MiB journal cap reports GAP rather than silently discarding; sustained
30-day retention has not been demonstrated. Tenant is bound by machine config,
not copied from an arbitrary request parameter.

