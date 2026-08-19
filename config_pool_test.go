package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
)

func validTestConfig(t *testing.T) Config {
	t.Helper()
	rules := filepath.Join(t.TempDir(), "coraza.conf")
	if err := os.WriteFile(rules, []byte("SecRuleEngine Off\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	c := defaultConfig()
	c.Policies[0].RulesPath = rules
	return c
}

func TestMigrateConfigPreservesSitesAndAddsLegacyDefaults(t *testing.T) {
	c := Config{
		LegacyListen:     "127.0.0.1:9443",
		Rules:            "/legacy/coraza.conf",
		RequestBodyLimit: 12345,
		Sites:            []SiteConfig{{Name: "legacy"}},
	}
	migrateConfig(&c)
	if c.LegacyListen != "" {
		t.Fatalf("legacy listen was not cleared: %q", c.LegacyListen)
	}
	if len(c.Policies) != 1 || c.Policies[0].RulesPath != "/legacy/coraza.conf" || c.Policies[0].RequestBodyLimit != 12345 {
		t.Fatalf("unexpected migrated policy: %#v", c.Policies)
	}
	if c.Sites[0].Listen != "127.0.0.1:9443" || c.Sites[0].Policy != "default" {
		t.Fatalf("unexpected migrated site: %#v", c.Sites[0])
	}
}

func TestValidateRejectsBrokenReferencesAndManagedIPShape(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Config)
		want string
	}{
		{"unknown pool", func(c *Config) { c.Sites[0].Pool = "missing" }, "unknown pool"},
		{"unknown policy", func(c *Config) { c.Sites[0].Policy = "missing" }, "unknown policy"},
		{"managed wildcard", func(c *Config) { c.Sites[0].ManageIP = true }, "concrete IP"},
		{"interface without management", func(c *Config) { c.Sites[0].ManageInterface = "eth1" }, "requires manage_ip"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := validTestConfig(t)
			tc.edit(&c)
			if err := c.validate(); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("validate error = %v, want substring %q", err, tc.want)
			}
		})
	}
}

func TestValidateDraftAllowsIncompleteReferencesButRejectsUnsafeEnums(t *testing.T) {
	c := validTestConfig(t)
	c.Sites[0].Pool = "not-created-yet"
	c.Sites[0].Policy = "not-created-yet"
	if err := c.validateDraft(); err != nil {
		t.Fatalf("incomplete draft rejected: %v", err)
	}
	c.Pools[0].LBMethod = "injected\nvalue"
	if err := c.validateDraft(); err == nil {
		t.Fatal("unsafe lb_method accepted")
	}
}

func testMember(name string, weight int, healthy bool) *memberRuntime {
	m := &memberRuntime{node: name, weight: weight}
	if healthy {
		atomic.StoreInt32(&m.healthy, 1)
	}
	return m
}

func TestPoolWeightedRoundRobinAndHealthFiltering(t *testing.T) {
	a, b := testMember("a", 1, true), testMember("b", 2, true)
	p := &poolRuntime{method: "round_robin", members: []*memberRuntime{a, b}}
	var got []string
	for i := 0; i < 6; i++ {
		got = append(got, p.pick("").node)
	}
	if want := []string{"b", "b", "a", "b", "b", "a"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("weighted sequence = %#v, want %#v", got, want)
	}
	atomic.StoreInt32(&b.healthy, 0)
	for i := 0; i < 3; i++ {
		if picked := p.pick(""); picked != a {
			t.Fatalf("picked unhealthy member: %#v", picked)
		}
	}
}

func TestPoolLeastConnectionsIPHashAndFailOpen(t *testing.T) {
	a, b := testMember("a", 1, true), testMember("b", 1, true)
	atomic.StoreInt64(&a.active, 3)
	atomic.StoreInt64(&b.active, 1)
	p := &poolRuntime{method: "least_conn", members: []*memberRuntime{a, b}}
	if got := p.pick("192.0.2.1"); got != b {
		t.Fatalf("least_conn picked %q", got.node)
	}
	p.method = "ip_hash"
	first := p.pick("192.0.2.44")
	for i := 0; i < 10; i++ {
		if got := p.pick("192.0.2.44"); got != first {
			t.Fatal("ip_hash was not stable")
		}
	}
	atomic.StoreInt32(&a.healthy, 0)
	atomic.StoreInt32(&b.healthy, 0)
	if got := p.pick("192.0.2.44"); got == nil {
		t.Fatal("all-down pool did not fail open")
	}
}
