package main

import (
	"testing"
	"time"
)

func TestCIDRPolicyExpiryRFC3339IsApplied(t *testing.T) {
	future := time.Now().UTC().Add(time.Hour).Format(time.RFC3339)
	engine, err := newCIDRPolicyEngine(CIDRPolicyConfig{Enabled: true, Rules: []CIDRPolicyEntry{{
		Name: "temporary-block", CIDR: "203.0.113.0/24", Action: "deny", ExpiresAt: future,
	}}})
	if err != nil {
		t.Fatalf("newCIDRPolicyEngine: %v", err)
	}
	if got := engine.evaluate("203.0.113.10"); !got.Matched || got.Allowed {
		t.Fatalf("unexpired deny rule not applied: %#v", got)
	}
}

func TestCIDRPolicyExpiredRuleIsSkipped(t *testing.T) {
	past := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)
	engine, err := newCIDRPolicyEngine(CIDRPolicyConfig{Enabled: true, Rules: []CIDRPolicyEntry{{
		Name: "expired-block", CIDR: "203.0.113.0/24", Action: "deny", Priority: 100, ExpiresAt: past,
	}, {
		Name: "fallback-allow", CIDR: "203.0.113.0/24", Action: "allow", Priority: 10,
	}}})
	if err != nil {
		t.Fatalf("newCIDRPolicyEngine: %v", err)
	}
	if got := engine.evaluate("203.0.113.10"); !got.Matched || !got.Allowed || got.Rule != "fallback-allow" {
		t.Fatalf("expired rule was not skipped: %#v", got)
	}
}

func TestCIDRPolicyRejectsInvalidExpiry(t *testing.T) {
	_, err := newCIDRPolicyEngine(CIDRPolicyConfig{Rules: []CIDRPolicyEntry{{
		Name: "bad", CIDR: "203.0.113.0/24", Action: "deny", ExpiresAt: "tomorrow",
	}}})
	if err == nil {
		t.Fatal("invalid expires_at accepted")
	}
}

func TestCIDRPolicyDisabledDoesNotMatch(t *testing.T) {
	engine, err := newCIDRPolicyEngine(CIDRPolicyConfig{Enabled: false, Rules: []CIDRPolicyEntry{{
		Name: "disabled-block", CIDR: "203.0.113.0/24", Action: "deny",
	}}})
	if err != nil {
		t.Fatalf("newCIDRPolicyEngine: %v", err)
	}
	if got := engine.evaluate("203.0.113.10"); got.Matched {
		t.Fatalf("disabled CIDR policy unexpectedly matched: %#v", got)
	}
}
