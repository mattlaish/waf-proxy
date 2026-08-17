// waf-proxy: a multi-site, load-balancing reverse proxy with an embedded
// Coraza WAF engine and a local admin console.
//
//	                          ┌ site blog (:443, blog.example.com) ─┐
//	client ─▶ listener :443 ──┤                                     ├─▶ pool ─▶ member (node:port)
//	          (SNI, WAF)       └ site shop (:443, shop.example.com) ┘         └▶ member (node:port)
//	client ─▶ listener :8443 ─ site api  (:8443, api.example.com) ──▶ pool ─▶ ...
//
// Objects (F5-style): node → member → pool → site.
//   node   = backend server address (reusable)
//   member = node + port + weight in a pool
//   pool   = members + LB method + health monitor
//   site   = listen address + hostnames + one pool
//
// Each site has its own WAF instance and its own listen address; sites that
// share an address are demultiplexed by Host (and SNI for TLS). Pools, nodes,
// members, LB method, monitors, engine modes, and certificates are all hot.
// Only the SET of listen addresses (and whether one is TLS) needs a restart.
package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/corazawaf/coraza/v3"
	txhttp "github.com/corazawaf/coraza/v3/http"
	"github.com/corazawaf/coraza/v3/types"
)

// ── configuration ───────────────────────────────────────────────────────

type NodeConfig struct {
	Name string `json:"name"`
	Host string `json:"host"` // IP or hostname, no port, no scheme
}

type MemberConfig struct {
	Node   string `json:"node"`   // references NodeConfig.Name
	Port   int    `json:"port"`   // backend port
	Weight int    `json:"weight"` // relative weight for round-robin (default 1)
}

type MonitorConfig struct {
	Type         string `json:"type"` // none | tcp | http
	Path         string `json:"path,omitempty"`
	IntervalSec  int    `json:"interval_sec,omitempty"`
	TimeoutSec   int    `json:"timeout_sec,omitempty"`
	ExpectStatus int    `json:"expect_status,omitempty"`
	Rise         int    `json:"rise,omitempty"`
	Fall         int    `json:"fall,omitempty"`
}

type PoolConfig struct {
	Name     string         `json:"name"`
	Scheme   string         `json:"scheme"`    // http | https (to the backend)
	LBMethod string         `json:"lb_method"` // round_robin | least_conn | ip_hash | random
	Monitor  MonitorConfig  `json:"monitor"`
	Members  []MemberConfig `json:"members"`
}

// PolicyExclusion removes a CRS rule (or a specific target of it), optionally
// scoped to a URL path prefix. This is exactly what the site-map policy
// learner emits when it recommends tuning for a page.
type PolicyExclusion struct {
	PathPrefix string   `json:"path_prefix,omitempty"` // "" or "/" = whole site
	RuleIDs    []int    `json:"rule_ids"`
	Targets    []string `json:"targets,omitempty"` // e.g. "ARGS:body_html"; if set, remove targets not whole rule
	Note       string   `json:"note,omitempty"`
}

// PolicyConfig is a named WAF ruleset + tuning that sites reference. Each site
// compiles its own engine from its policy (so per-site engine mode and correct
// per-site match attribution are preserved).
type PolicyConfig struct {
	Name             string            `json:"name"`
	RulesPath        string            `json:"rules_path"`         // SecLang file (usually includes CRS)
	ParanoiaLevel    int               `json:"paranoia_level"`     // 1-4, 0 = leave file default
	RequestBodyLimit int               `json:"request_body_limit"` // 0 = leave file default
	Exclusions       []PolicyExclusion `json:"exclusions,omitempty"`
}

// PageExcludeTarget removes a specific target of a rule (vs the whole rule).
type PageExcludeTarget struct {
	RuleID int    `json:"rule_id"`
	Target string `json:"target"`
}

// FieldPolicy validates one request field on one page. Profiles are deliberately
// allow-listed: the UI cannot inject raw SecLang or regular expressions.
type FieldPolicy struct {
	Name           string `json:"name"`
	Source         string `json:"source,omitempty"`  // ARGS_POST (default) | ARGS
	Profile        string `json:"profile,omitempty"` // identifier | password | free_text | email | numeric
	Required       bool   `json:"required,omitempty"`
	MinLength      int    `json:"min_length,omitempty"`
	MaxLength      int    `json:"max_length,omitempty"`
	ExcludeRuleIDs []int  `json:"exclude_rule_ids,omitempty"`
}

// PagePolicy is a URL-scoped ruleset bound to a path within a site. It compiles
// to path-gated SecLang inside the site's engine, so each URL effectively has
// its own policy without needing its own engine.
type PagePolicy struct {
	Path           string              `json:"path"`            // path to bind to
	Match          string              `json:"match,omitempty"` // prefix (default) | exact
	Action         string              `json:"action,omitempty"` // "" tune (default) | deny (virtual patch)
	DenyStatus     int                 `json:"deny_status,omitempty"` // 403 (default) or 404
	Mode           string              `json:"mode,omitempty"`  // "" inherit | Off | DetectionOnly | On
	ParanoiaLevel  int                 `json:"paranoia_level,omitempty"` // 0 = inherit
	ExcludeRuleIDs []int               `json:"exclude_rule_ids,omitempty"`
	ExcludeTargets []PageExcludeTarget `json:"exclude_targets,omitempty"`
	Methods        []string            `json:"methods,omitempty"` // defaults to POST when fields are present
	Fields         []FieldPolicy       `json:"fields,omitempty"`
	Note           string              `json:"note,omitempty"`
	Source         string              `json:"source,omitempty"` // learned | manual
}

type SiteConfig struct {
	Name      string   `json:"name"`
	Listen    string   `json:"listen"` // address THIS site binds, e.g. ":443"
	Hostnames []string `json:"hostnames"`
	Pool      string   `json:"pool"`   // references PoolConfig.Name
	Policy    string   `json:"policy"` // references PolicyConfig.Name (base ruleset)
	PreserveHost bool  `json:"preserve_host"`
	EngineMode   string `json:"engine_mode,omitempty"` // "" inherits global
	AIMode       string `json:"ai_mode,omitempty"`     // off | advisory | block ("" = off)
	TLSCert      string `json:"tls_cert,omitempty"`
	TLSKey       string `json:"tls_key,omitempty"`
	// ManageIP: when true and Listen has a concrete IP not on any interface,
	// the WAF assigns it to the matching NIC on Apply (and removes it when the
	// site goes away). Requires CAP_NET_ADMIN. Off for VIPs owned by keepalived.
	ManageIP bool `json:"manage_ip,omitempty"`
	// ManageInterface pins the listen IP to a specific data-plane NIC. When
	// empty, the legacy same-subnet auto-detection remains as a fallback.
	ManageInterface string `json:"manage_interface,omitempty"`
	// ManagePrefixLen is the CIDR prefix used when assigning the address.
	// Zero means: inherit the selected interface's IPv4/IPv6 prefix.
	ManagePrefixLen int `json:"manage_prefix_len,omitempty"`
	// PagePolicies are URL-scoped overrides managed from the Site Map.
	PagePolicies []PagePolicy `json:"page_policies,omitempty"`
}

