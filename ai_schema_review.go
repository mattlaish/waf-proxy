package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

func schemaReviewOutputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"recommendation": map[string]any{"type": "string", "enum": []string{"approve", "revise", "insufficient_evidence"}},
			"confidence":     map[string]any{"type": "integer", "minimum": 0, "maximum": 100},
			"reason":         map[string]any{"type": "string"},
		},
		"required":             []string{"recommendation", "confidence", "reason"},
		"additionalProperties": false,
	}
}

func (e *aiEngine) reviewSchemaCandidate(ctx context.Context, candidate *SchemaCandidate) (SchemaReviewRecommendation, error) {
	if e == nil || candidate == nil {
		return SchemaReviewRecommendation{}, fmt.Errorf("schema candidate unavailable")
	}
	cfg := e.snapshotCfg()
	if !cfg.Enabled {
		return SchemaReviewRecommendation{}, fmt.Errorf("AI connector disabled")
	}
	b, err := json.Marshal(candidate)
	if err != nil {
		return SchemaReviewRecommendation{}, err
	}
	system := `You are reviewing a deterministic API schema-learning candidate for a web application firewall. ` +
		`The evidence contains only field names and aggregate metadata. Assess whether required-field and enum candidates are sufficiently supported. ` +
		`Do not authorize blocking or policy activation. Return only the requested structured JSON.`
	user := "schema_candidate:\n" + string(b)
	var txt string
	if cfg.Provider == "openai" && cfg.effectiveOpenAIAPIStyle() == "responses" {
		txt, err = e.callOpenAIResponses(ctx, cfg, system, user, "waf_api_schema_review", schemaReviewOutputSchema())
	} else {
		txt, err = e.callLLMRaw(ctx, cfg, system, user)
	}
	if err != nil {
		return SchemaReviewRecommendation{}, err
	}
	var out struct {
		Recommendation string `json:"recommendation"`
		Confidence     int    `json:"confidence"`
		Reason         string `json:"reason"`
	}
	if err := decodeJSONObject(txt, &out); err != nil {
		return SchemaReviewRecommendation{}, err
	}
	switch out.Recommendation {
	case "approve", "revise", "insufficient_evidence":
	default:
		return SchemaReviewRecommendation{}, fmt.Errorf("invalid schema review recommendation %q", out.Recommendation)
	}
	if out.Confidence < 0 {
		out.Confidence = 0
	}
	if out.Confidence > 100 {
		out.Confidence = 100
	}
	out.Reason = strings.TrimSpace(out.Reason)
	if out.Reason == "" {
		return SchemaReviewRecommendation{}, fmt.Errorf("schema review reason is required")
	}
	if len(out.Reason) > 4000 {
		out.Reason = out.Reason[:4000]
	}
	return SchemaReviewRecommendation{Recommendation: out.Recommendation, Confidence: out.Confidence, Reason: out.Reason, ReviewedAt: time.Now().UTC(), Source: "openai"}, nil
}
