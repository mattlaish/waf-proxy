package vectoraccel

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	shared "waf-proxy/internal/capability"
)

const maxRuleSourceBytes = 64 << 20
const semanticAdapterVersion = "vectorscan-learning-v2-coraza37-context-truth"

type SourceKind string

const (
	SourceRequestURI      SourceKind = "REQUEST_URI"
	SourceRequestFilename SourceKind = "REQUEST_FILENAME"
	SourceRequestMethod   SourceKind = "REQUEST_METHOD"
	SourceRequestProtocol SourceKind = "REQUEST_PROTOCOL"
	SourceRequestHeader   SourceKind = "REQUEST_HEADERS"
)

type RuleSpec struct {
	ID        int        `json:"id"`
	Phase     int        `json:"phase"`
	Source    SourceKind `json:"source"`
	Header    string     `json:"header,omitempty"`
	Lowercase bool       `json:"lowercase,omitempty"`
	Pattern   string     `json:"pattern"`
}

type GroupSpec struct {
	Key         string     `json:"key"`
	Source      SourceKind `json:"source"`
	Header      string     `json:"header,omitempty"`
	Lowercase   bool       `json:"lowercase,omitempty"`
	Phase       int        `json:"phase"`
	Rules       []RuleSpec `json:"rules"`
	Fingerprint string     `json:"fingerprint"`
}

var (
	idRE    = regexp.MustCompile(`(?i)(?:^|,)\s*id\s*:\s*([0-9]+)`)
	phaseRE = regexp.MustCompile(`(?i)(?:^|,)\s*phase\s*:\s*([1-5])`)
)

// ParseRules conservatively extracts standalone positive @rx rules that can be
// evaluated against exactly one request value. Unsupported rules remain Coraza-only.
func ParseRules(path string) ([]GroupSpec, map[int]struct{}, error) {
	stmts, err := loadStatements(path)
	if err != nil {
		return nil, nil, err
	}
	used := map[int]struct{}{}
	groups := map[string]*GroupSpec{}
	for _, st := range stmts {
		// Reserve every explicit SecLang ID, including SecAction/SecMarker-like
		// statements, so internal control rules never collide with user/CRS IDs.
		for _, m := range idRE.FindAllStringSubmatch(st, -1) {
			if len(m) == 2 {
				if id, err := strconv.Atoi(m[1]); err == nil && id > 0 {
					used[id] = struct{}{}
				}
			}
		}
		if !strings.HasPrefix(strings.TrimSpace(st), "SecRule ") {
			continue
		}
		vars, op, actions, ok := splitSecRule(st)
		if !ok {
			continue
		}
		idm := idRE.FindStringSubmatch(actions)
		if len(idm) != 2 {
			continue
		}
		id, _ := strconv.Atoi(idm[1])
		if id <= 0 {
			continue
		}
		used[id] = struct{}{}
		spec, ok := classifyRule(id, vars, op, actions)
		if !ok {
			continue
		}
		key := fmt.Sprintf("p%d|%s|%s|lc=%t", spec.Phase, spec.Source, strings.ToLower(spec.Header), spec.Lowercase)
		g := groups[key]
		if g == nil {
			g = &GroupSpec{Key: key, Source: spec.Source, Header: spec.Header, Lowercase: spec.Lowercase, Phase: spec.Phase}
			groups[key] = g
		}
		g.Rules = append(g.Rules, spec)
	}
	out := make([]GroupSpec, 0, len(groups))
	for _, g := range groups {
		sort.Slice(g.Rules, func(i, j int) bool { return g.Rules[i].ID < g.Rules[j].ID })
		h := sha256.New()
		fmt.Fprintf(h, "%s|%s|", semanticAdapterVersion, g.Key)
		for _, r := range g.Rules {
			fmt.Fprintf(h, "%d:%s\x00", r.ID, r.Pattern)
		}
		g.Fingerprint = hex.EncodeToString(h.Sum(nil))
		out = append(out, *g)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out, used, nil
}

func classifyRule(id int, vars, op, actions string) (RuleSpec, bool) {
	operator := strings.TrimSpace(op)
	negated := strings.HasPrefix(operator, "!")
	if negated {
		operator = strings.TrimSpace(strings.TrimPrefix(operator, "!"))
	}
	operatorName := ""
	pattern := ""
	if strings.HasPrefix(operator, "@") {
		name := strings.TrimPrefix(operator, "@")
		if i := strings.IndexAny(name, " \t"); i >= 0 {
			operatorName = strings.ToLower(strings.TrimSpace(name[:i]))
			pattern = strings.TrimSpace(name[i+1:])
		} else {
			operatorName = strings.ToLower(strings.TrimSpace(name))
		}
	}

	phase := 0
	if pm := phaseRE.FindStringSubmatch(actions); len(pm) == 2 {
		phase, _ = strconv.Atoi(pm[1])
	}
	transforms := []string{}
	hasChain := false
	for _, raw := range splitActions(actions) {
		a := strings.TrimSpace(raw)
		name, value := splitActionForCapability(a)
		switch strings.ToLower(name) {
		case "t":
			if v := strings.ToLower(trimQuotedActionValue(value)); v != "" {
				transforms = append(transforms, v)
			}
		case "chain":
			hasChain = true
		}
	}

	selectors := []string{}
	for _, v := range strings.Split(vars, "|") {
		v = strings.TrimSpace(v)
		if v != "" {
			selectors = append(selectors, v)
		}
	}
	result := shared.Classify(shared.Input{
		RuleID:     strconv.Itoa(id),
		Phase:      phase,
		Operator:   operatorName,
		Pattern:    pattern,
		Variables:  selectors,
		Transforms: transforms,
		HasChain:   hasChain,
		Negated:    negated,
	})
	if !result.Eligible {
		return RuleSpec{}, false
	}

	spec := RuleSpec{
		ID:        id,
		Phase:     result.Phase,
		Source:    SourceKind(result.Source),
		Header:    result.Header,
		Lowercase: result.Lowercase,
		Pattern:   pattern,
	}
	return spec, true
}

func splitActionForCapability(s string) (string, string) {
	if i := strings.IndexByte(s, ':'); i >= 0 {
		return strings.TrimSpace(s[:i]), strings.TrimSpace(s[i+1:])
	}
	return strings.TrimSpace(s), ""
}

func trimQuotedActionValue(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && ((s[0] == '\'' && s[len(s)-1] == '\'') || (s[0] == '"' && s[len(s)-1] == '"')) {
		s = s[1 : len(s)-1]
	}
	return strings.TrimSpace(s)
}

func splitActions(s string) []string {
	var out []string
	start := 0
	quote := byte(0)
	esc := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if esc {
			esc = false
			continue
		}
		if c == '\\' {
			esc = true
			continue
		}
		if quote != 0 {
			if c == quote {
				quote = 0
			}
			continue
		}
		if c == '\'' || c == '"' {
			quote = c
			continue
		}
		if c == ',' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	out = append(out, s[start:])
	return out
}

func splitSecRule(st string) (string, string, string, bool) {
	s := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(st), "SecRule"))
	if s == "" {
		return "", "", "", false
	}
	vars, rest, ok := nextToken(s)
	if !ok {
		return "", "", "", false
	}
	op, rest, ok := nextToken(strings.TrimSpace(rest))
	if !ok {
		return "", "", "", false
	}
	actions, _, ok := nextToken(strings.TrimSpace(rest))
	if !ok {
		return "", "", "", false
	}
	return vars, op, actions, true
}