type Config struct {
	Rules             string       `json:"rules"`
	EngineMode        string       `json:"engine_mode"`
	RequestBodyLimit  int          `json:"request_body_limit"`
	ReadTimeoutSec    int          `json:"read_timeout_sec"`
	IdleTimeoutSec    int          `json:"idle_timeout_sec"`
	BackendTimeoutSec int          `json:"backend_timeout_sec"`
	Nodes             []NodeConfig   `json:"nodes"`
	Pools             []PoolConfig   `json:"pools"`
	Policies          []PolicyConfig `json:"policies"`
	Sites             []SiteConfig   `json:"sites"`
	Profiles          []Profile      `json:"profiles,omitempty"` // custom page profiles (built-ins always available)
	Users             []UserConfig   `json:"users,omitempty"`
	AI                AIConfig       `json:"ai"`
	Notify            NotifyConfig   `json:"notify"`
	HA                HAConfig       `json:"ha"`
	Syslog            SyslogConfig   `json:"syslog"`

	// Legacy v2 fields, migrated on load.
	LegacyListen string `json:"listen,omitempty"`
}

// Build metadata, stamped by build.sh via -ldflags -X.
var (
	buildVersion = "dev"
	buildCommit  = "unknown"
)

func defaultConfig() Config {
	return Config{
		Rules:             "coraza.conf",
		EngineMode:        "DetectionOnly",
		ReadTimeoutSec:    15,
		IdleTimeoutSec:    60,
		BackendTimeoutSec: 30,
		Nodes:             []NodeConfig{{Name: "app1", Host: "127.0.0.1"}},
		Pools: []PoolConfig{{
			Name:     "default-pool",
			Scheme:   "http",
			LBMethod: "round_robin",
			Monitor:  MonitorConfig{Type: "tcp", IntervalSec: 5, TimeoutSec: 2, Rise: 2, Fall: 3},
			Members:  []MemberConfig{{Node: "app1", Port: 8080, Weight: 1}},
		}},
		Policies: []PolicyConfig{{
			Name:          "default",
			RulesPath:     "coraza.conf",
			ParanoiaLevel: 1,
		}},
		Sites: []SiteConfig{{
			Name:         "default",
			Listen:       ":8443",
			Hostnames:    []string{"*"},
			Pool:         "default-pool",
			Policy:       "default",
			PreserveHost: true,
		}},
		AI: defaultAIConfig(),
		Notify: defaultNotifyConfig(),
		HA:     defaultHAConfig(),
		Syslog: defaultSyslogConfig(),
	}
}

func validEngineMode(m string, allowEmpty bool) bool {
	switch m {
	case "On", "DetectionOnly", "Off":
		return true
	case "":
		return allowEmpty
	}
	return false
}

func validLBMethod(m string) bool {
	switch m {
	case "round_robin", "least_conn", "ip_hash", "random":
		return true
	}
	return false
}

func validMonitorType(t string) bool {
	switch t {
	case "", "none", "tcp", "http", "https":
		return true
	}
	return false
}

func validFieldSource(v string) bool {
	return v == "" || v == "ARGS_POST" || v == "ARGS"
}

func validFieldProfile(v string) bool {
	switch v {
	case "", "identifier", "password", "free_text", "email", "numeric":
		return true
	}
	return false
}

func validPolicyMethod(v string) bool {
	switch strings.ToUpper(v) {
	case "GET", "POST", "PUT", "PATCH", "DELETE":
		return true
	}
	return false
}

func validFieldName(v string) bool {
	if v == "" || len(v) > 128 {
		return false
	}
	for _, r := range v {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || strings.ContainsRune("_.-", r) {
			continue
		}
		return false
	}
	return true
}

// validate is also the injection barrier between the admin API and SecLang:
// engine modes are interpolated into directives, so they are allow-listed.
// validateDraft is the light check for a DRAFT save (work-in-progress). It
// verifies each entity is internally well-formed — names present, no duplicate
// names, valid enum values, sane numbers — but deliberately does NOT enforce
// cross-references (member->node, site->pool/policy), the "at least one policy"
// rule, or that rules files exist on disk. That's what lets you build a config
// incrementally and save at every step. Full validate() still gates apply.
func (c Config) validateDraft() error {
	if c.EngineMode != "" && !validEngineMode(c.EngineMode, false) {
		return fmt.Errorf("engine_mode must be On, DetectionOnly, or Off (got %q)", c.EngineMode)
	}
	seen := func(kind string) func(string) error {
		m := map[string]bool{}
		return func(name string) error {
			if name == "" {
				return nil // empty names are allowed mid-draft
			}
			if m[name] {
				return fmt.Errorf("duplicate %s name %q", kind, name)
			}
			m[name] = true
			return nil
		}
	}
	np, npool, nnode := seen("policy"), seen("pool"), seen("node")
	for _, p := range c.Policies {
		if err := np(p.Name); err != nil {
			return err
		}
		if p.ParanoiaLevel < 0 || p.ParanoiaLevel > 4 {
			return fmt.Errorf("policy %q: paranoia_level must be 0-4", p.Name)
		}
	}
	for _, n := range c.Nodes {
		if err := nnode(n.Name); err != nil {
			return err
		}
		if n.Host != "" && (strings.Contains(n.Host, "/") || strings.Contains(n.Host, ":")) {
			return fmt.Errorf("node %q: host must be a bare IP or hostname (no scheme, no port)", n.Name)
		}
	}
	for _, p := range c.Pools {
		if err := npool(p.Name); err != nil {
			return err
		}
		if p.Scheme != "" && p.Scheme != "http" && p.Scheme != "https" {
			return fmt.Errorf("pool %q: scheme must be http or https", p.Name)
		}
		if p.LBMethod != "" && !validLBMethod(p.LBMethod) {
			return fmt.Errorf("pool %q: invalid lb_method", p.Name)
		}
		for j, m := range p.Members {
			if m.Port != 0 && (m.Port < 1 || m.Port > 65535) {
				return fmt.Errorf("pool %q member %d: port must be 1-65535", p.Name, j+1)
			}
		}
	}
	nsite := seen("site")
	for _, s := range c.Sites {
		if err := nsite(s.Name); err != nil {
			return err
		}
		if s.ManagePrefixLen < 0 || s.ManagePrefixLen > 128 {
			return fmt.Errorf("site %q: manage_prefix_len must be 0-128", s.Name)
		}
	}
	return nil
}

