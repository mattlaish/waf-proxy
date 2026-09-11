package vectoraccel

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

type regexFactory struct{}
type regexDB struct {
	rules []RuleSpec
	rx    []*regexp.Regexp
}

func (regexFactory) Available() bool { return true }
func (regexFactory) Version() string { return "test" }
func (regexFactory) Compile(rs []RuleSpec) (compiledScanner, error) {
	d := &regexDB{rules: rs}
	for _, r := range rs {
		x, e := regexp.Compile(r.Pattern)
		if e != nil {
			return nil, e
		}
		d.rx = append(d.rx, x)
	}
	return d, nil
}
func (d *regexDB) Scan(b []byte) ([]int, error) {
	var ids []int
	for i, r := range d.rx {
		if r.Match(b) {
			ids = append(ids, d.rules[i].ID)
		}
	}
	return ids, nil
}
func (d *regexDB) Close() error { return nil }

func TestParseRulesConservative(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "rules.conf")
	os.WriteFile(p, []byte(`SecRule REQUEST_URI "@rx (?i)admin" "id:1001,phase:1,pass,t:none"
SecRule ARGS "@rx bad" "id:1002,phase:2,deny,t:none"
SecRule REQUEST_METHOD "@rx ^POST$" "id:1003,phase:1,pass,t:none,t:lowercase"
`), 0600)
	g, used, e := ParseRules(p)
	if e != nil {
		t.Fatal(e)
	}
	if len(g) != 2 {
		t.Fatalf("groups=%d", len(g))
	}
	if _, ok := used[1002]; !ok {
		t.Fatal("all ids must be reserved even unsupported")
	}
}

func TestLearningFailsafeOnFalseNegative(t *testing.T) {
	cfg := Effective(Config{Mode: ModeAuto, StatePath: filepath.Join(t.TempDir(), "s.json"), MinSamples: 1, MinCorazaMatches: 0, MinLearningSec: 0, VerificationSampleRate: 1})
	m := &Manager{cfg: cfg, factory: regexFactory{}, groups: map[string]*groupRuntime{}, persisted: map[string]persistedGroup{}, stop: make(chan struct{}), done: make(chan struct{}), native: true, version: "test"}
	close(m.done)
	d := t.TempDir()
	p := filepath.Join(d, "rules.conf")
	os.WriteFile(p, []byte(`SecRule REQUEST_METHOD "@rx ^GET$" "id:2001,phase:1,pass,t:none"`), 0600)
	plan, e := m.BuildSite("s", p)
	if e != nil {
		t.Fatal(e)
	}
	r := httptest.NewRequest("POST", "http://x/", nil)
	obs := plan.analyze(r)
	plan.Observe(obs, []int{2001})
	if plan.groups[0].getState() != StateFailsafe {
		t.Fatalf("state=%s", plan.groups[0].getState())
	}
}

func TestParseRulesPhase2Coverage(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "phase2.conf")
	rules := `SecRule QUERY_STRING "@rx ^mixed$" "id:3001,phase:1,pass,t:none,t:trim,t:lowercase"
SecRule SERVER_NAME "@rx ^EXAMPLE.COM$" "id:3002,phase:1,pass,t:none,t:uppercase"
SecRule REQUEST_HEADERS:Host "@rx ^example" "id:3003,phase:1,pass,t:none,t:lowercase"
SecRule REQUEST_HEADERS:Transfer-Encoding "@rx chunked" "id:3004,phase:1,pass,t:none,t:lowercase"
SecRule REQUEST_URI "@rx ^/x$" "id:3005,phase:1,pass,t:none,t:trimLeft,t:trimRight"
SecRule REQUEST_URI "@rx x" "id:3006,phase:1,pass,t:none,t:urlDecode"
SecRule ARGS:test "@rx x" "id:3007,phase:2,pass,t:none,t:lowercase"
SecRule REQUEST_URI "!@rx x" "id:3008,phase:1,pass,t:none"
SecRule REQUEST_URI "@rx x" "id:3009,phase:1,pass,t:none,chain"
SecRule REQUEST_URI_RAW "@rx ^/path" "id:3010,phase:1,pass,t:none"
SecRule REQUEST_LINE "@rx ^POST " "id:3011,phase:1,pass,t:none"
SecRule REQUEST_BASENAME "@rx ^file[.]txt$" "id:3012,phase:1,pass,t:none"
SecRule REMOTE_ADDR "@rx ^203[.]0[.]113[.]9$" "id:3013,phase:1,pass,t:none"
SecRule REMOTE_PORT "@rx ^45678$" "id:3014,phase:1,pass,t:none"
SecRule REQUEST_HEADERS:X-Encoding "@rx ^YWJj$" "id:3015,phase:1,pass,t:none,t:base64Encode"
SecRule REQUEST_HEADERS:X-Hex "@rx ^616263$" "id:3016,phase:1,pass,t:none,t:hexEncode"
SecRule REQUEST_URI "@rx x" "id:3017,phase:1,pass,t:none,t:base64Decode"
SecRule REQUEST_URI "@rx x" "id:3018,phase:1,pass,t:none,t:hexDecode"
`
	if err := os.WriteFile(p, []byte(rules), 0600); err != nil {
		t.Fatal(err)
	}
	groups, used, err := ParseRules(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 12 {
		t.Fatalf("groups=%d want 12: %#v", len(groups), groups)
	}
	for _, id := range []int{3006, 3007, 3008, 3009, 3017, 3018} {
		if _, ok := used[id]; !ok {
			t.Fatalf("unsupported rule id %d must still be reserved", id)
		}
	}
	seen := map[int]RuleSpec{}
	for _, g := range groups {
		for _, r := range g.Rules {
			seen[r.ID] = r
		}
	}
	for _, id := range []int{3001, 3002, 3003, 3004, 3005, 3010, 3011, 3012, 3013, 3014, 3015, 3016} {
		if _, ok := seen[id]; !ok {
			t.Fatalf("eligible Phase 2 rule %d missing", id)
		}
	}
	for _, id := range []int{3006, 3007, 3008, 3009, 3017, 3018} {
		if _, ok := seen[id]; ok {
			t.Fatalf("unsupported rule %d unexpectedly eligible", id)
		}
	}
	if got := transformKey(seen[3001].Transforms); got != "trim>lowercase" {
		t.Fatalf("rule 3001 transforms=%q", got)
	}
}

