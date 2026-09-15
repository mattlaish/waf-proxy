package capability

import "strings"

// Input is the normalized semantic subset shared by offline coverage analysis
// and the live VectorScan rule classifier. Callers may parse SecLang using
// different front-ends, but eligibility is decided only here.
type Input struct {
	RuleID     string
	Phase      int
	Operator   string
	Pattern    string
	Variables  []string
	Transforms []string
	HasChain   bool
	Negated    bool
}

type Result struct {
	Eligible  bool
	Reason    string
	Phase     int
	Source    string
	Header    string
	Lowercase bool
}

const (
	SourceRequestURI      = "REQUEST_URI"
	SourceRequestFilename = "REQUEST_FILENAME"
	SourceRequestMethod   = "REQUEST_METHOD"
	SourceRequestProtocol = "REQUEST_PROTOCOL"
	SourceRequestHeader   = "REQUEST_HEADERS"
)

// Classify is the single source of truth for the exact request-value subset
// that the current VectorScan adapter can reproduce. Eligibility is only a
// prerequisite for learning/validation; it never promotes a rule by itself.
func Classify(in Input) Result {
	if strings.TrimSpace(in.RuleID) == "" {
		return reject("missing rule id")
	}
	if strings.ToLower(strings.TrimSpace(in.Operator)) != "rx" {
		return reject("unsupported operator: " + strings.TrimSpace(in.Operator))
	}
	if strings.TrimSpace(in.Pattern) == "" {
		return reject("empty regex pattern")
	}
	if in.HasChain {
		return reject("chains remain Coraza-only")
	}
	if in.Negated {
		return reject("negated operators remain Coraza-only")
	}
	phase := in.Phase
	if phase == 0 {
		// SecRule defaults to phase 2 when no phase action overrides it.
		phase = 2
	}
	if phase != 1 && phase != 2 {
		return reject("unsupported phase")
	}
	if len(in.Variables) != 1 {
		return reject("multi-variable selectors remain Coraza-only")
	}
	source, header, ok := classifySelector(in.Variables[0])
	if !ok {
		return reject("unsupported variable: " + in.Variables[0])
	}
	if len(in.Transforms) == 0 || strings.ToLower(strings.TrimSpace(in.Transforms[0])) != "none" {
		return reject("explicit t:none is required")
	}
	lowercase := false
	for i, raw := range in.Transforms {
		t := strings.ToLower(strings.TrimSpace(raw))
		if i == 0 && t == "none" {
			continue
		}
		switch t {
		case "lowercase":
			lowercase = true
		default:
			return reject("unsupported transform: " + raw)
		}
	}
	return Result{
		Eligible:  true,
		Reason:    "exact supported runtime scope",
		Phase:     phase,
		Source:    source,
		Header:    header,
		Lowercase: lowercase,
	}
}

func reject(reason string) Result { return Result{Eligible: false, Reason: reason} }

func classifySelector(raw string) (source, header string, ok bool) {
	v := strings.TrimSpace(raw)
	// Selector combination/exclusion/count syntax cannot be reproduced by the
	// current single-value adapter.
	if strings.ContainsAny(v, "|!&") {
		return "", "", false
	}
	switch v {
	case SourceRequestURI, SourceRequestFilename, SourceRequestMethod, SourceRequestProtocol:
		return v, "", true
	}
	const prefix = SourceRequestHeader + ":"
	if !strings.HasPrefix(v, prefix) {
		return "", "", false
	}
	h := strings.TrimSpace(strings.TrimPrefix(v, prefix))
	// Preserve the runtime adapter's established conservative fixed-header
	// boundary. It is intentionally narrower than the full RFC token grammar.
	if h == "" || strings.ContainsAny(h, " /\\\"'|!&") {
		return "", "", false
	}
	return SourceRequestHeader, h, true
}