func (c Config) validate() error {
	if !validEngineMode(c.EngineMode, false) {
		return fmt.Errorf("engine_mode must be On, DetectionOnly, or Off (got %q)", c.EngineMode)
	}
	if c.ReadTimeoutSec < 1 || c.IdleTimeoutSec < 1 || c.BackendTimeoutSec < 1 {
		return errors.New("timeouts must be >= 1 second")
	}
	if c.RequestBodyLimit < 0 {
		return errors.New("request_body_limit must be >= 0")
	}

	// policies
	if len(c.Policies) == 0 {
		return errors.New("at least one policy is required")
	}
	policySet := map[string]bool{}
	for i, p := range c.Policies {
		where := fmt.Sprintf("policy %d (%q)", i+1, p.Name)
		if strings.TrimSpace(p.Name) == "" {
			return fmt.Errorf("policy %d: name is required", i+1)
		}
		if policySet[p.Name] {
			return fmt.Errorf("duplicate policy name %q", p.Name)
		}
		policySet[p.Name] = true
		if p.RulesPath == "" {
			return fmt.Errorf("%s: rules_path is required", where)
		}
		if _, err := os.Stat(p.RulesPath); err != nil {
			return fmt.Errorf("%s: rules_path: %w", where, err)
		}
		if p.ParanoiaLevel < 0 || p.ParanoiaLevel > 4 {
			return fmt.Errorf("%s: paranoia_level must be 0-4", where)
		}
		if p.RequestBodyLimit < 0 {
			return fmt.Errorf("%s: request_body_limit must be >= 0", where)
		}
		for j, ex := range p.Exclusions {
			for _, id := range ex.RuleIDs {
				if id <= 0 {
					return fmt.Errorf("%s exclusion %d: rule ids must be positive", where, j+1)
				}
			}
		}
	}

	// nodes
	nodeSet := map[string]bool{}
	for i, n := range c.Nodes {
		if strings.TrimSpace(n.Name) == "" {
			return fmt.Errorf("node %d: name is required", i+1)
		}
		if nodeSet[n.Name] {
			return fmt.Errorf("duplicate node name %q", n.Name)
		}
		if strings.TrimSpace(n.Host) == "" {
			return fmt.Errorf("node %q: host is required", n.Name)
		}
		if strings.Contains(n.Host, "/") || strings.Contains(n.Host, ":") {
			return fmt.Errorf("node %q: host must be a bare IP or hostname (no scheme, no port)", n.Name)
		}
		nodeSet[n.Name] = true
	}

	// pools
	poolSet := map[string]bool{}
	for i, p := range c.Pools {
		where := fmt.Sprintf("pool %d (%q)", i+1, p.Name)
		if strings.TrimSpace(p.Name) == "" {
			return fmt.Errorf("pool %d: name is required", i+1)
		}
		if poolSet[p.Name] {
			return fmt.Errorf("duplicate pool name %q", p.Name)
		}
		poolSet[p.Name] = true
		if p.Scheme != "http" && p.Scheme != "https" {
			return fmt.Errorf("%s: scheme must be http or https", where)
		}
		if !validLBMethod(p.LBMethod) {
			return fmt.Errorf("%s: lb_method must be round_robin, least_conn, ip_hash, or random", where)
		}
		if !validMonitorType(p.Monitor.Type) {
			return fmt.Errorf("%s: monitor type must be none, tcp, or http", where)
		}
		if len(p.Members) == 0 {
			return fmt.Errorf("%s: at least one member is required", where)
		}
		for j, m := range p.Members {
			if strings.TrimSpace(m.Node) == "" {
				return fmt.Errorf("%s member %d: no node selected — pick a node", where, j+1)
			}
			if !nodeSet[m.Node] {
				return fmt.Errorf("%s member %d: unknown node %q (not in your nodes list)", where, j+1, m.Node)
			}
			if m.Port < 1 || m.Port > 65535 {
				return fmt.Errorf("%s member %d: port must be 1-65535", where, j+1)
			}
		}
	}

	// sites
	if len(c.Sites) == 0 {
		return errors.New("at least one site is required")
	}
	// hostname uniqueness is PER LISTEN ADDRESS (same name can live on two ports)
	claimed := map[string]map[string]string{} // listen -> host -> site
	for i, s := range c.Sites {
		where := fmt.Sprintf("site %d (%q)", i+1, s.Name)
		if strings.TrimSpace(s.Name) == "" {
			return fmt.Errorf("site %d: name is required", i+1)
		}
		if strings.TrimSpace(s.Listen) == "" {
			return fmt.Errorf("%s: listen address is required (e.g. :443)", where)
		}
		if len(s.Hostnames) == 0 {
			return fmt.Errorf("%s: at least one hostname is required", where)
		}
		if strings.TrimSpace(s.Pool) == "" {
			return fmt.Errorf("%s: no pool selected — pick a pool", where)
		}
		if !poolSet[s.Pool] {
			return fmt.Errorf("%s: unknown pool %q (not in your pools list)", where, s.Pool)
		}
		if strings.TrimSpace(s.Policy) == "" {
			return fmt.Errorf("%s: no policy selected — pick a policy", where)
		}
		if !policySet[s.Policy] {
			return fmt.Errorf("%s: unknown policy %q (not in your policies list)", where, s.Policy)
		}
		if !validEngineMode(s.EngineMode, true) {
			return fmt.Errorf("%s: engine_mode must be On, DetectionOnly, Off, or empty to inherit", where)
		}
		if !validAIMode(s.AIMode) {
			return fmt.Errorf("%s: ai_mode must be off, advisory, block, or empty", where)
		}
		if s.ManageInterface != "" && !s.ManageIP {
			return fmt.Errorf("%s: manage_interface requires manage_ip", where)
		}
		if s.ManagePrefixLen != 0 && !s.ManageIP {
			return fmt.Errorf("%s: manage_prefix_len requires manage_ip", where)
		}
		if s.ManageIP {
			ip := net.ParseIP(listenIP(s.Listen))
			if ip == nil {
				return fmt.Errorf("%s: manage_ip requires a concrete IP listen address", where)
			}
			maxPrefix := 128
			if ip.To4() != nil {
				maxPrefix = 32
			}
			if s.ManagePrefixLen < 0 || s.ManagePrefixLen > maxPrefix {
				return fmt.Errorf("%s: manage_prefix_len must be 0-%d for this address", where, maxPrefix)
			}
		}
		for j, pp := range s.PagePolicies {
			pw := fmt.Sprintf("%s page policy %d", where, j+1)
			if strings.TrimSpace(pp.Path) == "" {
				return fmt.Errorf("%s: path is required", pw)
			}
			if pp.Match != "" && pp.Match != "prefix" && pp.Match != "exact" {
				return fmt.Errorf("%s: match must be prefix or exact", pw)
			}
			if pp.Action != "" && pp.Action != "deny" {
				return fmt.Errorf("%s: action must be empty (tune) or deny", pw)
			}
			if pp.DenyStatus != 0 && pp.DenyStatus != 403 && pp.DenyStatus != 404 {
				return fmt.Errorf("%s: deny_status must be 403 or 404", pw)
			}
			if !validEngineMode(pp.Mode, true) {
				return fmt.Errorf("%s: mode must be On, DetectionOnly, Off, or empty", pw)
			}
			if pp.ParanoiaLevel < 0 || pp.ParanoiaLevel > 4 {
				return fmt.Errorf("%s: paranoia_level must be 0-4", pw)
			}
			for _, method := range pp.Methods {
				if !validPolicyMethod(method) {
					return fmt.Errorf("%s: unsupported method %q", pw, method)
				}
			}
			fieldNames := map[string]bool{}
			for k, fp := range pp.Fields {
				fw := fmt.Sprintf("%s field %d", pw, k+1)
				if !validFieldName(fp.Name) {
					return fmt.Errorf("%s: name must use only letters, digits, dot, underscore, or hyphen", fw)
				}
				key := fp.Source + ":" + fp.Name
				if fieldNames[key] {
					return fmt.Errorf("%s: duplicate field %q", pw, fp.Name)
				}
				fieldNames[key] = true
				if !validFieldSource(fp.Source) {
					return fmt.Errorf("%s: source must be ARGS_POST or ARGS", fw)
				}
				if !validFieldProfile(fp.Profile) {
					return fmt.Errorf("%s: unsupported profile %q", fw, fp.Profile)
				}
				if fp.MinLength < 0 || fp.MaxLength < 0 || fp.MaxLength > 1048576 {
					return fmt.Errorf("%s: lengths must be between 0 and 1048576", fw)
				}
				if fp.MaxLength > 0 && fp.MinLength > fp.MaxLength {
					return fmt.Errorf("%s: min_length cannot exceed max_length", fw)
				}
				for _, rid := range fp.ExcludeRuleIDs {
					if rid <= 0 {
						return fmt.Errorf("%s: exclude rule ids must be positive", fw)
					}
				}
			}
			for _, id := range pp.ExcludeRuleIDs {
				if id <= 0 {
					return fmt.Errorf("%s: exclude rule ids must be positive", pw)
				}
			}
		}
		if (s.TLSCert == "") != (s.TLSKey == "") {
			return fmt.Errorf("%s: tls_cert and tls_key must both be set or both be empty", where)
		}
		if claimed[s.Listen] == nil {
			claimed[s.Listen] = map[string]string{}
		}
		for _, h := range s.Hostnames {
			key := normalizeHost(h)
			if key == "" {
				return fmt.Errorf("%s: empty hostname", where)
			}
			if owner, dup := claimed[s.Listen][key]; dup {
				return fmt.Errorf("%s: hostname %q on %s already claimed by site %q", where, h, s.Listen, owner)
			}
			claimed[s.Listen][key] = s.Name
		}
	}
	if err := c.AI.validate(); err != nil {
		return err
	}
	if err := c.HA.validate(); err != nil {
		return err
	}
	if err := c.validateUsers(); err != nil {
		return err
	}
	if err := c.Syslog.validate(); err != nil {
		return err
	}
	return nil
}

