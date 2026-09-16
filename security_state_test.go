package main

import (
	"testing"
	"time"
)

func TestMemorySecurityStateStore(t *testing.T) {
	s := newMemorySecurityStateStore()
	s.PutIndicator(SecurityIndicator{Type:"ip", Value:"192.0.2.1", State:"active"})
	s.PutAudit(SecurityAuditEvent{ID:"1", Type:"blocked", CreatedAt:time.Now()})
	if len(s.ListIndicators()) != 1 || len(s.ListAudits()) != 1 {
		t.Fatal("security state persistence failed")
	}
}
