package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

type doctorCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

type buildModuleEvidence struct {
	Path    string `json:"path"`
	Version string `json:"version,omitempty"`
	Sum     string `json:"sum,omitempty"`
}

func buildModuleList() []buildModuleEvidence {
	bi, ok := debug.ReadBuildInfo()
	if !ok || bi == nil {
		return nil
	}
	out := make([]buildModuleEvidence, 0, len(bi.Deps)+1)
	if bi.Main.Path != "" {
		out = append(out, buildModuleEvidence{Path: bi.Main.Path, Version: bi.Main.Version, Sum: bi.Main.Sum})
	}
	for _, d := range bi.Deps {
		if d == nil {
			continue
		}
		m := d
		if d.Replace != nil {
			m = d.Replace
		}
		out = append(out, buildModuleEvidence{Path: d.Path, Version: m.Version, Sum: m.Sum})
	}
	return out
}

func moduleVersion(path string, mods []buildModuleEvidence) string {
	for _, m := range mods {
		if m.Path == path {
			return m.Version
		}
	}
	return ""
}

func (a *adminServer) handleDoctor(w http.ResponseWriter, _ *http.Request) {
	rt := a.srv.rt.Load()
	if rt == nil {
		http.Error(w, "runtime unavailable", http.StatusServiceUnavailable)
		return
	}
	mods := buildModuleList()
	checks := []doctorCheck{
		{Name: "runtime.go", Status: "PASS", Detail: runtime.Version()},
		{Name: "runtime.waf", Status: "PASS", Detail: buildVersion + " commit=" + buildCommit},
	}
	corazaVersion := moduleVersion("github.com/corazawaf/coraza/v3", mods)
	if corazaVersion == "" {
		checks = append(checks, doctorCheck{Name: "coraza.module", Status: "WARN", Detail: "version unavailable from build metadata"})
	} else {
		checks = append(checks, doctorCheck{Name: "coraza.module", Status: "PASS", Detail: corazaVersion})
	}
	if _, err := os.Stat(a.srv.configPath); err != nil {
		checks = append(checks, doctorCheck{Name: "config", Status: "FAIL", Detail: err.Error()})
	} else {
		checks = append(checks, doctorCheck{Name: "config", Status: "PASS", Detail: a.srv.configPath})
	}
	rules := rt.cfg.Rules
	if rules == "" && len(rt.cfg.Policies) > 0 {
		rules = rt.cfg.Policies[0].RulesPath
	}
	if rules == "" {
		checks = append(checks, doctorCheck{Name: "rules", Status: "WARN", Detail: "no rules path configured"})
	} else if _, err := os.Stat(rules); err != nil {
		checks = append(checks, doctorCheck{Name: "rules", Status: "FAIL", Detail: err.Error()})
	} else {
		checks = append(checks, doctorCheck{Name: "rules", Status: "PASS", Detail: rules})
	}
	vector := map[string]any{"mode": "off", "native_available": false, "groups": []any{}}
	if rt.vector != nil {
		vs := rt.vector.Status()
		vector = map[string]any{
			"mode": vs.Mode, "native_available": vs.NativeAvailable,
			"native_version": vs.NativeVersion, "groups": vs.Groups,
		}
		switch {
		case vs.NativeAvailable:
			checks = append(checks, doctorCheck{Name: "vectorscan.native", Status: "PASS", Detail: vs.NativeVersion})
		case vs.Mode == "off":
			checks = append(checks, doctorCheck{Name: "vectorscan.native", Status: "NOT_RUN", Detail: "acceleration disabled"})
		default:
			checks = append(checks, doctorCheck{Name: "vectorscan.native", Status: "WARN", Detail: "native libhs unavailable; Coraza-only fallback"})
		}
	}
	writeJSON(w, map[string]any{
		"generated_at":        time.Now().UTC().Format(time.RFC3339Nano),
		"checks":              checks,
		"vector_acceleration": vector,
		"tls_frontend":        a.srv.tlsFrontend.status(),
		"modules":             mods,
		"truth_boundary": map[string]string{
			"real_coraza_gate":        "requires release-host qualification transcript",
			"real_vectorscan_gate":    "requires vectorscan-tag native qualification transcript",
			"production_crs_learning": "requires Phase 1 corpus qualification with zero false negatives",
		},
	})
}

