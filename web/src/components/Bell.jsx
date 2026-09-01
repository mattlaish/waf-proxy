import { useState, useEffect, useRef } from "preact/hooks";
import { getNotifications, notifyRead, notifyDismiss, notifyApply } from "../api.js";

export function Bell({ unread, onToast }) {
  const [open, setOpen] = useState(false);
  const [items, setItems] = useState([]);
  const ref = useRef();

  async function load() {
    try {
      const r = await getNotifications();
      setItems(r.items || []);
      await notifyRead(true); // opening marks read
    } catch {}
  }

  useEffect(() => {
    function onDoc(e) {
      if (open && ref.current && !ref.current.contains(e.target)) setOpen(false);
    }
    document.addEventListener("click", onDoc);
    return () => document.removeEventListener("click", onDoc);
  }, [open]);

  return (
    <div class="bellwrap" ref={ref}>
      <button
        class="bell"
        onClick={(e) => {
          e.stopPropagation();
          const n = !open;
          setOpen(n);
          if (n) load();
        }}
      >
        🔔{unread > 0 && <span class="badge">{unread}</span>}
      </button>
      {open && (
        <div class="bellpanel">
          <div class="bellhead">
            <span>Notifications</span>
            <button class="btn sm" onClick={async () => { await notifyRead(true); load(); }}>Mark all read</button>
            <button class="btn sm" onClick={async () => { await notifyDismiss(0, true); load(); }}>Clear</button>
          </div>
          {items.length === 0 && <div class="empty">Nothing yet.</div>}
          {items.map((n) => (
            <div class={`bellitem ${n.level} ${n.read ? "" : "unread"}`} key={n.id}>
              <div class="bt"><b>{n.title}</b><span class="tm">{n.time}</span></div>
              <div class="bb">{n.body}</div>
              <div class="ba">
                {n.action === "apply_exclusion" && (
                  <button class="btn sm" onClick={async () => {
                    try { await notifyApply(n.id); onToast && onToast("applied"); load(); }
                    catch (e) { onToast && onToast(e.message, "err"); }
                  }}>Apply</button>
                )}
                <button class="btn sm danger" onClick={async () => { await notifyDismiss(n.id); load(); }}>Dismiss</button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
