package vectorscan

import "time"

type QualificationReport struct {
	Format         string   `json:"format"`
	GeneratedAt    string   `json:"generated_at"`
	Status         string   `json:"status"`
	FalseNegatives int      `json:"false_negatives"`
	Evidence       []string `json:"evidence,omitempty"`
}

func NewReport() QualificationReport {
	return QualificationReport{
		Format:      "waf-proxy-vectorscan-qualification-v1",
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Status:      "NOT_RUN",
	}
}
