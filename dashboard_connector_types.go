package main

import "time"

const (
	owiContract       = "operator-workspace.integration.v1"
	owiSchemaVersion  = "1.0"
	owiSourceProduct  = "WAF"
	owiBasePath       = "/api/integrations/dashboard/v1"
	owiCursorTTL      = 24 * time.Hour
	owiHistoryRetain  = 30 * 24 * time.Hour
	owiStateLogRetain = 7 * 24 * time.Hour
)

const (
	owiScopeHealth      = "dashboard:read:health"
	owiScopeCaps        = "dashboard:read:capabilities"
	owiScopeAssets      = "dashboard:read:assets"
	owiScopeDetections  = "dashboard:read:detections"
	owiScopePolicies    = "dashboard:read:policies"
	owiScopeHealthObs   = "dashboard:read:health-observations"
	owiScopeEvents      = "dashboard:read:events"
	owiScopeActionState = "dashboard:read:action-status"
)

type owiRef struct {
	SourceProduct    string `json:"source_product"`
	SourceInstanceID string `json:"source_instance_id"`
	ResourceKind     string `json:"resource_kind"`
	ExternalID       string `json:"external_id"`
}

type owiEntityRef struct {
	Ref              owiRef  `json:"ref"`
	Role             string  `json:"role"`
	ObservedAt       string  `json:"observed_at"`
	ValidUntil       *string `json:"valid_until"`
	NetworkNamespace *string `json:"network_namespace"`
	Confidence       float64 `json:"confidence"`
}

type owiEvidenceRef struct {
	Ref        owiRef  `json:"ref"`
	Revision   string  `json:"revision"`
	Kind       string  `json:"kind"`
	Summary    string  `json:"summary"`
	DetailPath *string `json:"detail_path"`
}

type owiProvenance struct {
	Origin         owiRef              `json:"origin"`
	OriginRevision string              `json:"origin_revision"`
	OriginEventID  *string             `json:"origin_event_id"`
	ForwardedBy    []map[string]string `json:"forwarded_by"`
	DerivedFrom    []owiRef            `json:"derived_from"`
	EvidenceRefs   []owiEvidenceRef    `json:"evidence_refs"`
	DetailPath     *string             `json:"detail_path"`
}

type owiAttention struct {
	Code                   string   `json:"code"`
	Severity               string   `json:"severity"`
	Confidence             *float64 `json:"confidence"`
	Status                 string   `json:"status"`
	ReasonCodes            []string `json:"reason_codes"`
	RecommendedActionCodes []string `json:"recommended_action_codes"`
}

type owiIdentity struct {
	Kind       string  `json:"kind"`
	Value      string  `json:"value"`
	Namespace  string  `json:"namespace"`
	ObservedAt string  `json:"observed_at"`
	ValidUntil *string `json:"valid_until"`
	Assurance  string  `json:"assurance"`
}

type owiMetric struct {
	Name              string   `json:"name"`
	Value             *float64 `json:"value"`
	Unit              string   `json:"unit"`
	WindowSeconds     int      `json:"window_seconds"`
	ObservedAt        string   `json:"observed_at"`
	Threshold         *float64 `json:"threshold"`
	ThresholdOperator *string  `json:"threshold_operator"`
	Quality           string   `json:"quality"`
}

type owiRecord struct {
	ExternalID       string         `json:"external_id"`
	Revision         string         `json:"revision"`
	StreamSequence   string         `json:"stream_sequence"`
	Operation        string         `json:"operation"`
	SourceObservedAt string         `json:"source_observed_at"`
	SourceUpdatedAt  *string        `json:"source_updated_at"`
	Provenance       owiProvenance  `json:"provenance"`
	Payload          map[string]any `json:"payload"`
}

type owiCoverage struct {
	State       string   `json:"state"`
	ReasonCodes []string `json:"reason_codes"`
}

type owiPage struct {
	Contract         string      `json:"contract"`
	SchemaVersion    string      `json:"schema_version"`
	SourceProduct    string      `json:"source_product"`
	SourceInstanceID string      `json:"source_instance_id"`
	ResourceKind     string      `json:"resource_kind"`
	SyncMode         string      `json:"sync_mode"`
	SourceSnapshotID *string     `json:"source_snapshot_id"`
	Coverage         owiCoverage `json:"coverage"`
	GeneratedAt      string      `json:"generated_at"`
	Records          []owiRecord `json:"records"`
	NextCursor       string      `json:"next_cursor"`
	HasMore          bool        `json:"has_more"`
}

