package crs

import (
	"strconv"
	"strings"
)

// ParseRule preserves the Slice B API for focused callers.
func ParseRule(line string) Rule {
	return ParseRuleAt(line, "", 0)
}

// ParseRuleAt parses the subset of SecLang metadata required for conservative
// coverage analysis. It does not attempt to execute or reinterpret a rule.
func ParseRuleAt(statement, file string, line int) Rule {
	r := Rule{File: file, Line: line, Raw: statement}
	s := strings.TrimSpace(statement)
	if len(s) < len("SecRule") || !strings.EqualFold(s[:len("SecRule")], "SecRule") {
		return r
	}
	s = strings.TrimSpace(s[len("SecRule"):])
	targets, rest, ok := nextToken(s)
	if !ok {
		return r
	}
	op, rest, ok := nextToken(strings.TrimSpace(rest))
	if !ok {
		return r
	}
	actions, _, ok := nextToken(strings.TrimSpace(rest))
	if !ok {
		return r
	}

	for _, target := range strings.Split(targets, "|") {
		target = strings.TrimSpace(target)
		if target != "" {
			r.Variables = append(r.Variables, target)
		}
	}

	r.RawOperator = strings.TrimSpace(op)
	opWork := r.RawOperator
	if strings.HasPrefix(opWork, "!") {
		r.Negated = true
		opWork = strings.TrimSpace(strings.TrimPrefix(opWork, "!"))
	}
	if strings.HasPrefix(opWork, "@") {
		name := strings.TrimPrefix(opWork, "@")
		if i := strings.IndexAny(name, " \t"); i >= 0 {
			r.Operator = strings.ToLower(strings.TrimSpace(name[:i]))
			r.Pattern = strings.TrimSpace(name[i+1:])
		} else {
			r.Operator = strings.ToLower(strings.TrimSpace(name))
		}
	}

	for _, rawAction := range splitActions(actions) {
		a := strings.TrimSpace(rawAction)
		if a == "" {
			continue
		}
		r.Actions = append(r.Actions, a)
		name, value := splitAction(a)
		switch strings.ToLower(name) {
		case "id":
			r.ID = trimActionValue(value)
		case "phase":
			if n, err := strconv.Atoi(trimActionValue(value)); err == nil {
				r.Phase = n
			}
		case "t":
			v := strings.ToLower(trimActionValue(value))
			if v != "" {
				r.Transforms = append(r.Transforms, v)
			}
		case "tag":
			if v := trimActionValue(value); v != "" {
				r.Tags = append(r.Tags, v)
			}
		case "severity":
			r.Severity = trimActionValue(value)
		case "chain":
			r.HasChain = true
		}
	}
	return r
}

func nextToken(s string) (string, string, bool) {
	if s == "" {
		return "", "", false
	}
	if s[0] == '"' || s[0] == '\'' {
		q := s[0]
		var b strings.Builder
		escaped := false
		for i := 1; i < len(s); i++ {
			c := s[i]
			if escaped {
				b.WriteByte(c)
				escaped = false
				continue
			}
			if c == '\\' {
				escaped = true
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
	if i := strings.IndexAny(s, " \t\r\n"); i >= 0 {
		return s[:i], s[i:], true
	}
	return s, "", true
}

func splitActions(s string) []string {
	var out []string
	start := 0
	quote := byte(0)
	escaped := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if escaped {
			escaped = false
			continue
		}
		if c == '\\' {
			escaped = true
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
	return append(out, s[start:])
}

func splitAction(s string) (string, string) {
	if i := strings.IndexByte(s, ':'); i >= 0 {
		return strings.TrimSpace(s[:i]), strings.TrimSpace(s[i+1:])
	}
	return strings.TrimSpace(s), ""
}

func trimActionValue(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && ((s[0] == '\'' && s[len(s)-1] == '\'') || (s[0] == '"' && s[len(s)-1] == '"')) {
		s = s[1 : len(s)-1]
	}
	return strings.TrimSpace(s)
}
