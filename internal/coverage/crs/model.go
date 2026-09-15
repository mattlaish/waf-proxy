package crs

// Rule describes normalized CRS SecRule metadata consumed by the capability
// analyzer. Raw is diagnostic only and is not emitted by inventory JSON.
type Rule struct {
	ID          string   `json:"id"`
	File        string   `json:"file,omitempty"`
	Line        int      `json:"line,omitempty"`
	Phase       int      `json:"phase,omitempty"`
	Operator    string   `json:"operator"`
	RawOperator string   `json:"raw_operator,omitempty"`
	Pattern     string   `json:"pattern,omitempty"`
	Variables   []string `json:"variables"`
	Transforms  []string `json:"transforms"`
	Tags        []string `json:"tags,omitempty"`
	Severity    string   `json:"severity,omitempty"`
	Actions     []string `json:"actions,omitempty"`
	HasChain    bool     `json:"has_chain"`
	Negated     bool     `json:"negated"`
	Raw         string   `json:"-"`
}

// Warning records non-fatal ingestion diagnostics. Mandatory include failures
// and I/O failures are returned as errors instead.
type Warning struct {
	File    string `json:"file"`
	Line    int    `json:"line,omitempty"`
	Message string `json:"message"`
}

// RuleLocation identifies a rule occurrence for duplicate-ID reporting.
type RuleLocation struct {
	File string `json:"file"`
	Line int    `json:"line"`
}

// Ruleset is the normalized result of loading a CRS entrypoint or directory.
type Ruleset struct {
	Root             string                    `json:"root"`
	Files            []string                  `json:"files"`
	Rules            []Rule                    `json:"rules"`
	DuplicateRuleIDs map[string][]RuleLocation `json:"duplicate_rule_ids,omitempty"`
	Warnings         []Warning                 `json:"warnings,omitempty"`
}