func loadConfig(path string) (Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	c := defaultConfig()
	c.Nodes, c.Pools, c.Policies, c.Sites = nil, nil, nil, nil
	if err := json.Unmarshal(b, &c); err != nil {
		return Config{}, err
	}
	migrateConfig(&c)
	return c, nil
}

// migrateConfig upgrades older single-backend / global-listen configs into
// the node/pool/site model so existing files keep working.
func migrateConfig(c *Config) {
	// Synthesize a default policy from the legacy top-level rules/body-limit,
	// and point any policy-less site at it.
	if len(c.Policies) == 0 {
		rules := firstNonEmpty(c.Rules, "coraza.conf")
		c.Policies = []PolicyConfig{{
			Name:             "default",
			RulesPath:        rules,
			ParanoiaLevel:    1,
			RequestBodyLimit: c.RequestBodyLimit,
		}}
	}
	if len(c.Sites) > 0 {
		for i := range c.Sites {
			if c.Sites[i].Listen == "" {
				c.Sites[i].Listen = firstNonEmpty(c.LegacyListen, ":8443")
			}
			if c.Sites[i].Policy == "" {
				c.Sites[i].Policy = c.Policies[0].Name
			}
		}
	}
	c.LegacyListen = ""
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func saveConfig(path string, c Config) error {
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func normalizeHost(h string) string {
	h = strings.ToLower(strings.TrimSpace(h))
	if host, _, err := net.SplitHostPort(h); err == nil {
		h = host
	}
	return strings.TrimSuffix(h, ".")
}

// listenerSet returns the distinct listen addresses and whether each is TLS
// (any site on it has a certificate).
func listenerSet(c Config) map[string]bool {
	out := map[string]bool{}
	for _, s := range c.Sites {
		if _, ok := out[s.Listen]; !ok {
			out[s.Listen] = false
		}
		if s.TLSCert != "" {
			out[s.Listen] = true
		}
	}
	return out
}

// ── hot-swappable runtime ───────────────────────────────────────────────

type siteRuntime struct {
	handler http.Handler
	cfg     SiteConfig
	cert    *tls.Certificate
	mode    string
}

type listenerRuntime struct {
	exact       map[string]*siteRuntime
	wildcard    map[string]*siteRuntime // ".example.com"
	catchAll    *siteRuntime
	defaultCert *tls.Certificate
	tls         bool
}

func (lr *listenerRuntime) lookup(host string) *siteRuntime {
	host = normalizeHost(host)
	if s, ok := lr.exact[host]; ok {
		return s
	}
	rest := host
	for {
		i := strings.IndexByte(rest, '.')
		if i < 0 {
			break
		}
		if s, ok := lr.wildcard[rest[i:]]; ok {
			return s
		}
		rest = rest[i+1:]
	}
	return lr.catchAll
}

type runtimeState struct {
	listeners map[string]*listenerRuntime
	pools     map[string]*poolRuntime
	cfg       Config
	builtAt   time.Time
	cancel    context.CancelFunc // stops this runtime's health monitors
}

type server struct {
	rt            atomic.Pointer[runtimeState]
	matches       *matchRing
	access        *accessRing
	maps          *siteMaps
	signals       *signalStore
	ai            *aiEngine
	learn         *learnStore
	notify        *notifier
	ha            *haEngine
	syslog        *syslogEngine
	hosts         *hostObserver
	metrics       *metrics
	ipmgr         *ipManager
	listenMgr     *listenerManager
	draining      atomic.Bool
	log           *slog.Logger
	configPath    string
	tlsBrowseRoot string
	bootCfg       Config
}

// restartPending is retained for API compatibility but is always false now:
// data-plane listeners are opened/closed live on apply (see listenerManager),
// so changing a listen address or flipping HTTP<->TLS no longer needs a restart.
func (s *server) restartPending(cfg Config) bool {
	return false
}

func (s *server) apply(cfg Config) error {
	return s.applyEx(cfg, false)
}

// applyEx applies config; when fromSync is true the change originated from the
// HA peer, so we don't push it back (loop guard).
func (s *server) applyEx(cfg Config, fromSync bool) error {
	if err := cfg.validate(); err != nil {
		return err
	}
	rt, err := s.buildRuntime(cfg)
	if err != nil {
		return err
	}
	// Network ownership must succeed before the runtime becomes live. Otherwise
	// the API could report Apply failure after already swapping the config.
	if s.ipmgr != nil {
		if err := s.ipmgr.reconcile(cfg); err != nil {
			if rt.cancel != nil {
				rt.cancel()
			}
			return err
		}
	}
	old := s.rt.Swap(rt)
	if old != nil && old.cancel != nil {
		old.cancel() // stop the previous runtime's monitors
	}
	s.ai.configure(cfg.AI)
	s.notify.configure(cfg.Notify)
	s.ha.configure(cfg.HA)
	s.syslog.configure(cfg.Syslog)
	if s.listenMgr != nil {
		s.listenMgr.reconcile(cfg) // open/close data-plane sockets live — no restart
	}
	if !fromSync {
		s.ha.pushConfig(cfg) // propagate local changes to the peer
	}
	return nil
}

func (s *server) buildRuntime(cfg Config) (*runtimeState, error) {
	ctx, cancel := context.WithCancel(context.Background())
	rt := &runtimeState{
		listeners: map[string]*listenerRuntime{},
		pools:     map[string]*poolRuntime{},
		cfg:       cfg,
		builtAt:   time.Now(),
		cancel:    cancel,
	}

	nodeHost := map[string]string{}
	for _, n := range cfg.Nodes {
		nodeHost[n.Name] = n.Host
	}

	// Build pools and start their monitors.
	for _, pc := range cfg.Pools {
		pr := &poolRuntime{name: pc.Name, method: pc.LBMethod, monitor: pc.Monitor}
		for _, mc := range pc.Members {
			w := mc.Weight
			if w < 1 {
				w = 1
			}
			target := &url.URL{
				Scheme: pc.Scheme,
				Host:   net.JoinHostPort(nodeHost[mc.Node], fmt.Sprintf("%d", mc.Port)),
			}
			pr.members = append(pr.members, &memberRuntime{node: mc.Node, target: target, weight: w})
		}
		pr.startMonitor(ctx, s.log, s.notify)
		rt.pools[pc.Name] = pr
	}

	// Index policies for lookup.
	policyByName := map[string]PolicyConfig{}
	for _, p := range cfg.Policies {
		policyByName[p.Name] = p
	}

	// Build sites onto their listeners.
	for _, sc := range cfg.Sites {
		mode := sc.EngineMode
		if mode == "" {
			mode = cfg.EngineMode
		}
		policy := policyByName[sc.Policy]
		waf, err := s.buildWAF(policy, sc.PagePolicies, mode, sc.AIMode, sc.Name)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("site %q: %w", sc.Name, err)
		}
		pool := rt.pools[sc.Pool]
		siteName := sc.Name
		record := func(method, path, rawQuery, contentType string, code int) {
			s.maps.record(siteName, method, path, code, srcObserved)
			s.learn.noteRequest(siteName, path, code)
			s.signals.noteRequestShape(siteName, path, method, rawQuery, contentType)
		}
		sr := &siteRuntime{
			handler: s.logWrap(sc.Name, s.ai.wrap(sc, txhttp.WrapHandler(waf, buildProxy(pool, sc, cfg, s.log, record)))),
			cfg:     sc,
			mode:    mode,
		}
		if sc.TLSCert != "" {
			cert, err := tls.LoadX509KeyPair(sc.TLSCert, sc.TLSKey)
			if err != nil {
				cancel()
				return nil, fmt.Errorf("site %q: tls: %w", sc.Name, err)
			}
			sr.cert = &cert
		}

		lr := rt.listeners[sc.Listen]
		if lr == nil {
			lr = &listenerRuntime{exact: map[string]*siteRuntime{}, wildcard: map[string]*siteRuntime{}}
			rt.listeners[sc.Listen] = lr
		}
		if sr.cert != nil {
			lr.tls = true
			if lr.defaultCert == nil {
				lr.defaultCert = sr.cert
			}
		}
		for _, h := range sc.Hostnames {
			key := normalizeHost(h)
			switch {
			case key == "*":
				lr.catchAll = sr
			case strings.HasPrefix(key, "*."):
				lr.wildcard[key[1:]] = sr
			default:
				lr.exact[key] = sr
			}
		}
	}
	return rt, nil
}

// logWrap is the outermost handler layer: it records every request to the
// access log with the final status (after WAF/AI decisions) and client IP.
// It reuses statusRecorder (defined in ai.go) to capture the status code.
// adminPlaneCheck enforces management/data-plane separation at startup. It
// returns a fatal error when the admin console would be reachable somewhere it
// must not be — fail-closed, because a mistake here exposes full control on the
// traffic-facing NIC. Rules:
//   - admin must not bind the same IP as any data-plane site listener
//   - admin bound off-loopback must have TLS
// Loopback admin (the default) is always allowed.
func adminPlaneCheck(adminAddr string, tls bool, sites []SiteConfig) error {
	aHost, aPort, err := net.SplitHostPort(adminAddr)
	if err != nil {
		return fmt.Errorf("admin address %q is not host:port: %w", adminAddr, err)
	}
	aHost = strings.TrimSpace(aHost)
	aIP := net.ParseIP(aHost)
	allIfaces := aHost == "" || aHost == "0.0.0.0" || aHost == "::" || aHost == "*"
	loopback := aIP != nil && aIP.IsLoopback()

	// Off-loopback (or all-interfaces) requires TLS.
	if (allIfaces || (aIP != nil && !loopback)) && !tls {
		return fmt.Errorf("admin console bound off-loopback (%s) without TLS: set -admin-cert/-admin-key (WAF_ADMIN_TLS_CERT/KEY)", adminAddr)
	}

	// Never share an IP:port or IP with a data-plane site.
	for _, s := range sites {
		dHost, dPort, err := net.SplitHostPort(s.Listen)
		if err != nil {
			continue
		}
		dHost = strings.TrimSpace(dHost)
		// All-interfaces admin would blanket every data NIC — forbidden.
		if allIfaces {
			return fmt.Errorf("admin console on all interfaces (%s) would also serve the data-plane NIC; bind it to the management IP instead", adminAddr)
		}
		// Same explicit IP as a data listener = admin riding the data NIC.
		if aIP != nil && dHost != "" && dHost != "0.0.0.0" && dHost != "::" {
			if dIP := net.ParseIP(dHost); dIP != nil && dIP.Equal(aIP) {
				return fmt.Errorf("admin console IP %s is also a data-plane listener (%s); the data NIC must not serve the console — use the management IP", aHost, s.Listen)
			}
		}
		// A data-plane site on all-interfaces will occupy the mgmt IP's port too.
		if (dHost == "" || dHost == "0.0.0.0" || dHost == "::") && dPort == aPort {
			return fmt.Errorf("data-plane site listens on all interfaces at port %s, which collides with the admin console; give sites explicit data-NIC IPs", aPort)
		}
	}
	return nil
}

// envOr returns the environment variable value or a fallback.
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// adminExposed reports whether the admin bind address is reachable off-loopback,
// so startup can warn about network exposure.
func adminExposed(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	host = strings.TrimSpace(host)
	if host == "" || host == "0.0.0.0" || host == "::" || host == "*" {
		return true // all-interfaces
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false // a hostname; don't cry wolf
	}
	return !ip.IsLoopback()
}

// drainDelay is how long to advertise unhealthy before closing listeners, so
// upstream failover (keepalived/VRRP, LB, bypass NIC) can react first.
// Configurable via WAF_DRAIN_SECONDS (default 3s, 0 to disable).
func drainDelay() time.Duration {
	if v := os.Getenv("WAF_DRAIN_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			return time.Duration(n) * time.Second
		}
	}
	return 3 * time.Second
}

