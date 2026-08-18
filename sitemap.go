package main

// Site content discovery.
//
// Two sources feed a per-site path tree:
//
//   1. Passive (always on): every response proxied through a site records its
//      path, method, and status. Zero extra load on the backend — it's just
//      bookkeeping on traffic that already flowed. This reflects what is
//      actually reached in practice.
//
//   2. Active crawl (opt-in, per site): a polite same-origin spider walks the
//      site's own backend starting from "/", following links only. It is a
//      MAPPER, not a scanner — it follows what the site links to, does not
//      brute-force paths from a wordlist, issues GET only, skips obviously
//      state-changing links, and is bounded by depth, page count, time, and a
//      politeness delay. It only ever targets a backend the operator
//      configured in this proxy.
//
// The tree is bounded (maxNodes per site) so neither a chatty site nor a deep
// crawl can grow memory without limit.

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	srcObserved = "seen"
	srcCrawled  = "crawled"
)

// ── path tree ───────────────────────────────────────────────────────────

type pathNode struct {
	name     string
	full     string
	hits     int64
	methods  map[string]struct{}
	lastCode int
	lastSeen time.Time
	seen     bool // observed in live traffic
	crawled  bool // discovered by the crawler
	children map[string]*pathNode
}

func newNode(name, full string) *pathNode {
	return &pathNode{name: name, full: full, methods: map[string]struct{}{}, children: map[string]*pathNode{}}
}

type siteMap struct {
	mu    sync.Mutex
	root  *pathNode
	nodes int
	crawl crawlState
}

type crawlState struct {
	Running  bool      `json:"running"`
	Started  time.Time `json:"started,omitempty"`
	Finished string    `json:"finished,omitempty"`
	Pages    int       `json:"pages"`
	Depth    int       `json:"depth"`
	Forms    int       `json:"forms"`
	Fields   int       `json:"fields"`
	Err      string    `json:"error,omitempty"`
}

type siteMaps struct {
	mu       sync.Mutex
	byName   map[string]*siteMap
	maxNodes int
}

func newSiteMaps(maxNodes int) *siteMaps {
	return &siteMaps{byName: map[string]*siteMap{}, maxNodes: maxNodes}
}

func (m *siteMaps) forSite(name string) *siteMap {
	m.mu.Lock()
	defer m.mu.Unlock()
	sm := m.byName[name]
	if sm == nil {
		sm = &siteMap{root: newNode("", "/")}
		m.byName[name] = sm
	}
	return sm
}

func (m *siteMaps) clear(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if sm := m.byName[name]; sm != nil {
		sm.mu.Lock()
		running := sm.crawl.Running
		sm.mu.Unlock()
		if !running {
			delete(m.byName, name)
		}
	}
}

// record inserts a path into a site's tree. Cheap and lock-scoped so it can sit
// in the hot response path.
func (m *siteMaps) record(site, method, path string, code int, source string) {
	if path == "" {
		path = "/"
	}
	sm := m.forSite(site)
	sm.mu.Lock()
	defer sm.mu.Unlock()

	segs := splitPath(path)
	node := sm.root
	touch(node, method, code, source, sm.root == node)
	full := ""
	for _, seg := range segs {
		full += "/" + seg
		child, ok := node.children[seg]
		if !ok {
			if sm.nodes >= m.maxNodes {
				// Tree is full: keep counting on the deepest existing node
				// rather than growing. record() never allocates past the cap.
				break
			}
			child = newNode(seg, full)
			node.children[seg] = child
			sm.nodes++
		}
		node = child
		touch(node, method, code, source, false)
	}
}

func touch(n *pathNode, method string, code int, source string, isRoot bool) {
	if !isRoot {
		n.hits++
	}
	if method != "" {
		n.methods[method] = struct{}{}
	}
	if code != 0 {
		n.lastCode = code
	}
	n.lastSeen = time.Now()
	switch source {
	case srcObserved:
		n.seen = true
	case srcCrawled:
		n.crawled = true
	}
}

