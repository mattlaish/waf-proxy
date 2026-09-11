package main

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/corazawaf/coraza/v3"
	txhttp "github.com/corazawaf/coraza/v3/http"
	"github.com/corazawaf/coraza/v3/types"
)

func TestPhase4RequestIDBecomesCorazaTransactionID(t *testing.T) {
	var gotTxID string
	waf, err := coraza.NewWAF(coraza.NewWAFConfig().
		WithDirectives(`SecRuleEngine DetectionOnly
SecRule REQUEST_URI "@contains /phase4-probe" "id:990401,phase:1,pass,log"`).
		WithErrorCallback(func(mr types.MatchedRule) {
			gotTxID = mr.TransactionID()
		}))
	if err != nil {
		t.Fatal(err)
	}
	wrapped := observeCorazaWAF(waf, nil)
	h := txhttp.WrapHandler(wrapped, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	s := &server{metrics: newMetrics(), access: newAccessRing(10), syslog: newSyslogEngine(log)}
	req := httptest.NewRequest(http.MethodGet, "http://example.test/phase4-probe", nil)
	req.Header.Set(requestIDHeader, "attacker-controlled")
	rr := httptest.NewRecorder()
	s.logWrap("phase4", h).ServeHTTP(rr, req)
	id := rr.Header().Get(requestIDHeader)
	if id == "" || id == "attacker-controlled" {
		t.Fatalf("unexpected response request ID %q", id)
	}
	if gotTxID != id {
		t.Fatalf("Coraza transaction ID %q != WAF request ID %q", gotTxID, id)
	}
}
