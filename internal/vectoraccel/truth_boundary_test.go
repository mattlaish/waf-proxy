package vectoraccel

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	coveragecrs "waf-proxy/internal/coverage/crs"
)

func TestCoverageAndRuntimeClassifierParity(t *testing.T) {
	cases := []string{
		`SecRule REQUEST_URI "@rx normal" "id:810001,phase:1,pass,t:none"`,
		`SecRule REQUEST_HEADERS:X!Foo "@rx x" "id:810002,phase:1,pass,t:none"`,
		`SecRule REQUEST_HEADERS:X&Foo "@rx x" "id:810003,phase:1,pass,t:none"`,
		`SecRule REQUEST_URI "@rx " "id:810004,phase:1,pass,t:none"`,
		`SecRule REQUEST_URI "@rx implicit" "id:810005,pass,t:none"`,
		`SecRule REQUEST_URI "@rx chain-word" "id:810006,phase:1,pass,t:none,tag:'not-a-chain-action'"`,
		`SecRule REQUEST_URI "@rx chained" "id:810007,phase:1,pass,t:none,chain"`,
		`SecRule REQUEST_URI "!@rx negated" "id:810008,phase:1,pass,t:none"`,
		`SecRule REQUEST_URI|REQUEST_METHOD "@rx multi" "id:810009,phase:1,pass,t:none"`,
		`SecRule REQUEST_URI "@rx upper" "id:810010,phase:1,pass,t:none,t:uppercase"`,
	}
	for _, statement := range cases {
		r := coveragecrs.ParseRule(statement)
		coverageEligible := coveragecrs.Analyze(r).Eligible
		id, err := strconv.Atoi(r.ID)
		if err != nil {
			t.Fatalf("parse id %q: %v", r.ID, err)
		}
		vars, op, actions, ok := splitSecRule(statement)
		if !ok {
			t.Fatalf("split failed: %s", statement)
		}
		_, runtimeEligible := classifyRule(id, vars, op, actions)
		if coverageEligible != runtimeEligible {
			t.Fatalf("classifier parity mismatch id=%s coverage=%v runtime=%v statement=%s", r.ID, coverageEligible, runtimeEligible, statement)
		}
	}
}

func TestIncludeOptionalParity(t *testing.T) {
	d := t.TempDir()
	child := filepath.Join(d, "optional.conf")
	if err := os.WriteFile(child, []byte(`SecRule REQUEST_URI "@rx optional" "id:820001,phase:1,pass,t:none"`+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	entry := filepath.Join(d, "entry.conf")
	if err := os.WriteFile(entry, []byte("IncludeOptional optional.conf\nIncludeOptional missing-*.conf\n"), 0600); err != nil {
		t.Fatal(err)
	}
	rs, err := coveragecrs.LoadRuleset(entry)
	if err != nil {
		t.Fatal(err)
	}
	inv := coveragecrs.BuildInventory(rs)
	if len(inv.Rules) != 1 || !inv.Rules[0].Eligible {
		t.Fatalf("unexpected coverage inventory: %#v", inv)
	}
	groups, _, err := ParseRules(entry)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, g := range groups {
		count += len(g.Rules)
	}
	if count != 1 {
		t.Fatalf("runtime parsed %d eligible rules, want 1", count)
	}
}

func TestDynamicMandatoryIncludeFailsClosedInBothPaths(t *testing.T) {
	d := t.TempDir()
	entry := filepath.Join(d, "entry.conf")
	if err := os.WriteFile(entry, []byte("Include %{ENV.RULES}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := coveragecrs.LoadRuleset(entry); err == nil {
		t.Fatal("coverage loader must fail closed on dynamic mandatory Include")
	}
	if _, _, err := ParseRules(entry); err == nil {
		t.Fatal("runtime loader must fail closed on dynamic mandatory Include")
	}
}
