package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	schemaCandidateMinSamples = int64(20)
	schemaMaxEnumValues       = 16
	schemaMaxDistinctHashes   = 256
	schemaMaxDepth            = 12
	schemaMaxFields           = 256
	schemaObservationBodyMax  = 32 << 10
)

var regexpSafeEnum = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_.-]{0,31}$`)

type SchemaTypeEvidence struct {
	Type       string  `json:"type"`
	Count      int64   `json:"count"`
	Confidence float64 `json:"confidence"`
}

type SchemaFieldObservation struct {
	Path                string               `json:"path"`
	Location            string               `json:"location,omitempty"` // body | query | path | header
	TypeCounts          map[string]int64     `json:"type_counts"`
	Types               []SchemaTypeEvidence `json:"types,omitempty"`
	Formats             []string             `json:"formats,omitempty"`
	Samples             int64                `json:"samples"`
	PresenceRate        float64              `json:"presence_rate"`
	DistinctCardinality int                  `json:"distinct_value_cardinality,omitempty"`
	RequiredCandidate   bool                 `json:"required_candidate,omitempty"`
	EnumCandidate       []string             `json:"enum_candidate,omitempty"`
	Sensitive           bool                 `json:"sensitive"`
}

type SchemaReviewRecommendation struct {
	Recommendation   string    `json:"recommendation"`
	Confidence       int       `json:"confidence"`
	Reason           string    `json:"reason"`
	ReviewedAt       time.Time `json:"reviewed_at"`
	Source           string    `json:"source"` // openai | operator
	CandidateVersion int       `json:"candidate_version,omitempty"`
}

type SchemaCandidate struct {
	ID          string                      `json:"id"`
	OperationID string                      `json:"operation_id"`
	Status      string                      `json:"status"` // LEARNING | CANDIDATE | REVIEWED
	Confidence  float64                     `json:"confidence"`
	SampleCount int64                       `json:"sample_count"`
	Fields      []SchemaFieldObservation    `json:"fields"`
	Review      *SchemaReviewRecommendation `json:"review,omitempty"`
	CreatedAt   time.Time                   `json:"created_at"`
	UpdatedAt   time.Time                   `json:"updated_at"`
	Version     int                         `json:"version"`
}

type schemaFieldSample struct {
	Path        string
	Location    string
	Type        string
	Format      string
	EnumValue   string
	Sensitive   bool
	ValueHash   string
	objectValue string // transient API-7.1 input; never persisted
}

type apiObservationMeta struct {
	Host          string
	AuthScheme    string
	Headers       map[string][]string
	BodyPrefix    []byte
	BodyTruncated bool
}

type schemaFieldAggregate struct {
	Path           string           `json:"path"`
	Location       string           `json:"location"`
	SeenRequests   int64            `json:"seen_requests"`
	TypeCounts     map[string]int64 `json:"type_counts"`
	FormatCounts   map[string]int64 `json:"format_counts"`
	EnumCounts     map[string]int64 `json:"enum_counts,omitempty"`
	DistinctHashes map[string]bool  `json:"distinct_hashes,omitempty"`
	Sensitive      bool             `json:"sensitive"`
}

type schemaAggregateState struct {
	OperationID string                 `json:"operation_id"`
	Requests    int64                  `json:"requests"`
	Fields      []schemaFieldAggregate `json:"fields"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type schemaAggregate struct {
	OperationID string
	Requests    int64
	Fields      map[string]*schemaFieldAggregate // location|path
	UpdatedAt   time.Time
}

type schemaStore struct {
	mu         sync.RWMutex
	candidates map[string]*SchemaCandidate // candidate ID -> candidate
	aggregates map[string]*schemaAggregate // operation ID -> aggregate
}

func newSchemaStore() *schemaStore {
	return &schemaStore{candidates: map[string]*SchemaCandidate{}, aggregates: map[string]*schemaAggregate{}}
}

