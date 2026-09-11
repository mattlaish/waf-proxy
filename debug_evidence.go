package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// DebugEvidenceCapture is bounded diagnostic evidence. Capture is never on the
// WAF verdict critical path. Failures must not affect Coraza decisions.
var activeDebugEvidenceCapture atomic.Pointer[DebugEvidenceCapture]

func SetDebugEvidenceCapture(c *DebugEvidenceCapture) { activeDebugEvidenceCapture.Store(c) }

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

func maskDebugValue(v any) any {
	s, ok := v.(string)
	if !ok {
		return v
	}
	lower := strings.ToLower(s)
	if strings.Contains(lower, "authorization") || strings.Contains(lower, "password") || strings.Contains(lower, "cookie") {
		return "[masked]"
	}
	if len(s) > 4096 {
		return s[:4096] + "[truncated]"
	}
	return s
}

func (d *DebugEvidenceCapture) Capture(id string, evidence map[string]any) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if !d.enabled || id == "" {
		return
	}
	if len(d.items) >= d.max {
		return
	}
	now := time.Now().UTC()
	e := map[string]any{"captured_at": now.Format(time.RFC3339Nano)}
	for k, v := range evidence {
		e[k] = maskDebugValue(v)
	}
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
	b, _ := json.MarshalIndent(map[string]any{"files": len(d.items), "evidence": d.items}, "", "  ")
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
