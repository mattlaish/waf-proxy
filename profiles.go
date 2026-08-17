package main

// Content- and structure-driven page profiling.
//
// The pipeline: LEARN (content signals from the crawler + structure from the
// path tree) → SUGGEST a page profile per URL → REVIEW (LLM second opinion or
// human) → APPLY (bind the profile as a page policy, optionally auto-applied
// above a confidence threshold).
//
// A "profile" is a pre-built, named bundle of CRS-family tuning (which rule
// families to strengthen/relax and at what paranoia) that maps onto the
// existing per-URL page-policy compiler. It is NOT a separate engine per page —
// it compiles to path-gated SecLang inside the site's one engine.

import (
	"regexp"
	"sort"
	"strings"
	"sync"
)

// ── profile catalog ─────────────────────────────────────────────────────

// A Profile expresses intent as CRS-family tuning. Families we care about:
//
//	sqli  = 942xxx   xss = 941xxx   rce = 932xxx   lfi = 930xxx
//	rfi   = 931xxx   php = 933xxx   session = 943xxx   protocol = 920/921xxx
//	scanner = 913xxx
//
// A profile sets a paranoia level and a mode, and can relax families that are
// irrelevant to a page (to cut false positives) via CRS family rule-id ranges.
type Profile struct {
	Name          string   `json:"name"`
	Label         string   `json:"label"`
	Mode          string   `json:"mode,omitempty"` // "" inherit | On | DetectionOnly | Off
	ParanoiaLevel int      `json:"paranoia_level"` // 0 inherit
	RelaxFamilies []string `json:"relax_families,omitempty"`
	Description   string   `json:"description"`
	BuiltIn       bool     `json:"built_in"`
}

// crsFamilyRanges maps a family key to the CRS rule-id ranges to remove when a
// profile relaxes that family. Removing whole ranges is coarse but predictable.
var crsFamilyRanges = map[string][]int{
	// family: representative CRS rule ids (PL1 anchors); relaxing removes these
	"sqli":     {942100, 942110, 942150, 942190, 942200, 942260, 942300, 942370, 942410, 942430},
	"xss":      {941100, 941110, 941130, 941160, 941170, 941180, 941320, 941350},
	"rce":      {932100, 932105, 932115, 932130, 932160, 932170},
	"lfi":      {930100, 930110, 930120},
	"rfi":      {931100, 931110, 931120, 931130},
	"php":      {933100, 933110, 933120, 933150, 933160},
	"session":  {943100, 943110, 943120},
	"protocol": {920100, 920160, 920170, 920270, 920280, 921110, 921150},
	"scanner":  {913100, 913110, 913120},
}

func builtinProfiles() []Profile {
	return []Profile{
		{Name: "form-sqli-xss", Label: "Form (SQLi+XSS strict)", Mode: "On", ParanoiaLevel: 3,
			Description: "Pages that accept form input. Strong SQLi/XSS coverage at PL3.", BuiltIn: true},
		{Name: "query-hardened", Label: "Query-hardened", Mode: "On", ParanoiaLevel: 2,
			Description: "Pages driven by query parameters. Injection coverage at PL2.", BuiltIn: true},
		{Name: "upload-strict", Label: "Upload (strict)", Mode: "On", ParanoiaLevel: 3,
			Description: "Pages with file uploads. Keeps RCE/PHP/LFI families hot.", BuiltIn: true},
		{Name: "api-json", Label: "API / JSON", Mode: "On", ParanoiaLevel: 2,
			RelaxFamilies: []string{"xss"},
			Description:   "JSON endpoints. Injection coverage; XSS relaxed (not HTML-rendered).", BuiltIn: true},
		{Name: "static-lenient", Label: "Static (lenient)", Mode: "DetectionOnly", ParanoiaLevel: 1,
			RelaxFamilies: []string{"sqli", "rce", "php", "session"},
			Description:   "Static assets/content with no inputs. Minimal rules to avoid FPs.", BuiltIn: true},
	}
}

// toPagePolicy converts a profile bound at path into the page-policy the engine
// compiles. Relaxed families become rule-id exclusions.
func (p Profile) toPagePolicy(path string) PagePolicy {
	var excl []int
	for _, fam := range p.RelaxFamilies {
		excl = append(excl, crsFamilyRanges[fam]...)
	}
	sort.Ints(excl)
	return PagePolicy{
		Path: path, Match: "prefix", Mode: p.Mode, ParanoiaLevel: p.ParanoiaLevel,
		ExcludeRuleIDs: excl, Note: "profile: " + p.Name, Source: "profile:" + p.Name,
	}
}

func profileByName(list []Profile, name string) (Profile, bool) {
	for _, p := range list {
		if p.Name == name {
			return p, true
		}
	}
	return Profile{}, false
}

// ── content signals (learned by the crawler) ────────────────────────────

