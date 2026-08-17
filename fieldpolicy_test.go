package main

import (
	"strings"
	"testing"

	"github.com/corazawaf/coraza/v3"
)

func TestPagePolicyDirectivesFieldPolicy(t *testing.T) {
	got := pagePolicyDirectives([]PagePolicy{{
		Path: "/register", Match: "exact", Methods: []string{"POST"},
		Fields: []FieldPolicy{
			{Name: "user_id", Profile: "identifier", Required: true, MinLength: 3, MaxLength: 64},
			{Name: "password", Profile: "password", Required: true, MinLength: 12, MaxLength: 256, ExcludeRuleIDs: []int{942100}},
		},
	}})
	for _, want := range []string{
		"ctl:ruleRemoveTargetById=942100;ARGS_POST:password",
		"required field missing: user_id",
		"SecRule ARGS_POST:user_id \"!@rx ^[A-Za-z0-9._-]*$\"",
		"SecRule ARGS_POST:password \"!@rx (?s)^.{12,256}$\"",
		"SecRule REQUEST_METHOD \"@rx ^(?:POST)$\"",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("directives missing %q:\n%s", want, got)
		}
	}

	// The generated directives must be accepted by the same Coraza parser used
	// at Apply time; this catches broken chain/action syntax.
	cfg := coraza.NewWAFConfig().WithDirectives("SecRuleEngine On\nSecRequestBodyAccess On\n" + got)
	if _, err := coraza.NewWAF(cfg); err != nil {
		t.Fatalf("generated field directives do not compile: %v\n%s", err, got)
	}
}

func TestDiscoverFields(t *testing.T) {
	html := `<form method="post" action="/register">
		<input name="user_id" required>
		<input type="password" name="password" required>
		<textarea name="comment"></textarea>
	</form>`
	got := discoverFields(html)
	if len(got) != 3 {
		t.Fatalf("got %d fields, want 3: %#v", len(got), got)
	}
	if got[0].Name != "user_id" || got[0].Method != "POST" || !got[0].Required {
		t.Fatalf("unexpected first field: %#v", got[0])
	}
	if got[1].Type != "password" || got[2].Type != "textarea" {
		t.Fatalf("types not detected: %#v", got)
	}
}

func TestFieldPolicyValidationRejectsUnsafeName(t *testing.T) {
	if validFieldName(`user",ctl:ruleEngine=Off`) {
		t.Fatal("unsafe field name accepted")
	}
	if !validFieldName("user_id") {
		t.Fatal("normal field name rejected")
	}
}
