package main

// Policy learner.
//
// Watches per-page signals — request volume, response outcomes, and which CRS
// rules fire, from how many distinct clients — and turns them into concrete,
// reviewable recommendations for each page under a site:
//
//   * suggested path-scoped exclusions for rules that look like false
//     positives (fired on a meaningful share of traffic, from many distinct
//     clients — i.e. legitimate users tripping a rule), and
//   * a per-page risk class (benign / normal / elevated / hostile) that rolls
//     up to a suggested paranoia level for the site's policy.
//
// The heuristics are deliberately simple and transparent — no black box. A
// true per-PAGE engine isn't practical (one Coraza instance per site), so the
// unit of action is a page-scoped exclusion written into the site's policy,
// plus a policy-level paranoia suggestion.

import (
	"sort"
	"sync"
	"time"
)

const (
	learnMaxPathsPerSite = 2000
	learnMaxClientsPerRule = 64
)

type ruleAgg struct {
	count    int64
	clients  map[string]struct{}
	severity string
}

type pathAgg struct {
	hits    int64
	blocked int64
	ok2xx   int64
	rules   map[int]*ruleAgg
	lastSeen time.Time
}

type siteLearn struct {
	mu    sync.Mutex
	paths map[string]*pathAgg
}

type learnStore struct {
	mu    sync.Mutex
	sites map[string]*siteLearn
}

func newLearnStore() *learnStore { return &learnStore{sites: map[string]*siteLearn{}} }

func (l *learnStore) forSite(site string) *siteLearn {
	l.mu.Lock()
	defer l.mu.Unlock()
	sl := l.sites[site]
	if sl == nil {
		sl = &siteLearn{paths: map[string]*pathAgg{}}
		l.sites[site] = sl
	}
	return sl
}

func (sl *siteLearn) path(p string) *pathAgg {
	if p == "" {
		p = "/"
	}
	pa := sl.paths[p]
	if pa == nil {
		if len(sl.paths) >= learnMaxPathsPerSite {
			return nil // cap: stop tracking new paths
		}
		pa = &pathAgg{rules: map[int]*ruleAgg{}}
		sl.paths[p] = pa
	}
	return pa
}

func (l *learnStore) noteRequest(site, path string, code int) {
	sl := l.forSite(site)
	sl.mu.Lock()
	defer sl.mu.Unlock()
	pa := sl.path(path)
	if pa == nil {
		return
	}
	pa.hits++
	pa.lastSeen = time.Now()
	switch {
	case code == 403:
		pa.blocked++
	case code >= 200 && code < 300:
		pa.ok2xx++
	}
}

func (l *learnStore) noteMatch(site, uri string, ruleID int, client, severity string) {
	sl := l.forSite(site)
	sl.mu.Lock()
	defer sl.mu.Unlock()
	pa := sl.path(uri)
	if pa == nil {
		return
	}
	ra := pa.rules[ruleID]
	if ra == nil {
		ra = &ruleAgg{clients: map[string]struct{}{}, severity: severity}
		pa.rules[ruleID] = ra
	}
	ra.count++
	if len(ra.clients) < learnMaxClientsPerRule {
		ra.clients[client] = struct{}{}
	}
}

func (l *learnStore) clear(site string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.sites, site)
}

// ── recommendations ─────────────────────────────────────────────────────

type ruleFinding struct {
	RuleID   int    `json:"rule_id"`
	Count    int64  `json:"count"`
	Clients  int    `json:"clients"`
	Severity string `json:"severity"`
	Kind     string `json:"kind"` // likely_fp | attack | noise
}

type pageRecommendation struct {
	Path         string        `json:"path"`
	Hits         int64         `json:"hits"`
	Blocked      int64         `json:"blocked"`
	Risk         string        `json:"risk"` // benign | normal | elevated | hostile
	Findings     []ruleFinding `json:"findings"`
	SuggestExcl  []int         `json:"suggest_exclusions"` // rule ids to remove for this path
	SuggestNote  string        `json:"suggest_note"`
}