type pageSignal struct {
	hasForm     bool
	postForm    bool
	inputs      int
	hasPassword bool
	hasFile     bool
	hasHidden   bool
	queryParams int
	json        bool
	static      bool
	seenPost    bool
	crawled     bool
	fields      []DiscoveredField
}

type DiscoveredField struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Method   string `json:"method"`
	Action   string `json:"action,omitempty"`
	Required bool   `json:"required,omitempty"`
}

var (
	formTagRE  = regexp.MustCompile(`(?is)<form\b([^>]*)>(.*?)</form>`)
	fieldTagRE = regexp.MustCompile(`(?is)<(input|textarea|select)\b([^>]*)>`)
	attrRE     = regexp.MustCompile(`(?i)([a-z_:][-a-z0-9_:.]*)\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s>]+))`)
)

func htmlAttrs(raw string) map[string]string {
	out := map[string]string{}
	for _, match := range attrRE.FindAllStringSubmatch(raw, -1) {
		value := match[2]
		if value == "" {
			value = match[3]
		}
		if value == "" {
			value = match[4]
		}
		out[strings.ToLower(match[1])] = value
	}
	return out
}

func discoverFields(body string) []DiscoveredField {
	seen := map[string]bool{}
	var out []DiscoveredField
	for _, form := range formTagRE.FindAllStringSubmatch(body, -1) {
		formAttrs := htmlAttrs(form[1])
		method := strings.ToUpper(formAttrs["method"])
		if method == "" {
			method = "GET"
		}
		for _, tag := range fieldTagRE.FindAllStringSubmatch(form[2], -1) {
			attrs := htmlAttrs(tag[2])
			name := strings.TrimSpace(attrs["name"])
			if !validFieldName(name) || seen[method+"\x00"+name] {
				continue
			}
			seen[method+"\x00"+name] = true
			kind := strings.ToLower(attrs["type"])
			if kind == "" {
				kind = strings.ToLower(tag[1])
				if kind == "input" {
					kind = "text"
				}
			}
			_, required := attrs["required"]
			if !required {
				required = regexp.MustCompile(`(?i)(^|\s)required(?:\s|$)`).MatchString(tag[2])
			}
			out = append(out, DiscoveredField{Name: name, Type: kind, Method: method, Action: formAttrs["action"], Required: required})
		}
	}
	return out
}

type siteSignals struct {
	mu   sync.Mutex
	byPath map[string]*pageSignal
}

type signalStore struct {
	mu    sync.Mutex
	sites map[string]*siteSignals
}

func newSignalStore() *signalStore { return &signalStore{sites: map[string]*siteSignals{}} }

func (s *signalStore) forSite(site string) *siteSignals {
	s.mu.Lock()
	defer s.mu.Unlock()
	ss := s.sites[site]
	if ss == nil {
		ss = &siteSignals{byPath: map[string]*pageSignal{}}
		s.sites[site] = ss
	}
	return ss
}

func (s *signalStore) get(site, path string) *pageSignal {
	ss := s.forSite(site)
	ss.mu.Lock()
	defer ss.mu.Unlock()
	sig := ss.byPath[path]
	if sig == nil {
		sig = &pageSignal{}
		ss.byPath[path] = sig
	}
	return sig
}

func (s *signalStore) clear(site string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sites, site)
}

// noteContent records HTML-derived signals for a crawled page.
func (s *signalStore) noteContent(site, path, contentType, body string) {
	ss := s.forSite(site)
	ss.mu.Lock()
	defer ss.mu.Unlock()
	sig := ss.byPath[path]
	if sig == nil {
		sig = &pageSignal{}
		ss.byPath[path] = sig
	}
	sig.crawled = true
	ct := strings.ToLower(contentType)
	sig.json = strings.Contains(ct, "json")
	isHTML := strings.Contains(ct, "html")
	sig.static = !isHTML && !sig.json && ct != ""

	if isHTML {
		low := strings.ToLower(body)
		if strings.Contains(low, "<form") {
			sig.hasForm = true
			// crude: does any form use method=post
			if strings.Contains(low, "method=\"post\"") || strings.Contains(low, "method=post") || strings.Contains(low, "method='post'") {
				sig.postForm = true
			}
		}
		sig.inputs = strings.Count(low, "<input")
		sig.hasPassword = strings.Contains(low, "type=\"password\"") || strings.Contains(low, "type=password")
		sig.hasFile = strings.Contains(low, "type=\"file\"") || strings.Contains(low, "type=file")
		sig.hasHidden = strings.Contains(low, "type=\"hidden\"")
		sig.fields = discoverFields(body)
	}
}

