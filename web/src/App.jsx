import { useState, useEffect } from "preact/hooks";
import { getStatus } from "./api.js";
import { useConfig } from "./store.js";
import { Bell } from "./components/Bell.jsx";
import { HATab } from "./tabs/HATab.jsx";

// Tab registry. HATab is a fully-ported exemplar; the others are stubs that
// share the same { cfg, set, save, onToast } contract — port each from the
// single-file console (static/admin.html) one at a time. See PORTING.md.
const TABS = [
  ["config", "CONFIG"],
  ["pools", "POOLS & NODES"],
  ["policies", "POLICIES"],
  ["setup", "SETUP / AI · HA"],
  ["map", "SITE MAP"],
];

function Stub({ name }) {
  return (
    <section>
      <div class="eyebrow">{name}</div>
      <div class="hint">
        Not yet ported to the component build. The working implementation lives in
        <code> static/admin.html</code>; port it into <code>src/tabs/</code> using the
        same <code>{"{ cfg, set, save, onToast }"}</code> contract as <code>HATab.jsx</code>.
      </div>
    </section>
  );
}

export function App() {
  const [tab, setTab] = useState("setup");
  const [toast, setToast] = useState(null);
  const [status, setStatus] = useState(null);
  const onToast = (msg, cls = "") => setToast({ msg, cls, t: Date.now() });
  const { cfg, set, save } = useConfig(onToast);

  useEffect(() => {
    if (!toast) return;
    const id = setTimeout(() => setToast(null), 3500);
    return () => clearTimeout(id);
  }, [toast]);

  useEffect(() => {
    const id = setInterval(() => getStatus().then(setStatus).catch(() => {}), 5000);
    getStatus().then(setStatus).catch(() => {});
    return () => clearInterval(id);
  }, []);

  return (
    <div class="wrap">
      <header>
        <h1>waf-proxy <span>/ console</span></h1>
        <div class="up">{status ? "up " + status.uptime : "—"}</div>
        <Bell unread={(status && status.notify_unread) || 0} onToast={onToast} />
      </header>

      <nav class="tabs">
        {TABS.map(([id, lbl]) => (
          <button key={id} class={`tab ${tab === id ? "sel" : ""}`} onClick={() => setTab(id)}>{lbl}</button>
        ))}
      </nav>

      {!cfg ? (
        <section><div class="hint">loading…</div></section>
      ) : tab === "setup" ? (
        <HATab cfg={cfg} set={set} save={save} onToast={onToast} />
      ) : (
        <Stub name={TABS.find((t) => t[0] === tab)[1]} />
      )}

      {toast && <div class={`show ${toast.cls}`} id="toast">{toast.msg}</div>}
    </div>
  );
}