type siteRecommendation struct {
	Site            string               `json:"site"`
	Pages           []pageRecommendation `json:"pages"`
	SuggestParanoia int                  `json:"suggest_paranoia"`
	Summary         string               `json:"summary"`
}

// classifyRule decides whether a rule firing on a path looks like a false
// positive (broad, many clients), a targeted attack (few clients), or noise.
func classifyRule(ra *ruleAgg, pathHits int64) string {
	clients := len(ra.clients)
	// Fired on a meaningful fraction of this page's traffic, spread across
	// several distinct clients ⇒ legitimate users tripping it ⇒ likely FP.
	if clients >= 3 && pathHits > 0 && float64(ra.count) >= 0.15*float64(pathHits) {
		return "likely_fp"
	}
	// Concentrated on a few sources ⇒ looks targeted.
	if clients <= 2 && ra.count >= 3 {
		return "attack"
	}
	return "noise"
}

func (l *learnStore) recommend(site string) siteRecommendation {
	sl := l.forSite(site)
	sl.mu.Lock()
	defer sl.mu.Unlock()

	out := siteRecommendation{Site: site}
	attackPages, fpOnlyPages := 0, 0

	for p, pa := range sl.paths {
		if len(pa.rules) == 0 {
			continue // nothing fired here; no recommendation needed
		}
		rec := pageRecommendation{Path: p, Hits: pa.hits, Blocked: pa.blocked}
		hasAttack, hasFP := false, false
		for id, ra := range pa.rules {
			kind := classifyRule(ra, pa.hits)
			rec.Findings = append(rec.Findings, ruleFinding{
				RuleID: id, Count: ra.count, Clients: len(ra.clients),
				Severity: ra.severity, Kind: kind,
			})
			switch kind {
			case "likely_fp":
				hasFP = true
				rec.SuggestExcl = append(rec.SuggestExcl, id)
			case "attack":
				hasAttack = true
			}
		}
		sort.Slice(rec.Findings, func(i, j int) bool { return rec.Findings[i].Count > rec.Findings[j].Count })
		sort.Ints(rec.SuggestExcl)

		switch {
		case hasAttack && hasFP:
			rec.Risk = "elevated"
			rec.SuggestNote = "under attack and generating false positives — exclude the FP rules for this page, keep paranoia high"
			attackPages++
		case hasAttack:
			rec.Risk = "hostile"
			rec.SuggestNote = "targeted attack traffic — keep or raise paranoia; do not exclude"
			attackPages++
		case hasFP:
			rec.Risk = "benign"
			rec.SuggestNote = "legitimate traffic tripping rules — safe to exclude these rule ids for this page"
			fpOnlyPages++
		default:
			rec.Risk = "normal"
		}
		out.Pages = append(out.Pages, rec)
	}

	sort.Slice(out.Pages, func(i, j int) bool {
		ri, rj := riskRank(out.Pages[i].Risk), riskRank(out.Pages[j].Risk)
		if ri != rj {
			return ri > rj
		}
		return out.Pages[i].Hits > out.Pages[j].Hits
	})

	// Site-level paranoia suggestion.
	switch {
	case attackPages >= 3:
		out.SuggestParanoia = 3
		out.Summary = "multiple pages under attack — a strict policy (PL3) is warranted"
	case attackPages >= 1:
		out.SuggestParanoia = 2
		out.Summary = "some attack pressure — consider PL2 for this site"
	case fpOnlyPages >= 1:
		out.SuggestParanoia = 1
		out.Summary = "mostly benign; tune out the flagged false positives and PL1 is fine"
	default:
		out.SuggestParanoia = 0
		out.Summary = "not enough signal yet — send more traffic"
	}
	return out
}

func riskRank(r string) int {
	switch r {
	case "hostile":
		return 3
	case "elevated":
		return 2
	case "benign":
		return 1
	default:
		return 0
	}
}