func (s *signalStore) discoveredFields(site, path string) []DiscoveredField {
	ss := s.forSite(site)
	ss.mu.Lock()
	defer ss.mu.Unlock()
	sig := ss.byPath[path]
	if sig == nil || len(sig.fields) == 0 {
		return []DiscoveredField{}
	}
	out := make([]DiscoveredField, len(sig.fields))
	copy(out, sig.fields)
	return out
}

// noteRequestShape records passive signals available without crawling.
func (s *signalStore) noteRequestShape(site, path, method, rawQuery, contentType string) {
	ss := s.forSite(site)
	ss.mu.Lock()
	defer ss.mu.Unlock()
	sig := ss.byPath[path]
	if sig == nil {
		sig = &pageSignal{}
		ss.byPath[path] = sig
	}
	if strings.EqualFold(method, "POST") || strings.EqualFold(method, "PUT") || strings.EqualFold(method, "PATCH") {
		sig.seenPost = true
	}
	if strings.Contains(strings.ToLower(contentType), "application/json") {
		sig.json = true // live JSON request body — real API signal, no crawl needed
	}
	if rawQuery != "" {
		n := strings.Count(rawQuery, "=")
		if n > sig.queryParams {
			sig.queryParams = n
		}
	}
}

// summary builds a short, non-sensitive description of a page's signals for
// LLM review (no field values, no bodies — just shape).
func (s *signalStore) summary(site, path string) string {
	ss := s.forSite(site)
	ss.mu.Lock()
	defer ss.mu.Unlock()
	sig := ss.byPath[path]
	if sig == nil {
		return "no signals"
	}
	var parts []string
	if sig.crawled {
		parts = append(parts, "crawled")
	}
	if sig.hasForm {
		parts = append(parts, "has <form>")
	}
	if sig.postForm {
		parts = append(parts, "POST form")
	}
	if sig.inputs > 0 {
		parts = append(parts, "input fields")
	}
	if sig.hasPassword {
		parts = append(parts, "password field")
	}
	if sig.hasFile {
		parts = append(parts, "file upload")
	}
	if sig.json {
		parts = append(parts, "JSON content-type")
	}
	if sig.static {
		parts = append(parts, "static asset")
	}
	if sig.queryParams > 0 {
		parts = append(parts, "query params")
	}
	if sig.seenPost {
		parts = append(parts, "observed POST/PUT")
	}
	if len(parts) == 0 {
		return "no notable signals"
	}
	return strings.Join(parts, ", ")
}

type profileSuggestion struct {
	Path       string `json:"path"`
	Profile    string `json:"profile"`
	Confidence int    `json:"confidence"` // 0-100 (content-signal strength)
	Rationale  string `json:"rationale"`
	Crawled    bool   `json:"crawled"`
	Bound      string `json:"bound,omitempty"` // currently bound profile, if any
}

// classify maps a page's signals to a recommended profile + confidence.
func classify(sig *pageSignal) (profile, rationale string, confidence int) {
	switch {
	case sig.hasFile:
		return "upload-strict", "file upload input present", 90
	case sig.hasPassword:
		return "form-sqli-xss", "password field — likely a login/credential form", 92
	case sig.postForm || (sig.hasForm && sig.inputs > 0):
		return "form-sqli-xss", "HTML form with input fields", 85
	case sig.seenPost && sig.json:
		return "api-json", "observed POST/PUT to a JSON endpoint", 82
	case sig.seenPost:
		// Live POST/PUT/PATCH traffic is real write-path evidence and must
		// outrank crawl-derived "static" conclusions — a crawler only issues
		// GETs, so it can never see a write endpoint's true shape.
		return "form-sqli-xss", "observed POST/PUT/PATCH traffic (write endpoint)", 65
	case sig.json:
		return "api-json", "JSON content type", 80
	case sig.queryParams >= 1:
		return "query-hardened", "driven by query parameters", 70
	case sig.static:
		return "static-lenient", "static asset, no inputs", 75
	case sig.crawled:
		return "static-lenient", "crawled HTML with no forms or inputs", 55
	default:
		return "", "not enough signal — crawl this page (Discover)", 20
	}
}

// suggestProfiles walks a site's known paths and produces suggestions, marking
// any that already have a bound profile page-policy.
func (s *signalStore) suggestProfiles(site string, bound map[string]string) []profileSuggestion {
	ss := s.forSite(site)
	ss.mu.Lock()
	defer ss.mu.Unlock()
	out := make([]profileSuggestion, 0, len(ss.byPath))
	for path, sig := range ss.byPath {
		prof, why, conf := classify(sig)
		if prof == "" {
			continue
		}
		out = append(out, profileSuggestion{
			Path: path, Profile: prof, Confidence: conf, Rationale: why,
			Crawled: sig.crawled, Bound: bound[path],
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Confidence != out[j].Confidence {
			return out[i].Confidence > out[j].Confidence
		}
		return out[i].Path < out[j].Path
	})
	return out
}
