package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	positiveSchemaModeLearn   = "LEARN"
	positiveSchemaModeDetect  = "DETECT"
	positiveSchemaModeEnforce = "ENFORCE"

	positiveSchemaUnknownAllow = "allow"
	positiveSchemaUnknownDeny  = "deny"

	positiveSchemaMaxViolations = 2048
	positiveSchemaMaxHistory    = 32
	positiveSchemaMaxExceptions = 2048
)

type PositiveSchemaField struct {
	Path           string   `json:"path"`
	Location       string   `json:"location"`
	AllowedTypes   []string `json:"allowed_types,omitempty"`
	AllowedFormats []string `json:"allowed_formats,omitempty"`
	AllowedEnum    []string `json:"allowed_enum,omitempty"`
	Required       bool     `json:"required,omitempty"`
	Sensitive      bool     `json:"sensitive,omitempty"`
}

type PositiveSchemaProfileVersion struct {
	ID                 string                `json:"id"`
	OperationID        string                `json:"operation_id"`
	CandidateID        string                `json:"candidate_id"`
	CandidateVersion   int                   `json:"candidate_version"`
	Version            int                   `json:"version"`
	Fields             []PositiveSchemaField `json:"fields"`
	UnknownFields      string                `json:"unknown_fields"` // allow | deny
	EnforceFormats     bool                  `json:"enforce_formats"`
	EnforceEnums       bool                  `json:"enforce_enums"`
	EnforceContentType bool                  `json:"enforce_content_type"`
	AllowedContentType []string              `json:"allowed_content_types,omitempty"`
	CreatedAt          time.Time             `json:"created_at"`
	CreatedBy          string                `json:"created_by,omitempty"`
	Note               string                `json:"note,omitempty"`
}

type PositiveSchemaDeploymentRevision struct {
	ProfileVersionID string    `json:"profile_version_id"`
	Mode             string    `json:"mode"`
	BlockStatus      int       `json:"block_status"`
	ChangedAt        time.Time `json:"changed_at"`
	ChangedBy        string    `json:"changed_by,omitempty"`
	Reason           string    `json:"reason,omitempty"`
}

type PositiveSchemaDeployment struct {
	OperationID      string                             `json:"operation_id"`
	ProfileVersionID string                             `json:"profile_version_id"`
	Mode             string                             `json:"mode"`
	BlockStatus      int                                `json:"block_status"`
	UpdatedAt        time.Time                          `json:"updated_at"`
	UpdatedBy        string                             `json:"updated_by,omitempty"`
	History          []PositiveSchemaDeploymentRevision `json:"history,omitempty"`
}

