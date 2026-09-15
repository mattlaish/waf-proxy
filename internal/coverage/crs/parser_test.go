package crs

import "testing"

func TestParseRuleAtExtractsCRSMetadata(t *testing.T) {
	r := ParseRuleAt(`SecRule REQUEST_HEADERS:User-Agent "@rx attack" "id:941100,phase:1,pass,t:none,t:lowercase,tag:'attack-xss',severity:'CRITICAL'"`, "x.conf", 7)
	if r.ID != "941100" || r.Phase != 1 || r.Operator != "rx" || len(r.Variables) != 1 || r.Variables[0] != "REQUEST_HEADERS:User-Agent" {
		t.Fatalf("unexpected parse: %#v", r)
	}
	if len(r.Transforms) != 2 || r.Transforms[0] != "none" || r.Transforms[1] != "lowercase" {
		t.Fatalf("unexpected transforms: %#v", r.Transforms)
	}
	if r.Severity != "CRITICAL" || len(r.Tags) != 1 || r.Tags[0] != "attack-xss" {
		t.Fatalf("unexpected metadata: %#v", r)
	}
}

func TestAnalyzeMatchesRuntimeConservativeScope(t *testing.T) {
	r := ParseRule(`SecRule REQUEST_URI "@rx admin" "id:1001,phase:1,pass,t:none"`)
	got := Analyze(r)
	if !got.Eligible {
		t.Fatalf("expected eligible: %s", got.Reason)
	}
	bad := ParseRule(`SecRule REQUEST_URI|ARGS "@rx admin" "id:1002,phase:1,pass,t:none"`)
	if Analyze(bad).Eligible {
		t.Fatal("multi-variable selector must remain Coraza-only")
	}
}
