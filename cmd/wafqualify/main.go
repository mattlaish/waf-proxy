package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"waf-proxy/internal/vectoraccel"

	"github.com/corazawaf/coraza/v3"
)

type sample struct {
	Name        string            `json:"name"`
	Method      string            `json:"method"`
	URI         string            `json:"uri"`
	Headers     map[string]string `json:"headers,omitempty"`
	Body        string            `json:"body,omitempty"`
	ContentType string            `json:"content_type,omitempty"`
}

type sampleResult struct {
	Name             string `json:"name"`
	VectorCandidates []int  `json:"vectorscan_candidates"`
	CorazaMatches    []int  `json:"coraza_eligible_matches"`
	FalseNegatives   []int  `json:"false_negative_rules"`
	Passed           bool   `json:"passed"`
}

type report struct {
	Format              string         `json:"format"`
	GeneratedAt         string         `json:"generated_at"`
	RulesPath           string         `json:"rules_path"`
	RulesSHA256         string         `json:"rules_sha256"`
	CorpusPath          string         `json:"corpus_path"`
	CorpusSHA256        string         `json:"corpus_sha256"`
	NativeVersion       string         `json:"native_version"`
	EligibleRuleCount   int            `json:"eligible_rule_count"`
	Samples             int            `json:"samples"`
	EligibleMatchEvents int            `json:"eligible_match_events"`
	FalseNegativeCount  int            `json:"false_negative_count"`
	ZeroFalseNegatives  bool           `json:"zero_false_negatives"`
	Result              string         `json:"result"`
	Details             []sampleResult `json:"details"`
}

func main() {
	var rules, corpus, output, site string
	var minEligible int
	flag.StringVar(&rules, "rules", "", "production/representative Coraza/CRS rules file")
	flag.StringVar(&corpus, "corpus", "qualification/corpus/default.jsonl", "JSONL replay corpus")
	flag.StringVar(&output, "output", "phase1-qualification.json", "JSON qualification report")
	flag.StringVar(&site, "site", "qualification", "site label used for VectorScan plan")
	flag.IntVar(&minEligible, "min-eligible-matches", 1, "minimum eligible Coraza match events required for a qualifying PASS")
	flag.Parse()
	if strings.TrimSpace(rules) == "" {
		fmt.Fprintln(os.Stderr, "BLOCKED: --rules is required")
		os.Exit(3)
	}
	if minEligible < 0 {
		fmt.Fprintln(os.Stderr, "ERROR: --min-eligible-matches must be >=0")
		os.Exit(2)
	}
	r, code, err := run(rules, corpus, site, minEligible)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(code)
	}
	b, _ := json.MarshalIndent(r, "", "  ")
	b = append(b, '\n')
	if err := os.MkdirAll(filepath.Dir(output), 0750); err != nil && filepath.Dir(output) != "." {
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(output, b, 0640); err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		os.Exit(1)
	}
	fmt.Printf("RESULT %s samples=%d eligible_matches=%d false_negatives=%d native=%s output=%s\n", r.Result, r.Samples, r.EligibleMatchEvents, r.FalseNegativeCount, r.NativeVersion, output)
	if r.Result == "PASS" {
		return
	}
	if r.Result == "BLOCKED" {
		os.Exit(3)
	}
	os.Exit(1)
}

func run(rules, corpus, site string, minEligible int) (report, int, error) {
	rep := report{Format: "waf-phase1-vectorscan-differential-v1", GeneratedAt: time.Now().UTC().Format(time.RFC3339Nano), RulesPath: rules, CorpusPath: corpus}
	rulesHash, err := fileSHA256(rules)
	if err != nil {
		return rep, 3, fmt.Errorf("BLOCKED: rules unavailable: %w", err)
	}
	rep.RulesSHA256 = rulesHash
	corpusHash, err := fileSHA256(corpus)
	if err != nil {
		return rep, 3, fmt.Errorf("BLOCKED: corpus unavailable: %w", err)
	}
	rep.CorpusSHA256 = corpusHash
	samples, err := loadCorpus(corpus)
	if err != nil {
		return rep, 1, fmt.Errorf("FAIL: invalid corpus: %w", err)
	}
	if len(samples) == 0 {
		return rep, 3, errors.New("BLOCKED: corpus is empty")
	}
	stateDir, err := os.MkdirTemp("", "waf-vector-qualify-")
	if err != nil {
		return rep, 1, err
	}
	defer os.RemoveAll(stateDir)
	cfg := vectoraccel.Defaults()
	cfg.Mode = vectoraccel.ModeRequired
	cfg.StatePath = filepath.Join(stateDir, "state.json")
	cfg.MinSamples = 1_000_000_000
	cfg.MinCorazaMatches = 0
	cfg.MinLearningSec = 365 * 24 * 3600
	mgr, err := vectoraccel.NewManager(cfg)
	if err != nil {
		if errors.Is(err, vectoraccel.ErrNativeUnavailable) {
			return rep, 3, fmt.Errorf("BLOCKED: real VectorScan/libhs unavailable: %w", err)
		}
		return rep, 1, fmt.Errorf("FAIL: VectorScan manager: %w", err)
	}
	defer mgr.Close()
	if !mgr.NativeAvailable() {
		return rep, 3, errors.New("BLOCKED: native VectorScan scanner unavailable")
	}
	rep.NativeVersion = mgr.NativeVersion()
	plan, err := mgr.BuildSite(site, rules)
	if err != nil {
		if errors.Is(err, vectoraccel.ErrNativeUnavailable) {
			return rep, 3, fmt.Errorf("BLOCKED: real VectorScan/libhs unavailable: %w", err)
		}
		return rep, 1, fmt.Errorf("FAIL: compile eligible VectorScan rules: %w", err)
	}
	eligible := plan.EligibleRuleIDs()
	rep.EligibleRuleCount = len(eligible)
	if len(eligible) == 0 {
		rep.Result = "BLOCKED"
		return rep, 3, nil
	}

	wc := coraza.NewWAFConfig().WithDirectivesFromFile(rules).WithDirectives("\nSecRuleEngine DetectionOnly\n")
	waf, err := coraza.NewWAF(wc)
	if err != nil {
		return rep, 1, fmt.Errorf("FAIL: build real Coraza WAF: %w", err)
	}
	for _, sc := range samples {
		req, err := sampleRequest(sc)
		if err != nil {
			return rep, 1, fmt.Errorf("FAIL sample %q: %w", sc.Name, err)
		}
		candidates, err := plan.QualificationCandidates(req)
		if err != nil {
			return rep, 1, fmt.Errorf("FAIL sample %q native scan: %w", sc.Name, err)
		}
		actual, err := corazaMatches(waf, sc)
		if err != nil {
			return rep, 1, fmt.Errorf("FAIL sample %q Coraza: %w", sc.Name, err)
		}
		actual = plan.EligibleMatchedRuleIDs(actual)
		fn := missing(candidates, actual)
		rep.Details = append(rep.Details, sampleResult{Name: sc.Name, VectorCandidates: candidates, CorazaMatches: actual, FalseNegatives: fn, Passed: len(fn) == 0})
		rep.Samples++
		rep.EligibleMatchEvents += len(actual)
		rep.FalseNegativeCount += len(fn)
	}
	rep.ZeroFalseNegatives = rep.FalseNegativeCount == 0
	switch {
	case rep.FalseNegativeCount > 0:
		rep.Result = "FAIL"
		return rep, 1, nil
	case rep.EligibleMatchEvents < minEligible:
		rep.Result = "BLOCKED"
		return rep, 3, nil
	default:
		rep.Result = "PASS"
		return rep, 0, nil
	}
}

