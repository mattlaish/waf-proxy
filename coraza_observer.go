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
	if w == nil || plan == nil {
		return w
	}
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
	return &observedCorazaTx{Transaction: tx, plan: w.plan, obs: vectoraccel.ObservationFromContext(ctx)}
}

type observedCorazaTx struct {
	types.Transaction
	plan *vectoraccel.SitePlan
	obs  *vectoraccel.Observation
	once sync.Once
}

func (t *observedCorazaTx) observe() {
	t.once.Do(func() {
		if t.obs == nil || t.plan == nil {
			return
		}
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
		t.plan.Observe(t.obs, ids)
	})
}
func (t *observedCorazaTx) ProcessLogging() { t.Transaction.ProcessLogging(); t.observe() }
func (t *observedCorazaTx) Close() error    { t.observe(); return t.Transaction.Close() }
