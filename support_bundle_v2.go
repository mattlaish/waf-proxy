package main

import "time"

type SupportBundleProvenance struct {
	Version string `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	GoVersion string `json:"go_version,omitempty"`
	CorazaVersion string `json:"coraza_version,omitempty"`
	VectorScanEnabled bool `json:"vectorscan_enabled"`
}

type DependencyEvidence struct {
	Name string `json:"name"`
	Version string `json:"version"`
	Source string `json:"source,omitempty"`
}
