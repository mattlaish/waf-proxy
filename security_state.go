package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

const persistentSecurityStateVersion = 1

type persistedRuleAgg struct {
	RuleID   int      `json:"rule_id"`
	Count    int64    `json:"count"`
	Clients  []string `json:"clients,omitempty"`
	Severity string   `json:"severity,omitempty"`
}

type persistedPathAgg struct {
	Path     string             `json:"path"`
	Hits     int64              `json:"hits"`
	Blocked  int64              `json:"blocked"`
	OK2xx    int64              `json:"ok_2xx"`
	LastSeen time.Time          `json:"last_seen,omitempty"`
	Rules    []persistedRuleAgg `json:"rules,omitempty"`
}

type persistedSiteLearn struct {
	Site  string             `json:"site"`
	Paths []persistedPathAgg `json:"paths,omitempty"`
}

type persistedSession struct {
	TokenHash string    `json:"token_hash"`
	User      string    `json:"user"`
	Role      string    `json:"role"`
	Expires   time.Time `json:"expires"`
}

type persistentSecurityState struct {
	Version       int                  `json:"version"`
	SavedAt       time.Time            `json:"saved_at"`
	AIBlocks      []blockEntry         `json:"ai_blocks,omitempty"`
	Learner       []persistedSiteLearn `json:"learner,omitempty"`
	Notifications []notification       `json:"notifications,omitempty"`
	NotifyNextID  int64                `json:"notify_next_id,omitempty"`
	Sessions      []persistedSession   `json:"sessions,omitempty"`
	Audit         []auditEntry         `json:"audit,omitempty"`
	Security      securityCounters     `json:"security_counters"`
}

func effectiveSecurityStatePath(s *server) string {
	if s != nil {
		if rt := s.rt.Load(); rt != nil && rt.cfg.Security.StatePath != "" {
			return rt.cfg.Security.StatePath
		}
		if s.bootCfg.Security.StatePath != "" {
			return s.bootCfg.Security.StatePath
		}
	}
	return "/var/lib/waf-proxy/security-state.json"
}

func snapshotPersistentSecurityState(s *server, admin *adminServer) persistentSecurityState {
	out := persistentSecurityState{Version: persistentSecurityStateVersion, SavedAt: time.Now().UTC()}
	if s == nil {
		return out
	}
	if s.ai != nil {
		for _, be := range s.ai.loadBlocks() {
			if be.Expires.IsZero() || time.Now().Before(be.Expires) {
				out.AIBlocks = append(out.AIBlocks, be)
			}
		}
		sort.Slice(out.AIBlocks, func(i, j int) bool {
			if out.AIBlocks[i].Site != out.AIBlocks[j].Site {
				return out.AIBlocks[i].Site < out.AIBlocks[j].Site
			}
			return out.AIBlocks[i].IP < out.AIBlocks[j].IP
		})
	}
	if s.learn != nil {
		s.learn.mu.Lock()
		sites := make(map[string]*siteLearn, len(s.learn.sites))
		for name, sl := range s.learn.sites {
			sites[name] = sl
		}
		s.learn.mu.Unlock()
		names := make([]string, 0, len(sites))
		for name := range sites {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			sl := sites[name]
			sl.mu.Lock()
			ps := persistedSiteLearn{Site: name}
			paths := make([]string, 0, len(sl.paths))
			for path := range sl.paths {
				paths = append(paths, path)
			}
			sort.Strings(paths)
			for _, path := range paths {
				pa := sl.paths[path]
				pp := persistedPathAgg{Path: path, Hits: pa.hits, Blocked: pa.blocked, OK2xx: pa.ok2xx, LastSeen: pa.lastSeen}
				ids := make([]int, 0, len(pa.rules))
				for id := range pa.rules {
					ids = append(ids, id)
				}
				sort.Ints(ids)
				for _, id := range ids {
					ra := pa.rules[id]
					pr := persistedRuleAgg{RuleID: id, Count: ra.count, Severity: ra.severity}
					for client := range ra.clients {
						pr.Clients = append(pr.Clients, client)
					}
					sort.Strings(pr.Clients)
					pp.Rules = append(pp.Rules, pr)
				}
				ps.Paths = append(ps.Paths, pp)
			}
			sl.mu.Unlock()
			out.Learner = append(out.Learner, ps)
		}
	}
	if s.notify != nil {
		s.notify.mu.Lock()
		out.Notifications = append(out.Notifications, s.notify.items...)
		out.NotifyNextID = s.notify.nextID
		s.notify.mu.Unlock()
	}
	if s.security != nil {
		out.Security = s.security.counters()
	}
	if admin != nil {
		if admin.sessions != nil {
			admin.sessions.mu.Lock()
			now := time.Now()
			for tokenHash, sess := range admin.sessions.byID {
				if now.Before(sess.expires) {
					out.Sessions = append(out.Sessions, persistedSession{TokenHash: tokenHash, User: sess.user, Role: sess.role, Expires: sess.expires})
				}
			}
			admin.sessions.mu.Unlock()
			sort.Slice(out.Sessions, func(i, j int) bool { return out.Sessions[i].TokenHash < out.Sessions[j].TokenHash })
		}
		if admin.audit != nil {
			admin.audit.mu.Lock()
			out.Audit = append(out.Audit, admin.audit.recs...)
			admin.audit.mu.Unlock()
		}
	}
	return out
}