func schemaCandidateID(operationID string) string {
	h := sha256.Sum256([]byte("schema|" + operationID))
	return hex.EncodeToString(h[:16])
}

func (s *schemaStore) note(op apiOperation, samples []schemaFieldSample) {
	if s == nil || op.ID == "" || op.Ignored {
		return
	}
	now := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()

	agg := s.aggregates[op.ID]
	if agg == nil {
		agg = &schemaAggregate{OperationID: op.ID, Fields: map[string]*schemaFieldAggregate{}}
		s.aggregates[op.ID] = agg
	}
	agg.Requests++
	agg.UpdatedAt = now

	seenThisRequest := map[string]bool{}
	for _, sample := range samples {
		if sample.Path == "" || sample.Location == "" {
			continue
		}
		key := sample.Location + "|" + sample.Path
		field := agg.Fields[key]
		if field == nil {
			// Bound aggregate growth across requests as well as within one
			// request. Query/header field names are attacker-controlled, so a
			// per-request cap alone would still permit unbounded long-lived state.
			if len(agg.Fields) >= schemaMaxFields {
				continue
			}
			field = &schemaFieldAggregate{
				Path: sample.Path, Location: sample.Location,
				TypeCounts: map[string]int64{}, FormatCounts: map[string]int64{},
				EnumCounts: map[string]int64{}, DistinctHashes: map[string]bool{},
			}
			agg.Fields[key] = field
		}
		field.Sensitive = field.Sensitive || sample.Sensitive
		if !seenThisRequest[key] {
			field.SeenRequests++
			seenThisRequest[key] = true
		}
		if sample.Type != "" {
			field.TypeCounts[sample.Type]++
		}
		if sample.Format != "" {
			field.FormatCounts[sample.Format]++
		}
		if !field.Sensitive && sample.EnumValue != "" {
			if _, exists := field.EnumCounts[sample.EnumValue]; exists || len(field.EnumCounts) < schemaMaxEnumValues {
				field.EnumCounts[sample.EnumValue]++
			}
		}
		if !field.Sensitive && sample.ValueHash != "" && len(field.DistinctHashes) < schemaMaxDistinctHashes {
			field.DistinctHashes[sample.ValueHash] = true
		}
	}

	id := schemaCandidateID(op.ID)
	old := s.candidates[id]
	candidate := buildCandidateFromAggregate(agg, old, now)
	s.candidates[id] = &candidate
}

