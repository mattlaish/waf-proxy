package main

import "encoding/json"

// SecurityEvidenceExport provides structured evidence export.
type SecurityEvidenceExport struct {
	Timeline []SecurityTimelineEvent `json:"timeline"`
	Changes  []ChangeAuditEvent     `json:"changes"`
}

func (e SecurityEvidenceExport) JSON() ([]byte, error) {
	return json.Marshal(e)
}