func (s *server) logWrap(siteName string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sw := &statusRecorder{ResponseWriter: w, code: 200}
		next.ServeHTTP(sw, r)
		// metrics: count the request, approximate in/out bytes
		s.metrics.addReq()
		if r.ContentLength > 0 {
			s.metrics.addIn(r.ContentLength)
		}
		s.metrics.addOut(sw.nbytes)
		s.access.add(accessRec{
			Time:   time.Now().Format("15:04:05"),
			Site:   siteName,
			Client: clientIP(r),
			Method: r.Method,
			Path:   r.URL.Path,
			Status: sw.code,
		})
		s.syslog.forwardAccess(accessRec{Site: siteName, Client: clientIP(r), Method: r.Method, Path: r.URL.Path, Status: sw.code})
	})
}

func (s *server) buildWAF(policy PolicyConfig, pages []PagePolicy, mode, aiMode, siteName string) (coraza.WAF, error) {
	// Load directives from an absolute path (e.g. /etc/waf/coraza.conf) using
	// Coraza's default OS filesystem, which resolves absolute paths and their
	// absolute Include lines (the CRS files). Do NOT set WithRootFS(os.DirFS("/")):
	// os.DirFS follows io/fs rules that reject any leading slash as
	// "invalid argument", which breaks both this file and its CRS includes.
	wc := coraza.NewWAFConfig().
		WithDirectivesFromFile(policy.RulesPath).
		WithErrorCallback(func(rule types.MatchedRule) {
			rec := matchRec{
				Time:     time.Now().Format("15:04:05"),
				Site:     siteName,
				RuleID:   rule.Rule().ID(),
				Severity: rule.Rule().Severity().String(),
				Phase:    int(rule.Rule().Phase()),
				Client:   rule.ClientIPAddress(),
				URI:      rule.URI(),
				Msg:      rule.Message(),
				Data:     rule.Data(),
			}
			s.matches.add(rec)
			s.learn.noteMatch(siteName, rec.URI, rec.RuleID, rec.Client, rec.Severity)
			s.syslog.forwardMatch(rec)
			s.log.Warn("rule match",
				"site", rec.Site, "rule_id", rec.RuleID, "severity", rec.Severity,
				"phase", rec.Phase, "client", rec.Client, "uri", rec.URI,
				"msg", rec.Msg, "data", rec.Data)
			// Join the match to the live request so DetectionOnly analysis gets
			// method/query/headers/body as well as the rule and matched value.
			s.ai.enqueueMatch(siteName, aiMode, rec.Client, rec.URI, rec.RuleID, rec.Data)
		})

	wc = wc.WithDirectives(policyDirectives(policy, mode) + pagePolicyDirectives(pages))
	return coraza.NewWAF(wc)
}

