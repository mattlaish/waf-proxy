package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type owiNativeDetection struct {
	At       time.Time
	Site     string
	RuleID   int
	Severity string
	Path     string
}

type owiExportPlane struct {
	store     *owiStore
	log       *slog.Logger
	queue     chan owiNativeDetection
	stop      chan struct{}
	done      chan struct{}
	accepting atomic.Bool
	dropped   atomic.Uint64
	startOnce sync.Once
	stopOnce  sync.Once
}

func newOWIExportPlane(store *owiStore, log *slog.Logger, capacity int) *owiExportPlane {
	if capacity < 1 {
		capacity = 1
	}
	p := &owiExportPlane{store: store, log: log, queue: make(chan owiNativeDetection, capacity), stop: make(chan struct{}), done: make(chan struct{})}
	p.start()
	return p
}
func (p *owiExportPlane) start() { p.startOnce.Do(func() { p.accepting.Store(true); go p.run() }) }
func (p *owiExportPlane) enqueue(ev owiNativeDetection) bool {
	if p == nil || !p.accepting.Load() {
		return false
	}
	select {
	case p.queue <- ev:
		return true
	default:
		p.dropped.Add(1)
		return false
	}
}
func (p *owiExportPlane) run() {
	defer close(p.done)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	var reported uint64
	flushDrops := func() {
		d := p.dropped.Load()
		if d > reported {
			p.store.markDetectionDrop()
			if p.log != nil {
				p.log.Warn("dashboard detection export coverage gap", "dropped", d-reported, "dropped_total", d)
			}
			reported = d
		}
	}
	for {
		select {
		case ev := <-p.queue:
			rec := owiDetectionRecord(p.store.sourceInstance, ev)
			if err := p.store.appendDetection(rec, ev.At.UTC()); err != nil {
				p.store.markDetectionDrop()
				if p.log != nil {
					p.log.Error("dashboard detection export persistence failed", "err", err)
				}
			}
		case <-ticker.C:
			flushDrops()
		case <-p.stop:
			for {
				select {
				case ev := <-p.queue:
					if err := p.store.appendDetection(owiDetectionRecord(p.store.sourceInstance, ev), ev.At.UTC()); err != nil {
						p.store.markDetectionDrop()
						if p.log != nil {
							p.log.Error("dashboard detection export drain persistence failed", "err", err)
						}
					}
				default:
					flushDrops()
					return
				}
			}
		}
	}
}
func (p *owiExportPlane) stopAndDrain() {
	if p == nil {
		return
	}
	p.stopOnce.Do(func() { p.accepting.Store(false); close(p.stop) })
	<-p.done
}

type owiExportStats struct {
	QueueDepth    int
	QueueCapacity int
	Dropped       uint64
}

func (p *owiExportPlane) stats() owiExportStats {
	if p == nil {
		return owiExportStats{}
	}
	return owiExportStats{len(p.queue), cap(p.queue), p.dropped.Load()}
}

func (c *owiConnector) noteWAFMatch(rec matchRec, at time.Time) {
	if c == nil || c.export == nil {
		return
	}
	c.export.enqueue(owiNativeDetection{At: at.UTC(), Site: rec.Site, RuleID: rec.RuleID, Severity: rec.Severity, Path: safeOWIPath(rec.URI)})
}

var owiPathIDSegment = regexp.MustCompile(`(?i)^(?:[0-9]{2,}|[0-9a-f]{8}-[0-9a-f-]{8,}|[0-9a-f]{16,})$`)

func safeOWIPath(raw string) string {
	if raw == "" {
		return "/"
	}
	u, err := url.ParseRequestURI(raw)
	if err != nil {
		if i := strings.IndexByte(raw, '?'); i >= 0 {
			raw = raw[:i]
		}
		if !strings.HasPrefix(raw, "/") {
			return "/"
		}
		u = &url.URL{Path: raw}
	}
	parts := strings.Split(u.EscapedPath(), "/")
	for i, p := range parts {
		dec, _ := url.PathUnescape(p)
		if owiPathIDSegment.MatchString(dec) {
			parts[i] = "{id}"
		}
	}
	out := strings.Join(parts, "/")
	if out == "" {
		out = "/"
	}
	if len(out) > 512 {
		out = out[:512]
	}
	return out
}

