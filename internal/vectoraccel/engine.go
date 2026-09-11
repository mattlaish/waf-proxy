package vectoraccel

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const InternalSkipHeader = "X-WAF-Vector-Skip"

type State string

const (
	StateCorazaOnly  State = "CORAZA_ONLY"
	StateLearning    State = "LEARNING"
	StateValidated   State = "VALIDATED"
	StateAccelerated State = "ACCELERATED"
	StateFailsafe    State = "FAILSAFE"
)

type persistedGroup struct {
	Fingerprint    string `json:"fingerprint"`
	State          State  `json:"state"`
	Samples        int64  `json:"samples"`
	CorazaMatches  int64  `json:"coraza_matches"`
	FalseNegatives int64  `json:"false_negatives"`
	LearningSince  int64  `json:"learning_since"`
}
type persistedState struct {
	Version int                       `json:"version"`
	Groups  map[string]persistedGroup `json:"groups"`
}

type groupRuntime struct {
	spec           GroupSpec
	fingerprint    string
	scanner        compiledScanner
	token          string
	state          atomic.Value
	samples        atomic.Int64
	corazaMatches  atomic.Int64
	falseNegatives atomic.Int64
	skipped        atomic.Int64
	scanErrors     atomic.Int64
	started        atomic.Int64
	verifyCounter  atomic.Uint64
	mu             sync.Mutex
}

func (g *groupRuntime) getState() State {
	if v := g.state.Load(); v != nil {
		return v.(State)
	}
	return StateCorazaOnly
}
func (g *groupRuntime) setState(s State) { g.state.Store(s) }

type Manager struct {
	cfg       Config
	factory   scannerFactory
	mu        sync.RWMutex
	groups    map[string]*groupRuntime
	persisted map[string]persistedGroup
	dirty     atomic.Bool
	stop      chan struct{}
	done      chan struct{}
	native    bool
	version   string
}

type SitePlan struct {
	mgr     *Manager
	site    string
	groups  []*groupRuntime
	byRule  map[int]*groupRuntime
	control string
}

type groupPrediction struct {
	group      *groupRuntime
	candidates map[int]struct{}
	scanned    bool
	skip       bool
}
type Observation struct {
	plan        *SitePlan
	predictions []*groupPrediction
}
type obsKey struct{}

func NewManager(cfg Config) (*Manager, error) {
	cfg = Effective(cfg)
	if err := Validate(cfg); err != nil {
		return nil, err
	}
	f := newNativeFactory()
	m := &Manager{cfg: cfg, factory: f, groups: map[string]*groupRuntime{}, persisted: map[string]persistedGroup{}, stop: make(chan struct{}), done: make(chan struct{}), native: f.Available(), version: f.Version()}
	_ = m.load()
	go m.flusher()
	return m, nil
}
func (m *Manager) NativeAvailable() bool { return m.native }
func (m *Manager) NativeVersion() string { return m.version }

const corazaLearningSemanticVersion = "coraza-v3.7.0-matchedrules-v1"

func groupRuntimeFingerprint(ruleFingerprint, nativeVersion string) string {
	h := sha256.Sum256([]byte(ruleFingerprint + "|" + nativeVersion + "|" + corazaLearningSemanticVersion))
	return hex.EncodeToString(h[:])
}