func nextToken(s string) (string, string, bool) {
	if s == "" {
		return "", "", false
	}
	if s[0] == '"' || s[0] == '\'' {
		q := s[0]
		var b strings.Builder
		esc := false
		for i := 1; i < len(s); i++ {
			c := s[i]
			if esc {
				b.WriteByte(c)
				esc = false
				continue
			}
			if c == '\\' {
				esc = true
				b.WriteByte(c)
				continue
			}
			if c == q {
				return b.String(), s[i+1:], true
			}
			b.WriteByte(c)
		}
		return "", "", false
	}
	i := strings.IndexAny(s, " \t\r\n")
	if i < 0 {
		return s, "", true
	}
	return s[:i], s[i:], true
}

func loadStatements(path string) ([]string, error) {
	seen := map[string]bool{}
	var total int64
	var out []string
	var load func(string) error
	load = func(p string) error {
		abs, err := filepath.Abs(p)
		if err != nil {
			return err
		}
		if seen[abs] {
			return nil
		}
		seen[abs] = true
		fi, err := os.Stat(abs)
		if err != nil {
			return err
		}
		total += fi.Size()
		if total > maxRuleSourceBytes {
			return fmt.Errorf("VectorScan rule source exceeds %d bytes", maxRuleSourceBytes)
		}
		f, err := os.Open(abs)
		if err != nil {
			return err
		}
		defer f.Close()
		sc := bufio.NewScanner(f)
		buf := make([]byte, 64*1024)
		sc.Buffer(buf, 2<<20)
		var cur strings.Builder
		flush := func() error {
			st := strings.TrimSpace(cur.String())
			cur.Reset()
			if st == "" || strings.HasPrefix(st, "#") {
				return nil
			}
			lower := strings.ToLower(st)
			if strings.HasPrefix(lower, "include ") || strings.HasPrefix(lower, "includeoptional ") {
				optional := strings.HasPrefix(lower, "includeoptional ")
				prefix := "Include "
				if optional {
					prefix = "IncludeOptional "
				}
				arg := strings.TrimSpace(st[len(prefix):])
				arg = strings.Trim(arg, "\"'")
				if strings.Contains(arg, "%{") {
					if optional {
						return nil
					}
					return fmt.Errorf("dynamic Include expression unsupported for VectorScan: %s", arg)
				}
				if !filepath.IsAbs(arg) {
					arg = filepath.Join(filepath.Dir(abs), arg)
				}
				ms, err := filepath.Glob(arg)
				if err != nil {
					return err
				}
				if len(ms) == 0 && !optional {
					return fmt.Errorf("mandatory Include matched no files: %s", arg)
				}
				sort.Strings(ms)
				for _, m := range ms {
					if err := load(m); err != nil {
						return err
					}
				}
				return nil
			}
			out = append(out, st)
			return nil
		}
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			cont := strings.HasSuffix(line, "\\")
			if cont {
				line = strings.TrimSpace(strings.TrimSuffix(line, "\\"))
			}
			if cur.Len() > 0 {
				cur.WriteByte(' ')
			}
			cur.WriteString(line)
			if !cont {
				if err := flush(); err != nil {
					return err
				}
			}
		}
		if err := sc.Err(); err != nil {
			return err
		}
		if cur.Len() > 0 {
			return flush()
		}
		return nil
	}
	if err := load(path); err != nil {
		return nil, err
	}
	return out, nil
}
