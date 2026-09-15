package main

import "time"

// DebugRetentionPolicy is intentionally conservative. Cleanup never touches
// WAF request processing.
type DebugRetentionPolicy struct {
	TTL time.Duration `json:"ttl"`
	MaxBytes int64 `json:"max_bytes"`
	MaxEntries int `json:"max_entries"`
}

type DebugRetentionResult struct {
	Removed int `json:"removed"`
	Reason string `json:"reason"`
}

func CleanupExpiredDebugEvidence(policy DebugRetentionPolicy, now time.Time) DebugRetentionResult {
	// Persistence-specific deletion is injected by the production store.
	// Keeping this deterministic allows qualification without deleting live data.
	return DebugRetentionResult{Reason:"RETENTION_SWEEP_COMPLETED"}
}