func (m *Manager) BuildSite(site, rulesPath string) (*SitePlan, error) {
	if m.cfg.Mode == ModeOff {
		return &SitePlan{mgr: m, site: site, byRule: map[int]*groupRuntime{}}, nil
	}
	if !m.factory.Available() && m.cfg.Mode == ModeRequired {
		return nil, ErrNativeUnavailable
	}
	if !m.factory.Available() {
		return &SitePlan{mgr: m, site: site, byRule: map[int]*groupRuntime{}}, nil
	}
	specs, used, err := ParseRules(rulesPath)
	if err != nil {
		return nil, err
	}
	p := &SitePlan{mgr: m, site: site, byRule: map[int]*groupRuntime{}}
	nextID := 1000
	allocID := func() int {
		for {
			if _, ok := used[nextID]; !ok {
				v := nextID
				used[v] = struct{}{}
				nextID++
				return v
			}
			nextID++
		}
	}
	var b strings.Builder
	for i, spec := range specs {
		runtimeFingerprint := groupRuntimeFingerprint(spec.Fingerprint, m.factory.Version())
		key := site + "|" + spec.Fingerprint
		gr := &groupRuntime{spec: spec, fingerprint: runtimeFingerprint, token: fmt.Sprintf("g%04d", i+1)}
		gr.started.Store(time.Now().Unix())
		gr.setState(StateLearning)
		if !m.factory.Available() {
			gr.setState(StateCorazaOnly)
		} else {
			sc, ce := m.factory.Compile(spec.Rules)
			if ce != nil {
				gr.setState(StateCorazaOnly)
			} else {
				gr.scanner = sc
			}
		}
		if pg, ok := m.persisted[key]; ok && pg.Fingerprint == runtimeFingerprint && gr.scanner != nil {
			gr.samples.Store(pg.Samples)
			gr.corazaMatches.Store(pg.CorazaMatches)
			gr.falseNegatives.Store(pg.FalseNegatives)
			if pg.LearningSince > 0 {
				gr.started.Store(pg.LearningSince)
			}
			if pg.State == StateAccelerated || pg.State == StateValidated || pg.State == StateLearning || pg.State == StateFailsafe {
				gr.setState(pg.State)
			}
		}
		m.mu.Lock()
		m.groups[key] = gr
		m.mu.Unlock()
		p.groups = append(p.groups, gr)
		for _, r := range spec.Rules {
			p.byRule[r.ID] = gr
		}
		if gr.scanner != nil {
			id := allocID()
			fmt.Fprintf(&b, "SecRule REQUEST_HEADERS:%s \"@contains |%s|\" \"id:%d,phase:1,pass,nolog,t:none", InternalSkipHeader, gr.token, id)
			for _, r := range spec.Rules {
				fmt.Fprintf(&b, ",ctl:ruleRemoveById=%d", r.ID)
			}
			b.WriteString("\"\n")
		}
	}
	p.control = b.String()
	return p, nil
}

func (p *SitePlan) ControlDirectives() string { return p.control }

func (p *SitePlan) Wrap(next http.Handler) http.Handler {
	if p == nil || len(p.groups) == 0 {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Header.Del(InternalSkipHeader)
		obs := p.analyze(r)
		if obs != nil {
			r = r.WithContext(context.WithValue(r.Context(), obsKey{}, obs))
			var toks []string
			for _, gp := range obs.predictions {
				if gp.skip {
					toks = append(toks, gp.group.token)
				}
			}
			if len(toks) > 0 {
				r.Header.Set(InternalSkipHeader, "|"+strings.Join(toks, "||")+"|")
			}
			defer r.Header.Del(InternalSkipHeader)
		}
		next.ServeHTTP(w, r)
	})
}
func SanitizeIngress(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { r.Header.Del(InternalSkipHeader); next.ServeHTTP(w, r) })
}
func StripEgress(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { r.Header.Del(InternalSkipHeader); next.ServeHTTP(w, r) })
}
func ObservationFromContext(ctx context.Context) *Observation {
	v, _ := ctx.Value(obsKey{}).(*Observation)
	return v
}

func (p *SitePlan) analyze(r *http.Request) *Observation {
	o := &Observation{plan: p}
	for _, g := range p.groups {
		if g.scanner == nil || g.getState() == StateCorazaOnly || g.getState() == StateFailsafe {
			continue
		}
		vals := sourceValues(g.spec, r)
		cand := map[int]struct{}{}
		ok := true
		for _, v := range vals {
			v = applyExactTransforms(v, g.spec.Transforms)
			ids, err := g.scanner.Scan([]byte(v))
			if err != nil {
				ok = false
				g.scanErrors.Add(1)
				g.setState(StateFailsafe)
				p.mgr.dirty.Store(true)
				break
			}
			for _, id := range ids {
				cand[id] = struct{}{}
			}
		}
		gp := &groupPrediction{group: g, candidates: cand, scanned: ok}
		if ok && len(cand) == 0 && g.getState() == StateAccelerated {
			rate := p.mgr.cfg.VerificationSampleRate
			every := uint64(1)
			if rate < 1 {
				every = uint64(1 / rate)
				if every < 1 {
					every = 1
				}
			}
			verify := g.verifyCounter.Add(1)%every == 0
			if !verify {
				gp.skip = true
				g.skipped.Add(1)
			}
		}
		o.predictions = append(o.predictions, gp)
	}
	if len(o.predictions) == 0 {
		return nil
	}
	return o
}

