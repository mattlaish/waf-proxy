import { useState, useEffect } from "preact/hooks";
import { getStatus, haSync } from "../api.js";

// Fully-ported exemplar tab: HA peer sync + notification settings. Shows the
// pattern for the remaining tabs — read a slice of cfg, edit via set(), save().
export function HATab({ cfg, set, save, onToast }) {
  const ha = cfg.ha || {};
  const nt = cfg.notify || {};
  const [status, setStatus] = useState(null);

  useEffect(() => {
    const id = setInterval(() => getStatus().then(setStatus).catch(() => {}), 3000);
    getStatus().then(setStatus).catch(() => {});
    return () => clearInterval(id);
  }, []);

  const h = status && status.ha;
  return (
    <section>
      <div class="grid2">
        <div>
          <div class="eyebrow">High availability</div>
          <div class="field check">
            <input type="checkbox" checked={!!ha.enabled} onChange={(e) => set("ha", { enabled: e.target.checked })} />
            <label style="margin:0">Enable HA peer sync</label>
          </div>
          <div class="row">
            <div class="field"><label>Role</label>
              <select value={ha.role || "primary"} onChange={(e) => set("ha", { role: e.target.value })}>
                <option value="primary">primary</option>
                <option value="secondary">secondary</option>
              </select>
            </div>
            <div class="field" style="display:flex;align-items:flex-end">
              <div class="check">
                <input type="checkbox" checked={ha.sync_config !== false} onChange={(e) => set("ha", { sync_config: e.target.checked })} />
                <label style="margin:0">Sync config to peer</label>
              </div>
            </div>
          </div>
          <div class="field"><label>Peer admin URL</label>
            <input type="text" value={ha.peer_url || ""} placeholder="https://10.0.0.6:9090"
              onInput={(e) => set("ha", { peer_url: e.target.value })} />
          </div>
          <div class="field">
            <label>Peer admin token {status && (status.ha_token_set ? "· set" : "· not set")}</label>
            <input type="password" placeholder="leave blank to keep current"
              onInput={(e) => set("ha", { peer_token: e.target.value })} />
          </div>
          {status && (
            <div class="strip">
              {status.ha_enabled && h ? (
                <>
                  <div>peer <b style={{ color: h.peer_up ? "var(--green)" : "var(--red)" }}>{h.peer_up ? "up" : "down"}</b></div>
                  <div>role <b>{h.role || "solo"}</b></div>
                  {h.last_sync && <div>last sync <b>{h.last_sync}</b></div>}
                </>
              ) : <div>HA <b>disabled</b></div>}
            </div>
          )}
          <div class="actions">
            <button class="btn primary" onClick={() => save().then(() => onToast("HA applied")).catch(() => {})}>Save &amp; apply</button>
            <button class="btn" onClick={() => haSync().then(() => onToast("sync triggered")).catch((e) => onToast(e.message, "err"))}>Sync now</button>
          </div>
          <div class="hint">
            waf-proxy syncs config and computes an active/standby role from peer health.
            Real VIP failover belongs to keepalived/VRRP or your load balancer — point its
            health check at <code>/healthz</code> and read role from <code>/api/ha</code>.
          </div>
        </div>

        <div>
          <div class="eyebrow">Notifications</div>
          <div class="field"><label>Webhook URL (optional)</label>
            <input type="text" value={nt.webhook_url || ""} placeholder="https://hooks.slack.com/..."
              onInput={(e) => set("notify", { webhook_url: e.target.value })} />
          </div>
          <div class="field"><label>Webhook format</label>
            <select value={nt.webhook_kind || "slack"} onChange={(e) => set("notify", { webhook_kind: e.target.value })}>
              <option value="slack">Slack / Teams text</option>
              <option value="generic">Generic JSON</option>
            </select>
          </div>
          {[
            ["on_suggestion", "Policy-fit suggestions"],
            ["on_ai_block", "AI blocks"],
            ["on_member_down", "Pool member down"],
            ["on_sync", "Config sync results"],
            ["on_peer", "HA peer up/down"],
          ].map(([k, lbl]) => (
            <div class="field check" key={k}>
              <input type="checkbox" checked={nt[k] !== false} onChange={(e) => set("notify", { [k]: e.target.checked })} />
              <label style="margin:0">{lbl}</label>
            </div>
          ))}
          <div class="actions">
            <button class="btn primary" onClick={() => save().then(() => onToast("notifications applied")).catch(() => {})}>Save &amp; apply</button>
          </div>
          <div class="hint">Events always show in the bell; the webhook is an extra fan-out. Suggestions never auto-apply.</div>
        </div>
      </div>
    </section>
  );
}
