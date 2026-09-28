package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// API-1: normalized API operation discovery.
// This is telemetry/control-plane state only. It never participates in the
// blocking WAF verdict.

type APIPathParameter struct {
	Segment int    `json:"segment"`
	Name    string `json:"name"`
	Kind    string `json:"kind"`
	Format  string `json:"format,omitempty"`
}

type apiOperation struct {
	ID              string             `json:"id"`
	Site            string             `json:"site"`
	Host            string             `json:"host,omitempty"`
	Method          string             `json:"method"`
	Path            string             `json:"path"`
	Fingerprint     string             `json:"fingerprint"`
	Classification  string             `json:"classification,omitempty"`
	Ignored         bool               `json:"ignored,omitempty"`
	Confidence      float64            `json:"confidence"`
	Samples         int64              `json:"samples"`
	Status          map[string]int64   `json:"status_distribution,omitempty"`
	ContentTypes    map[string]int64   `json:"content_types,omitempty"`
	AuthObserved    map[string]int64   `json:"auth_observed,omitempty"`
	RawPathExamples []string           `json:"raw_path_examples,omitempty"`
	PathParameters  []APIPathParameter `json:"path_parameters,omitempty"`
	FirstSeen       string             `json:"first_seen"`
	LastSeen        string             `json:"last_seen"`
}

type apiOperationStore struct {
	mu  sync.RWMutex
	ops map[string]*apiOperation // ID -> operation
	cap int
}

func newAPIOperationStore() *apiOperationStore {
	return &apiOperationStore{ops: make(map[string]*apiOperation), cap: 10000}
}

