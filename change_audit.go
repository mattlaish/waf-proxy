package main

import "time"

// ChangeAuditEvent records administrative changes.
type ChangeAuditEvent struct {
	EventID    string    `json:"event_id"`
	Actor      string    `json:"actor"`
	Action     string    `json:"action"`
	ObjectType string    `json:"object_type"`
	Before     any       `json:"before,omitempty"`
	After      any       `json:"after,omitempty"`
	Reason     string    `json:"reason,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}
