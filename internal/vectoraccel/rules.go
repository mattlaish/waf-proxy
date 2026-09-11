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
)

const maxRuleSourceBytes = 64 << 20
const semanticAdapterVersion = "vectorscan-learning-v4-phase2-request-metadata-encodings"

type SourceKind string

const (
	SourceRequestURI      SourceKind = "REQUEST_URI"
	SourceRequestFilename SourceKind = "REQUEST_FILENAME"
	SourceRequestMethod   SourceKind = "REQUEST_METHOD"
	SourceRequestProtocol SourceKind = "REQUEST_PROTOCOL"
	SourceRequestURIRaw   SourceKind = "REQUEST_URI_RAW"
	SourceRequestLine     SourceKind = "REQUEST_LINE"
	SourceRequestBasename SourceKind = "REQUEST_BASENAME"
	SourceQueryString     SourceKind = "QUERY_STRING"
	SourceServerName      SourceKind = "SERVER_NAME"
	SourceRemoteAddr      SourceKind = "REMOTE_ADDR"
	SourceRemotePort      SourceKind = "REMOTE_PORT"
	SourceRequestHeader   SourceKind = "REQUEST_HEADERS"
)

type RuleSpec struct {
	ID         int             `json:"id"`
	Phase      int             `json:"phase"`
	Source     SourceKind      `json:"source"`
	Header     string          `json:"header,omitempty"`
	Transforms []TransformKind `json:"transforms,omitempty"`
	Pattern    string          `json:"pattern"`
}

type GroupSpec struct {
	Key         string          `json:"key"`
	Source      SourceKind      `json:"source"`
	Header      string          `json:"header,omitempty"`
	Transforms  []TransformKind `json:"transforms,omitempty"`
	Phase       int             `json:"phase"`
	Rules       []RuleSpec      `json:"rules"`
	Fingerprint string          `json:"fingerprint"`
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
		key := fmt.Sprintf("p%d|%s|%s|t=%s", spec.Phase, spec.Source, strings.ToLower(spec.Header), transformKey(spec.Transforms))
		g := groups[key]
		if g == nil {
			g = &GroupSpec{Key: key, Source: spec.Source, Header: spec.Header, Transforms: append([]TransformKind(nil), spec.Transforms...), Phase: spec.Phase}
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
	if strings.Contains(vars, "|") || strings.Contains(vars, "!") || strings.Contains(vars, "&") {
		return RuleSpec{}, false
	}
	if strings.Contains(strings.ToLower(actions), "chain") {
		return RuleSpec{}, false
	}
	if !strings.HasPrefix(op, "@rx ") || strings.HasPrefix(op, "!@") {
		return RuleSpec{}, false
	}
	pattern := strings.TrimSpace(strings.TrimPrefix(op, "@rx "))
	if pattern == "" {
		return RuleSpec{}, false
	}
	pm := phaseRE.FindStringSubmatch(actions)
	phase := 2
	if len(pm) == 2 {
		phase, _ = strconv.Atoi(pm[1])
	}
	if phase != 1 && phase != 2 {
		return RuleSpec{}, false
	}
	// Require explicit t:none so inherited SecDefaultAction transformations
	// cannot make our reconstructed input differ from Coraza's input.
	lowerActions := strings.ToLower(actions)
	if !strings.Contains(lowerActions, "t:none") {
		return RuleSpec{}, false
	}
	transforms := []string{}
	for _, a := range splitActions(actions) {
		a = strings.TrimSpace(strings.ToLower(a))
		if strings.HasPrefix(a, "t:") {
			transforms = append(transforms, strings.TrimPrefix(a, "t:"))
		}
	}
	if len(transforms) == 0 || transforms[0] != "none" {
		return RuleSpec{}, false
	}
	pipeline := make([]TransformKind, 0, len(transforms)-1)
	for _, t := range transforms[1:] {
		tk, ok := parseExactTransform(t)
		if !ok {
			return RuleSpec{}, false
		}
		pipeline = append(pipeline, tk)
	}
	v := strings.TrimSpace(vars)
	spec := RuleSpec{ID: id, Phase: phase, Transforms: pipeline, Pattern: pattern}
	switch {
	case v == "REQUEST_URI":
		spec.Source = SourceRequestURI
	case v == "REQUEST_FILENAME":
		spec.Source = SourceRequestFilename
	case v == "REQUEST_METHOD":
		spec.Source = SourceRequestMethod
	case v == "REQUEST_PROTOCOL":
		spec.Source = SourceRequestProtocol
	case v == "REQUEST_URI_RAW":
		spec.Source = SourceRequestURIRaw
	case v == "REQUEST_LINE":
		spec.Source = SourceRequestLine
	case v == "REQUEST_BASENAME":
		spec.Source = SourceRequestBasename
	case v == "QUERY_STRING":
		spec.Source = SourceQueryString
	case v == "SERVER_NAME":
		spec.Source = SourceServerName
	case v == "REMOTE_ADDR":
		spec.Source = SourceRemoteAddr
	case v == "REMOTE_PORT":
		spec.Source = SourceRemotePort
	case strings.HasPrefix(v, "REQUEST_HEADERS:"):
		h := strings.TrimSpace(strings.TrimPrefix(v, "REQUEST_HEADERS:"))
		if h == "" || strings.ContainsAny(h, " /\\\"'|!&") {
			return RuleSpec{}, false
		}
		spec.Source = SourceRequestHeader
		spec.Header = h
	default:
		return RuleSpec{}, false
	}
	return spec, true
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
			if strings.HasPrefix(strings.ToLower(st), "include ") {
				arg := strings.TrimSpace(st[len("Include "):])
				arg = strings.Trim(arg, "\"'")
				if !filepath.IsAbs(arg) {
					arg = filepath.Join(filepath.Dir(abs), arg)
				}
				ms, err := filepath.Glob(arg)
				if err != nil {
					return err
				}
				if len(ms) == 0 {
					return nil
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
