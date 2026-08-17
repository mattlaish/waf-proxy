import { useState, useEffect, useCallback } from "preact/hooks";
import { getConfig, putConfig } from "./api.js";

// useConfig loads the config once and exposes a mutable draft plus a save().
// Tabs receive { cfg, set, save } and edit slices of the draft.
export function useConfig(onToast) {
  const [cfg, setCfg] = useState(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    getConfig().then(setCfg).catch((e) => onToast && onToast(e.message, "err"));
  }, []);

  // set("ai", partial) shallow-merges into cfg[key]; set(null, fullCfg) replaces.
  const set = useCallback((key, val) => {
    setCfg((c) => {
      if (key === null) return val;
      const next = { ...c };
      next[key] = Array.isArray(val) ? val : { ...(c[key] || {}), ...val };
      return next;
    });
  }, []);

  const save = useCallback(async () => {
    setSaving(true);
    try {
      const res = await putConfig(cfg);
      setCfg(res.config);
      onToast && onToast(res.restart_required ? "saved — restart required" : "saved and applied",
        res.restart_required ? "warn" : "");
      return res;
    } catch (e) {
      onToast && onToast(e.message, "err");
      throw e;
    } finally {
      setSaving(false);
    }
  }, [cfg]);

  return { cfg, setCfg, set, save, saving };
}

// usePoll runs fn every ms while `active` is true.
export function usePoll(fn, ms, active = true) {
  useEffect(() => {
    if (!active) return;
    fn();
    const id = setInterval(fn, ms);
    return () => clearInterval(id);
  }, [active, ms]);
}
