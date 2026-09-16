package main

import (
	"sync"
	"time"
)

type SecurityIndicator struct {
	Type string `json:"type"`
	Value string `json:"value"`
	Confidence float64 `json:"confidence"`
	Source string `json:"source"`
	State string `json:"state"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

type LearningAggregate struct {
	RuleID int `json:"rule_id"`
	ObservationCount uint64 `json:"observation_count"`
	MatchCount uint64 `json:"match_count"`
	Confidence float64 `json:"confidence"`
	State string `json:"state"`
	LastSeen time.Time `json:"last_seen"`
}

type SecuritySession struct {
	ID string `json:"id"`
	Client string `json:"client"`
	CorrelationID string `json:"correlation_id"`
	FirstSeen time.Time `json:"first_seen"`
	LastSeen time.Time `json:"last_seen"`
	RequestCount uint64 `json:"request_count"`
	RiskState string `json:"risk_state"`
}

type SecurityAuditEvent struct {
	ID string `json:"id"`
	Type string `json:"type"`
	RequestID string `json:"request_id,omitempty"`
	Client string `json:"client,omitempty"`
	Action string `json:"action,omitempty"`
	Evidence map[string]string `json:"evidence,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type NotificationState struct {
	ID string `json:"id"`
	Type string `json:"type"`
	Severity string `json:"severity"`
	State string `json:"state"`
	RetryCount int `json:"retry_count"`
	CreatedAt time.Time `json:"created_at"`
}

type securityStateStore interface {
	PutIndicator(SecurityIndicator)
	ListIndicators() []SecurityIndicator
	PutAudit(SecurityAuditEvent)
	ListAudits() []SecurityAuditEvent
}

type memorySecurityStateStore struct {
	mu sync.RWMutex
	indicators []SecurityIndicator
	audits []SecurityAuditEvent
}

func newMemorySecurityStateStore() *memorySecurityStateStore {
	return &memorySecurityStateStore{}
}

func (s *memorySecurityStateStore) PutIndicator(v SecurityIndicator) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.indicators = append(s.indicators, v)
}

func (s *memorySecurityStateStore) ListIndicators() []SecurityIndicator {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]SecurityIndicator, len(s.indicators))
	copy(out, s.indicators)
	return out
}

func (s *memorySecurityStateStore) PutAudit(v SecurityAuditEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.audits = append(s.audits, v)
}

func (s *memorySecurityStateStore) ListAudits() []SecurityAuditEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]SecurityAuditEvent, len(s.audits))
	copy(out, s.audits)
	return out
}