type PositiveSchemaException struct {
	ID               string     `json:"id"`
	OperationID      string     `json:"operation_id"`
	ProfileVersionID string     `json:"profile_version_id,omitempty"`
	ViolationType    string     `json:"violation_type,omitempty"`
	Field            string     `json:"field,omitempty"` // location|path or empty for all fields
	Reason           string     `json:"reason"`
	Enabled          bool       `json:"enabled"`
	ExpiresAt        *time.Time `json:"expires_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	CreatedBy        string     `json:"created_by,omitempty"`
}

type PositiveSchemaViolation struct {
	ID               string    `json:"id"`
	Time             time.Time `json:"time"`
	OperationID      string    `json:"operation_id"`
	ProfileVersionID string    `json:"profile_version_id"`
	Mode             string    `json:"mode"`
	Action           string    `json:"action"` // learn | detect | block
	Type             string    `json:"type"`
	Location         string    `json:"location,omitempty"`
	Path             string    `json:"path,omitempty"`
	Expected         string    `json:"expected,omitempty"`
	Observed         string    `json:"observed,omitempty"`
	Method           string    `json:"method"`
	RequestPath      string    `json:"request_path"`
	ContentType      string    `json:"content_type,omitempty"`
	Exempted         bool      `json:"exempted,omitempty"`
	ExceptionID      string    `json:"exception_id,omitempty"`
}

type positiveSchemaRuntimePolicy struct {
	Profile    PositiveSchemaProfileVersion
	Deployment PositiveSchemaDeployment
	Exceptions []PositiveSchemaException
	FieldIndex map[string]PositiveSchemaField
}

type positiveSchemaRuntimeSnapshot struct {
	ByOperation map[string]positiveSchemaRuntimePolicy
}

type positiveSchemaStateFile struct {
	Version             int                                     `json:"version"`
	Saved               time.Time                               `json:"saved"`
	Profiles            map[string]PositiveSchemaProfileVersion `json:"profiles"`
	VersionsByOperation map[string][]string                     `json:"versions_by_operation"`
	Deployments         map[string]PositiveSchemaDeployment     `json:"deployments"`
	Exceptions          map[string]PositiveSchemaException      `json:"exceptions"`
	Violations          []PositiveSchemaViolation               `json:"violations,omitempty"`
}

type positiveSchemaStore struct {
	mu                  sync.RWMutex
	profiles            map[string]PositiveSchemaProfileVersion
	versionsByOperation map[string][]string
	deployments         map[string]PositiveSchemaDeployment
	exceptions          map[string]PositiveSchemaException
	violations          []PositiveSchemaViolation
	violationSeq        atomic.Uint64
	runtime             atomic.Pointer[positiveSchemaRuntimeSnapshot]
}

func newPositiveSchemaStore() *positiveSchemaStore {
	s := &positiveSchemaStore{
		profiles:            map[string]PositiveSchemaProfileVersion{},
		versionsByOperation: map[string][]string{},
		deployments:         map[string]PositiveSchemaDeployment{},
		exceptions:          map[string]PositiveSchemaException{},
	}
	s.runtime.Store(&positiveSchemaRuntimeSnapshot{ByOperation: map[string]positiveSchemaRuntimePolicy{}})
	return s
}

func positiveSchemaProfileID(operationID string, version int) string {
	h := sha256.Sum256([]byte("positive-schema|" + operationID + "|" + strconv.Itoa(version)))
	return hex.EncodeToString(h[:16])
}

func positiveSchemaExceptionID(operationID, profileVersionID, violationType, field string, now time.Time) string {
	h := sha256.Sum256([]byte(strings.Join([]string{"exception", operationID, profileVersionID, violationType, field, now.UTC().Format(time.RFC3339Nano)}, "|")))
	return hex.EncodeToString(h[:16])
}

func normalizePositiveSchemaMode(mode string) (string, error) {
	mode = strings.ToUpper(strings.TrimSpace(mode))
	switch mode {
	case positiveSchemaModeLearn, positiveSchemaModeDetect, positiveSchemaModeEnforce:
		return mode, nil
	default:
		return "", fmt.Errorf("mode must be LEARN, DETECT, or ENFORCE")
	}
}

func normalizeUnknownFieldPolicy(policy string) (string, error) {
	policy = strings.ToLower(strings.TrimSpace(policy))
	if policy == "" {
		return positiveSchemaUnknownAllow, nil
	}
	switch policy {
	case positiveSchemaUnknownAllow, positiveSchemaUnknownDeny:
		return policy, nil
	default:
		return "", fmt.Errorf("unknown_fields must be allow or deny")
	}
}

func normalizePositiveSchemaBlockStatus(code int) (int, error) {
	if code == 0 {
		return http.StatusBadRequest, nil
	}
	switch code {
	case http.StatusBadRequest, http.StatusForbidden, http.StatusUnprocessableEntity:
		return code, nil
	default:
		return 0, fmt.Errorf("block_status must be 400, 403, or 422")
	}
}

func candidateToPositiveFields(candidate *SchemaCandidate) []PositiveSchemaField {
	if candidate == nil {
		return nil
	}
	out := make([]PositiveSchemaField, 0, len(candidate.Fields))
	for _, field := range candidate.Fields {
		types := sortedCountKeys(field.TypeCounts)
		if len(types) == 0 {
			for _, t := range field.Types {
				if strings.TrimSpace(t.Type) != "" {
					types = append(types, t.Type)
				}
			}
			sort.Strings(types)
			types = uniqueStrings(types)
		}
		formats := append([]string(nil), field.Formats...)
		sort.Strings(formats)
		formats = uniqueStrings(formats)
		enums := append([]string(nil), field.EnumCandidate...)
		sort.Strings(enums)
		enums = uniqueStrings(enums)
		out = append(out, PositiveSchemaField{
			Path: field.Path, Location: field.Location, AllowedTypes: types,
			AllowedFormats: formats, AllowedEnum: enums, Required: field.RequiredCandidate, Sensitive: field.Sensitive,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Location != out[j].Location {
			return out[i].Location < out[j].Location
		}
		return out[i].Path < out[j].Path
	})
	return out
}

func uniqueStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := in[:0]
	var last string
	for _, v := range in {
		v = strings.TrimSpace(v)
		if v == "" || (len(out) > 0 && v == last) {
			continue
		}
		out = append(out, v)
		last = v
	}
	return out
}

type positiveSchemaPromoteOptions struct {
	UnknownFields      string
	EnforceFormats     bool
	EnforceEnums       bool
	EnforceContentType bool
	CreatedBy          string
	Note               string
	Activate           bool
	BlockStatus        int
}

func (s *positiveSchemaStore) promote(candidate *SchemaCandidate, op apiOperation, opts positiveSchemaPromoteOptions) (PositiveSchemaProfileVersion, PositiveSchemaDeployment, error) {
	if s == nil {
		return PositiveSchemaProfileVersion{}, PositiveSchemaDeployment{}, errors.New("positive schema store unavailable")
	}
	if candidate == nil || candidate.ID == "" || candidate.OperationID == "" {
		return PositiveSchemaProfileVersion{}, PositiveSchemaDeployment{}, errors.New("invalid schema candidate")
	}
	if candidate.Status != "REVIEWED" || candidate.Review == nil || candidate.Review.Source != "operator" || candidate.Review.Recommendation != "approved" || candidate.Review.CandidateVersion != candidate.Version {
		return PositiveSchemaProfileVersion{}, PositiveSchemaDeployment{}, errors.New("candidate must have a current operator approval before enforcement promotion")
	}
	if op.ID == "" || op.ID != candidate.OperationID || op.Ignored {
		return PositiveSchemaProfileVersion{}, PositiveSchemaDeployment{}, errors.New("matching active API operation is required")
	}
	if len(candidate.Fields) == 0 {
		return PositiveSchemaProfileVersion{}, PositiveSchemaDeployment{}, errors.New("reviewed candidate has no enforceable fields")
	}
	if opts.EnforceContentType && len(op.ContentTypes) == 0 {
		return PositiveSchemaProfileVersion{}, PositiveSchemaDeployment{}, errors.New("content-type enforcement requires observed content types")
	}
	unknown, err := normalizeUnknownFieldPolicy(opts.UnknownFields)
	if err != nil {
		return PositiveSchemaProfileVersion{}, PositiveSchemaDeployment{}, err
	}
	blockStatus, err := normalizePositiveSchemaBlockStatus(opts.BlockStatus)
	if err != nil {
		return PositiveSchemaProfileVersion{}, PositiveSchemaDeployment{}, err
	}
	now := time.Now().UTC()

	s.mu.Lock()
	defer s.mu.Unlock()
	version := len(s.versionsByOperation[candidate.OperationID]) + 1
	profile := PositiveSchemaProfileVersion{
		ID: positiveSchemaProfileID(candidate.OperationID, version), OperationID: candidate.OperationID,
		CandidateID: candidate.ID, CandidateVersion: candidate.Version, Version: version,
		Fields: candidateToPositiveFields(candidate), UnknownFields: unknown,
		EnforceFormats: opts.EnforceFormats, EnforceEnums: opts.EnforceEnums,
		EnforceContentType: opts.EnforceContentType, CreatedAt: now, CreatedBy: strings.TrimSpace(opts.CreatedBy), Note: strings.TrimSpace(opts.Note),
	}
	if opts.EnforceContentType {
		profile.AllowedContentType = sortedCountKeys(op.ContentTypes)
	}
	if _, exists := s.profiles[profile.ID]; exists {
		return PositiveSchemaProfileVersion{}, PositiveSchemaDeployment{}, errors.New("profile version already exists")
	}
	s.profiles[profile.ID] = profile
	s.versionsByOperation[candidate.OperationID] = append(s.versionsByOperation[candidate.OperationID], profile.ID)

	deployment := s.deployments[candidate.OperationID]
	if deployment.OperationID == "" || opts.Activate {
		if deployment.OperationID != "" {
			deployment = pushPositiveHistory(deployment, opts.CreatedBy, "activate new profile version")
		}
		deployment.OperationID = candidate.OperationID
		deployment.ProfileVersionID = profile.ID
		deployment.Mode = positiveSchemaModeLearn
		deployment.BlockStatus = blockStatus
		deployment.UpdatedAt = now
		deployment.UpdatedBy = strings.TrimSpace(opts.CreatedBy)
		s.deployments[candidate.OperationID] = deployment
	}
	s.rebuildRuntimeLocked()
	return clonePositiveProfile(profile), clonePositiveDeployment(deployment), nil
}

func pushPositiveHistory(deployment PositiveSchemaDeployment, who, reason string) PositiveSchemaDeployment {
	if deployment.ProfileVersionID == "" {
		return deployment
	}
	deployment.History = append(deployment.History, PositiveSchemaDeploymentRevision{
		ProfileVersionID: deployment.ProfileVersionID, Mode: deployment.Mode, BlockStatus: deployment.BlockStatus,
		ChangedAt: deployment.UpdatedAt, ChangedBy: strings.TrimSpace(who), Reason: strings.TrimSpace(reason),
	})
	if len(deployment.History) > positiveSchemaMaxHistory {
		deployment.History = append([]PositiveSchemaDeploymentRevision(nil), deployment.History[len(deployment.History)-positiveSchemaMaxHistory:]...)
	}
	return deployment
}

func validPositiveModeTransition(from, to string) bool {
	if from == to {
		return true
	}
	switch from {
	case positiveSchemaModeLearn:
		return to == positiveSchemaModeDetect
	case positiveSchemaModeDetect:
		return to == positiveSchemaModeLearn || to == positiveSchemaModeEnforce
	case positiveSchemaModeEnforce:
		return to == positiveSchemaModeDetect || to == positiveSchemaModeLearn
	default:
		return false
	}
}

func (s *positiveSchemaStore) setMode(operationID, mode, who, reason string, blockStatus int) (PositiveSchemaDeployment, error) {
	mode, err := normalizePositiveSchemaMode(mode)
	if err != nil {
		return PositiveSchemaDeployment{}, err
	}
	status, err := normalizePositiveSchemaBlockStatus(blockStatus)
	if err != nil {
		return PositiveSchemaDeployment{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	deployment, ok := s.deployments[operationID]
	if !ok || deployment.ProfileVersionID == "" {
		return PositiveSchemaDeployment{}, errors.New("no active positive schema deployment")
	}
	if !validPositiveModeTransition(deployment.Mode, mode) {
		return PositiveSchemaDeployment{}, fmt.Errorf("invalid mode transition %s -> %s", deployment.Mode, mode)
	}
	if mode == positiveSchemaModeEnforce && deployment.Mode != positiveSchemaModeDetect {
		return PositiveSchemaDeployment{}, errors.New("ENFORCE requires an active DETECT stage first")
	}
	if deployment.Mode != mode || deployment.BlockStatus != status {
		deployment = pushPositiveHistory(deployment, who, reason)
	}
	deployment.Mode = mode
	deployment.BlockStatus = status
	deployment.UpdatedAt = time.Now().UTC()
	deployment.UpdatedBy = strings.TrimSpace(who)
	s.deployments[operationID] = deployment
	s.rebuildRuntimeLocked()
	return clonePositiveDeployment(deployment), nil
}

func (s *positiveSchemaStore) activate(operationID, profileVersionID, who, reason string) (PositiveSchemaDeployment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	profile, ok := s.profiles[profileVersionID]
	if !ok || profile.OperationID != operationID {
		return PositiveSchemaDeployment{}, errors.New("profile version not found for operation")
	}
	deployment := s.deployments[operationID]
	if deployment.OperationID != "" {
		deployment = pushPositiveHistory(deployment, who, reason)
	}
	deployment.OperationID = operationID
	deployment.ProfileVersionID = profileVersionID
	deployment.Mode = positiveSchemaModeLearn
	if deployment.BlockStatus == 0 {
		deployment.BlockStatus = http.StatusBadRequest
	}
	deployment.UpdatedAt = time.Now().UTC()
	deployment.UpdatedBy = strings.TrimSpace(who)
	s.deployments[operationID] = deployment
	s.rebuildRuntimeLocked()
	return clonePositiveDeployment(deployment), nil
}

func (s *positiveSchemaStore) rollback(operationID, who, reason string) (PositiveSchemaDeployment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	deployment, ok := s.deployments[operationID]
	if !ok || len(deployment.History) == 0 {
		return PositiveSchemaDeployment{}, errors.New("no rollback history")
	}
	idx := len(deployment.History) - 1
	prev := deployment.History[idx]
	deployment.History = deployment.History[:idx]
	if _, exists := s.profiles[prev.ProfileVersionID]; !exists {
		return PositiveSchemaDeployment{}, errors.New("rollback profile version no longer exists")
	}
	deployment.ProfileVersionID = prev.ProfileVersionID
	deployment.Mode = prev.Mode
	deployment.BlockStatus = prev.BlockStatus
	deployment.UpdatedAt = time.Now().UTC()
	deployment.UpdatedBy = strings.TrimSpace(who)
	if strings.TrimSpace(reason) != "" {
		deployment = pushPositiveHistory(deployment, who, "rollback: "+strings.TrimSpace(reason))
		// pushPositiveHistory records the restored state; remove it because rollback
		// history is a stack of prior states, not an audit log.
		if len(deployment.History) > 0 {
			deployment.History = deployment.History[:len(deployment.History)-1]
		}
	}
	s.deployments[operationID] = deployment
	s.rebuildRuntimeLocked()
	return clonePositiveDeployment(deployment), nil
}

func validPositiveViolationType(value string) bool {
	switch value {
	case "", "BODY_TOO_LARGE_FOR_SCHEMA_VALIDATION", "MALFORMED_BODY", "CONTENT_TYPE_MISMATCH", "UNKNOWN_FIELD", "TYPE_MISMATCH", "FORMAT_MISMATCH", "ENUM_MISMATCH", "REQUIRED_FIELD_MISSING":
		return true
	default:
		return false
	}
}

func (s *positiveSchemaStore) addException(ex PositiveSchemaException) (PositiveSchemaException, error) {
	if s == nil {
		return PositiveSchemaException{}, errors.New("positive schema store unavailable")
	}
	ex.OperationID = strings.TrimSpace(ex.OperationID)
	ex.ProfileVersionID = strings.TrimSpace(ex.ProfileVersionID)
	ex.ViolationType = strings.ToUpper(strings.TrimSpace(ex.ViolationType))
	ex.Field = strings.TrimSpace(ex.Field)
	ex.Reason = strings.TrimSpace(ex.Reason)
	if ex.OperationID == "" || ex.Reason == "" {
		return PositiveSchemaException{}, errors.New("operation_id and reason are required")
	}
	if len(ex.Reason) > 512 {
		return PositiveSchemaException{}, errors.New("reason is too long")
	}
	if len(ex.Field) > 512 || strings.ContainsAny(ex.Field, "\r\n\x00") {
		return PositiveSchemaException{}, errors.New("field selector is invalid")
	}
	if !validPositiveViolationType(ex.ViolationType) {
		return PositiveSchemaException{}, errors.New("unsupported violation_type")
	}
	if ex.ExpiresAt != nil && !ex.ExpiresAt.After(time.Now().UTC()) {
		return PositiveSchemaException{}, errors.New("expires_at must be in the future")
	}
	now := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.exceptions) >= positiveSchemaMaxExceptions {
		return PositiveSchemaException{}, errors.New("exception limit reached")
	}
	if _, ok := s.deployments[ex.OperationID]; !ok {
		return PositiveSchemaException{}, errors.New("operation has no positive schema deployment")
	}
	if ex.ProfileVersionID != "" {
		profile, ok := s.profiles[ex.ProfileVersionID]
		if !ok || profile.OperationID != ex.OperationID {
			return PositiveSchemaException{}, errors.New("profile_version_id does not belong to operation")
		}
	}
	ex.ID = positiveSchemaExceptionID(ex.OperationID, ex.ProfileVersionID, ex.ViolationType, ex.Field, now)
	ex.Enabled = true
	ex.CreatedAt = now
	s.exceptions[ex.ID] = ex
	s.rebuildRuntimeLocked()
	return ex, nil
}

func (s *positiveSchemaStore) setExceptionEnabled(id string, enabled bool) (PositiveSchemaException, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ex, ok := s.exceptions[id]
	if !ok {
		return PositiveSchemaException{}, errors.New("exception not found")
	}
	ex.Enabled = enabled
	s.exceptions[id] = ex
	s.rebuildRuntimeLocked()
	return ex, nil
}

func (s *positiveSchemaStore) rebuildRuntimeLocked() {
	snap := &positiveSchemaRuntimeSnapshot{ByOperation: make(map[string]positiveSchemaRuntimePolicy, len(s.deployments))}
	for operationID, deployment := range s.deployments {
		profile, ok := s.profiles[deployment.ProfileVersionID]
		if !ok {
			continue
		}
		policy := positiveSchemaRuntimePolicy{Profile: clonePositiveProfile(profile), Deployment: clonePositiveDeployment(deployment), FieldIndex: map[string]PositiveSchemaField{}}
		for _, field := range profile.Fields {
			policy.FieldIndex[field.Location+"|"+field.Path] = field
		}
		for _, ex := range s.exceptions {
			if ex.OperationID != operationID || !ex.Enabled {
				continue
			}
			policy.Exceptions = append(policy.Exceptions, ex)
		}
		sort.Slice(policy.Exceptions, func(i, j int) bool { return policy.Exceptions[i].ID < policy.Exceptions[j].ID })
		snap.ByOperation[operationID] = policy
	}
	s.runtime.Store(snap)
}

func (s *positiveSchemaStore) runtimePolicy(operationID string) (positiveSchemaRuntimePolicy, bool) {
	if s == nil {
		return positiveSchemaRuntimePolicy{}, false
	}
	snap := s.runtime.Load()
	if snap == nil {
		return positiveSchemaRuntimePolicy{}, false
	}
	policy, ok := snap.ByOperation[operationID]
	return policy, ok
}

func clonePositiveProfile(in PositiveSchemaProfileVersion) PositiveSchemaProfileVersion {
	out := in
	out.AllowedContentType = append([]string(nil), in.AllowedContentType...)
	out.Fields = make([]PositiveSchemaField, len(in.Fields))
	for i := range in.Fields {
		out.Fields[i] = in.Fields[i]
		out.Fields[i].AllowedTypes = append([]string(nil), in.Fields[i].AllowedTypes...)
		out.Fields[i].AllowedFormats = append([]string(nil), in.Fields[i].AllowedFormats...)
		out.Fields[i].AllowedEnum = append([]string(nil), in.Fields[i].AllowedEnum...)
	}
	return out
}

func clonePositiveDeployment(in PositiveSchemaDeployment) PositiveSchemaDeployment {
	out := in
	out.History = append([]PositiveSchemaDeploymentRevision(nil), in.History...)
	return out
}

func (s *positiveSchemaStore) listDeployments() []PositiveSchemaDeployment {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]PositiveSchemaDeployment, 0, len(s.deployments))
	for _, d := range s.deployments {
		out = append(out, clonePositiveDeployment(d))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].OperationID < out[j].OperationID })
	return out
}

func (s *positiveSchemaStore) detail(operationID string) (map[string]any, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	deployment, ok := s.deployments[operationID]
	if !ok {
		return nil, false
	}
	versions := make([]PositiveSchemaProfileVersion, 0, len(s.versionsByOperation[operationID]))
	for _, id := range s.versionsByOperation[operationID] {
		if profile, exists := s.profiles[id]; exists {
			versions = append(versions, clonePositiveProfile(profile))
		}
	}
	exceptions := make([]PositiveSchemaException, 0)
	for _, ex := range s.exceptions {
		if ex.OperationID == operationID {
			exceptions = append(exceptions, ex)
		}
	}
	sort.Slice(exceptions, func(i, j int) bool { return exceptions[i].ID < exceptions[j].ID })
	return map[string]any{"deployment": clonePositiveDeployment(deployment), "versions": versions, "exceptions": exceptions}, true
}

func (s *positiveSchemaStore) recordViolations(violations []PositiveSchemaViolation) {
	if s == nil || len(violations) == 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range violations {
		v := violations[i]
		seq := s.violationSeq.Add(1)
		if v.Time.IsZero() {
			v.Time = time.Now().UTC()
		}
		if v.ID == "" {
			h := sha256.Sum256([]byte(fmt.Sprintf("violation|%s|%d|%s|%s|%s", v.OperationID, seq, v.Type, v.Location, v.Path)))
			v.ID = hex.EncodeToString(h[:16])
		}
		s.violations = append(s.violations, v)
	}
	if len(s.violations) > positiveSchemaMaxViolations {
		s.violations = append([]PositiveSchemaViolation(nil), s.violations[len(s.violations)-positiveSchemaMaxViolations:]...)
	}
}

func (s *positiveSchemaStore) listViolations(operationID string, limit int) []PositiveSchemaViolation {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]PositiveSchemaViolation, 0, limit)
	for i := len(s.violations) - 1; i >= 0 && len(out) < limit; i-- {
		v := s.violations[i]
		if operationID == "" || v.OperationID == operationID {
			out = append(out, v)
		}
	}
	return out
}

func containsString(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}

func positiveTypeAllowed(allowed []string, observed string) bool {
	if containsString(allowed, observed) {
		return true
	}
	return observed == "integer" && containsString(allowed, "number")
}

func positiveViolation(operationID string, policy positiveSchemaRuntimePolicy, typ, location, path, expected, observed string, r *http.Request) PositiveSchemaViolation {
	v := PositiveSchemaViolation{
		Time: time.Now().UTC(), OperationID: operationID, ProfileVersionID: policy.Profile.ID, Mode: policy.Deployment.Mode,
		Type: typ, Location: location, Path: path, Expected: expected, Observed: observed,
	}
	if r != nil {
		v.Method = r.Method
		if r.URL != nil {
			v.RequestPath = normalizeAPIOperationPath(r.URL.Path)
		}
		v.ContentType = normalizeMediaType(r.Header.Get("Content-Type"))
	}
	return v
}

func applyPositiveExceptions(policy positiveSchemaRuntimePolicy, violations []PositiveSchemaViolation, now time.Time) []PositiveSchemaViolation {
	for i := range violations {
		for _, ex := range policy.Exceptions {
			if !ex.Enabled || (ex.ExpiresAt != nil && !ex.ExpiresAt.After(now)) {
				continue
			}
			if ex.ProfileVersionID != "" && ex.ProfileVersionID != policy.Profile.ID {
				continue
			}
			if ex.ViolationType != "" && ex.ViolationType != violations[i].Type {
				continue
			}
			field := violations[i].Location + "|" + violations[i].Path
			if ex.Field != "" && ex.Field != field {
				continue
			}
			violations[i].Exempted = true
			violations[i].ExceptionID = ex.ID
			break
		}
	}
	return violations
}

func validatePositiveSchemaRequest(operationID string, policy positiveSchemaRuntimePolicy, r *http.Request) []PositiveSchemaViolation {
	if r == nil || r.URL == nil {
		return nil
	}
	var violations []PositiveSchemaViolation
	contentType := normalizeMediaType(r.Header.Get("Content-Type"))
	capture := requestCaptureFromRequest(r)
	bodyTruncated := capture != nil && capture.bodyTruncated
	hasBodyPolicy := false
	for _, field := range policy.Profile.Fields {
		if field.Location == "body" {
			hasBodyPolicy = true
			break
		}
	}
	if hasBodyPolicy && bodyTruncated {
		violations = append(violations, positiveViolation(operationID, policy, "BODY_TOO_LARGE_FOR_SCHEMA_VALIDATION", "body", "$", "complete body <= validation capture limit", "truncated", r))
	}
	if policy.Profile.EnforceContentType && hasBodyPolicy {
		if !containsString(policy.Profile.AllowedContentType, contentType) {
			violations = append(violations, positiveViolation(operationID, policy, "CONTENT_TYPE_MISMATCH", "header", "content-type", strings.Join(policy.Profile.AllowedContentType, ","), contentType, r))
		}
	}
	if hasBodyPolicy && !bodyTruncated && (contentType == "application/json" || contentType == "application/merge-patch+json") {
		prefix := requestBodyPrefixFromRequest(r)
		if len(prefix) > 0 && !json.Valid(prefix) {
			violations = append(violations, positiveViolation(operationID, policy, "MALFORMED_BODY", "body", "$", "valid JSON", "invalid JSON", r))
		}
	}
	meta := apiObservationMeta{
		Host: r.Host, AuthScheme: inferAuthScheme(r), Headers: selectedSchemaHeaders(r),
		BodyPrefix: append([]byte(nil), requestBodyPrefixFromRequest(r)...), BodyTruncated: bodyTruncated,
	}
	samples := collectSchemaSamplesFromObservation(r.URL.Path, r.URL.RawQuery, r.Header.Get("Content-Type"), meta)
	observed := map[string][]schemaFieldSample{}
	for _, sample := range samples {
		key := sample.Location + "|" + sample.Path
		observed[key] = append(observed[key], sample)
		field, known := policy.FieldIndex[key]
		if !known {
			if policy.Profile.UnknownFields == positiveSchemaUnknownDeny {
				violations = append(violations, positiveViolation(operationID, policy, "UNKNOWN_FIELD", sample.Location, sample.Path, "declared field", sample.Type, r))
			}
			continue
		}
		if !positiveTypeAllowed(field.AllowedTypes, sample.Type) {
			violations = append(violations, positiveViolation(operationID, policy, "TYPE_MISMATCH", sample.Location, sample.Path, strings.Join(field.AllowedTypes, ","), sample.Type, r))
			continue
		}
		if policy.Profile.EnforceFormats && len(field.AllowedFormats) > 0 && !containsString(field.AllowedFormats, sample.Format) {
			observedFormat := sample.Format
			if observedFormat == "" {
				observedFormat = "none"
			}
			violations = append(violations, positiveViolation(operationID, policy, "FORMAT_MISMATCH", sample.Location, sample.Path, strings.Join(field.AllowedFormats, ","), observedFormat, r))
		}
		if policy.Profile.EnforceEnums && len(field.AllowedEnum) > 0 {
			if sample.EnumValue == "" || !containsString(field.AllowedEnum, sample.EnumValue) {
				observedEnum := sample.EnumValue
				if observedEnum == "" {
					observedEnum = "outside reviewed enum"
				}
				violations = append(violations, positiveViolation(operationID, policy, "ENUM_MISMATCH", sample.Location, sample.Path, strings.Join(field.AllowedEnum, ","), observedEnum, r))
			}
		}
	}
	for key, field := range policy.FieldIndex {
		if field.Required && len(observed[key]) == 0 {
			violations = append(violations, positiveViolation(operationID, policy, "REQUIRED_FIELD_MISSING", field.Location, field.Path, "required", "missing", r))
		}
	}
	return applyPositiveExceptions(policy, violations, time.Now().UTC())
}

func unexemptedPositiveViolations(violations []PositiveSchemaViolation) int {
	n := 0
	for _, v := range violations {
		if !v.Exempted {
			n++
		}
	}
	return n
}

func (s *positiveSchemaStore) wrap(site string, next http.Handler) http.Handler {
	if s == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r == nil || r.URL == nil {
			next.ServeHTTP(w, r)
			return
		}
		operationID := apiOperationID(site, r.Method, normalizeAPIOperationPath(r.URL.Path))
		policy, ok := s.runtimePolicy(operationID)
		if !ok {
			next.ServeHTTP(w, r)
			return
		}
		violations := validatePositiveSchemaRequest(operationID, policy, r)
		if len(violations) == 0 {
			next.ServeHTTP(w, r)
			return
		}
		action := "learn"
		if policy.Deployment.Mode == positiveSchemaModeDetect {
			action = "detect"
		} else if policy.Deployment.Mode == positiveSchemaModeEnforce && unexemptedPositiveViolations(violations) > 0 {
			action = "block"
		}
		for i := range violations {
			violations[i].Action = action
			if violations[i].Exempted && action == "block" {
				violations[i].Action = "exception"
			}
		}
		s.recordViolations(violations)
		if action != "block" {
			next.ServeHTTP(w, r)
			return
		}
		status := policy.Deployment.BlockStatus
		if status == 0 {
			status = http.StatusBadRequest
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"error":"request rejected by API schema policy"}`))
	})
}