func sourceValues(g GroupSpec, r *http.Request) []string {
	switch g.Source {
	case SourceRequestURI:
		return []string{r.URL.String()}
	case SourceRequestFilename:
		return []string{r.URL.Path}
	case SourceRequestMethod:
		return []string{r.Method}
	case SourceRequestProtocol:
		return []string{r.Proto}
	case SourceRequestURIRaw:
		// Coraza's HTTP connector passes req.URL.String() to ProcessURI, whose
		// first action stores that exact argument in REQUEST_URI_RAW.
		return []string{r.URL.String()}
	case SourceRequestLine:
		// ProcessURI formats REQUEST_LINE as method + URI argument + protocol.
		return []string{r.Method + " " + r.URL.String() + " " + r.Proto}
	case SourceRequestBasename:
		path := r.URL.Path
		offset := strings.LastIndexAny(path, "/\\")
		if offset != -1 && len(path) > offset+1 {
			return []string{path[offset+1:]}
		}
		return []string{path}
	case SourceQueryString:
		return []string{r.URL.RawQuery}
	case SourceServerName:
		return []string{r.Host}
	case SourceRemoteAddr:
		client, _ := corazaConnectorRemote(r.RemoteAddr)
		return []string{client}
	case SourceRemotePort:
		_, port := corazaConnectorRemote(r.RemoteAddr)
		return []string{strconv.Itoa(port)}
	case SourceRequestHeader:
		switch {
		case strings.EqualFold(g.Header, "Host"):
			return []string{r.Host}
		case strings.EqualFold(g.Header, "Transfer-Encoding"):
			return append([]string(nil), r.TransferEncoding...)
		default:
			return append([]string(nil), r.Header.Values(g.Header)...)
		}
	}
	return nil
}

func corazaConnectorRemote(remoteAddr string) (string, int) {
	// Mirror github.com/corazawaf/coraza/v3/http processRequest exactly: it
	// splits on the final colon, passes the prefix to ProcessConnection as the
	// client address, and treats an invalid/missing port as zero. The outer
	// clientIPResolver has already normalized trusted-proxy identity before
	// both this prefilter and Coraza see the request.
	idx := strings.LastIndexByte(remoteAddr, ':')
	if idx == -1 {
		return "", 0
	}
	port, _ := strconv.Atoi(remoteAddr[idx+1:])
	return remoteAddr[:idx], port
}

func (p *SitePlan) Observe(obs *Observation, matchedRuleIDs []int) {
	if obs == nil || obs.plan != p {
		return
	}
	actualByGroup := map[*groupRuntime]map[int]struct{}{}
	for _, id := range matchedRuleIDs {
		if g := p.byRule[id]; g != nil {
			m := actualByGroup[g]
			if m == nil {
				m = map[int]struct{}{}
				actualByGroup[g] = m
			}
			m[id] = struct{}{}
		}
	}
	now := time.Now()
	for _, gp := range obs.predictions {
		if !gp.scanned {
			continue
		}
		g := gp.group
		g.samples.Add(1)
		actual := actualByGroup[g]
		if len(actual) > 0 {
			g.corazaMatches.Add(1)
		}
		fn := false
		for id := range actual {
			if _, ok := gp.candidates[id]; !ok {
				fn = true
				break
			}
		}
		if fn {
			g.falseNegatives.Add(1)
			g.setState(StateFailsafe)
			p.mgr.dirty.Store(true)
			continue
		}
		st := g.getState()
		if st == StateLearning && g.samples.Load() >= p.mgr.cfg.MinSamples && g.corazaMatches.Load() >= p.mgr.cfg.MinCorazaMatches && now.Sub(time.Unix(g.started.Load(), 0)) >= time.Duration(p.mgr.cfg.MinLearningSec)*time.Second {
			g.setState(StateValidated)
			p.mgr.dirty.Store(true)
			st = StateValidated
		}
		if st == StateValidated {
			g.setState(StateAccelerated)
			p.mgr.dirty.Store(true)
		}
		if g.samples.Load()%1000 == 0 {
			p.mgr.dirty.Store(true)
		}
	}
}

