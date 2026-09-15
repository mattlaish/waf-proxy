package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// DebugEvidenceCapture is the compatibility wrapper retained for callers from
// the first debug-evidence slice. New request-scoped capture uses
// DebugEvidenceStore in debug_bundle.go.
var activeDebugEvidenceCapture atomic.Pointer[DebugEvidenceCapture]

func SetDebugEvidenceCapture(c *DebugEvidenceCapture)    { activeDebugEvidenceCapture.Store(c) }
func currentDebugEvidenceCapture() *DebugEvidenceCapture { return activeDebugEvidenceCapture.Load() }

type DebugEvidenceCapture struct {
	mu      sync.RWMutex
	enabled bool
	max     int
	ttl     time.Duration
	items   map[string]map[string]any
}

func NewDebugEvidenceCapture(max int) *DebugEvidenceCapture {
	if max < 1 {
		max = 100
	}
	return &DebugEvidenceCapture{max: max, items: map[string]map[string]any{}}
}
func (d *DebugEvidenceCapture) Enable(v bool)          { d.mu.Lock(); d.enabled = v; d.mu.Unlock() }
func (d *DebugEvidenceCapture) SetTTL(v time.Duration) { d.mu.Lock(); d.ttl = v; d.mu.Unlock() }

func isSensitiveDebugKey(k string) bool {
	k = strings.ToLower(strings.TrimSpace(k))
	for _, part := range []string{"authorization", "proxy-authorization", "cookie", "set-cookie", "password", "passwd", "secret", "api_key", "apikey", "token", "private_key", "peer_token"} {
		if strings.Contains(k, part) {
			return true
		}
	}
	return false
}

func sanitizeDebugValue(key string, v any) any {
	if isSensitiveDebugKey(key) {
		return "[masked]"
	}
	switch x := v.(type) {
	case string:
		return truncateDebugString(x, 4096)
	case []string:
		out := make([]string, len(x))
		for i := range x {
			out[i] = truncateDebugString(x[i], 4096)
		}
		return out
	case map[string]any:
		return sanitizeDebugMap(x)
	case map[string]string:
		out := make(map[string]any, len(x))
		for k, v := range x {
			out[k] = sanitizeDebugValue(k, v)
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i := range x {
			out[i] = sanitizeDebugValue("", x[i])
		}
		return out
	default:
		return v
	}
}

func sanitizeDebugMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = sanitizeDebugValue(k, v)
	}
	return out
}

func (d *DebugEvidenceCapture) Capture(id string, evidence map[string]any) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if !d.enabled || id == "" {
		return
	}
	now := time.Now().UTC()
	if d.ttl > 0 {
		for k, v := range d.items {
			if raw, ok := v["captured_unix_nano"].(int64); ok && now.Sub(time.Unix(0, raw)) >= d.ttl {
				delete(d.items, k)
			}
		}
	}
	if len(d.items) >= d.max {
		return
	}
	e := sanitizeDebugMap(evidence)
	e["captured_at"] = now.Format(time.RFC3339Nano)
	e["captured_unix_nano"] = now.UnixNano()
	d.items[id] = e
}

func (d *DebugEvidenceCapture) Export(path string) error {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	z := zip.NewWriter(f)
	defer z.Close()
	b, err := json.MarshalIndent(map[string]any{"files": len(d.items), "evidence": d.items}, "", "  ")
	if err != nil {
		return err
	}
	w, err := z.Create("debug-evidence.json")
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}

func HashEvidence(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}

func sortedDebugKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
