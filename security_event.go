package main

type securityEvent struct {
	Type      string `json:"type"`
	RequestID string `json:"request_id,omitempty"`
	Client    string `json:"client,omitempty"`
	Reason    string `json:"reason,omitempty"`
	Action    string `json:"action,omitempty"`
}

type securityEventExporter interface{ ExportSecurityEvent(securityEvent) error }