var (
	apiUUID  = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	apiHex   = regexp.MustCompile(`(?i)^[0-9a-f]{16,}$`)
	apiNum   = regexp.MustCompile(`^[0-9]+$`)
	apiDate  = regexp.MustCompile(`^[0-9]{4}[-]?[0-9]{2}[-]?[0-9]{2}$`)
	apiULID  = regexp.MustCompile(`(?i)^[0-9ABCDEFGHJKMNPQRSTVWXYZ]{26}$`)
	apiSlug  = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]{2,63}$`)
	apiToken = regexp.MustCompile(`^[A-Za-z0-9_-]{20,128}$`)
)

func operationFingerprint(method, path string) string {
	h := sha256.Sum256([]byte(strings.ToUpper(method) + " " + path))
	return hex.EncodeToString(h[:])
}

func apiOperationID(site, method, path string) string {
	h := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(site)) + "|" + strings.ToUpper(method) + "|" + path))
	return hex.EncodeToString(h[:16])
}

func classifyPathSegment(p string) (placeholder, kind, format string, dynamic bool) {
	switch {
	case apiUUID.MatchString(p):
		return "{id}", "identifier", "uuid", true
	case apiULID.MatchString(p):
		return "{id}", "identifier", "ulid", true
	case apiDate.MatchString(p):
		return "{date}", "date", "date", true
	case apiNum.MatchString(p):
		return "{id}", "identifier", "integer", true
	case apiHex.MatchString(p):
		return "{id}", "identifier", "hex", true
	case apiToken.MatchString(p) && hasMixedCharacterClasses(p):
		return "{id}", "opaque_token", "opaque", true
	default:
		return p, "", "", false
	}
}

func hasMixedCharacterClasses(s string) bool {
	var alpha, digit bool
	for _, r := range s {
		if r >= '0' && r <= '9' {
			digit = true
		} else if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			alpha = true
		}
	}
	return alpha && digit
}

func normalizeAPIOperationPath(path string) string {
	if path == "" {
		return "/"
	}
	parts := strings.Split(path, "/")
	for i, p := range parts {
		if placeholder, _, _, dynamic := classifyPathSegment(p); dynamic {
			parts[i] = placeholder
		}
	}
	return strings.Join(parts, "/")
}

func inferPathParameters(raw, normalized string) []APIPathParameter {
	rawParts := strings.Split(raw, "/")
	normParts := strings.Split(normalized, "/")
	if len(rawParts) != len(normParts) {
		return nil
	}
	out := make([]APIPathParameter, 0, 2)
	ordinal := 0
	for i := range rawParts {
		if rawParts[i] == normParts[i] {
			continue
		}
		_, kind, format, dynamic := classifyPathSegment(rawParts[i])
		if !dynamic {
			kind = "dynamic"
		}
		ordinal++
		out = append(out, APIPathParameter{Segment: i, Name: fmt.Sprintf("param_%d", ordinal), Kind: kind, Format: format})
	}
	return out
}

func normalizeAuthScheme(s string) string {
	parts := strings.Split(strings.ToLower(strings.TrimSpace(s)), "+")
	seen := map[string]bool{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		switch part {
		case "bearer", "basic", "apikey", "mtls", "other":
			seen[part] = true
		case "none", "":
			// `none` is only meaningful when no positive auth family exists.
		default:
			seen["other"] = true
		}
	}
	if len(seen) == 0 {
		return "none"
	}
	keys := make([]string, 0, len(seen))
	for k := range seen {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return strings.Join(keys, "+")
}

func (s *apiOperationStore) note(site, host, method, path, contentType, authScheme string, code int) apiOperation {
	if s == nil {
		return apiOperation{}
	}
	now := time.Now().UTC().Format(time.RFC3339)
	method = strings.ToUpper(method)
	normalized := normalizeAPIOperationPath(path)
	id := apiOperationID(site, method, normalized)

	s.mu.Lock()
	defer s.mu.Unlock()
	op := s.ops[id]
	if op == nil {
		if len(s.ops) >= s.cap {
			return apiOperation{}
		}
		op = &apiOperation{
			ID:             id,
			Site:           site,
			Host:           normalizeHost(host),
			Method:         method,
			Path:           normalized,
			Fingerprint:    operationFingerprint(method, normalized),
			Classification: "API",
			Confidence:     normalizationConfidence(path, normalized),
			Status:         map[string]int64{},
			ContentTypes:   map[string]int64{},
			AuthObserved:   map[string]int64{},
			PathParameters: inferPathParameters(path, normalized),
			FirstSeen:      now,
		}
		s.ops[id] = op
	}
	if op.Host == "" {
		op.Host = normalizeHost(host)
	}
	op.Samples++
	op.Status[formatStatus(code)]++
	if ct := normalizeMediaType(contentType); ct != "" {
		op.ContentTypes[ct]++
	}
	op.AuthObserved[normalizeAuthScheme(authScheme)]++
	appendUniqueBounded(&op.RawPathExamples, path, 8)
	if len(op.PathParameters) == 0 {
		op.PathParameters = inferPathParameters(path, op.Path)
	}
	op.LastSeen = now
	return cloneAPIOperation(*op)
}

func normalizeMediaType(contentType string) string {
	contentType = strings.TrimSpace(strings.ToLower(contentType))
	if i := strings.IndexByte(contentType, ';'); i >= 0 {
		contentType = strings.TrimSpace(contentType[:i])
	}
	return contentType
}

func appendUniqueBounded(dst *[]string, value string, limit int) {
	value = strings.TrimSpace(value)
	if value == "" || limit <= 0 {
		return
	}
	for _, cur := range *dst {
		if cur == value {
			return
		}
	}
	if len(*dst) >= limit {
		return
	}
	*dst = append(*dst, value)
}

func formatStatus(code int) string {
	if code <= 0 {
		return "unknown"
	}
	return strconv.Itoa(code)
}

func cloneStringIntMap(in map[string]int64) map[string]int64 {
	if len(in) == 0 {
		return map[string]int64{}
	}
	out := make(map[string]int64, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneAPIOperation(in apiOperation) apiOperation {
	in.Status = cloneStringIntMap(in.Status)
	in.ContentTypes = cloneStringIntMap(in.ContentTypes)
	in.AuthObserved = cloneStringIntMap(in.AuthObserved)
	in.RawPathExamples = append([]string(nil), in.RawPathExamples...)
	in.PathParameters = append([]APIPathParameter(nil), in.PathParameters...)
	return in
}

func (s *apiOperationStore) snapshot() []apiOperation {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]apiOperation, 0, len(s.ops))
	for _, v := range s.ops {
		out = append(out, cloneAPIOperation(*v))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Samples != out[j].Samples {
			return out[i].Samples > out[j].Samples
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func (s *apiOperationStore) get(id string) (apiOperation, bool) {
	if s == nil {
		return apiOperation{}, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	op, ok := s.ops[id]
	if !ok {
		return apiOperation{}, false
	}
	return cloneAPIOperation(*op), true
}

func (s *apiOperationStore) ignore(id string, ignored bool) (apiOperation, error) {
	if s == nil {
		return apiOperation{}, errors.New("operation store unavailable")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	op := s.ops[id]
	if op == nil {
		return apiOperation{}, errors.New("operation not found")
	}
	op.Ignored = ignored
	return cloneAPIOperation(*op), nil
}

func (s *apiOperationStore) reclassify(id, normalizedPath, classification string) (apiOperation, error) {
	if s == nil {
		return apiOperation{}, errors.New("operation store unavailable")
	}
	normalizedPath = strings.TrimSpace(normalizedPath)
	if normalizedPath == "" || !strings.HasPrefix(normalizedPath, "/") || len(normalizedPath) > 2048 || strings.ContainsAny(normalizedPath, "\r\n\t ?#") {
		return apiOperation{}, errors.New("normalized_path must start with /")
	}
	classification = strings.ToUpper(strings.TrimSpace(classification))
	if classification == "" {
		classification = "API"
	}
	switch classification {
	case "API", "NON_API", "STATIC", "UNKNOWN":
	default:
		return apiOperation{}, errors.New("classification must be API, NON_API, STATIC, or UNKNOWN")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	op := s.ops[id]
	if op == nil {
		return apiOperation{}, errors.New("operation not found")
	}
	op.Path = normalizedPath
	op.Fingerprint = operationFingerprint(op.Method, normalizedPath)
	op.Classification = classification
	op.Confidence = 1.0
	return cloneAPIOperation(*op), nil
}

func normalizationConfidence(raw, normalized string) float64 {
	if raw == normalized {
		return 0.5
	}
	return 0.95
}

// classifyPotentialSlug is intentionally exposed to tests and future manual
// reclassification UI. Slugs are not auto-normalized because static resource
// names and API slugs are indistinguishable without cross-request evidence.
func classifyPotentialSlug(segment string) bool { return apiSlug.MatchString(segment) }