func loadCorpus(path string) ([]sample, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []sample
	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 64<<10), 2<<20)
	line := 0
	for s.Scan() {
		line++
		b := bytes.TrimSpace(s.Bytes())
		if len(b) == 0 {
			continue
		}
		var v sample
		if err := json.Unmarshal(b, &v); err != nil {
			return nil, fmt.Errorf("line %d: %w", line, err)
		}
		v.Method = strings.ToUpper(strings.TrimSpace(v.Method))
		if v.Method == "" {
			v.Method = http.MethodGet
		}
		if v.Name == "" || v.URI == "" || !strings.HasPrefix(v.URI, "/") {
			return nil, fmt.Errorf("line %d: name and absolute-path URI are required", line)
		}
		out = append(out, v)
	}
	return out, s.Err()
}

func sampleRequest(sc sample) (*http.Request, error) {
	u, err := url.Parse("http://qualification.local" + sc.URI)
	if err != nil {
		return nil, err
	}
	r, err := http.NewRequest(sc.Method, u.String(), strings.NewReader(sc.Body))
	if err != nil {
		return nil, err
	}
	r.Host = "qualification.local"
	for k, v := range sc.Headers {
		r.Header.Set(k, v)
	}
	if sc.ContentType != "" {
		r.Header.Set("Content-Type", sc.ContentType)
	}
	return r, nil
}

func corazaMatches(waf coraza.WAF, sc sample) ([]int, error) {
	tx := waf.NewTransaction()
	defer tx.Close()
	tx.ProcessConnection("192.0.2.10", 54321, "192.0.2.20", 443)
	tx.ProcessURI(sc.URI, sc.Method, "HTTP/1.1")
	tx.AddRequestHeader("Host", "qualification.local")
	tx.SetServerName("qualification.local")
	for k, v := range sc.Headers {
		tx.AddRequestHeader(k, v)
	}
	if sc.ContentType != "" {
		tx.AddRequestHeader("Content-Type", sc.ContentType)
	}
	if sc.Body != "" {
		tx.AddRequestHeader("Content-Length", strconv.Itoa(len(sc.Body)))
	}
	if it := tx.ProcessRequestHeaders(); it != nil {
		return nil, fmt.Errorf("DetectionOnly interrupted headers: %+v", it)
	}
	if sc.Body != "" && tx.IsRequestBodyAccessible() {
		if it, _, err := tx.WriteRequestBody([]byte(sc.Body)); err != nil {
			return nil, err
		} else if it != nil {
			return nil, fmt.Errorf("DetectionOnly interrupted body write: %+v", it)
		}
	}
	if it, err := tx.ProcessRequestBody(); err != nil {
		return nil, err
	} else if it != nil {
		return nil, fmt.Errorf("DetectionOnly interrupted request body: %+v", it)
	}
	tx.ProcessLogging()
	seen := map[int]struct{}{}
	for _, m := range tx.MatchedRules() {
		seen[m.Rule().ID()] = struct{}{}
	}
	ids := make([]int, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids, nil
}

func missing(candidates, actual []int) []int {
	have := map[int]struct{}{}
	for _, id := range candidates {
		have[id] = struct{}{}
	}
	var out []int
	for _, id := range actual {
		if _, ok := have[id]; !ok {
			out = append(out, id)
		}
	}
	sort.Ints(out)
	return out
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
