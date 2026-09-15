package main

import (
	"context"
	"sync"

	"waf-proxy/internal/vectoraccel"

	"github.com/corazawaf/coraza/v3"
	"github.com/corazawaf/coraza/v3/experimental"
	"github.com/corazawaf/coraza/v3/types"
)

// observedCorazaWAF preserves Coraza's public WAF contract while wrapping each
// transaction so the VectorScan learner gets the authoritative final
// MatchedRules set after ProcessLogging. The HTTP connector recognizes the
// experimental WAFWithOptions interface and therefore passes request.Context,
// giving us exact request<->transaction correlation even under HTTP/2.
type observedCorazaWAF struct {
	coraza.WAF
	plan *vectoraccel.SitePlan
}

func observeCorazaWAF(w coraza.WAF, plan *vectoraccel.SitePlan) coraza.WAF {
	if w == nil {
		return w
	}
	// Always wrap so transaction-final Coraza evidence remains available to the
	// debug plane even when VectorScan is disabled or no eligible plan exists.
	return &observedCorazaWAF{WAF: w, plan: plan}
}

func (w *observedCorazaWAF) NewTransaction() types.Transaction {
	return w.wrap(w.WAF.NewTransaction(), context.Background())
}
func (w *observedCorazaWAF) NewTransactionWithID(id string) types.Transaction {
	return w.wrap(w.WAF.NewTransactionWithID(id), context.Background())
}
func (w *observedCorazaWAF) NewTransactionWithOptions(opts experimental.Options) types.Transaction {
	var tx types.Transaction
	if ow, ok := w.WAF.(experimental.WAFWithOptions); ok {
		tx = ow.NewTransactionWithOptions(opts)
	} else {
		tx = w.WAF.NewTransaction()
	}
	return w.wrap(tx, opts.Context)
}
func (w *observedCorazaWAF) wrap(tx types.Transaction, ctx context.Context) types.Transaction {
	if ctx == nil {
		ctx = context.Background()
	}
	dc, _ := debugContextFrom(ctx)
	return &observedCorazaTx{Transaction: tx, plan: w.plan, obs: vectoraccel.ObservationFromContext(ctx), debug: dc}
}

type observedCorazaTx struct {
	types.Transaction
	plan  *vectoraccel.SitePlan
	obs   *vectoraccel.Observation
	debug debugRequestContext
	once  sync.Once
}

func (t *observedCorazaTx) observe() {
	t.once.Do(func() {
		matches := t.Transaction.MatchedRules()
		ids := make([]int, 0, len(matches))
		seen := make(map[int]struct{}, len(matches))
		for _, m := range matches {
			id := m.Rule().ID()
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			ids = append(ids, id)
		}

		var candidates, eligibleActual, falseNegatives []int
		vectorObserved := t.plan != nil && t.obs != nil
		if vectorObserved {
			candidates = t.obs.CandidateRuleIDs()
			eligibleActual = t.plan.EligibleMatchedRuleIDs(ids)
			falseNegatives = VectorScanDifferential(candidates, eligibleActual)
			t.plan.Observe(t.obs, ids)
		}

		if t.debug.TransactionID != "" {
			if store := currentDebugEvidenceStore(); store != nil {
				store.Merge(t.debug.Tenant, t.debug.TransactionID, func(b *DebugBundle) {
					b.Coraza = map[string]any{
						"matched_rules": ids,
						"source":        "tx.MatchedRules-after-ProcessLogging",
					}
					if vectorObserved {
						b.Coraza["eligible_matched_rules"] = eligibleActual
						b.VectorScan = map[string]any{
							"observed":             true,
							"candidate_rules":      candidates,
							"false_negative_rules": falseNegatives,
							"zero_false_negative":  len(falseNegatives) == 0,
						}
					} else {
						b.VectorScan = map[string]any{"observed": false}
					}
				})
			}
		}
		if c := currentDebugEvidenceCapture(); c != nil {
			c.Capture("coraza-tx", map[string]any{"matched_rules": ids, "source": "tx.MatchedRules-after-ProcessLogging"})
		}
	})
}
func (t *observedCorazaTx) ProcessLogging() { t.Transaction.ProcessLogging(); t.observe() }
func (t *observedCorazaTx) Close() error    { t.observe(); return t.Transaction.Close() }
