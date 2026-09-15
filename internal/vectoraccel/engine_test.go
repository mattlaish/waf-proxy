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
