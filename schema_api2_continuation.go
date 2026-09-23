package main

import "time"

// API-2 continuation foundation:
// This layer extends the existing candidate foundation with typed schema
// learning metadata. It intentionally stores metadata/statistics only and
// does not persist raw request values.

type SchemaTypeEvidence struct {
	Type       string
	Count      int64
	Confidence float64
}

type SchemaFieldLearning struct {
	Path             string
	Types            []SchemaTypeEvidence
	Formats          []string
	SampleCount      int64
	PresenceRate     float64
	RequiredCandidate bool
	EnumCandidate    []string
	Sensitive        bool
}

type SchemaObservationLearning struct {
	OperationID string
	Version     int
	Fields      []SchemaFieldLearning
	UpdatedAt   time.Time
}

type SchemaReviewRecommendation struct {
	Recommendation string
	Confidence     int
	Reason         string
}

// buildSchemaLearningCandidate is deliberately deterministic.
// AI review is handled separately and cannot activate policy.
func buildSchemaLearningCandidate(obs SchemaObservationLearning) SchemaCandidate {
	fields := make([]SchemaFieldObservation, 0, len(obs.Fields))
	for _, field := range obs.Fields {
		typeCounts := map[string]int64{}
		for _, t := range field.Types {
			typeCounts[t.Type] = t.Count
		}
		fields = append(fields, SchemaFieldObservation{
			Path:       field.Path,
			TypeCounts: typeCounts,
			Samples:    field.SampleCount,
			Sensitive:  field.Sensitive,
		})
	}

	return SchemaCandidate{
		OperationID: obs.OperationID,
		Status:      "CANDIDATE",
		Fields:      fields,
		CreatedAt:   obs.UpdatedAt,
	}
}
