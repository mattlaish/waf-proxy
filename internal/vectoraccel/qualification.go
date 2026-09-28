package vectoraccel

import (
	"fmt"
	"net/http"
	"sort"
)

// QualificationCandidates executes the same native group scans used by the
// runtime Learning path, but never creates a skip header and never suppresses
// Coraza. It is intended for the offline Phase 1 differential runner.
func (p *SitePlan) QualificationCandidates(r *http.Request) ([]int, error) {
	if p == nil {
		return nil, nil
	}
	obs := p.analyze(r)
	if obs == nil {
		return nil, nil
	}
	seen := map[int]struct{}{}
	for _, gp := range obs.predictions {
		if !gp.scanned {
			return nil, fmt.Errorf("native VectorScan scan failed for group %s", gp.group.spec.Key)
		}
		for id := range gp.candidates {
			seen[id] = struct{}{}
		}
	}
	ids := make([]int, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids, nil
}

// EligibleRuleIDs returns exactly the conservative rule set that this plan may
// accelerate. Differential qualification must compare Coraza matches only
// within this set; unsupported CRS semantics intentionally remain Coraza-only.
func (p *SitePlan) EligibleRuleIDs() []int {
	if p == nil {
		return nil
	}
	ids := make([]int, 0, len(p.byRule))
	for id := range p.byRule {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids
}

// CandidateRuleIDs exposes the native candidates already computed for one
// request observation so debug evidence can be correlated with the final
// transaction without rescanning request bytes.
func (o *Observation) CandidateRuleIDs() []int {
	if o == nil {
		return nil
	}
	seen := map[int]struct{}{}
	for _, gp := range o.predictions {
		for id := range gp.candidates {
			seen[id] = struct{}{}
		}
	}
	ids := make([]int, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids
}

// EligibleMatchedRuleIDs filters authoritative Coraza matches to the rules
// actually represented by this acceleration plan.
func (p *SitePlan) EligibleMatchedRuleIDs(ids []int) []int {
	if p == nil {
		return nil
	}
	out := make([]int, 0, len(ids))
	seen := map[int]struct{}{}
	for _, id := range ids {
		if _, ok := p.byRule[id]; !ok {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	sort.Ints(out)
	return out
}