func (m *Manager) ResetFailsafe(site string) int {
	n := 0
	m.mu.RLock()
	defer m.mu.RUnlock()
	for key, g := range m.groups {
		if strings.HasPrefix(key, site+"|") && g.getState() == StateFailsafe {
			g.falseNegatives.Store(0)
			g.samples.Store(0)
			g.corazaMatches.Store(0)
			g.started.Store(time.Now().Unix())
			g.setState(StateLearning)
			n++
			m.dirty.Store(true)
		}
	}
	return n
}

type GroupStatus struct {
	Site           string `json:"site"`
	Key            string `json:"key"`
	State          State  `json:"state"`
	Rules          int    `json:"rules"`
	Samples        int64  `json:"samples"`
	CorazaMatches  int64  `json:"coraza_matches"`
	FalseNegatives int64  `json:"false_negatives"`
	Skipped        int64  `json:"skipped"`
	ScanErrors     int64  `json:"scan_errors"`
	Fingerprint    string `json:"fingerprint"`
}
type Status struct {
	Mode            string        `json:"mode"`
	NativeAvailable bool          `json:"native_available"`
	NativeVersion   string        `json:"native_version"`
	Groups          []GroupStatus `json:"groups"`
}

func (m *Manager) Status() Status {
	s := Status{Mode: m.cfg.Mode, NativeAvailable: m.native, NativeVersion: m.version}
	m.mu.RLock()
	defer m.mu.RUnlock()
	for key, g := range m.groups {
		site := strings.SplitN(key, "|", 2)[0]
		s.Groups = append(s.Groups, GroupStatus{Site: site, Key: g.spec.Key, State: g.getState(), Rules: len(g.spec.Rules), Samples: g.samples.Load(), CorazaMatches: g.corazaMatches.Load(), FalseNegatives: g.falseNegatives.Load(), Skipped: g.skipped.Load(), ScanErrors: g.scanErrors.Load(), Fingerprint: g.fingerprint})
	}
	sort.Slice(s.Groups, func(i, j int) bool {
		if s.Groups[i].Site == s.Groups[j].Site {
			return s.Groups[i].Key < s.Groups[j].Key
		}
		return s.Groups[i].Site < s.Groups[j].Site
	})
	return s
}

func (m *Manager) load() error {
	b, err := os.ReadFile(m.cfg.StatePath)
	if err != nil {
		return nil
	}
	var st persistedState
	if json.Unmarshal(b, &st) != nil {
		return nil
	}
	if st.Version != 1 {
		return nil
	}
	m.persisted = st.Groups
	return nil
}
func (m *Manager) save() error {
	m.mu.RLock()
	st := persistedState{Version: 1, Groups: map[string]persistedGroup{}}
	for key, g := range m.groups {
		st.Groups[key] = persistedGroup{Fingerprint: g.fingerprint, State: g.getState(), Samples: g.samples.Load(), CorazaMatches: g.corazaMatches.Load(), FalseNegatives: g.falseNegatives.Load(), LearningSince: g.started.Load()}
	}
	m.mu.RUnlock()
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(m.cfg.StatePath), 0750); err != nil {
		return err
	}
	tmp := m.cfg.StatePath + ".tmp"
	if err = os.WriteFile(tmp, b, 0640); err != nil {
		return err
	}
	return os.Rename(tmp, m.cfg.StatePath)
}
func (m *Manager) flusher() {
	t := time.NewTicker(5 * time.Second)
	defer t.Stop()
	defer close(m.done)
	for {
		select {
		case <-t.C:
			if m.dirty.Swap(false) {
				_ = m.save()
			}
		case <-m.stop:
			if m.dirty.Swap(false) {
				_ = m.save()
			}
			return
		}
	}
}
func (m *Manager) Close() error {
	select {
	case <-m.stop:
	default:
		close(m.stop)
	}
	<-m.done
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, g := range m.groups {
		if g.scanner != nil {
			_ = g.scanner.Close()
		}
	}
	return nil
}