func (s *positiveSchemaStore) save(configPath string) error {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	profiles := make(map[string]PositiveSchemaProfileVersion, len(s.profiles))
	for id, profile := range s.profiles {
		profiles[id] = clonePositiveProfile(profile)
	}
	deployments := make(map[string]PositiveSchemaDeployment, len(s.deployments))
	for id, deployment := range s.deployments {
		deployments[id] = clonePositiveDeployment(deployment)
	}
	state := positiveSchemaStateFile{
		Version: apiSecurityStateVersion, Saved: time.Now().UTC(),
		Profiles: profiles, VersionsByOperation: cloneStringSliceMap(s.versionsByOperation),
		Deployments: deployments, Exceptions: cloneMap(s.exceptions),
		Violations: append([]PositiveSchemaViolation(nil), s.violations...),
	}
	s.mu.RUnlock()
	return atomicWriteJSON(apiSecurityStatePath(configPath, "api-positive-schema.json"), state)
}

func (s *positiveSchemaStore) load(configPath string) error {
	if s == nil {
		return nil
	}
	var state positiveSchemaStateFile
	if err := readJSONIfExists(apiSecurityStatePath(configPath, "api-positive-schema.json"), &state); err != nil {
		return err
	}
	if err := validateAPISecurityStateVersion("api-positive-schema", state.Version); err != nil {
		return err
	}
	if len(state.Profiles) == 0 && len(state.Deployments) == 0 && len(state.Exceptions) == 0 && len(state.Violations) == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if state.Profiles != nil {
		s.profiles = state.Profiles
	}
	if state.VersionsByOperation != nil {
		s.versionsByOperation = state.VersionsByOperation
	}
	if state.Deployments != nil {
		s.deployments = state.Deployments
	}
	if state.Exceptions != nil {
		s.exceptions = state.Exceptions
	}
	if len(state.Violations) > positiveSchemaMaxViolations {
		state.Violations = state.Violations[len(state.Violations)-positiveSchemaMaxViolations:]
	}
	s.violations = append([]PositiveSchemaViolation(nil), state.Violations...)
	s.rebuildRuntimeLocked()
	return nil
}

