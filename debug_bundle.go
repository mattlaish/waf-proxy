package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type DebugBundle struct {
	Tenant        string         `json:"tenant"`
	TransactionID string         `json:"transaction_id"`
	Request       map[string]any `json:"request,omitempty"`
	Response      map[string]any `json:"response,omitempty"`
	TLS           map[string]any `json:"tls,omitempty"`
	Proxy         map[string]any `json:"proxy,omitempty"`
	Coraza        map[string]any `json:"coraza,omitempty"`
	CapturedAt    time.Time      `json:"captured_at"`
}

type DebugEvidenceStore struct {
	mu    sync.Mutex
	ttl   time.Duration
	max   int
	items map[string]DebugBundle
}

func NewDebugEvidenceStore(max int, ttl time.Duration) *DebugEvidenceStore {
	if max < 1 {
		max = 100
	}
	return &DebugEvidenceStore{max: max, ttl: ttl, items: map[string]DebugBundle{}}
}
func (s *DebugEvidenceStore) Put(b DebugBundle) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if b.Tenant == "" {
		return
	}
	if len(s.items) >= s.max {
		return
	}
	b.CapturedAt = time.Now().UTC()
	s.items[b.TransactionID] = b
}
func (s *DebugEvidenceStore) Cleanup(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ttl <= 0 {
		return
	}
	for k, v := range s.items {
		if now.Sub(v.CapturedAt) > s.ttl {
			delete(s.items, k)
		}
	}
}
func (s *DebugEvidenceStore) Export(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
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
	w, err := z.Create("debug-bundle.json")
	if err != nil {
		return err
	}
	return json.NewEncoder(w).Encode(s.items)
}
func (s *DebugEvidenceStore) TenantExport(tenant, path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, v := range s.items {
		if v.Tenant == tenant {
			return s.Export(path)
		}
	}
	return fmt.Errorf("no evidence for tenant")
}