func buildCandidateFromAggregate(agg *schemaAggregate, old *SchemaCandidate, now time.Time) SchemaCandidate {
	candidate := SchemaCandidate{
		ID: schemaCandidateID(agg.OperationID), OperationID: agg.OperationID,
		Status: "LEARNING", SampleCount: agg.Requests, UpdatedAt: now, Version: 1,
	}
	if old != nil {
		candidate.CreatedAt = old.CreatedAt
		candidate.Review = old.Review
		candidate.Version = old.Version
		if candidate.Version < 1 {
			candidate.Version = 1
		}
	}
	if candidate.CreatedAt.IsZero() {
		candidate.CreatedAt = now
	}

	keys := make([]string, 0, len(agg.Fields))
	for k := range agg.Fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var confidenceTotal float64
	for _, key := range keys {
		f := agg.Fields[key]
		obs := SchemaFieldObservation{
			Path: f.Path, Location: f.Location, Samples: sumCounts(f.TypeCounts), Sensitive: f.Sensitive,
			TypeCounts: cloneStringIntMap(f.TypeCounts), DistinctCardinality: len(f.DistinctHashes),
		}
		if agg.Requests > 0 {
			obs.PresenceRate = float64(f.SeenRequests) / float64(agg.Requests)
		}
		obs.RequiredCandidate = agg.Requests >= schemaCandidateMinSamples && obs.PresenceRate >= 0.995
		obs.Formats = sortedCountKeys(f.FormatCounts)
		totalTypes := sumCounts(f.TypeCounts)
		for _, typ := range sortedCountKeys(f.TypeCounts) {
			count := f.TypeCounts[typ]
			conf := 0.0
			if totalTypes > 0 {
				conf = float64(count) / float64(totalTypes)
			}
			obs.Types = append(obs.Types, SchemaTypeEvidence{Type: typ, Count: count, Confidence: conf})
		}
		if !f.Sensitive && len(f.EnumCounts) > 0 && len(f.EnumCounts) <= schemaMaxEnumValues && sumCounts(f.EnumCounts) >= schemaCandidateMinSamples {
			obs.EnumCandidate = sortedCountKeys(f.EnumCounts)
		}
		fieldConf := dominantCountFraction(f.TypeCounts)
		sampleFactor := float64(f.SeenRequests) / float64(schemaCandidateMinSamples)
		if sampleFactor > 1 {
			sampleFactor = 1
		}
		confidenceTotal += fieldConf * sampleFactor
		candidate.Fields = append(candidate.Fields, obs)
	}
	if len(candidate.Fields) > 0 {
		candidate.Confidence = confidenceTotal / float64(len(candidate.Fields))
	}

	eligible := agg.Requests >= schemaCandidateMinSamples
	if old == nil {
		if eligible {
			candidate.Status = "CANDIDATE"
		}
		return candidate
	}

	semanticChanged := !schemaCandidateSemanticallyEqual(old, &candidate)
	thresholdChanged := old.Status == "LEARNING" && eligible
	if semanticChanged || thresholdChanged {
		candidate.Version = old.Version + 1
		if candidate.Version < 1 {
			candidate.Version = 1
		}
		if eligible {
			candidate.Status = "CANDIDATE"
		}
		return candidate
	}

	// New samples that reinforce the same learned schema update confidence and
	// counters but do not invalidate an operator's review. A review is tied to
	// a semantic candidate version, not to an exact request count.
	candidate.Status = old.Status
	return candidate
}

func schemaCandidateSemanticallyEqual(a, b *SchemaCandidate) bool {
	if a == nil || b == nil || len(a.Fields) != len(b.Fields) {
		return false
	}
	for i := range a.Fields {
		af, bf := a.Fields[i], b.Fields[i]
		if af.Path != bf.Path || af.Location != bf.Location || af.Sensitive != bf.Sensitive || af.RequiredCandidate != bf.RequiredCandidate {
			return false
		}
		if !sameSortedStrings(sortedCountKeys(af.TypeCounts), sortedCountKeys(bf.TypeCounts)) ||
			dominantObservedType(af.TypeCounts) != dominantObservedType(bf.TypeCounts) ||
			!sameSortedStrings(af.Formats, bf.Formats) ||
			!sameSortedStrings(af.EnumCandidate, bf.EnumCandidate) {
			return false
		}
	}
	return true
}

func sameSortedStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func sumCounts(m map[string]int64) int64 {
	var total int64
	for _, n := range m {
		total += n
	}
	return total
}

func dominantCountFraction(m map[string]int64) float64 {
	var total, best int64
	for _, n := range m {
		total += n
		if n > best {
			best = n
		}
	}
	if total == 0 {
		return 0
	}
	return float64(best) / float64(total)
}

func dominantObservedType(counts map[string]int64) string {
	var best string
	var bestCount int64
	for typ, count := range counts {
		if count > bestCount || (count == bestCount && typ < best) {
			best = typ
			bestCount = count
		}
	}
	return best
}