func owiDetectionRecord(source string, ev owiNativeDetection) owiRecord {
	at := ev.At.UTC().Format(time.RFC3339)
	h := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%d\x00%d\x00%s", ev.Site, ev.RuleID, ev.At.UnixNano(), ev.Path)))
	id := "det-" + hex.EncodeToString(h[:12])
	siteID := safeOWIExternalID("site", ev.Site)
	conf := 0.85
	sev := owiSeverity(ev.Severity)
	var attention any = nil
	if sev == "CRITICAL" || sev == "HIGH" {
		attention = map[string]any{"code": "WAF.HIGH_SIGNAL_ATTACK", "severity": sev, "confidence": conf, "status": "OPEN", "reason_codes": []string{"WAF_RULE_MATCH"}, "recommended_action_codes": []string{}}
	}
	payload := map[string]any{
		"entity_refs": []owiEntityRef{{Ref: owiRef{owiSourceProduct, source, "ASSET", siteID}, Role: "SUBJECT", ObservedAt: at, ValidUntil: nil, NetworkNamespace: nil, Confidence: 1}},
		"summary":     fmt.Sprintf("WAF rule %d matched on site %s at path %s.", ev.RuleID, safeOWILabel(ev.Site), ev.Path),
		"attention":   attention, "detector_id": fmt.Sprintf("coraza:%d", ev.RuleID), "rule_version": nil, "category": "WAF_RULE_MATCH", "confidence": conf, "status": "OPEN", "lure_type": nil, "match_strength": "EXACT",
	}
	eid := id
	return owiRecord{ExternalID: id, SourceObservedAt: at, SourceUpdatedAt: nil, Provenance: owiProvenance{Origin: owiRef{owiSourceProduct, source, "DETECTION", id}, OriginEventID: &eid, ForwardedBy: []map[string]string{}, DerivedFrom: []owiRef{}, EvidenceRefs: []owiEvidenceRef{}, DetailPath: nil}, Payload: payload}
}

func owiSeverity(v string) string {
	switch strings.ToUpper(strings.TrimSpace(v)) {
	case "EMERGENCY", "ALERT", "CRITICAL":
		return "CRITICAL"
	case "ERROR", "HIGH":
		return "HIGH"
	case "WARNING", "WARN", "MEDIUM":
		return "MEDIUM"
	case "NOTICE", "LOW":
		return "LOW"
	case "INFO", "INFORMATIONAL":
		return "INFO"
	default:
		return "UNKNOWN"
	}
}

func safeOWILabel(v string) string {
	v = strings.TrimSpace(v)
	if len(v) > 128 {
		v = v[:128]
	}
	return v
}
func safeOWIExternalID(prefix, native string) string {
	base := strings.TrimSpace(native)
	if base != "" && len(prefix)+1+len(base) <= 128 && validOWIID(prefix+":"+base) {
		return prefix + ":" + base
	}
	h := sha256.Sum256([]byte(prefix + "\x00" + native))
	return prefix + ":" + hex.EncodeToString(h[:16])
}

