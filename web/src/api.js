// Thin API client. Mirrors the single-file console's auth model: a bearer
// token kept in sessionStorage, prompted once, re-prompted on 401.

function token(fresh) {
  if (fresh) sessionStorage.removeItem("waf_token");
  let t = sessionStorage.getItem("waf_token");
  if (!t) {
    t = prompt("Admin token (printed at waf-proxy startup):") || "";
    sessionStorage.setItem("waf_token", t);
  }
  return t;
}

export async function api(path, opts = {}) {
  opts.headers = Object.assign(
    { Authorization: "Bearer " + token(), "Content-Type": "application/json" },
    opts.headers || {}
  );
  const r = await fetch(path, opts);
  if (r.status === 401) {
    token(true);
    throw new Error("unauthorized — token rejected");
  }
  if (!r.ok) throw new Error((await r.text()).trim() || String(r.status));
  return r.json();
}

export const getConfig = () => api("/api/config");
export const putConfig = (cfg) => api("/api/config", { method: "PUT", body: JSON.stringify(cfg) });
export const getStatus = () => api("/api/status");
export const getNotifications = () => api("/api/notifications?limit=100");
export const notifyRead = (all = true, id = 0) =>
  api("/api/notifications/read", { method: "POST", body: JSON.stringify({ all, id }) });
export const notifyDismiss = (id, all = false) =>
  api("/api/notifications/dismiss", { method: "POST", body: JSON.stringify({ id, all }) });
export const notifyApply = (id) =>
  api("/api/notifications/apply", { method: "POST", body: JSON.stringify({ id }) });
export const haSync = () => api("/api/ha/sync", { method: "POST" });
