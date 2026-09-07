//go:build realcoraza

package main

import (
	"sync/atomic"
	"testing"

	"github.com/corazawaf/coraza/v3"
	"github.com/corazawaf/coraza/v3/types"
)

// This is a release gate, not a unit-test substitute. It proves that the Coraza
// version selected by go.mod reports a nolog disruptive rule in MatchedRules()
// while DetectionOnly prevents the disruption and the logging callback remains
// silent. VectorScan Learning uses MatchedRules(), never the callback, as truth.
func TestRealCorazaDetectionOnlyNologMatchedRulesTruth(t *testing.T) {
	var callbacks atomic.Int64
	cfg := coraza.NewWAFConfig().
		WithDirectives(`
SecRuleEngine DetectionOnly
SecRequestBodyAccess On
SecRule REQUEST_URI "@contains vector-truth-sentinel" "id:187001,phase:1,deny,status:403,nolog,t:none"
`).
		WithErrorCallback(func(types.MatchedRule) { callbacks.Add(1) })
	waf, err := coraza.NewWAF(cfg)
	if err != nil {
		t.Fatalf("NewWAF: %v", err)
	}
	tx := waf.NewTransaction()
	defer tx.Close()
	tx.ProcessConnection("127.0.0.1", 12345, "127.0.0.1", 443)
	tx.ProcessURI("/vector-truth-sentinel", "GET", "HTTP/1.1")
	if it := tx.ProcessRequestHeaders(); it != nil {
		t.Fatalf("DetectionOnly unexpectedly interrupted request: %+v", it)
	}
	if it, err := tx.ProcessRequestBody(); err != nil || it != nil {
		t.Fatalf("ProcessRequestBody interruption=%+v err=%v", it, err)
	}
	tx.ProcessLogging()
	if got := callbacks.Load(); got != 0 {
		t.Fatalf("nolog rule unexpectedly reached ErrorCallback: %d", got)
	}
	found := false
	for _, mr := range tx.MatchedRules() {
		if mr.Rule().ID() == 187001 {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("MatchedRules omitted DetectionOnly+nolog rule 187001")
	}
}
