package crs

import (
	"strconv"
	"strings"

	shared "waf-proxy/internal/capability"
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
	targets, rest, ok := shared.NextToken(s)
	if !ok {
		return r
	}
	op, rest, ok := shared.NextToken(strings.TrimSpace(rest))
	if !ok {
		return r
	}
	actions, _, ok := shared.NextToken(strings.TrimSpace(rest))
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

	for _, rawAction := range shared.SplitActions(actions) {
		a := strings.TrimSpace(rawAction)
		if a == "" {
			continue
		}
		r.Actions = append(r.Actions, a)
		name, value := shared.SplitAction(a)
		switch strings.ToLower(name) {
		case "id":
			r.ID = shared.TrimActionValue(value)
		case "phase":
			if n, err := strconv.Atoi(shared.TrimActionValue(value)); err == nil {
				r.Phase = n
			}
		case "t":
			v := strings.ToLower(shared.TrimActionValue(value))
			if v != "" {
				r.Transforms = append(r.Transforms, v)
			}
		case "tag":
			if v := shared.TrimActionValue(value); v != "" {
				r.Tags = append(r.Tags, v)
			}
		case "severity":
			r.Severity = shared.TrimActionValue(value)
		case "chain":
			r.HasChain = true
		}
	}
	return r
}