func (a *adminServer) debugTenantExists(tenant string) bool {
	rt := a.srv.rt.Load()
	if rt == nil {
		return false
	}
	for _, site := range rt.cfg.Sites {
		if site.Name == tenant {
			return true
		}
	}
	return false
}

func (a *adminServer) handleDebugStatus(w http.ResponseWriter, _ *http.Request) {
	if a.srv.debug == nil {
		writeJSON(w, map[string]any{"available": false})
		return
	}
	status := a.srv.debug.CaptureStatus(time.Now().UTC())
	status["available"] = true
	writeJSON(w, status)
}

func (a *adminServer) handleDebugCapture(w http.ResponseWriter, r *http.Request) {
	if a.srv.debug == nil {
		http.Error(w, "debug evidence store unavailable", http.StatusServiceUnavailable)
		return
	}
	var in struct {
		Tenant      string `json:"tenant"`
		Enabled     bool   `json:"enabled"`
		DurationSec int    `json:"duration_sec"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192)).Decode(&in); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	in.Tenant = strings.TrimSpace(in.Tenant)
	if !a.debugTenantExists(in.Tenant) {
		http.Error(w, "unknown tenant/site", http.StatusNotFound)
		return
	}
	if !in.Enabled {
		a.srv.debug.DisableTenant(in.Tenant)
		a.audit.add(who(r).user, "debug.capture_stop", in.Tenant)
		writeJSON(w, map[string]any{"ok": true, "tenant": in.Tenant, "enabled": false})
		return
	}
	if in.DurationSec == 0 {
		in.DurationSec = 300
	}
	if in.DurationSec < 1 || in.DurationSec > 3600 {
		http.Error(w, "duration_sec must be 1..3600", http.StatusBadRequest)
		return
	}
	if err := a.srv.debug.EnableTenant(in.Tenant, time.Duration(in.DurationSec)*time.Second); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.audit.add(who(r).user, "debug.capture_start", fmt.Sprintf("%s duration=%ds", in.Tenant, in.DurationSec))
	writeJSON(w, map[string]any{"ok": true, "tenant": in.Tenant, "enabled": true, "duration_sec": in.DurationSec})
}

func (a *adminServer) handleDebugEvidence(w http.ResponseWriter, r *http.Request) {
	if a.srv.debug == nil {
		http.Error(w, "debug evidence store unavailable", http.StatusServiceUnavailable)
		return
	}
	tenant := strings.TrimSpace(r.URL.Query().Get("tenant"))
	if !a.debugTenantExists(tenant) {
		http.Error(w, "unknown tenant/site", http.StatusNotFound)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	writeJSON(w, a.srv.debug.List(tenant, limit))
}

func (a *adminServer) handleDebugExport(w http.ResponseWriter, r *http.Request) {
	if a.srv.debug == nil {
		http.Error(w, "debug evidence store unavailable", http.StatusServiceUnavailable)
		return
	}
	tenant := strings.TrimSpace(r.URL.Query().Get("tenant"))
	txid := strings.TrimSpace(r.URL.Query().Get("transaction_id"))
	if !a.debugTenantExists(tenant) {
		http.Error(w, "unknown tenant/site", http.StatusNotFound)
		return
	}
	if txid == "" || strings.ContainsAny(txid, "\r\n/\\") {
		http.Error(w, "valid transaction_id is required", http.StatusBadRequest)
		return
	}
	if _, ok := a.srv.debug.Get(tenant, txid); !ok {
		http.Error(w, "evidence not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", "waf-debug-"+txid+".zip"))
	w.Header().Set("Cache-Control", "no-store")
	a.audit.add(who(r).user, "debug.export", fmt.Sprintf("%s transaction=%s", tenant, txid))
	if err := a.srv.debug.ExportIncident(tenant, txid, w); err != nil {
		a.log.Error("debug export failed", "err", err, "tenant", tenant, "transaction_id", txid)
	}
}