func TestPhase2SourceValuesMirrorCorazaHTTPConnector(t *testing.T) {
	r := httptest.NewRequest("POST", "/dir/file.txt?A=One&B=Two", nil)
	r.Host = "Public.Example:8443"
	r.RemoteAddr = "203.0.113.9:45678"
	r.TransferEncoding = []string{"gzip", "chunked"}
	r.Header.Set("X-Probe", "one")
	r.Header.Add("X-Probe", "two")

	cases := []struct {
		name string
		g    GroupSpec
		want []string
	}{
		{name: "request-uri-raw", g: GroupSpec{Source: SourceRequestURIRaw}, want: []string{"/dir/file.txt?A=One&B=Two"}},
		{name: "request-line", g: GroupSpec{Source: SourceRequestLine}, want: []string{"POST /dir/file.txt?A=One&B=Two HTTP/1.1"}},
		{name: "request-basename", g: GroupSpec{Source: SourceRequestBasename}, want: []string{"file.txt"}},
		{name: "query", g: GroupSpec{Source: SourceQueryString}, want: []string{"A=One&B=Two"}},
		{name: "server-name", g: GroupSpec{Source: SourceServerName}, want: []string{"Public.Example:8443"}},
		{name: "remote-addr", g: GroupSpec{Source: SourceRemoteAddr}, want: []string{"203.0.113.9"}},
		{name: "remote-port", g: GroupSpec{Source: SourceRemotePort}, want: []string{"45678"}},
		{name: "host-header", g: GroupSpec{Source: SourceRequestHeader, Header: "Host"}, want: []string{"Public.Example:8443"}},
		{name: "transfer-encoding", g: GroupSpec{Source: SourceRequestHeader, Header: "Transfer-Encoding"}, want: []string{"gzip", "chunked"}},
		{name: "ordinary-header", g: GroupSpec{Source: SourceRequestHeader, Header: "X-Probe"}, want: []string{"one", "two"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := sourceValues(tc.g, r)
			if len(got) != len(tc.want) {
				t.Fatalf("values=%q want %q", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("values=%q want %q", got, tc.want)
				}
			}
		})
	}
}

func TestPhase2RequestBasenameAndRemoteConnectorEdges(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.URL.Path = `/dir\\windows\\name.txt`
	if got := sourceValues(GroupSpec{Source: SourceRequestBasename}, r); len(got) != 1 || got[0] != "name.txt" {
		t.Fatalf("backslash basename=%q", got)
	}
	r.URL.Path = "/dir/"
	if got := sourceValues(GroupSpec{Source: SourceRequestBasename}, r); len(got) != 1 || got[0] != "/dir/" {
		t.Fatalf("trailing-separator basename=%q", got)
	}

	cases := []struct {
		remote string
		addr   string
		port   int
	}{
		{remote: "203.0.113.7:1234", addr: "203.0.113.7", port: 1234},
		{remote: "[2001:db8::7]:8443", addr: "[2001:db8::7]", port: 8443},
		{remote: "203.0.113.7", addr: "", port: 0},
		{remote: "203.0.113.7:not-a-port", addr: "203.0.113.7", port: 0},
	}
	for _, tc := range cases {
		addr, port := corazaConnectorRemote(tc.remote)
		if addr != tc.addr || port != tc.port {
			t.Fatalf("corazaConnectorRemote(%q)=(%q,%d) want (%q,%d)", tc.remote, addr, port, tc.addr, tc.port)
		}
	}
}

func TestPhase2DifferentTransformPipelinesDoNotShareGroup(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "rules.conf")
	if err := os.WriteFile(p, []byte(`SecRule REQUEST_HEADERS:X-Test "@rx one" "id:3101,phase:1,pass,t:none,t:lowercase"
SecRule REQUEST_HEADERS:X-Test "@rx TWO" "id:3102,phase:1,pass,t:none,t:uppercase"
`), 0600); err != nil {
		t.Fatal(err)
	}
	groups, _, err := ParseRules(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 2 {
		t.Fatalf("groups=%d want 2", len(groups))
	}
	if groups[0].Fingerprint == groups[1].Fingerprint {
		t.Fatal("different transform semantics must not share fingerprint")
	}
}
