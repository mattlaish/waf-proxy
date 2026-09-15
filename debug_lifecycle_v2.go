package main

import (
	"sync"
	"time"
)

type DebugCaptureState string

const (
	DebugCaptureCreated   DebugCaptureState = "CREATED"
	DebugCaptureActive    DebugCaptureState = "ACTIVE"
	DebugCaptureStopping  DebugCaptureState = "STOPPING"
	DebugCaptureCompleted DebugCaptureState = "COMPLETED"
	DebugCaptureFailed    DebugCaptureState = "FAILED"
)

type DebugCaptureSession struct {
	ID        string            `json:"id"`
	Tenant    string            `json:"tenant"`
	Site      string            `json:"site,omitempty"`
	State     DebugCaptureState `json:"state"`
	Owner     string            `json:"owner,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	ExpiresAt time.Time         `json:"expires_at"`
	MaxEvents int64             `json:"max_events"`
	Events    int64             `json:"events"`
	MaxBytes  int64             `json:"max_bytes"`
	Bytes     int64             `json:"bytes"`
}

type DebugCaptureManager struct {
	mu       sync.RWMutex
	sessions map[string]*DebugCaptureSession
}

func NewDebugCaptureManager() *DebugCaptureManager {
	return &DebugCaptureManager{sessions: map[string]*DebugCaptureSession{}}
}
func (m *DebugCaptureManager) Add(s DebugCaptureSession) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if s.State == "" {
		s.State = DebugCaptureCreated
	}
	m.sessions[s.ID] = &s
}
func (m *DebugCaptureManager) Stop(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := m.sessions[id]
	if s == nil {
		return false
	}
	s.State = DebugCaptureStopping
	s.State = DebugCaptureCompleted
	return true
}
func (m *DebugCaptureManager) PurgeExpired(now time.Time) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for id, s := range m.sessions {
		if !s.ExpiresAt.IsZero() && now.After(s.ExpiresAt) {
			delete(m.sessions, id)
			n++
		}
	}
	return n
}