func (a *adminServer) handlePositiveSchemaPromote(w http.ResponseWriter, r *http.Request) {
	candidate, ok := a.srv.schema.get(r.PathValue("id"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	op, ok := a.srv.apiOps.get(candidate.OperationID)
	if !ok {
		http.Error(w, "matching API operation not found", http.StatusConflict)
		return
	}
	var req struct {
		UnknownFields      string `json:"unknown_fields"`
		EnforceFormats     bool   `json:"enforce_formats"`
		EnforceEnums       *bool  `json:"enforce_enums"`
		EnforceContentType bool   `json:"enforce_content_type"`
		Activate           bool   `json:"activate"`
		BlockStatus        int    `json:"block_status"`
		Note               string `json:"note"`
	}
	if r.Body != nil {
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16))
		if err := dec.Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}
	}
	enforceEnums := true
	if req.EnforceEnums != nil {
		enforceEnums = *req.EnforceEnums
	}
	profile, deployment, err := a.srv.positiveSchema.promote(candidate, op, positiveSchemaPromoteOptions{
		UnknownFields: req.UnknownFields, EnforceFormats: req.EnforceFormats, EnforceEnums: enforceEnums,
		EnforceContentType: req.EnforceContentType, CreatedBy: who(r).user, Note: req.Note, Activate: req.Activate, BlockStatus: req.BlockStatus,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	if err := a.srv.positiveSchema.save(a.srv.configPath); err != nil {
		http.Error(w, "positive schema persistence failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	a.audit.add(who(r).user, "api_schema.promote", candidate.OperationID+" profile="+profile.ID)
	writeJSON(w, map[string]any{"profile": profile, "deployment": deployment})
}

func (a *adminServer) handlePositiveSchemaList(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, a.srv.positiveSchema.listDeployments())
}

func (a *adminServer) handlePositiveSchemaDetail(w http.ResponseWriter, r *http.Request) {
	detail, ok := a.srv.positiveSchema.detail(r.PathValue("operation_id"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, detail)
}

func (a *adminServer) handlePositiveSchemaMode(w http.ResponseWriter, r *http.Request) {
	operationID := r.PathValue("operation_id")
	var req struct {
		Mode        string `json:"mode"`
		Reason      string `json:"reason"`
		BlockStatus int    `json:"block_status"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	deployment, err := a.srv.positiveSchema.setMode(operationID, req.Mode, who(r).user, req.Reason, req.BlockStatus)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	if err := a.srv.positiveSchema.save(a.srv.configPath); err != nil {
		http.Error(w, "positive schema persistence failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	a.audit.add(who(r).user, "api_schema.mode", operationID+" -> "+deployment.Mode)
	writeJSON(w, deployment)
}

func (a *adminServer) handlePositiveSchemaActivate(w http.ResponseWriter, r *http.Request) {
	operationID := r.PathValue("operation_id")
	var req struct {
		ProfileVersionID string `json:"profile_version_id"`
		Reason           string `json:"reason"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	deployment, err := a.srv.positiveSchema.activate(operationID, strings.TrimSpace(req.ProfileVersionID), who(r).user, req.Reason)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	if err := a.srv.positiveSchema.save(a.srv.configPath); err != nil {
		http.Error(w, "positive schema persistence failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	a.audit.add(who(r).user, "api_schema.activate", operationID+" profile="+deployment.ProfileVersionID)
	writeJSON(w, deployment)
}

func (a *adminServer) handlePositiveSchemaRollback(w http.ResponseWriter, r *http.Request) {
	operationID := r.PathValue("operation_id")
	var req struct {
		Reason string `json:"reason"`
	}
	_ = json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req)
	deployment, err := a.srv.positiveSchema.rollback(operationID, who(r).user, req.Reason)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	if err := a.srv.positiveSchema.save(a.srv.configPath); err != nil {
		http.Error(w, "positive schema persistence failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	a.audit.add(who(r).user, "api_schema.rollback", operationID+" profile="+deployment.ProfileVersionID)
	writeJSON(w, deployment)
}

func (a *adminServer) handlePositiveSchemaExceptionCreate(w http.ResponseWriter, r *http.Request) {
	operationID := r.PathValue("operation_id")
	var req struct {
		ProfileVersionID string `json:"profile_version_id"`
		ViolationType    string `json:"violation_type"`
		Field            string `json:"field"`
		Reason           string `json:"reason"`
		ExpiresAt        string `json:"expires_at"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	var expires *time.Time
	if strings.TrimSpace(req.ExpiresAt) != "" {
		t, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err != nil {
			http.Error(w, "expires_at must be RFC3339", http.StatusBadRequest)
			return
		}
		t = t.UTC()
		expires = &t
	}
	ex, err := a.srv.positiveSchema.addException(PositiveSchemaException{
		OperationID: operationID, ProfileVersionID: req.ProfileVersionID, ViolationType: req.ViolationType,
		Field: req.Field, Reason: req.Reason, ExpiresAt: expires, CreatedBy: who(r).user,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	if err := a.srv.positiveSchema.save(a.srv.configPath); err != nil {
		http.Error(w, "positive schema persistence failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	a.audit.add(who(r).user, "api_schema.exception_create", operationID+" exception="+ex.ID)
	writeJSON(w, ex)
}

func (a *adminServer) handlePositiveSchemaExceptionToggle(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	ex, err := a.srv.positiveSchema.setExceptionEnabled(r.PathValue("exception_id"), req.Enabled)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err := a.srv.positiveSchema.save(a.srv.configPath); err != nil {
		http.Error(w, "positive schema persistence failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	a.audit.add(who(r).user, "api_schema.exception_toggle", ex.ID+" enabled="+strconv.FormatBool(ex.Enabled))
	writeJSON(w, ex)
}

func (a *adminServer) handlePositiveSchemaViolations(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	writeJSON(w, a.srv.positiveSchema.listViolations(r.URL.Query().Get("operation_id"), limit))
}
