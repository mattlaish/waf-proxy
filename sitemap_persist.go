package main

// Site-map persistence.
//
// The site content map is built in memory as traffic flows (and by the
// crawler). Without persistence it resets on every restart — and since a
// binary upgrade requires a restart, rebuilding the tool throws the map away.
// This saves all site maps to a JSON file next to config.json and reloads them
// at startup, so the observed structure survives restarts and upgrades.
//
// The file is written atomically (temp + rename) into the same dir as the
// config, which is already group-writable by the waf user.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type siteMapsFile struct {
	Saved time.Time      `json:"saved"`
	Sites []siteMapJSON  `json:"sites"`
}

func (m *siteMaps) statePath(configPath string) string {
	return filepath.Join(filepath.Dir(configPath), "sitemap.json")
}

// save writes every site's tree to disk atomically.
func (m *siteMaps) save(configPath string) error {
	m.mu.Lock()
	names := make([]string, 0, len(m.byName))
	for n := range m.byName {
		names = append(names, n)
	}
	m.mu.Unlock()
	sort.Strings(names)

	out := siteMapsFile{Saved: time.Now()}
	for _, n := range names {
		out.Sites = append(out.Sites, m.snapshot(n))
	}
	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	path := m.statePath(configPath)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o640); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// load rebuilds site maps from disk. Missing file is not an error.
func (m *siteMaps) load(configPath string) error {
	b, err := os.ReadFile(m.statePath(configPath))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var in siteMapsFile
	if err := json.Unmarshal(b, &in); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, sj := range in.Sites {
		sm := &siteMap{root: newNode("", "/")}
		sm.nodes = sj.Nodes
		fromJSON(sm.root, sj.Tree)
		m.byName[sj.Site] = sm
	}
	return nil
}

// fromJSON rebuilds a pathNode subtree from its serialized form (inverse of
// pathNode.toJSON).
func fromJSON(n *pathNode, j nodeJSON) {
	n.full = j.Full
	n.hits = j.Hits
	n.lastCode = j.LastCode
	n.methods = map[string]struct{}{}
	for _, mth := range j.Methods {
		n.methods[mth] = struct{}{}
	}
	switch j.Source {
	case "both":
		n.seen, n.crawled = true, true
	case srcObserved:
		n.seen = true
	case srcCrawled:
		n.crawled = true
	}
	if j.LastSeen != "" {
		// Stored as clock time only; anchor to today so ordering is sane.
		if t, err := time.Parse("15:04:05", j.LastSeen); err == nil {
			now := time.Now()
			n.lastSeen = time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), t.Second(), 0, now.Location())
		}
	}
	n.children = map[string]*pathNode{}
	for _, cj := range j.Children {
		name := cj.Name
		if name == "/" {
			name = ""
		}
		child := newNode(name, cj.Full)
		fromJSON(child, cj)
		n.children[strings.ToLower(cj.Name)] = child
	}
}

// startAutosave periodically flushes site maps to disk until ctx is done, and
// writes a final snapshot on exit. Interval is generous — this is observed
// state, not critical data, so we trade freshness for negligible I/O.
func (m *siteMaps) startAutosave(configPath string, every time.Duration, stop <-chan struct{}, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		t := time.NewTicker(every)
		defer t.Stop()
		for {
			select {
			case <-stop:
				_ = m.save(configPath)
				return
			case <-t.C:
				_ = m.save(configPath)
			}
		}
	}()
}
