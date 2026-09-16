package qualification

import (
	"encoding/json"
	"time"
)

// RuntimeReport is an evidence container for real release-host qualification.
// It intentionally does not claim PASS unless a real runtime gate populates it.
type RuntimeReport struct {
	Format      string    `json:"format"`
	GeneratedAt string    `json:"generated_at"`
	Component   string    `json:"component"`
	Status      string    `json:"status"` // PASS, FAIL, BLOCKED, NOT_RUN
	Evidence    []Evidence `json:"evidence,omitempty"`
}

type Evidence struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

func NewReport(component string) RuntimeReport {
	return RuntimeReport{
		Format: "waf-proxy-runtime-qualification-v1",
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Component: component,
		Status: "NOT_RUN",
	}
}

func (r RuntimeReport) MarshalJSONDocument() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}