func sortedCountKeys(m map[string]int64) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (s *schemaStore) list(filter string) []*SchemaCandidate {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*SchemaCandidate, 0, len(s.candidates))
	for _, c := range s.candidates {
		if filter == "" || strings.Contains(c.OperationID, filter) || strings.Contains(c.ID, filter) {
			cp := cloneSchemaCandidate(c)
			out = append(out, cp)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Confidence != out[j].Confidence {
			return out[i].Confidence > out[j].Confidence
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func cloneSchemaCandidate(c *SchemaCandidate) *SchemaCandidate {
	if c == nil {
		return nil
	}
	cp := *c
	cp.Fields = make([]SchemaFieldObservation, len(c.Fields))
	for i := range c.Fields {
		cp.Fields[i] = c.Fields[i]
		cp.Fields[i].TypeCounts = cloneStringIntMap(c.Fields[i].TypeCounts)
		cp.Fields[i].Types = append([]SchemaTypeEvidence(nil), c.Fields[i].Types...)
		cp.Fields[i].Formats = append([]string(nil), c.Fields[i].Formats...)
		cp.Fields[i].EnumCandidate = append([]string(nil), c.Fields[i].EnumCandidate...)
	}
	if c.Review != nil {
		r := *c.Review
		cp.Review = &r
	}
	return &cp
}

func (s *schemaStore) get(id string) (*SchemaCandidate, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.candidates[id]
	return cloneSchemaCandidate(c), ok
}

func (s *schemaStore) setReview(id, status string, review SchemaReviewRecommendation) (*SchemaCandidate, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.candidates[id]
	if !ok {
		return nil, false
	}
	if status != "" {
		c.Status = status
	}
	review.CandidateVersion = c.Version
	c.Review = &review
	c.UpdatedAt = time.Now().UTC()
	return cloneSchemaCandidate(c), true
}

func (s *schemaStore) persistenceSnapshot() schemaStateFile {
	s.mu.RLock()
	defer s.mu.RUnlock()
	state := schemaStateFile{Candidates: make([]*SchemaCandidate, 0, len(s.candidates)), Aggregates: make([]schemaAggregateState, 0, len(s.aggregates))}
	for _, c := range s.candidates {
		state.Candidates = append(state.Candidates, cloneSchemaCandidate(c))
	}
	for _, agg := range s.aggregates {
		as := schemaAggregateState{OperationID: agg.OperationID, Requests: agg.Requests, UpdatedAt: agg.UpdatedAt}
		for _, f := range agg.Fields {
			fc := *f
			fc.TypeCounts = cloneStringIntMap(f.TypeCounts)
			fc.FormatCounts = cloneStringIntMap(f.FormatCounts)
			fc.EnumCounts = cloneStringIntMap(f.EnumCounts)
			fc.DistinctHashes = map[string]bool{}
			for k, v := range f.DistinctHashes {
				fc.DistinctHashes[k] = v
			}
			as.Fields = append(as.Fields, fc)
		}
		sort.Slice(as.Fields, func(i, j int) bool {
			if as.Fields[i].Location != as.Fields[j].Location {
				return as.Fields[i].Location < as.Fields[j].Location
			}
			return as.Fields[i].Path < as.Fields[j].Path
		})
		state.Aggregates = append(state.Aggregates, as)
	}
	return state
}

func (s *schemaStore) restorePersistence(state schemaStateFile) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, c := range state.Candidates {
		if c == nil || c.ID == "" {
			continue
		}
		s.candidates[c.ID] = cloneSchemaCandidate(c)
	}
	for _, as := range state.Aggregates {
		if as.OperationID == "" {
			continue
		}
		agg := &schemaAggregate{OperationID: as.OperationID, Requests: as.Requests, UpdatedAt: as.UpdatedAt, Fields: map[string]*schemaFieldAggregate{}}
		for i := range as.Fields {
			f := as.Fields[i]
			if f.TypeCounts == nil {
				f.TypeCounts = map[string]int64{}
			}
			if f.FormatCounts == nil {
				f.FormatCounts = map[string]int64{}
			}
			if f.EnumCounts == nil {
				f.EnumCounts = map[string]int64{}
			}
			if f.DistinctHashes == nil {
				f.DistinctHashes = map[string]bool{}
			}
			fc := f
			agg.Fields[f.Location+"|"+f.Path] = &fc
		}
		s.aggregates[as.OperationID] = agg
	}
	return nil
}

func (a *adminServer) handleSchemaCandidates(w http.ResponseWriter, r *http.Request) {
	filter := r.URL.Query().Get("q")
	writeJSON(w, a.srv.schema.list(filter))
}

func (a *adminServer) handleSchemaDetail(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		id = strings.TrimPrefix(r.URL.Path, "/api/security/schema/")
	}
	c, ok := a.srv.schema.get(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, c)
}

func inferAuthScheme(r *http.Request) string {
	if r == nil {
		return "none"
	}
	var families []string
	if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
		families = append(families, "mtls")
	}
	if r.Header.Get("X-API-Key") != "" || r.Header.Get("Api-Key") != "" {
		families = append(families, "apikey")
	}
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if auth != "" {
		parts := strings.Fields(auth)
		if len(parts) == 0 {
			families = append(families, "other")
		} else {
			families = append(families, normalizeAuthScheme(parts[0]))
		}
	}
	return normalizeAuthScheme(strings.Join(families, "+"))
}

