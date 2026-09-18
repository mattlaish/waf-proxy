package main

import "time"

// InvestigationQuery defines operator search filters.
type InvestigationQuery struct {
	RequestID string
	ClientIP string
	RuleID string
	Hostname string
	Action string
	From time.Time
	To time.Time
}

// InvestigationResult returns correlated events.
type InvestigationResult struct {
	Events []SecurityTimelineEvent `json:"events"`
}