func splitPath(p string) []string {
	p = strings.TrimPrefix(p, "/")
	if p == "" {
		return nil
	}
	raw := strings.Split(p, "/")
	out := raw[:0]
	for _, s := range raw {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

// ── serialization for the API ───────────────────────────────────────────

type nodeJSON struct {
	Name     string     `json:"name"`
	Full     string     `json:"full"`
	Hits     int64      `json:"hits"`
	Methods  []string   `json:"methods"`
	LastCode int        `json:"last_code"`
	LastSeen string     `json:"last_seen"`
	Source   string     `json:"source"` // seen | crawled | both
	IsDir    bool       `json:"is_dir"`
	Children []nodeJSON `json:"children,omitempty"`
}

func (n *pathNode) toJSON() nodeJSON {
	methods := make([]string, 0, len(n.methods))
	for k := range n.methods {
		methods = append(methods, k)
	}
	sort.Strings(methods)

	kids := make([]nodeJSON, 0, len(n.children))
	for _, c := range n.children {
		kids = append(kids, c.toJSON())
	}
	sort.Slice(kids, func(i, j int) bool {
		// directories first, then alphabetical
		if kids[i].IsDir != kids[j].IsDir {
			return kids[i].IsDir
		}
		return kids[i].Name < kids[j].Name
	})

	src := ""
	switch {
	case n.seen && n.crawled:
		src = "both"
	case n.seen:
		src = srcObserved
	case n.crawled:
		src = srcCrawled
	}
	name := n.name
	if name == "" {
		name = "/"
	}
	ls := ""
	if !n.lastSeen.IsZero() {
		ls = n.lastSeen.Format("15:04:05")
	}
	return nodeJSON{
		Name: name, Full: n.full, Hits: n.hits, Methods: methods,
		LastCode: n.lastCode, LastSeen: ls, Source: src,
		IsDir: len(n.children) > 0, Children: kids,
	}
}

type siteMapJSON struct {
	Site  string     `json:"site"`
	Nodes int        `json:"nodes"`
	Crawl crawlState `json:"crawl"`
	Tree  nodeJSON   `json:"tree"`
}

func (m *siteMaps) snapshot(name string) siteMapJSON {
	sm := m.forSite(name)
	sm.mu.Lock()
	defer sm.mu.Unlock()
	return siteMapJSON{Site: name, Nodes: sm.nodes, Crawl: sm.crawl, Tree: sm.root.toJSON()}
}

func collectSeenGETPaths(node *pathNode, out *[]string) {
	if node.full != "" && node.full != "/" && node.seen {
		if _, ok := node.methods[http.MethodGet]; ok && !skipLink(node.full) {
			*out = append(*out, node.full)
		}
	}
	for _, child := range node.children {
		collectSeenGETPaths(child, out)
	}
}

func (sm *siteMap) seenGETPaths() []string {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	var out []string
	collectSeenGETPaths(sm.root, &out)
	sort.Strings(out)
	return out
}

// ── crawler ─────────────────────────────────────────────────────────────

type crawlOpts struct {
	maxPages int
	maxDepth int
}

// hrefRe pulls href/src targets out of HTML. This is a link mapper, not a
// parser — good enough to discover the site's own linked structure without
// pulling in an HTML dependency.
var hrefRe = regexp.MustCompile(`(?i)(?:href|src)\s*=\s*["']([^"']+)["']`)

// skipLink avoids following links that commonly mutate state on GET.
func skipLink(p string) bool {
	low := strings.ToLower(p)
	for _, bad := range []string{"logout", "signout", "sign-out", "log-out",
		"/delete", "/destroy", "/remove", "action=delete", "action=logout"} {
		if strings.Contains(low, bad) {
			return true
		}
	}
	return false
}

// startCrawl kicks off an async crawl of a site's backend. Only one crawl per
// site runs at a time. backend is resolved from the site's pool by the caller.
// Returns false if a crawl is already in flight.
func (s *server) startCrawl(site SiteConfig, backend *url.URL, opts crawlOpts) bool {
	sm := s.maps.forSite(site.Name)
	sm.mu.Lock()
	if sm.crawl.Running {
		sm.mu.Unlock()
		return false
	}
	sm.crawl = crawlState{Running: true, Started: time.Now()}
	sm.mu.Unlock()

	go s.runCrawl(site, backend, opts, sm)
	return true
}

func (s *server) runCrawl(site SiteConfig, backend *url.URL, opts crawlOpts, sm *siteMap) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	finish := func(errMsg string) {
		sm.mu.Lock()
		sm.crawl.Running = false
		sm.crawl.Err = errMsg
		sm.crawl.Finished = time.Now().Format(time.RFC3339)
		sm.mu.Unlock()
	}

	if backend == nil {
		finish("no reachable pool member to crawl")
		return
	}
	// Public origin is used to resolve and same-origin-filter links; the
	// primary hostname is sent as Host so vhosted backends serve the right
	// site. Fetches always go to the configured backend, never elsewhere.
	primary := primaryHost(site)
	scheme := "http"
	if site.TLSCert != "" {
		scheme = "https"
	}
	publicHost := primary
	if publicHost == "" {
		publicHost = backend.Host
	}
	publicOrigin := &url.URL{Scheme: scheme, Host: publicHost}

	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // record redirects, don't chase offsite
		},
	}

	type item struct {
		path  string
		depth int
	}
	seen := map[string]bool{"/": true}
	queue := []item{{"/", 0}}
	// Previously observed GET paths are safe, high-value seeds for login and
	// SPA routes that are not linked from the public root page.
	for _, path := range sm.seenGETPaths() {
		if !seen[path] {
			seen[path] = true
			queue = append(queue, item{path, 0})
		}
	}
	pages := 0

	for len(queue) > 0 {
		if ctx.Err() != nil {
			finish("timed out")
			return
		}
		if pages >= opts.maxPages {
			break
		}
		cur := queue[0]
		queue = queue[1:]

		body, code, ct := s.fetchPage(ctx, client, backend, primary, cur.path)
		pages++
		s.maps.record(site.Name, "GET", cur.path, code, srcCrawled)
		fields := s.signals.noteContent(site.Name, cur.path, ct, body)
		var actions []string
		formCount := 0
		if strings.Contains(strings.ToLower(ct), "html") {
			actions = discoverFormActions(body, cur.path)
			formCount = len(formTagRE.FindAllStringSubmatch(body, -1))
		}

		sm.mu.Lock()
		sm.crawl.Pages = pages
		sm.crawl.Depth = cur.depth
		sm.crawl.Forms += formCount
		sm.crawl.Fields += len(fields)
		sm.mu.Unlock()

		time.Sleep(60 * time.Millisecond) // politeness

		if cur.depth >= opts.maxDepth || !strings.Contains(ct, "html") || body == "" {
			continue
		}
		// Resolve links against the CURRENT page so relative hrefs work.
		base := *publicOrigin
		base.Path = cur.path
		var targets []string
		for _, m := range hrefRe.FindAllStringSubmatch(body, -1) {
			targets = append(targets, m[1])
		}
		// Form actions are discovery targets too. Fetches remain GET-only, so
		// this cannot submit a form or mutate a conforming backend.
		targets = append(targets, actions...)
		for _, target := range targets {
			raw := strings.TrimSpace(target)
			if raw == "" || strings.HasPrefix(raw, "#") ||
				strings.HasPrefix(raw, "mailto:") || strings.HasPrefix(raw, "javascript:") ||
				strings.HasPrefix(raw, "tel:") || strings.HasPrefix(raw, "data:") {
				continue
			}
			ref, err := url.Parse(raw)
			if err != nil {
				continue
			}
			abs := base.ResolveReference(ref)
			if abs.Host != publicOrigin.Host { // same-origin only
				continue
			}
			np := abs.Path
			if np == "" {
				np = "/"
			}
			if seen[np] || skipLink(np) {
				continue
			}
			seen[np] = true
			queue = append(queue, item{np, cur.depth + 1})
		}
	}
	finish("")
}

func (s *server) fetchPage(ctx context.Context, client *http.Client, backend *url.URL, host, path string) (body string, code int, contentType string) {
	u := *backend
	u.Path = singleJoin(backend.Path, path)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", 0, ""
	}
	if host != "" {
		req.Host = host
	}
	req.Header.Set("User-Agent", "waf-proxy-mapper/1.0 (+local admin)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := client.Do(req)
	if err != nil {
		return "", 0, ""
	}
	defer resp.Body.Close()
	ct := resp.Header.Get("Content-Type")
	// Cap body read so a huge page can't blow memory during a crawl.
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	return string(b), resp.StatusCode, ct
}

func primaryHost(site SiteConfig) string {
	for _, h := range site.Hostnames {
		k := normalizeHost(h)
		if k != "" && k != "*" && !strings.HasPrefix(k, "*.") {
			return k
		}
	}
	return ""
}

func singleJoin(a, b string) string {
	switch {
	case a == "" || a == "/":
		return b
	case strings.HasSuffix(a, "/") && strings.HasPrefix(b, "/"):
		return a + b[1:]
	case !strings.HasSuffix(a, "/") && !strings.HasPrefix(b, "/"):
		return a + "/" + b
	default:
		return a + b
	}
}
