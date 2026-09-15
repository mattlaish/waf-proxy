package main

import "sync"

type VectorScanAuditStore struct {
	mu sync.RWMutex
	Events []VectorScanTransitionAudit
}

func (s *VectorScanAuditStore) Add(e VectorScanTransitionAudit) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Events = append(s.Events, e)
}

func (s *VectorScanAuditStore) List() []VectorScanTransitionAudit {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]VectorScanTransitionAudit,len(s.Events))
	copy(out,s.Events)
	return out
}
