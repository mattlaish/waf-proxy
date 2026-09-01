# Experimental console migration: single-file → Vite + preact

> **Status: experimental and non-shipping.** Do not deploy `static/app` or
> replace `static/admin.html`. This tree is a partial migration reference until
> every tab is ported, dependencies are made reproducible with a lockfile, and
> feature-parity tests pass.

The shipping console is still `static/admin.html` (one self-contained file, no build
step — ideal for an air-gapped box). This `web/` project is the migration target for
when that file gets too big to maintain comfortably.

## What's done here

- **Build + dev**: `vite.config.js` (dev proxies `/api` and `/healthz` to `127.0.0.1:9090`; build emits to `../static/app`).
- **Shared infra**: `src/api.js` (bearer-token client + endpoint helpers), `src/store.js` (`useConfig` draft/save hook + `usePoll`), `src/style.css` (design tokens ported 1:1).
- **App shell**: `src/App.jsx` — header, tab routing, toast.
- **Bell**: `src/components/Bell.jsx` — notification dropdown with apply/dismiss/mark-read.
- **Exemplar tab**: `src/tabs/HATab.jsx` — the HA + Notifications settings, fully ported. Use it as the template.

## What's left

Port these tabs from `static/admin.html` into `src/tabs/`, each taking `{ cfg, set, save, onToast }`:

- [ ] `ConfigTab` — sites (listen, hostnames, pool, policy, engine mode, AI mode, TLS + file browser), global timeouts, engine selector.
- [ ] `PoolsTab` — nodes + pools (LB method, monitor, members) with live `/api/pools` health.
- [ ] `PoliciesTab` — rules path (file browser), paranoia, body limit, exclusions editor.
- [ ] `SetupTab` (AI half) — LLM connector + analysis policy + verdicts/blocklist feeds. Merge with `HATab` under the "setup" route, or keep them as sub-sections.
- [ ] `SiteMapTab` — path tree + crawl + "suggest policy fit" learner panel.
- [ ] File browser — extract the cert/rules picker modal into `components/FileBrowser.jsx`.

## Pattern

Each tab reads its slice of `cfg` and edits with `set(key, partial)`:

```jsx
const ai = cfg.ai || {};
<input value={ai.model || ""} onInput={(e) => set("ai", { model: e.target.value })} />
<button onClick={() => save()}>Save</button>
```

`set(key, partial)` shallow-merges into `cfg[key]`; pass an array to replace (e.g. `set("sites", nextSites)`).

## Build into the binary

```bash
cd web && npm install && npm run build   # emits ../static/app/
```

Then either point the Go admin server at `static/app/index.html`, or add it to the
`go:embed` set. Until the port is complete, keep serving `static/admin.html` (which
remains fully functional and is what `admin.go` embeds today).

## Note on `@preact/preset-vite`

`vite.config.js` uses the preset if present (Fast Refresh) and silently falls back
without it. If `npm install` doesn't pull it, add it: `npm i -D @preact/preset-vite`,
or remove the preset import and rely on the `react → preact/compat` alias alone.