func restorePersistentSecurityState(st persistentSecurityState, s *server, admin *adminServer) error {
	if st.Version != persistentSecurityStateVersion {
		return fmt.Errorf("unsupported security state version %d", st.Version)
	}
	now := time.Now()
	if s != nil && s.ai != nil {
		next := blockSnapshot{}
		for _, be := range st.AIBlocks {
			if be.IP == "" || be.Site == "" || (!be.Expires.IsZero() && !now.Before(be.Expires)) {
				continue
			}
			next[blockKey(be.Site, be.IP)] = be
		}
		s.ai.blockMu.Lock()
		s.ai.block.Store(&next)
		s.ai.blockMu.Unlock()
	}
	if s != nil && s.learn != nil {
		restored := map[string]*siteLearn{}
		for _, ps := range st.Learner {
			if ps.Site == "" {
				continue
			}
			sl := &siteLearn{paths: map[string]*pathAgg{}}
			for _, pp := range ps.Paths {
				if pp.Path == "" || len(sl.paths) >= learnMaxPathsPerSite {
					continue
				}
				pa := &pathAgg{hits: pp.Hits, blocked: pp.Blocked, ok2xx: pp.OK2xx, lastSeen: pp.LastSeen, rules: map[int]*ruleAgg{}}
				for _, pr := range pp.Rules {
					if pr.RuleID <= 0 {
						continue
					}
					ra := &ruleAgg{count: pr.Count, severity: pr.Severity, clients: map[string]struct{}{}}
					for _, client := range pr.Clients {
						if len(ra.clients) >= learnMaxClientsPerRule {
							break
						}
						ra.clients[client] = struct{}{}
					}
					pa.rules[pr.RuleID] = ra
				}
				sl.paths[pp.Path] = pa
			}
			restored[ps.Site] = sl
		}
		s.learn.mu.Lock()
		s.learn.sites = restored
		s.learn.mu.Unlock()
	}
	if s != nil && s.notify != nil {
		s.notify.mu.Lock()
		items := st.Notifications
		if len(items) > s.notify.cap {
			items = items[len(items)-s.notify.cap:]
		}
		s.notify.items = append([]notification(nil), items...)
		s.notify.nextID = st.NotifyNextID
		s.notify.mu.Unlock()
	}
	if s != nil && s.security != nil {
		s.security.restoreCounters(st.Security)
	}
	if admin != nil {
		if admin.sessions != nil {
			admin.sessions.mu.Lock()
			admin.sessions.byID = map[string]session{}
			for _, ps := range st.Sessions {
				if ps.TokenHash != "" && ps.User != "" && now.Before(ps.Expires) {
					admin.sessions.byID[ps.TokenHash] = session{user: ps.User, role: ps.Role, expires: ps.Expires}
				}
			}
			admin.sessions.mu.Unlock()
		}
		if admin.audit != nil {
			admin.audit.mu.Lock()
			recs := st.Audit
			if len(recs) > admin.audit.cap {
				recs = recs[len(recs)-admin.audit.cap:]
			}
			admin.audit.recs = append([]auditEntry(nil), recs...)
			admin.audit.mu.Unlock()
		}
	}
	return nil
}

func loadPersistentSecurityState(path string, s *server, admin *adminServer) error {
	if path == "" {
		return nil
	}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(b) > 16<<20 {
		return errors.New("security state exceeds 16 MiB")
	}
	var st persistentSecurityState
	if err := json.Unmarshal(b, &st); err != nil {
		return fmt.Errorf("decode security state: %w", err)
	}
	return restorePersistentSecurityState(st, s, admin)
}

func savePersistentSecurityState(path string, s *server, admin *adminServer) error {
	if path == "" {
		return nil
	}
	st := snapshotPersistentSecurityState(s, admin)
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".security-state-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	ok := false
	defer func() {
		_ = tmp.Close()
		if !ok {
			_ = os.Remove(tmpName)
		}
	}()
	if err := tmp.Chmod(0o600); err != nil {
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
	ok = true
	return nil
}

func startPersistentSecurityState(s *server, admin *adminServer, interval time.Duration) (chan struct{}, *sync.WaitGroup) {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	stop := make(chan struct{})
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				if err := savePersistentSecurityState(effectiveSecurityStatePath(s), s, admin); err != nil && s != nil && s.log != nil {
					s.log.Warn("security state autosave failed", "err", err)
				}
			case <-stop:
				_ = savePersistentSecurityState(effectiveSecurityStatePath(s), s, admin)
				return
			}
		}
	}()
	return stop, wg
}
