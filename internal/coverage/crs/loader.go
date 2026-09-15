package crs

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	maxRulesetBytes  = int64(128 << 20)
	maxStatementSize = 4 << 20
)

type loader struct {
	seen     map[string]bool
	files    []string
	rules    []Rule
	warnings []Warning
	total    int64
}

// LoadRuleset loads either a CRS configuration entrypoint (following Include
// and IncludeOptional recursively) or every .conf file below a directory.
// Files are visited once in deterministic order and total source bytes are
// bounded to protect the operator tool from accidental unbounded ingestion.
func LoadRuleset(path string) (Ruleset, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return Ruleset{}, err
	}
	l := &loader{seen: map[string]bool{}}
	st, err := os.Stat(abs)
	if err != nil {
		return Ruleset{}, err
	}
	if st.IsDir() {
		var files []string
		err := filepath.WalkDir(abs, func(p string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				return nil
			}
			if strings.EqualFold(filepath.Ext(d.Name()), ".conf") {
				files = append(files, p)
			}
			return nil
		})
		if err != nil {
			return Ruleset{}, err
		}
		sort.Strings(files)
		for _, f := range files {
			if err := l.loadFile(f); err != nil {
				return Ruleset{}, err
			}
		}
	} else if err := l.loadFile(abs); err != nil {
		return Ruleset{}, err
	}

	rs := Ruleset{Root: abs, Files: append([]string(nil), l.files...), Rules: l.rules, Warnings: l.warnings}
	rs.DuplicateRuleIDs = findDuplicates(rs.Rules)
	return rs, nil
}

func (l *loader) loadFile(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if l.seen[abs] {
		return nil
	}
	l.seen[abs] = true
	st, err := os.Stat(abs)
	if err != nil {
		return err
	}
	if !st.Mode().IsRegular() {
		return fmt.Errorf("CRS source is not a regular file: %s", abs)
	}
	l.total += st.Size()
	if l.total > maxRulesetBytes {
		return fmt.Errorf("CRS coverage source exceeds %d bytes", maxRulesetBytes)
	}
	l.files = append(l.files, abs)

	f, err := os.Open(abs)
	if err != nil {
		return err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), maxStatementSize)
	var cur strings.Builder
	startLine := 0
	lineNo := 0
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
					l.warnings = append(l.warnings, Warning{File: abs, Line: startLine, Message: "dynamic IncludeOptional expression not expanded: " + arg})
					return nil
				}
				return fmt.Errorf("dynamic mandatory Include expression unsupported for coverage/runtime parity: %s", arg)
			}
			if !filepath.IsAbs(arg) {
				arg = filepath.Join(filepath.Dir(abs), arg)
			}
			matches, err := filepath.Glob(arg)
			if err != nil {
				return fmt.Errorf("include glob %s: %w", arg, err)
			}
			if len(matches) == 0 {
				if optional {
					return nil
				}
				return fmt.Errorf("mandatory Include matched no files: %s", arg)
			}
			sort.Strings(matches)
			for _, match := range matches {
				if err := l.loadFile(match); err != nil {
					return err
				}
			}
			return nil
		}
		if strings.HasPrefix(lower, "secrule ") {
			r := ParseRuleAt(st, abs, startLine)
			if r.ID == "" {
				l.warnings = append(l.warnings, Warning{File: abs, Line: startLine, Message: "SecRule missing parseable id"})
			}
			l.rules = append(l.rules, r)
		}
		return nil
	}

	for sc.Scan() {
		lineNo++
		line := strings.TrimSpace(sc.Text())
		if line == "" || (cur.Len() == 0 && strings.HasPrefix(line, "#")) {
			continue
		}
		if cur.Len() == 0 {
			startLine = lineNo
		}
		continued := strings.HasSuffix(line, "\\")
		if continued {
			line = strings.TrimSpace(strings.TrimSuffix(line, "\\"))
		}
		if cur.Len() > 0 {
			cur.WriteByte(' ')
		}
		cur.WriteString(line)
		if !continued {
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

func findDuplicates(rules []Rule) map[string][]RuleLocation {
	locs := map[string][]RuleLocation{}
	for _, r := range rules {
		if r.ID == "" {
			continue
		}
		locs[r.ID] = append(locs[r.ID], RuleLocation{File: r.File, Line: r.Line})
	}
	for id, entries := range locs {
		if len(entries) < 2 {
			delete(locs, id)
		}
	}
	if len(locs) == 0 {
		return nil
	}
	return locs
}