// pagePolicyDirectives compiles URL-scoped page policies into path-gated
// SecLang, appended after the base policy so page rules win. Each page emits
// one phase-1 SecRule matching REQUEST_FILENAME that can flip engine mode
// (ctl:ruleEngine), remove rules/targets (ctl:ruleRemove*), and set paranoia.
//
// Reliability note: ctl:ruleEngine and ctl:ruleRemoveById take effect for the
// whole transaction from phase 1, so mode + exclusions are exact. Paranoia is
// set late (after CRS's own phase-1 init) so it reliably affects phase-2
// (body) rules; a handful of phase-1 rules may already have evaluated. For
// strict phase-1 paranoia, set it on the base policy/crs-setup.
func pagePolicyDirectives(pages []PagePolicy) string {
	var b strings.Builder
	id := 9100000
	for _, pp := range pages {
		op := "@beginsWith"
		if pp.Match == "exact" {
			op = "@streq"
		}
		path := sanitizePrefix(pp.Path)

		if pp.Action == "deny" {
			status := pp.DenyStatus
			if status != 404 {
				status = 403
			}
			// Two rules: force the engine On for this transaction first, so the
			// virtual patch blocks even when the site runs DetectionOnly; then
			// deny with the chosen status. Logged so it shows in the match feed.
			fmt.Fprintf(&b, "SecRule REQUEST_FILENAME \"%s %s\" \"id:%d,phase:1,pass,nolog,ctl:ruleEngine=On\"\n",
				op, path, id)
			id++
			fmt.Fprintf(&b, "SecRule REQUEST_FILENAME \"%s %s\" \"id:%d,phase:1,deny,status:%d,log,msg:'virtual patch: blocked path %s'\"\n",
				op, path, id, status, path)
			id++
			continue
		}

		var acts strings.Builder
		if pp.Mode != "" {
			fmt.Fprintf(&acts, ",ctl:ruleEngine=%s", pp.Mode) // On|Off|DetectionOnly
		}
		for _, rid := range sanitizeIDs(pp.ExcludeRuleIDs) {
			fmt.Fprintf(&acts, ",ctl:ruleRemoveById=%d", rid)
		}
		for _, et := range pp.ExcludeTargets {
			if et.RuleID <= 0 {
				continue
			}
			t := sanitizeTarget(et.Target)
			if t == "" {
				continue
			}
			fmt.Fprintf(&acts, ",ctl:ruleRemoveTargetById=%d;%s", et.RuleID, t)
		}
		for _, fp := range pp.Fields {
			target := fieldTarget(fp)
			for _, rid := range sanitizeIDs(fp.ExcludeRuleIDs) {
				fmt.Fprintf(&acts, ",ctl:ruleRemoveTargetById=%d;%s", rid, target)
			}
		}
		if pp.ParanoiaLevel >= 1 && pp.ParanoiaLevel <= 4 {
			fmt.Fprintf(&acts, ",setvar:tx.paranoia_level=%d,setvar:tx.blocking_paranoia_level=%d,setvar:tx.detection_paranoia_level=%d",
				pp.ParanoiaLevel, pp.ParanoiaLevel, pp.ParanoiaLevel)
		}
		if acts.Len() == 0 && len(pp.Fields) == 0 {
			continue // nothing to do for this page
		}
		if acts.Len() > 0 {
			fmt.Fprintf(&b, "SecRule REQUEST_FILENAME \"%s %s\" \"id:%d,phase:1,pass,nolog%s\"\n",
				op, path, id, acts.String())
			id++
		}

		// Field validation runs in phase 2, after URL-encoded, multipart, and
		// supported structured request bodies have been parsed into ARGS.
		for _, fp := range pp.Fields {
			methods := policyMethods(pp)
			variable := fieldTarget(fp)
			methodRX := "^(?:" + strings.Join(methods, "|") + ")$"
			if fp.Required {
				fmt.Fprintf(&b, "SecRule REQUEST_FILENAME \"%s %s\" \"id:%d,phase:2,deny,status:403,log,msg:'required field missing: %s',chain\"\n",
					op, path, id, fp.Name)
				fmt.Fprintf(&b, " SecRule REQUEST_METHOD \"@rx %s\" \"chain\"\n", methodRX)
				fmt.Fprintf(&b, "  SecRule &%s \"@eq 0\"\n", variable)
				id++
			}
			for _, pattern := range fieldPatterns(fp) {
				fmt.Fprintf(&b, "SecRule REQUEST_FILENAME \"%s %s\" \"id:%d,phase:2,deny,status:403,log,msg:'field policy violation: %s',chain\"\n",
					op, path, id, fp.Name)
				fmt.Fprintf(&b, " SecRule REQUEST_METHOD \"@rx %s\" \"chain\"\n", methodRX)
				fmt.Fprintf(&b, "  SecRule %s \"!@rx %s\"\n", variable, pattern)
				id++
			}
		}
	}
	return b.String()
}

func fieldTarget(fp FieldPolicy) string {
	source := fp.Source
	if source == "" {
		source = "ARGS_POST"
	}
	return source + ":" + fp.Name
}

func policyMethods(pp PagePolicy) []string {
	if len(pp.Methods) == 0 {
		return []string{"POST"}
	}
	seen := map[string]bool{}
	var out []string
	for _, method := range pp.Methods {
		method = strings.ToUpper(method)
		if validPolicyMethod(method) && !seen[method] {
			seen[method] = true
			out = append(out, method)
		}
	}
	return out
}

func fieldPatterns(fp FieldPolicy) []string {
	var patterns []string
	minLen := fp.MinLength
	if fp.Required && minLen < 1 {
		minLen = 1
	}
	maxLen := fp.MaxLength
	if maxLen == 0 {
		maxLen = 1048576
	}
	if fp.Required || fp.MinLength > 0 || fp.MaxLength > 0 {
		patterns = append(patterns, "(?s)^."+fmt.Sprintf("{%d,%d}", minLen, maxLen)+"$")
	}
	switch fp.Profile {
	case "identifier":
		patterns = append(patterns, "^[A-Za-z0-9._-]*$")
	case "numeric":
		patterns = append(patterns, "^[0-9]*$")
	case "email":
		// Kept intentionally practical rather than pretending to implement all
		// of RFC 5322. Applications remain responsible for canonical validation.
		patterns = append(patterns, "^[^@[:space:]]+@[^@[:space:]]+\\.[^@[:space:]]+$")
	}
	return patterns
}