func buildAPIObservationMeta(r *http.Request, rawQuery, contentType string) apiObservationMeta {
	meta := apiObservationMeta{AuthScheme: inferAuthScheme(r)}
	if r != nil {
		meta.Host = r.Host
		meta.Headers = selectedSchemaHeaders(r)
		prefix := requestBodyPrefixFromRequest(r)
		if len(prefix) > schemaObservationBodyMax {
			meta.BodyTruncated = true
			prefix = prefix[:schemaObservationBodyMax]
		}
		if len(prefix) > 0 {
			// Keep the observation event independent from the request capture's
			// backing array. The bounded queue owns this small immutable copy.
			meta.BodyPrefix = append([]byte(nil), prefix...)
		}
	}
	return meta
}

func collectSchemaSamplesFromObservation(rawPath, rawQuery, contentType string, meta apiObservationMeta) []schemaFieldSample {
	out := make([]schemaFieldSample, 0, 32)
	addPathParameterSamples(&out, rawPath)
	addHeaderValueSamples(&out, meta.Headers)
	addQuerySamples(&out, rawQuery)
	if len(meta.BodyPrefix) > 0 && !meta.BodyTruncated {
		addBodySamples(&out, contentType, meta.BodyPrefix)
	}
	if len(out) > schemaMaxFields {
		out = out[:schemaMaxFields]
	}
	return out
}

func addPathParameterSamples(out *[]schemaFieldSample, rawPath string) {
	normalized := normalizeAPIOperationPath(rawPath)
	raw := strings.Split(rawPath, "/")
	norm := strings.Split(normalized, "/")
	if len(raw) != len(norm) {
		return
	}
	ordinal := 0
	for i := range raw {
		if raw[i] == norm[i] {
			continue
		}
		ordinal++
		typ, format := inferScalarType(raw[i])
		appendSchemaSample(out, fmt.Sprintf("param_%d", ordinal), "path", typ, format, "", raw[i])
	}
}

func addQuerySamples(out *[]schemaFieldSample, rawQuery string) {
	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		return
	}
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, name := range keys {
		for _, value := range values[name] {
			typ, format := inferScalarType(value)
			appendSchemaSample(out, name, "query", typ, format, safeEnumValue(name, value), value)
		}
	}
}

func selectedSchemaHeaders(r *http.Request) map[string][]string {
	if r == nil {
		return nil
	}
	out := map[string][]string{}
	for k := range r.Header {
		lk := strings.ToLower(k)
		if isSensitiveFieldName(lk) {
			continue
		}
		if lk == "content-type" || lk == "accept" || strings.HasPrefix(lk, "x-") {
			vals := r.Header.Values(k)
			if len(vals) > 0 {
				out[lk] = append([]string(nil), vals...)
			}
		}
	}
	return out
}

