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
	Saved time.Time     `json:"saved"`
	Sites []siteMapJSON `json:"sites"`
}

func (m *siteMaps) statePath(configPath string) string {
	return filepath.Join(filepath.Dir(configPath), "sitemap.json")
}

// save writes every site's tree to disk atomically. Persistence is serialized
// independently from the hot-path tree mutex so periodic autosave and explicit
// operator clears cannot race on the same state file or temporary path.
func (m *siteMaps) save(configPath string) error {
	m.persistMu.Lock()
	defer m.persistMu.Unlock()

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
	return writeSiteMapsFile(m.statePath(configPath), b)
}

func writeSiteMapsFile(path string, b []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".sitemap.json.tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	keep := false
	defer func() {
		_ = tmp.Close()
		if !keep {
			_ = os.Remove(tmpName)
		}
	}()
	if err := tmp.Chmod(0o640); err != nil {
		return err
	}
	if _, err := tmp.Write(b); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	keep = true
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}

// restoreSnapshot reinstates one site map after a failed destructive persistence
// operation. It is intentionally narrow: clear handlers use it to avoid leaving
// in-memory state cleared when the authoritative snapshot could not be updated.
func (m *siteMaps) restoreSnapshot(sj siteMapJSON) {
	sm := &siteMap{root: newNode("", "/")}
	sm.nodes = sj.Nodes
	sm.crawl = sj.Crawl
	fromJSON(sm.root, sj.Tree)
	m.mu.Lock()
	m.byName[sj.Site] = sm
	m.mu.Unlock()
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