// policyDirectives builds the SecLang overrides appended AFTER the policy's
// rules file (last directive wins): engine mode, paranoia level, body limit,
// and path-scoped exclusions. Values are allow-listed / integer, never raw
// user strings, so this is not a directive-injection surface.
func policyDirectives(p PolicyConfig, mode string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "SecRuleEngine %s\n", mode)
	if p.RequestBodyLimit > 0 {
		fmt.Fprintf(&b, "SecRequestBodyLimit %d\n", p.RequestBodyLimit)
	}
	if p.ParanoiaLevel >= 1 && p.ParanoiaLevel <= 4 {
		// CRS 4 reads these tx vars; setting them after crs-setup overrides it.
		fmt.Fprintf(&b, "SecAction \"id:9000001,phase:1,pass,nolog,"+
			"setvar:tx.paranoia_level=%d,setvar:tx.blocking_paranoia_level=%d,setvar:tx.detection_paranoia_level=%d\"\n",
			p.ParanoiaLevel, p.ParanoiaLevel, p.ParanoiaLevel)
	}
	id := 9000100
	for _, ex := range p.Exclusions {
		ids := sanitizeIDs(ex.RuleIDs)
		if len(ids) == 0 {
			continue
		}
		prefix := ex.PathPrefix
		global := prefix == "" || prefix == "/"
		if len(ex.Targets) > 0 {
			// remove specific targets from each rule
			for _, id2 := range ids {
				for _, t := range ex.Targets {
					t = sanitizeTarget(t)
					if t == "" {
						continue
					}
					if global {
						fmt.Fprintf(&b, "SecRuleUpdateTargetById %d \"!%s\"\n", id2, t)
					} else {
						fmt.Fprintf(&b, "SecRule REQUEST_URI \"@beginsWith %s\" \"id:%d,phase:2,pass,nolog,ctl:ruleRemoveTargetById=%d;%s\"\n",
							sanitizePrefix(prefix), id, id2, t)
						id++
					}
				}
			}
			continue
		}
		if global {
			fmt.Fprintf(&b, "SecRuleRemoveById %s\n", joinIDs(ids))
		} else {
			var ctl strings.Builder
			for _, id2 := range ids {
				fmt.Fprintf(&ctl, ",ctl:ruleRemoveById=%d", id2)
			}
			fmt.Fprintf(&b, "SecRule REQUEST_URI \"@beginsWith %s\" \"id:%d,phase:1,pass,nolog%s\"\n",
				sanitizePrefix(prefix), id, ctl.String())
			id++
		}
	}
	return b.String()
}

func sanitizeIDs(ids []int) []int {
	out := make([]int, 0, len(ids))
	for _, id := range ids {
		if id > 0 && id < 1_000_000_000 {
			out = append(out, id)
		}
	}
	return out
}

func joinIDs(ids []int) string {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = fmt.Sprintf("%d", id)
	}
	return strings.Join(parts, " ")
}

// sanitizePrefix strips anything that could break out of the @beginsWith
// argument or the double-quoted directive.
func sanitizePrefix(p string) string {
	p = strings.ReplaceAll(p, "\"", "")
	p = strings.ReplaceAll(p, "\n", "")
	p = strings.ReplaceAll(p, " ", "")
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return p
}

// sanitizeTarget keeps only a conservative charset for a rule target.
func sanitizeTarget(t string) string {
	t = strings.TrimSpace(t)
	for _, r := range t {
		if !(r == ':' || r == '_' || r == '-' || r == '.' ||
			(r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')) {
			return ""
		}
	}
	return t
}

// ── reverse proxy (pooled) ──────────────────────────────────────────────

func buildProxy(pool *poolRuntime, site SiteConfig, cfg Config, log *slog.Logger, record func(method, path, rawQuery, contentType string, code int)) *httputil.ReverseProxy {
	backendTO := time.Duration(cfg.BackendTimeoutSec) * time.Second
	base := &http.Transport{
		Proxy: nil,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          256,
		MaxIdleConnsPerHost:   32,
		IdleConnTimeout:       90 * time.Second,
		ResponseHeaderTimeout: backendTO,
		ExpectContinueTimeout: 1 * time.Second,
		ForceAttemptHTTP2:     true,
	}
	return &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			ip := clientIP(pr.In)
			m := pool.pick(ip)
			if m != nil {
				pr.SetURL(m.target)
				// stash chosen member for the counting transport
				pr.Out = pr.Out.WithContext(context.WithValue(pr.Out.Context(), memberCtxKey, m))
			}
			pr.SetXForwarded()
			if site.PreserveHost {
				pr.Out.Host = pr.In.Host
			}
			for _, h := range []string{"X-Real-IP", "Forwarded"} {
				pr.Out.Header.Del(h)
			}
			pr.Out.Header.Set("X-Real-IP", ip)
		},
		Transport: lbTransport{base: base},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			log.Error("backend error", "site", site.Name, "pool", pool.name,
				"err", err, "uri", r.URL.RequestURI(), "client", clientIP(r))
			http.Error(w, "backend unavailable", http.StatusBadGateway)
		},
		ModifyResponse: func(resp *http.Response) error {
			resp.Header.Del("Server")
			resp.Header.Del("X-Powered-By")
			if resp.Header.Get("X-Content-Type-Options") == "" {
				resp.Header.Set("X-Content-Type-Options", "nosniff")
			}
			if record != nil && resp.Request != nil && resp.Request.URL != nil {
				record(resp.Request.Method, resp.Request.URL.Path, resp.Request.URL.RawQuery, resp.Request.Header.Get("Content-Type"), resp.StatusCode)
			}
			return nil
		},
		FlushInterval: 100 * time.Millisecond,
		ErrorLog:      slog.NewLogLogger(log.Handler(), slog.LevelWarn),
	}
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return strings.Trim(host, "[]")
}

// getCertificate selects a certificate for a given listener by SNI.
func (s *server) getCertificate(addr string) func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	return func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
		rt := s.rt.Load()
		lr := rt.listeners[addr]
		if lr == nil {
			return nil, fmt.Errorf("no listener for %s", addr)
		}
		if site := lr.lookup(hello.ServerName); site != nil && site.cert != nil {
			return site.cert, nil
		}
		if lr.defaultCert != nil {
			return lr.defaultCert, nil
		}
		return nil, fmt.Errorf("no certificate for %q", hello.ServerName)
	}
}

// ── main ────────────────────────────────────────────────────────────────