func addHeaderValueSamples(out *[]schemaFieldSample, headers map[string][]string) {
	keys := make([]string, 0, len(headers))
	for k := range headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, name := range keys {
		for _, value := range headers[name] {
			typ, format := inferScalarType(value)
			enum := ""
			if strings.EqualFold(name, "Content-Type") || strings.EqualFold(name, "Accept") {
				typ, format = "string", ""
				enum = normalizeMediaType(value)
			}
			// Header values can carry identity, forwarding, tracing, or vendor
			// secrets even when the name itself looks harmless. Learn shape only;
			// never retain a header-value fingerprint for cardinality.
			appendSchemaSample(out, name, "header", typ, format, enum, "")
		}
	}
}

func addBodySamples(out *[]schemaFieldSample, contentType string, body []byte) {
	media, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return
	}
	switch strings.ToLower(media) {
	case "application/json", "application/merge-patch+json":
		var v any
		dec := json.NewDecoder(bytes.NewReader(body))
		dec.UseNumber()
		if dec.Decode(&v) == nil {
			flattenJSONSamples(out, "", v, 0)
		}
	case "application/x-www-form-urlencoded":
		values, err := url.ParseQuery(string(body))
		if err != nil {
			return
		}
		keys := make([]string, 0, len(values))
		for k := range values {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, name := range keys {
			for _, value := range values[name] {
				typ, format := inferScalarType(value)
				appendSchemaSample(out, name, "body", typ, format, safeEnumValue(name, value), value)
			}
		}
	case "multipart/form-data":
		boundary := params["boundary"]
		if boundary == "" {
			return
		}
		mr := multipart.NewReader(bytes.NewReader(body), boundary)
		for len(*out) < schemaMaxFields {
			part, err := mr.NextPart()
			if err != nil {
				break
			}
			name := part.FormName()
			if name == "" {
				_ = part.Close()
				continue
			}
			if part.FileName() != "" {
				appendSchemaSample(out, name, "body", "file", "", "", "")
			} else {
				b, _ := io.ReadAll(io.LimitReader(part, 4096))
				value := string(b)
				typ, format := inferScalarType(value)
				appendSchemaSample(out, name, "body", typ, format, safeEnumValue(name, value), value)
			}
			_ = part.Close()
		}
	}
}

func schemaFallbackPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "$"
	}
	return path
}

func flattenJSONSamples(out *[]schemaFieldSample, prefix string, v any, depth int) {
	if depth > schemaMaxDepth || len(*out) >= schemaMaxFields {
		return
	}
	switch x := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			p := k
			if prefix != "" {
				p = prefix + "." + k
			}
			flattenJSONSamples(out, p, x[k], depth+1)
		}
	case []any:
		// Record the array container itself and, for non-empty arrays, a
		// bounded sample of item shape. This lets the learned schema preserve
		// that `items` is an array instead of only exposing `items[].field`.
		appendSchemaSample(out, schemaFallbackPath(prefix), "body", "array", "", "", "")
		if len(x) == 0 {
			return
		}
		p := prefix + "[]"
		limit := len(x)
		if limit > 8 {
			limit = 8
		}
		for i := 0; i < limit; i++ {
			flattenJSONSamples(out, p, x[i], depth+1)
		}
	case nil:
		appendSchemaSample(out, schemaFallbackPath(prefix), "body", "null", "", "", "")
	case bool:
		appendSchemaSample(out, schemaFallbackPath(prefix), "body", "boolean", "", strconv.FormatBool(x), strconv.FormatBool(x))
	case json.Number:
		typ := "number"
		if _, err := x.Int64(); err == nil {
			typ = "integer"
		}
		appendSchemaSample(out, schemaFallbackPath(prefix), "body", typ, "", "", x.String())
	case string:
		typ, format := inferScalarType(x)
		appendSchemaSample(out, schemaFallbackPath(prefix), "body", typ, format, safeEnumValue(prefix, x), x)
	default:
		appendSchemaSample(out, schemaFallbackPath(prefix), "body", "unknown", "", "", "")
	}
}

