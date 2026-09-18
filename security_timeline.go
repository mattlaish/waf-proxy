package main

import "time"

// SecurityTimelineEvent represents a correlated security operation event.
type SecurityTimelineEvent struct {
	EventID     string    `json:"event_id"`
	RequestID   string    `json:"request_id,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
	EventType   string    `json:"event_type"`
	Source      string    `json:"source,omitempty"`
	Decision    string    `json:"decision,omitempty"`
	Severity    string    `json:"severity,omitempty"`
	Evidence    any       `json:"evidence,omitempty"`
}