type owiHealthResponse struct {
	Contract         string   `json:"contract"`
	SchemaVersion    string   `json:"schema_version"`
	SourceProduct    string   `json:"source_product"`
	SourceInstanceID string   `json:"source_instance_id"`
	Status           string   `json:"status"`
	ObservedAt       string   `json:"observed_at"`
	ExportReady      bool     `json:"export_ready"`
	ReasonCodes      []string `json:"reason_codes"`
}

type owiCapabilityResource struct {
	Path                   string   `json:"path"`
	ResourceKind           string   `json:"resource_kind"`
	Capability             string   `json:"capability"`
	Enabled                bool     `json:"enabled"`
	Qualification          string   `json:"qualification"`
	SyncModes              []string `json:"sync_modes"`
	StreamSemantics        string   `json:"stream_semantics"`
	BootstrapRetentionDays *int     `json:"bootstrap_retention_days"`
	CursorTTLSeconds       int      `json:"cursor_ttl_seconds"`
	MaxPageSize            int      `json:"max_page_size"`
	ReasonCodes            []string `json:"reason_codes"`
}

type owiCapabilitiesResponse struct {
	Contract         string                  `json:"contract"`
	SchemaVersion    string                  `json:"schema_version"`
	SourceProduct    string                  `json:"source_product"`
	SourceInstanceID string                  `json:"source_instance_id"`
	AuthProfile      string                  `json:"auth_profile"`
	Resources        []owiCapabilityResource `json:"resources"`
	Actions          []string                `json:"actions"`
}

type owiErrorBody struct {
	Code              string `json:"code"`
	Message           string `json:"message"`
	RequestID         string `json:"request_id"`
	Retryable         bool   `json:"retryable"`
	RetryAfterSeconds *int   `json:"retry_after_seconds"`
}

type owiErrorResponse struct {
	Contract string       `json:"contract"`
	Error    owiErrorBody `json:"error"`
}

type owiReaderToken struct {
	ID           string   `json:"id"`
	Principal    string   `json:"principal"`
	TenantID     string   `json:"tenant_id"`
	DigestSHA256 string   `json:"digest_sha256"`
	Scopes       []string `json:"scopes"`
	IssuedAt     string   `json:"issued_at"`
	NotBefore    string   `json:"not_before,omitempty"`
	ExpiresAt    string   `json:"expires_at"`
	RevokedAt    *string  `json:"revoked_at"`
}

type owiTokenRegistry struct {
	Version int              `json:"version"`
	Tokens  []owiReaderToken `json:"tokens"`
}

type owiPrincipal struct {
	TokenID   string
	Principal string
	TenantID  string
	Scopes    map[string]struct{}
}

type owiCursor struct {
	Version          int    `json:"v"`
	TenantID         string `json:"t"`
	Principal        string `json:"p"`
	ScopeHash        string `json:"s"`
	SourceInstanceID string `json:"i"`
	ResourceKind     string `json:"k"`
	Mode             string `json:"m"`
	SnapshotID       string `json:"n,omitempty"`
	Offset           int    `json:"o,omitempty"`
	Watermark        uint64 `json:"w"`
	Limit            int    `json:"l"`
	IssuedAtUnix     int64  `json:"a"`
	ExpiresAtUnix    int64  `json:"e"`
}

type owiSnapshot struct {
	ID        string      `json:"id"`
	TenantID  string      `json:"tenant_id"`
	Principal string      `json:"principal"`
	ScopeHash string      `json:"scope_hash"`
	Kind      string      `json:"kind"`
	CreatedAt time.Time   `json:"created_at"`
	ExpiresAt time.Time   `json:"expires_at"`
	Watermark uint64      `json:"watermark"`
	Records   []owiRecord `json:"records"`
}

type owiPersistedState struct {
	Version        int                             `json:"version"`
	SourceInstance string                          `json:"source_instance_id"`
	SavedAt        time.Time                       `json:"saved_at"`
	CleanShutdown  bool                            `json:"clean_shutdown"`
	NextSequence   map[string]uint64               `json:"next_sequence"`
	Current        map[string]map[string]owiRecord `json:"current"`
	Journal        map[string][]owiRecord          `json:"journal"`
	Snapshots      map[string]owiSnapshot          `json:"snapshots"`
	DetectionGapAt *time.Time                      `json:"detection_gap_at,omitempty"`
	DetectionDrops uint64                          `json:"detection_drops"`
}