func appendSchemaSample(out *[]schemaFieldSample, path, location, typ, format, enumValue, raw string) {
	if len(*out) >= schemaMaxFields || path == "" {
		return
	}
	sensitive := isSensitiveFieldName(path)
	if sensitive || format == "email" || format == "ip" || format == "uuid" || format == "date-time" {
		enumValue = ""
		raw = ""
	}
	hash := ""
	if raw != "" && !sensitive && (typ != "string" || enumValue != "") {
		h := sha256.Sum256([]byte(raw))
		hash = hex.EncodeToString(h[:8])
	}
	*out = append(*out, schemaFieldSample{Path: path, Location: location, Type: typ, Format: format, EnumValue: enumValue, Sensitive: sensitive, ValueHash: hash, objectValue: raw})
}

func isSensitiveFieldName(path string) bool {
	p := strings.ToLower(path)
	// Normalize common separators before matching so credential names such as
	// X-API-Key, api_key, access-key, and private.key cannot evade the
	// privacy boundary simply by changing punctuation.
	canonical := strings.NewReplacer("_", "", "-", "", ".", "", " ", "", "[", "", "]", "").Replace(p)
	for _, token := range []string{"password", "passwd", "secret", "token", "authorization", "cookie", "session", "apikey", "accesskey", "privatekey", "creditcard", "cardnumber", "cvv", "ssn", "email", "phone", "address", "name", "memo", "comment", "message", "note"} {
		if strings.Contains(canonical, token) {
			return true
		}
	}
	return false
}

func safeEnumValue(path, value string) string {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 64 || isSensitiveFieldName(path) || !isEnumLikeFieldName(path) {
		return ""
	}
	if !regexpSafeEnum.MatchString(value) {
		return ""
	}
	_, format := inferScalarType(value)
	if format != "" && format != "date" {
		return ""
	}
	if strings.Contains(value, "@") || net.ParseIP(value) != nil {
		return ""
	}
	if apiUUID.MatchString(value) || apiULID.MatchString(value) || apiHex.MatchString(value) || apiToken.MatchString(value) {
		return ""
	}
	return value
}

func isEnumLikeFieldName(path string) bool {
	p := strings.ToLower(path)
	parts := strings.FieldsFunc(p, func(r rune) bool { return r == '.' || r == '_' || r == '-' || r == '[' || r == ']' })
	for _, part := range parts {
		switch part {
		case "status", "state", "type", "kind", "role", "scope", "currency", "country", "region", "locale", "language", "mode", "tier", "level", "category", "action", "channel", "provider", "environment", "format", "code":
			return true
		}
	}
	return false
}

func inferScalarType(value string) (string, string) {
	v := strings.TrimSpace(value)
	if v == "" {
		return "string", ""
	}
	if apiUUID.MatchString(v) {
		return "string", "uuid"
	}
	if apiDate.MatchString(v) {
		return "string", "date"
	}
	if t, err := time.Parse(time.RFC3339, v); err == nil && !t.IsZero() {
		return "string", "date-time"
	}
	if net.ParseIP(v) != nil {
		return "string", "ip"
	}
	if strings.Contains(v, "@") && strings.Contains(strings.SplitN(v, "@", 2)[1], ".") {
		return "string", "email"
	}
	if v == "true" || v == "false" {
		return "boolean", ""
	}
	if _, err := strconv.ParseInt(v, 10, 64); err == nil {
		return "integer", ""
	}
	if _, err := strconv.ParseFloat(v, 64); err == nil {
		return "number", ""
	}
	return "string", ""
}

func (s *schemaStore) findByOperationID(operationID string) (*SchemaCandidate, bool) {
	if s == nil {
		return nil, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, c := range s.candidates {
		if c.OperationID == operationID {
			return cloneSchemaCandidate(c), true
		}
	}
	return nil, false
}