func main() {
	var (
		configPath = flag.String("config", "config.json", "path to JSON config (created with defaults if missing)")
		adminAddr  = flag.String("admin", envOr("WAF_ADMIN_ADDR", "127.0.0.1:9090"), "admin console listen address, e.g. 127.0.0.1:9090, 0.0.0.0:9090, or a mgmt IP")
		adminCert  = flag.String("admin-cert", os.Getenv("WAF_ADMIN_TLS_CERT"), "TLS cert for the admin console (recommended when binding off-loopback)")
		adminKey   = flag.String("admin-key", os.Getenv("WAF_ADMIN_TLS_KEY"), "TLS key for the admin console")
		adminToken = flag.String("admin-token", os.Getenv("WAF_ADMIN_TOKEN"), "bearer token for the admin API (random if empty)")
		browseRoot = flag.String("tls-browse-root", "/etc", "directory the console's cert/key file browser is allowed to list (read-only)")
	)
	flag.Parse()

	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(log)

	cfg, err := loadConfig(*configPath)
	if errors.Is(err, os.ErrNotExist) {
		cfg = defaultConfig()
		if err := saveConfig(*configPath, cfg); err != nil {
			log.Error("could not write default config", "err", err)
			os.Exit(1)
		}
		log.Info("wrote default config", "path", *configPath)
	} else if err != nil {
		log.Error("could not load config", "path", *configPath, "err", err)
		os.Exit(1)
	}

	root, err := filepath.Abs(*browseRoot)
	if err != nil {
		root = "/etc"
	}
	notifier := newNotifier(log)
	aiEng := newAIEngine(log)
	aiEng.notify = notifier
	s := &server{
		matches:       newMatchRing(250),
		access:        newAccessRing(1000),
		maps:          newSiteMaps(5000),
		signals:       newSignalStore(),
		ai:            aiEng,
		learn:         newLearnStore(),
		notify:        notifier,
		ha:            newHAEngine(log, notifier),
		syslog:        newSyslogEngine(log),
		hosts:         newHostObserver(log),
		metrics:       newMetrics(),
		ipmgr:         newIPManager(log, *adminAddr, os.Getenv("WAF_DATA_INTERFACE")),
		log:           log,
		configPath:    *configPath,
		tlsBrowseRoot: filepath.Clean(root),
		bootCfg:       cfg,
	}
	notifier.sink = s.syslog.forwardNotify // fan notifications out to syslog
	if err := s.apply(cfg); err != nil {
		log.Error("initial build failed", "err", err)
		os.Exit(1)
	}

	// ── data-plane listeners (dynamic: opened/closed on config apply) ──
	s.listenMgr = newListenerManager(s, log)
	s.listenMgr.startAll(cfg)

	// ── site-map persistence: reload observed structure, autosave periodically ──
	if err := s.maps.load(*configPath); err != nil {
		log.Warn("could not load saved site map", "err", err)
	}
	sitemapStop := make(chan struct{})
	var sitemapWG sync.WaitGroup
	s.maps.startAutosave(*configPath, 60*time.Second, sitemapStop, &sitemapWG)

	// ── metrics sampler (CPU/mem/throughput/rates for the dashboard) ──
	metricsStop := make(chan struct{})
	s.metrics.startSampler(3*time.Second, metricsStop)

	// ── admin listener ──
	admin := newAdminServer(s, *adminToken, log)
	adminSrv := &http.Server{
		Addr:              *adminAddr,
		Handler:           admin.handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	adminTLS := *adminCert != "" && *adminKey != ""
	if err := adminPlaneCheck(*adminAddr, adminTLS, cfg.Sites); err != nil {
		log.Error("refusing to start: management/data-plane separation violated", "err", err)
		os.Exit(1)
	}
	if adminExposed(*adminAddr) {
		if adminTLS {
			log.Warn("admin console is bound OFF-LOOPBACK — reachable from the network",
				"addr", *adminAddr, "tls", true,
				"reminder", "restrict with a firewall/mgmt-VLAN; signed-update endpoints remain localhost-only")
		} else {
			log.Warn("admin console is bound OFF-LOOPBACK WITHOUT TLS — session tokens and the LLM key would traverse the network in cleartext",
				"addr", *adminAddr,
				"fix", "set -admin-cert/-admin-key (or WAF_ADMIN_TLS_CERT/KEY), and firewall this port")
		}
	}
	go func() {
		scheme := "http"
		if adminTLS {
			scheme = "https"
		}
		log.Info("admin console", "addr", scheme+"://"+*adminAddr)
		var err error
		if adminTLS {
			err = adminSrv.ListenAndServeTLS(*adminCert, *adminKey)
		} else {
			err = adminSrv.ListenAndServe()
		}
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("admin server error", "err", err)
		}
	}()

	log.Info("waf-proxy starting",
		"version", buildVersion, "commit", buildCommit,
		"listeners", s.listenMgr.count(), "sites", len(cfg.Sites),
		"pools", len(cfg.Pools), "nodes", len(cfg.Nodes), "engine", cfg.EngineMode)

	// ── graceful shutdown ──
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go s.ha.run(ctx)       // peer health polling + role computation
	go s.learnScanner(ctx) // periodic learner → notification (never auto-applies)
	wdStop := make(chan struct{})
	go s.runWatchdog(wdStop) // opt-in hardware watchdog feeder (no-op unless configured)
	<-ctx.Done()
	log.Info("shutting down")
	close(sitemapStop) // triggers a final site-map flush
	close(metricsStop)
	sitemapWG.Wait()
	// Managed service IPs intentionally survive a routine stop/restart. They are
	// removed when manage_ip is disabled or the site is deleted, not merely
	// because the process is being upgraded.
	// Mark unhealthy first and pause so keepalived/VRRP / a bypass NIC poller
	// sees /healthz go 503 and fails over BEFORE we stop accepting connections.
	s.draining.Store(true)
	close(wdStop) // stop feeding the hardware watchdog on clean shutdown
	if d := drainDelay(); d > 0 {
		log.Info("draining", "seconds", int(d.Seconds()))
		time.Sleep(d)
	}
	shutCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	_ = adminSrv.Shutdown(shutCtx)
	s.listenMgr.shutdown(shutCtx)
	if rt := s.rt.Load(); rt != nil && rt.cancel != nil {
		rt.cancel()
	}
	log.Info("stopped cleanly")
}

// learnScanner periodically checks each site for actionable policy-fit
// suggestions and raises a notification (with an apply action payload) for new
// ones. It never applies anything — the operator acts from the console.
func (s *server) learnScanner(ctx context.Context) {
	t := time.NewTicker(90 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			cfg := s.rt.Load().cfg
			for _, sc := range cfg.Sites {
				rec := s.learn.recommend(sc.Name)
				for _, pg := range rec.Pages {
					if len(pg.SuggestExcl) == 0 && pg.Risk != "hostile" {
						continue
					}
					key := "learn:" + sc.Name + "|" + pg.Path + "|" + fmt.Sprint(pg.SuggestExcl)
					title := "Policy suggestion: " + sc.Name
					body := pg.Path + " — " + pg.Risk + "; " + pg.SuggestNote
					var action string
					var payload map[string]any
					if len(pg.SuggestExcl) > 0 {
						action = "apply_exclusion"
						payload = map[string]any{
							"site": sc.Name, "path": pg.Path, "rule_ids": pg.SuggestExcl,
						}
					}
					s.notify.push(notifySuggestion, "info", title, body, key, action, payload)
				}
			}
		}
	}
}

// sortedListenAddrs is used by the admin API for stable display.
func sortedListenAddrs(c Config) []string {
	set := map[string]bool{}
	for _, s := range c.Sites {
		set[s.Listen] = true
	}
	out := make([]string, 0, len(set))
	for a := range set {
		out = append(out, a)
	}
	sort.Strings(out)
	return out
}
