package main

import "time"

// Compatibility input used by deterministic unit tests and migration helpers.
type SchemaFieldLearning struct {
	Path              string
	Location          string
	Types             []SchemaTypeEvidence
	Formats           []string
	SampleCount       int64
	PresenceRate      float64
	RequiredCandidate bool
	EnumCandidate     []string
	Sensitive         bool
}

type SchemaObservationLearning struct {
	OperationID string
	Version     int
	SampleCount int64
	Fields      []SchemaFieldLearning
	UpdatedAt   time.Time
}

// buildSchemaLearningCandidate remains deterministic and does not invoke AI.
func buildSchemaLearningCandidate(obs SchemaObservationLearning) SchemaCandidate {
	fields := make([]SchemaFieldObservation, 0, len(obs.Fields))
	var confidence float64
	for _, field := range obs.Fields {
		typeCounts := map[string]int64{}
		for _, t := range field.Types {
			typeCounts[t.Type] = t.Count
			confidence += t.Confidence
		}
		fields = append(fields, SchemaFieldObservation{
			Path: field.Path, Location: field.Location, TypeCounts: typeCounts, Types: append([]SchemaTypeEvidence(nil), field.Types...),
			Formats: append([]string(nil), field.Formats...), Samples: field.SampleCount, PresenceRate: field.PresenceRate,
			RequiredCandidate: field.RequiredCandidate, EnumCandidate: append([]string(nil), field.EnumCandidate...), Sensitive: field.Sensitive,
		})
	}
	status := "LEARNING"
	if obs.SampleCount >= schemaCandidateMinSamples {
		status = "CANDIDATE"
	}
	if obs.UpdatedAt.IsZero() {
		obs.UpdatedAt = time.Now().UTC()
	}
	c := SchemaCandidate{ID: schemaCandidateID(obs.OperationID), OperationID: obs.OperationID, Status: status, SampleCount: obs.SampleCount, Fields: fields, CreatedAt: obs.UpdatedAt, UpdatedAt: obs.UpdatedAt, Version: obs.Version}
	if len(fields) > 0 {
		c.Confidence = confidence / float64(len(fields))
	}
	return c
}