func (c *owiConnector) syncNativeState(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	if c.srv == nil {
		return errorsNewOWI("runtime unavailable")
	}
	rt := c.srv.rt.Load()
	if rt == nil {
		return errorsNewOWI("runtime unavailable")
	}
	observed := time.Now().UTC()
	if c.srv.configPath != "" {
		if st, err := os.Stat(c.srv.configPath); err == nil {
			observed = st.ModTime().UTC()
		}
	}
	desired := map[string][]owiRecord{
		"ASSET":  c.assetRecords(rt.cfg, observed),
		"POLICY": c.policyRecords(rt.cfg, observed),
		"HEALTH": {c.healthRecord(rt.cfg, time.Now().UTC().Truncate(time.Minute))},
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	return c.store.syncStateBatchContext(ctx, desired, time.Now().UTC())
}

func errorsNewOWI(s string) error { return fmt.Errorf("dashboard connector: %s", s) }

func (c *owiConnector) baseRecord(kind, id string, observed time.Time, payload map[string]any) owiRecord {
	return owiRecord{ExternalID: id, SourceObservedAt: observed.UTC().Format(time.RFC3339), SourceUpdatedAt: nil, Provenance: owiProvenance{Origin: owiRef{owiSourceProduct, c.cfg.SourceInstanceID, kind, id}, ForwardedBy: []map[string]string{}, DerivedFrom: []owiRef{}, EvidenceRefs: []owiEvidenceRef{}, DetailPath: nil}, Payload: payload}
}

func owiIdentityValue(kind, value, namespace string, observed time.Time) owiIdentity {
	return owiIdentity{kind, value, namespace, observed.UTC().Format(time.RFC3339), nil, "VERIFIED"}
}

func (c *owiConnector) assetRecords(cfg Config, observed time.Time) []owiRecord {
	var out []owiRecord
	for _, n := range cfg.Nodes {
		id := safeOWIExternalID("node", n.Name)
		ik := "HOSTNAME"
		if net.ParseIP(n.Host) != nil {
			ik = "IP"
		}
		p := map[string]any{"entity_refs": []owiEntityRef{}, "summary": "WAF origin node " + safeOWILabel(n.Name) + ".", "attention": nil, "asset_type": "HOST", "display_name": safeOWILabel(n.Name), "identities": []owiIdentity{owiIdentityValue("PRODUCT_OBJECT_ID", n.Name, c.cfg.SourceInstanceID, observed), owiIdentityValue(ik, n.Host, c.cfg.SourceInstanceID, observed)}, "lifecycle": "ACTIVE", "criticality": "UNKNOWN", "last_seen_at": nil}
		out = append(out, c.baseRecord("ASSET", id, observed, p))
	}
	for _, pCfg := range cfg.Pools {
		id := safeOWIExternalID("pool", pCfg.Name)
		p := map[string]any{"entity_refs": []owiEntityRef{}, "summary": "WAF origin pool " + safeOWILabel(pCfg.Name) + ".", "attention": nil, "asset_type": "SERVICE", "display_name": safeOWILabel(pCfg.Name), "identities": []owiIdentity{owiIdentityValue("PRODUCT_OBJECT_ID", pCfg.Name, c.cfg.SourceInstanceID, observed)}, "lifecycle": "ACTIVE", "criticality": "UNKNOWN", "last_seen_at": nil}
		out = append(out, c.baseRecord("ASSET", id, observed, p))
	}
	for _, site := range cfg.Sites {
		siteID := safeOWIExternalID("site", site.Name)
		ids := []owiIdentity{owiIdentityValue("PRODUCT_OBJECT_ID", site.Name, c.cfg.SourceInstanceID, observed)}
		for _, h := range site.Hostnames {
			if h != "*" && h != "" {
				ids = append(ids, owiIdentityValue("FQDN", strings.TrimPrefix(h, "*."), c.cfg.SourceInstanceID, observed))
			}
		}
		p := map[string]any{"entity_refs": []owiEntityRef{}, "summary": "Protected WAF site " + safeOWILabel(site.Name) + ".", "attention": nil, "asset_type": "SITE", "display_name": safeOWILabel(site.Name), "identities": ids, "lifecycle": "ACTIVE", "criticality": "UNKNOWN", "last_seen_at": nil}
		out = append(out, c.baseRecord("ASSET", siteID, observed, p))
		for _, host := range site.Hostnames {
			if host == "" {
				continue
			}
			native := site.Name + "@" + host
			id := safeOWIExternalID("virtual-service", native)
			vids := []owiIdentity{owiIdentityValue("PRODUCT_OBJECT_ID", native, c.cfg.SourceInstanceID, observed)}
			if host != "*" {
				vids = append(vids, owiIdentityValue("FQDN", strings.TrimPrefix(host, "*."), c.cfg.SourceInstanceID, observed))
			}
			ref := owiEntityRef{Ref: owiRef{owiSourceProduct, c.cfg.SourceInstanceID, "ASSET", siteID}, Role: "SUBJECT", ObservedAt: observed.UTC().Format(time.RFC3339), ValidUntil: nil, NetworkNamespace: nil, Confidence: 1}
			vp := map[string]any{"entity_refs": []owiEntityRef{ref}, "summary": "WAF virtual service for site " + safeOWILabel(site.Name) + ".", "attention": nil, "asset_type": "APPLICATION", "display_name": safeOWILabel(native), "identities": vids, "lifecycle": "ACTIVE", "criticality": "UNKNOWN", "last_seen_at": nil}
			out = append(out, c.baseRecord("ASSET", id, observed, vp))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ExternalID < out[j].ExternalID })
	return out
}

func (c *owiConnector) policyRecords(cfg Config, observed time.Time) []owiRecord {
	var out []owiRecord
	for _, pCfg := range cfg.Policies {
		b, _ := json.Marshal(pCfg)
		sum := sha256.Sum256(b)
		digest := "sha256:" + hex.EncodeToString(sum[:])
		gen := hex.EncodeToString(sum[:8])
		var refs []owiRef
		enforcing := false
		nonEnforcing := false
		for _, site := range cfg.Sites {
			if site.Policy == pCfg.Name {
				refs = append(refs, owiRef{owiSourceProduct, c.cfg.SourceInstanceID, "ASSET", safeOWIExternalID("site", site.Name)})
				mode := site.EngineMode
				if mode == "" {
					mode = cfg.EngineMode
				}
				if strings.EqualFold(mode, "On") {
					enforcing = true
				} else {
					nonEnforcing = true
				}
			}
		}
		state := "UNKNOWN"
		if enforcing && !nonEnforcing {
			state = "ENFORCING"
		} else if !enforcing && nonEnforcing {
			state = "NOT_ENFORCING"
		} else if enforcing && nonEnforcing {
			state = "PARTIAL"
		}
		id := safeOWIExternalID("policy", pCfg.Name)
		payload := map[string]any{"entity_refs": []owiEntityRef{}, "summary": "WAF policy " + safeOWILabel(pCfg.Name) + ".", "attention": nil, "policy_type": "CORAZA_CRS_POLICY", "desired_generation": gen, "active_generation": gen, "enforcement_state": state, "scope_refs": refs, "content_digest": digest}
		out = append(out, c.baseRecord("POLICY", id, observed, payload))
	}
	return out
}

func ptrFloat(v float64) *float64 { return &v }
func ptrString(v string) *string  { return &v }

func (c *owiConnector) healthRecord(cfg Config, observed time.Time) owiRecord {
	state := "HEALTHY"
	reasons := []string{}
	var att any = nil
	if c.srv.draining.Load() {
		state = "DEGRADED"
		reasons = append(reasons, "DRAINING")
		conf := 1.0
		att = map[string]any{"code": "WAF.PROTECTION_DEGRADED", "severity": "HIGH", "confidence": conf, "status": "OPEN", "reason_codes": []string{"DRAINING"}, "recommended_action_codes": []string{}}
	}
	obs := observationSnapshot{}
	if c.srv.observations != nil {
		obs = c.srv.observations.snapshot()
	}
	mlog := matchLogSnapshot{}
	if c.srv.matchLogs != nil {
		mlog = c.srv.matchLogs.snapshot()
	}
	exp := c.export.stats()
	obsRatio := 0.0
	if obs.QueueCapacity > 0 {
		obsRatio = float64(obs.QueueDepth) / float64(obs.QueueCapacity)
	}
	logRatio := 0.0
	if mlog.QueueCapacity > 0 {
		logRatio = float64(mlog.QueueDepth) / float64(mlog.QueueCapacity)
	}
	expRatio := 0.0
	if exp.QueueCapacity > 0 {
		expRatio = float64(exp.QueueDepth) / float64(exp.QueueCapacity)
	}
	if (obsRatio >= .8 || logRatio >= .8 || expRatio >= .8) && att == nil {
		state = "DEGRADED"
		reasons = append(reasons, "CAPACITY_PRESSURE")
		conf := 0.9
		att = map[string]any{"code": "WAF.CAPACITY_PRESSURE", "severity": "MEDIUM", "confidence": conf, "status": "OPEN", "reason_codes": []string{"QUEUE_UTILIZATION_HIGH"}, "recommended_action_codes": []string{}}
	}
	metrics := []owiMetric{
		{"observation_queue_utilization", ptrFloat(obsRatio), "ratio", 60, observed.Format(time.RFC3339), ptrFloat(.8), ptrString("GTE"), "MEASURED"},
		{"match_log_queue_utilization", ptrFloat(logRatio), "ratio", 60, observed.Format(time.RFC3339), ptrFloat(.8), ptrString("GTE"), "MEASURED"},
		{"dashboard_export_queue_utilization", ptrFloat(expRatio), "ratio", 60, observed.Format(time.RFC3339), ptrFloat(.8), ptrString("GTE"), "MEASURED"},
	}
	if h := c.srv.metrics.history(); len(h) > 0 {
		last := h[len(h)-1]
		metrics = append(metrics, owiMetric{"requests_per_second", ptrFloat(last.ReqPerSec), "requests_per_second", 60, observed.Format(time.RFC3339), nil, nil, "MEASURED"}, owiMetric{"tls_handshake_p95_ms", ptrFloat(last.TLSHandshakeP95MS), "milliseconds", 60, observed.Format(time.RFC3339), nil, nil, "MEASURED"})
	}
	id := "health:waf-proxy"
	payload := map[string]any{"entity_refs": []owiEntityRef{}, "summary": "WAF proxy runtime health observation.", "attention": att, "component": "waf-proxy", "state": state, "observed_at": observed.Format(time.RFC3339), "reason_codes": reasons, "metrics": metrics}
	return c.baseRecord("HEALTH", id, observed, payload)
}
