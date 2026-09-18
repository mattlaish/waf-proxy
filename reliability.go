package main

import (
	"encoding/json"
	"time"
)

// ReliabilityScenario describes a controlled failure or recovery qualification case.
// It records expected behavior without claiming that execution happened.
type ReliabilityScenario struct {
	Name        string `json:"name"`
	Category    string `json:"category"`
	Expected    string `json:"expected"`
	Status      string `json:"status"`
	Evidence    string `json:"evidence,omitempty"`
	GeneratedAt string `json:"generated_at"`
}

type ReliabilityReport struct {
	Format    string               `json:"format"`
	Status    string               `json:"status"`
	Scenarios []ReliabilityScenario `json:"scenarios"`
}

func NewReliabilityReport() ReliabilityReport {
	now := time.Now().UTC().Format(time.RFC3339)
	return ReliabilityReport{
		Format: "waf-proxy-reliability-qualification-v1",
		Status: "NOT_RUN",
		Scenarios: []ReliabilityScenario{
			{Name: "crs_unavailable", Category: "dependency_failure", Expected: "fail_closed_no_bypass", Status: "NOT_RUN", GeneratedAt: now},
			{Name: "crl_unavailable", Category: "pki_failure", Expected: "use_last_known_good", Status: "NOT_RUN", GeneratedAt: now},
			{Name: "storage_unavailable", Category: "storage_failure", Expected: "degraded_visible_state", Status: "NOT_RUN", GeneratedAt: now},
			{Name: "process_restart", Category: "recovery", Expected: "recover_configuration_and_state", Status: "NOT_RUN", GeneratedAt: now},
			{Name: "resource_exhaustion", Category: "resource", Expected: "controlled_failure", Status: "NOT_RUN", GeneratedAt: now},
		},
	}
}

func MarshalReliabilityReport() ([]byte, error) {
	r := NewReliabilityReport()
	return json.MarshalIndent(r, "", "  ")
}
