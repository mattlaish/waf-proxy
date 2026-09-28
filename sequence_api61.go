package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	sequenceStateVersion                 = 4
	sequenceDefaultSessionTTL            = 30 * time.Minute
	sequenceDefaultSessionAbsoluteTTL    = 8 * time.Hour
	sequenceDefaultTransitionTTL         = 24 * time.Hour
	sequenceDefaultWorkflowTTL           = 7 * 24 * time.Hour
	sequenceDefaultStaleAfter            = 6 * time.Hour
	sequenceDefaultMaxSessions           = 4096
	sequenceDefaultMaxTransitions        = 8192
	sequenceDefaultMaxWorkflows          = 512
	sequenceDefaultMaxWorkflowOperations = 256
	sequenceDefaultRecentOps             = 32
	sequenceDefaultSeenTransitions       = 128
	sequenceDefaultRecentSessionTTL      = 24 * time.Hour
	sequenceDefaultMaxRecentSessions     = 1024
	sequenceDefaultMinObservations       = 20
	sequenceDefaultMinSessions           = 5
	sequenceDefaultMinLearningDuration   = 15 * time.Minute
	sequenceDefaultConfidenceThreshold   = 0.05
	sequenceObservationQueue             = 4096
	sequencePublishEvery                 = 64
)

// SequenceSession is an API-6.1 privacy-preserving request-flow session. Key
// is an HMAC digest: raw JWTs, claims, cookies, client addresses and user-agent
// values are never retained. Operation IDs are the normalized API-1 IDs.
type SequenceSession struct {
	Key                string    `json:"key"`
	Site               string    `json:"site"`
	CorrelationKind    string    `json:"correlation_kind"`
	WorkflowID         string    `json:"workflow_id,omitempty"`
	CurrentOperationID string    `json:"current_operation_id"`
	EntryOperationID   string    `json:"entry_operation_id,omitempty"`
	RecentOperationIDs []string  `json:"recent_operation_ids,omitempty"`
	SeenTransitionIDs  []string  `json:"seen_transition_ids,omitempty"`
	RequestCount       uint64    `json:"request_count"`
	WorkflowDepth      uint64    `json:"workflow_depth"`
	StartedAt          time.Time `json:"started_at"`
	LastSeen           time.Time `json:"last_seen"`
	ExpiresAt          time.Time `json:"expires_at"`
	AbsoluteExpiresAt  time.Time `json:"absolute_expires_at"`
}

// SequenceTransition is the bounded structural edge recorded by API-6.1.
// Workflow confidence, anomaly scoring and enforcement are deliberately not
// part of this slice.
type SequenceTransition struct {
	ID               string    `json:"id"`
	Site             string    `json:"site"`
	WorkflowID       string    `json:"workflow_id,omitempty"`
	FromOperationID  string    `json:"from_operation_id"`
	ToOperationID    string    `json:"to_operation_id"`
	Count            uint64    `json:"count"` // API-6.1 compatibility alias for ObservationCount.
	ObservationCount uint64    `json:"observation_count"`
	SessionCount     uint64    `json:"session_count"`
	FirstSeen        time.Time `json:"first_seen"`
	LastSeen         time.Time `json:"last_seen"`
	ExpiresAt        time.Time `json:"expires_at"`
	Confidence       float64   `json:"confidence"`
	Maturity         string    `json:"maturity"`
}

// SequenceModel is the immutable snapshot used by persistence and the admin
// visibility API. The runtime publishes a fully cloned model atomically.
type SequenceModel struct {
	Version                         int                     `json:"version"`
	GeneratedAt                     time.Time               `json:"generated_at"`
	Sessions                        []SequenceSession       `json:"sessions"`
	Transitions                     []SequenceTransition    `json:"transitions"`
	Workflows                       []SequenceWorkflowModel `json:"workflows"`
	Violations                      []SequenceViolation     `json:"violations"`
	Exceptions                      []SequenceException     `json:"exceptions"`
	SessionTTLSeconds               int64                   `json:"session_ttl_seconds"`
	SessionAbsoluteSeconds          int64                   `json:"session_absolute_seconds"`
	TransitionTTLSeconds            int64                   `json:"transition_ttl_seconds"`
	WorkflowTTLSeconds              int64                   `json:"workflow_ttl_seconds"`
	StaleAfterSeconds               int64                   `json:"stale_after_seconds"`
	MaxSessions                     int                     `json:"max_sessions"`
	MaxTransitions                  int                     `json:"max_transitions"`
	MaxWorkflows                    int                     `json:"max_workflows"`
	MaxWorkflowOperations           int                     `json:"max_workflow_operations"`
	MaxRecentOperations             int                     `json:"max_recent_operations"`
	MinObservations                 uint64                  `json:"min_observations"`
	MinSessions                     uint64                  `json:"min_sessions"`
	MinLearningSeconds              int64                   `json:"min_learning_duration_seconds"`
	ConfidenceThreshold             float64                 `json:"confidence_threshold"`
	QueueCapacity                   int                     `json:"queue_capacity"`
	QueueDepth                      int                     `json:"queue_depth"`
	DroppedObservations             uint64                  `json:"dropped_observations"`
	EvictedSessions                 uint64                  `json:"evicted_sessions"`
	EvictedTransitions              uint64                  `json:"evicted_transitions"`
	EvictedWorkflows                uint64                  `json:"evicted_workflows"`
	EvictedViolations               uint64                  `json:"evicted_violations"`
	ViolationTTLSeconds             int64                   `json:"violation_ttl_seconds"`
	MaxViolations                   int                     `json:"max_violations"`
	MaxExceptions                   int                     `json:"max_exceptions"`
	EntryConfidenceThreshold        float64                 `json:"entry_confidence_threshold"`
	DivergenceExpectedMinConfidence float64                 `json:"divergence_expected_min_confidence"`
	DivergenceObservedMaxConfidence float64                 `json:"divergence_observed_max_confidence"`
	RepetitionTailProbability       float64                 `json:"repetition_tail_probability"`
	RepetitionMinimumRun            uint64                  `json:"repetition_minimum_run"`
	RecentSessions                  []SequenceRecentSession `json:"recent_sessions"`
	Controls                        []SequenceSiteControl   `json:"controls"`
	RecentSessionTTLSeconds         int64                   `json:"recent_session_ttl_seconds"`
	MaxRecentSessions               int                     `json:"max_recent_sessions"`
	EvictedRecentSessions           uint64                  `json:"evicted_recent_sessions"`
}

type sequenceSettings struct {
	SessionTTL                      time.Duration
	SessionAbsoluteTTL              time.Duration
	TransitionTTL                   time.Duration
	WorkflowTTL                     time.Duration
	StaleAfter                      time.Duration
	MaxSessions                     int
	MaxTransitions                  int
	MaxWorkflows                    int
	MaxWorkflowOperations           int
	MaxRecentOps                    int
	MaxSeenTransitions              int
	MinObservations                 uint64
	MinSessions                     uint64
	MinLearningDuration             time.Duration
	ConfidenceThreshold             float64
	QueueCapacity                   int
	ViolationTTL                    time.Duration
	MaxViolations                   int
	MaxExceptions                   int
	EntryConfidenceThreshold        float64
	DivergenceExpectedMinConfidence float64
	DivergenceObservedMaxConfidence float64
	RepetitionTailProbability       float64
	RepetitionMinimumRun            uint64
	RecentSessionTTL                time.Duration
	MaxRecentSessions               int
}

func defaultSequenceSettings() sequenceSettings {
	return sequenceSettings{
		SessionTTL: sequenceDefaultSessionTTL, SessionAbsoluteTTL: sequenceDefaultSessionAbsoluteTTL,
		TransitionTTL: sequenceDefaultTransitionTTL, WorkflowTTL: sequenceDefaultWorkflowTTL, StaleAfter: sequenceDefaultStaleAfter,
		MaxSessions: sequenceDefaultMaxSessions, MaxTransitions: sequenceDefaultMaxTransitions, MaxWorkflows: sequenceDefaultMaxWorkflows,
		MaxWorkflowOperations: sequenceDefaultMaxWorkflowOperations, MaxRecentOps: sequenceDefaultRecentOps, MaxSeenTransitions: sequenceDefaultSeenTransitions,
		MinObservations: sequenceDefaultMinObservations, MinSessions: sequenceDefaultMinSessions, MinLearningDuration: sequenceDefaultMinLearningDuration,
		ConfidenceThreshold: sequenceDefaultConfidenceThreshold, QueueCapacity: sequenceObservationQueue,
		ViolationTTL: sequenceDefaultViolationTTL, MaxViolations: sequenceDefaultMaxViolations, MaxExceptions: sequenceDefaultMaxExceptions,
		EntryConfidenceThreshold: sequenceDefaultEntryConfidenceThreshold, DivergenceExpectedMinConfidence: sequenceDefaultDivergenceExpectedMinConfidence,
		DivergenceObservedMaxConfidence: sequenceDefaultDivergenceObservedMaxConfidence, RepetitionTailProbability: sequenceDefaultRepetitionTailProbability,
		RepetitionMinimumRun: sequenceDefaultRepetitionMinimumRun,
		RecentSessionTTL:     sequenceDefaultRecentSessionTTL, MaxRecentSessions: sequenceDefaultMaxRecentSessions,
	}
}

type sequenceObservation struct {
	SessionKey      string
	Site            string
	CorrelationKind string
	WorkflowID      string
	IdentityKind    string
	OperationID     string
	At              time.Time
}

type sequenceStateFile struct {
	Version        int                     `json:"version"`
	Saved          time.Time               `json:"saved"`
	Sessions       []SequenceSession       `json:"sessions"`
	Transitions    []SequenceTransition    `json:"transitions"`
	Workflows      []SequenceWorkflowModel `json:"workflows,omitempty"`
	Violations     []SequenceViolation     `json:"violations,omitempty"`
	Exceptions     []SequenceException     `json:"exceptions,omitempty"`
	RecentSessions []SequenceRecentSession `json:"recent_sessions,omitempty"`
	Controls       []SequenceSiteControl   `json:"controls,omitempty"`
}

type sequenceStore struct {
	settings sequenceSettings
	now      func() time.Time

	keyMu          sync.RWMutex
	correlationKey []byte

	stateMu        sync.Mutex
	publishMu      sync.Mutex
	sessions       map[string]SequenceSession
	transitions    map[string]SequenceTransition
	workflows      map[string]*sequenceWorkflowRuntime
	violations     []SequenceViolation
	exceptions     map[string]SequenceException
	recentSessions []SequenceRecentSession
	controls       map[string]SequenceSiteControl
	processed      uint64

	runtime               atomic.Pointer[SequenceModel]
	dropped               atomic.Uint64
	evictedSessions       atomic.Uint64
	evictedTransitions    atomic.Uint64
	evictedWorkflows      atomic.Uint64
	evictedViolations     atomic.Uint64
	evictedRecentSessions atomic.Uint64

	queue     chan sequenceObservation
	stop      chan struct{}
	done      chan struct{}
	accepting atomic.Bool
	startOnce sync.Once
	stopOnce  sync.Once
}

func newSequenceStore() *sequenceStore {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		panic("sequence correlation key generation failed: " + err.Error())
	}
	return newSequenceStoreWithSettings(defaultSequenceSettings(), key)
}

func newSequenceStoreWithSettings(settings sequenceSettings, key []byte) *sequenceStore {
	if settings.SessionTTL <= 0 {
		settings.SessionTTL = sequenceDefaultSessionTTL
	}
	if settings.SessionAbsoluteTTL <= 0 {
		settings.SessionAbsoluteTTL = sequenceDefaultSessionAbsoluteTTL
	}
	if settings.TransitionTTL <= 0 {
		settings.TransitionTTL = sequenceDefaultTransitionTTL
	}
	if settings.WorkflowTTL <= 0 {
		settings.WorkflowTTL = sequenceDefaultWorkflowTTL
	}
	if settings.StaleAfter <= 0 {
		settings.StaleAfter = sequenceDefaultStaleAfter
	}
	if settings.MaxSessions <= 0 {
		settings.MaxSessions = sequenceDefaultMaxSessions
	}
	if settings.MaxTransitions <= 0 {
		settings.MaxTransitions = sequenceDefaultMaxTransitions
	}
	if settings.MaxWorkflows <= 0 {
		settings.MaxWorkflows = sequenceDefaultMaxWorkflows
	}
	if settings.MaxWorkflowOperations <= 0 {
		settings.MaxWorkflowOperations = sequenceDefaultMaxWorkflowOperations
	}
	if settings.MaxRecentOps <= 0 {
		settings.MaxRecentOps = sequenceDefaultRecentOps
	}
	if settings.MaxSeenTransitions <= 0 {
		settings.MaxSeenTransitions = sequenceDefaultSeenTransitions
	}
	if settings.MinObservations == 0 {
		settings.MinObservations = sequenceDefaultMinObservations
	}
	if settings.MinSessions == 0 {
		settings.MinSessions = sequenceDefaultMinSessions
	}
	if settings.MinLearningDuration <= 0 {
		settings.MinLearningDuration = sequenceDefaultMinLearningDuration
	}
	if settings.ConfidenceThreshold <= 0 || settings.ConfidenceThreshold > 1 {
		settings.ConfidenceThreshold = sequenceDefaultConfidenceThreshold
	}
	if settings.QueueCapacity <= 0 {
		settings.QueueCapacity = sequenceObservationQueue
	}
	if settings.ViolationTTL <= 0 {
		settings.ViolationTTL = sequenceDefaultViolationTTL
	}
	if settings.MaxViolations <= 0 {
		settings.MaxViolations = sequenceDefaultMaxViolations
	}
	if settings.MaxExceptions <= 0 {
		settings.MaxExceptions = sequenceDefaultMaxExceptions
	}
	if settings.EntryConfidenceThreshold <= 0 || settings.EntryConfidenceThreshold > 1 {
		settings.EntryConfidenceThreshold = sequenceDefaultEntryConfidenceThreshold
	}
	if settings.DivergenceExpectedMinConfidence <= 0 || settings.DivergenceExpectedMinConfidence > 1 {
		settings.DivergenceExpectedMinConfidence = sequenceDefaultDivergenceExpectedMinConfidence
	}
	if settings.DivergenceObservedMaxConfidence < 0 || settings.DivergenceObservedMaxConfidence > 1 {
		settings.DivergenceObservedMaxConfidence = sequenceDefaultDivergenceObservedMaxConfidence
	}
	if settings.RepetitionTailProbability <= 0 || settings.RepetitionTailProbability >= 1 {
		settings.RepetitionTailProbability = sequenceDefaultRepetitionTailProbability
	}
	if settings.RepetitionMinimumRun < 2 {
		settings.RepetitionMinimumRun = sequenceDefaultRepetitionMinimumRun
	}
	if settings.RecentSessionTTL <= 0 {
		settings.RecentSessionTTL = sequenceDefaultRecentSessionTTL
	}
	if settings.MaxRecentSessions <= 0 {
		settings.MaxRecentSessions = sequenceDefaultMaxRecentSessions
	}
	if len(key) != 32 {
		panic("sequence correlation key must be exactly 32 bytes")
	}
	s := &sequenceStore{
		settings: settings, now: func() time.Time { return time.Now().UTC() },
		correlationKey: append([]byte(nil), key...),
		sessions:       make(map[string]SequenceSession), transitions: make(map[string]SequenceTransition), workflows: make(map[string]*sequenceWorkflowRuntime),
		violations: make([]SequenceViolation, 0, settings.MaxViolations), exceptions: make(map[string]SequenceException),
		recentSessions: make([]SequenceRecentSession, 0, settings.MaxRecentSessions), controls: make(map[string]SequenceSiteControl),
		queue: make(chan sequenceObservation, settings.QueueCapacity), stop: make(chan struct{}), done: make(chan struct{}),
	}
	s.publishSnapshot()
	return s
}

func (s *sequenceStore) start() {
	if s == nil {
		return
	}
	s.startOnce.Do(func() {
		s.accepting.Store(true)
		go s.run()
	})
}

func (s *sequenceStore) stopAndDrain() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		s.accepting.Store(false)
		close(s.stop)
	})
	<-s.done
}

func (s *sequenceStore) run() {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	defer close(s.done)
	for {
		select {
		case ev := <-s.queue:
			s.process(ev)
		case <-ticker.C:
			s.publishSnapshot()
		case <-s.stop:
			for {
				select {
				case ev := <-s.queue:
					s.process(ev)
				default:
					s.publishSnapshot()
					return
				}
			}
		}
	}
}

func (s *sequenceStore) wrap(site string, next http.Handler) http.Handler {
	if s == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.noteRequest(site, r)
		next.ServeHTTP(w, r)
	})
}

func (s *sequenceStore) noteRequest(site string, r *http.Request) {
	if s == nil || r == nil || r.URL == nil || !s.accepting.Load() {
		return
	}
	ev := s.observation(site, r, s.now())
	if ev.SessionKey == "" || ev.OperationID == "" {
		return
	}
	select {
	case s.queue <- ev:
	default:
		s.dropped.Add(1)
	}
}

func (s *sequenceStore) observation(site string, r *http.Request, at time.Time) sequenceObservation {
	if s == nil || r == nil || r.URL == nil {
		return sequenceObservation{}
	}
	site = strings.TrimSpace(site)
	opID := apiOperationID(site, r.Method, normalizeAPIOperationPath(r.URL.Path))
	kind, material := "anonymous", anonymousSequenceMaterial(r)
	workflowMaterial := "anonymous"
	identityKind := "anonymous"
	if id, ok := verifiedAPIIdentity(r); ok && (strings.TrimSpace(id.Subject) != "" || strings.TrimSpace(id.ClientID) != "") {
		kind = "verified_identity"
		identityKind = "authenticated"
		material = strings.Join([]string{id.Issuer, id.Subject, id.TenantID, id.ClientID}, "\x00")
		workflowMaterial = verifiedWorkflowMaterial(id)
	}
	return sequenceObservation{
		SessionKey: s.digest(kind + "\x00" + site + "\x00" + material), Site: site,
		CorrelationKind: kind, WorkflowID: s.digest("workflow\x00" + site + "\x00" + identityKind + "\x00" + workflowMaterial),
		IdentityKind: identityKind, OperationID: opID, At: at.UTC(),
	}
}

func anonymousSequenceMaterial(r *http.Request) string {
	for _, name := range []string{"session", "sessionid", "sid", "JSESSIONID", "PHPSESSID"} {
		if c, err := r.Cookie(name); err == nil && strings.TrimSpace(c.Value) != "" {
			return "cookie\x00" + name + "\x00" + c.Value
		}
	}
	// clientIP has already applied the trusted-proxy policy. The value remains
	// only as HMAC input and is never copied into a session or persistence file.
	return "client\x00" + clientIP(r) + "\x00" + r.UserAgent()
}

func (s *sequenceStore) digest(material string) string {
	s.keyMu.RLock()
	key := append([]byte(nil), s.correlationKey...)
	s.keyMu.RUnlock()
	h := hmac.New(sha256.New, key)
	_, _ = h.Write([]byte(material))
	return hex.EncodeToString(h.Sum(nil)[:20])
}

func sequenceTransitionID(site, from, to string) string {
	h := sha256.Sum256([]byte(site + "\x00" + from + "\x00" + to))
	return hex.EncodeToString(h[:20])
}

func sequenceWorkflowTransitionID(site, workflowID, from, to string) string {
	h := sha256.Sum256([]byte(site + "\x00" + workflowID + "\x00" + from + "\x00" + to))
	return hex.EncodeToString(h[:20])
}

func (s *sequenceStore) process(ev sequenceObservation) {
	if s == nil || ev.SessionKey == "" || ev.OperationID == "" || ev.At.IsZero() {
		return
	}
	if ev.WorkflowID == "" {
		ev.WorkflowID = s.digest("workflow\x00" + ev.Site + "\x00legacy")
	}
	if ev.IdentityKind == "" {
		ev.IdentityKind = "anonymous"
	}
	s.stateMu.Lock()
	s.pruneLocked(ev.At)
	prev, exists := s.sessions[ev.SessionKey]
	if exists && !prev.AbsoluteExpiresAt.IsZero() && !prev.AbsoluteExpiresAt.After(ev.At) {
		s.finalizeSessionLocked(prev, ev.At)
		delete(s.sessions, ev.SessionKey)
		exists = false
	}
	if !exists {
		s.detectUnexpectedEntryLocked(ev)
		s.ensureSessionCapacityLocked(ev.At)
		prev = SequenceSession{
			Key: ev.SessionKey, Site: ev.Site, CorrelationKind: ev.CorrelationKind, WorkflowID: ev.WorkflowID,
			EntryOperationID: ev.OperationID, StartedAt: ev.At, LastSeen: ev.At, ExpiresAt: ev.At.Add(s.settings.SessionTTL),
			AbsoluteExpiresAt: ev.At.Add(s.settings.SessionAbsoluteTTL),
		}
		s.noteWorkflowSessionStartLocked(ev)
	}
	from := prev.CurrentOperationID
	if from != "" {
		s.detectTransitionAnomaliesLocked(ev, prev, from)
	}
	prev.CurrentOperationID = ev.OperationID
	prev.LastSeen = ev.At
	prev.ExpiresAt = ev.At.Add(s.settings.SessionTTL)
	if prev.ExpiresAt.After(prev.AbsoluteExpiresAt) {
		prev.ExpiresAt = prev.AbsoluteExpiresAt
	}
	if prev.RequestCount < ^uint64(0) {
		prev.RequestCount++
	}
	prev.WorkflowDepth = prev.RequestCount
	prev.RecentOperationIDs = append(prev.RecentOperationIDs, ev.OperationID)
	if len(prev.RecentOperationIDs) > s.settings.MaxRecentOps {
		prev.RecentOperationIDs = append([]string(nil), prev.RecentOperationIDs[len(prev.RecentOperationIDs)-s.settings.MaxRecentOps:]...)
	}
	s.noteWorkflowObservationLocked(ev, prev.WorkflowDepth)
	if from != "" {
		id := sequenceWorkflowTransitionID(ev.Site, ev.WorkflowID, from, ev.OperationID)
		edge, ok := s.transitions[id]
		if !ok {
			s.ensureTransitionCapacityLocked()
			edge = SequenceTransition{ID: id, Site: ev.Site, WorkflowID: ev.WorkflowID, FromOperationID: from, ToOperationID: ev.OperationID, FirstSeen: ev.At, Maturity: sequenceMaturityLearning}
		}
		if edge.ObservationCount < ^uint64(0) {
			edge.ObservationCount++
		}
		edge.Count = edge.ObservationCount
		if !containsSequenceID(prev.SeenTransitionIDs, id) && len(prev.SeenTransitionIDs) < s.settings.MaxSeenTransitions {
			if edge.SessionCount < ^uint64(0) {
				edge.SessionCount++
			}
			prev.SeenTransitionIDs = append(prev.SeenTransitionIDs, id)
		}
		edge.LastSeen = ev.At
		edge.ExpiresAt = ev.At.Add(s.settings.TransitionTTL)
		s.transitions[id] = edge
	}
	s.sessions[ev.SessionKey] = prev
	s.processed++
	publish := s.processed%sequencePublishEvery == 0
	s.stateMu.Unlock()
	if publish {
		s.publishSnapshot()
	}
}

func (s *sequenceStore) pruneLocked(now time.Time) {
	s.pruneDetectionLocked(now)
	s.pruneOperationsLocked(now)
	for k, v := range s.sessions {
		if !v.ExpiresAt.After(now) || (!v.AbsoluteExpiresAt.IsZero() && !v.AbsoluteExpiresAt.After(now)) {
			s.finalizeSessionLocked(v, now)
			delete(s.sessions, k)
		}
	}
	for k, v := range s.transitions {
		if !v.ExpiresAt.After(now) {
			delete(s.transitions, k)
		}
	}
	for k, v := range s.workflows {
		if v == nil || (!v.LastSeen.IsZero() && !v.LastSeen.Add(s.settings.WorkflowTTL).After(now)) {
			delete(s.workflows, k)
		}
	}
}

func (s *sequenceStore) ensureSessionCapacityLocked(now time.Time) {
	if len(s.sessions) < s.settings.MaxSessions {
		return
	}
	oldestKey := ""
	var oldest time.Time
	for k, v := range s.sessions {
		if oldestKey == "" || v.LastSeen.Before(oldest) || (v.LastSeen.Equal(oldest) && k < oldestKey) {
			oldestKey, oldest = k, v.LastSeen
		}
	}
	if oldestKey != "" {
		s.finalizeSessionLocked(s.sessions[oldestKey], now)
		delete(s.sessions, oldestKey)
		s.evictedSessions.Add(1)
	}
}

func (s *sequenceStore) ensureTransitionCapacityLocked() {
	if len(s.transitions) < s.settings.MaxTransitions {
		return
	}
	oldestKey := ""
	var oldest time.Time
	for k, v := range s.transitions {
		if oldestKey == "" || v.LastSeen.Before(oldest) || (v.LastSeen.Equal(oldest) && k < oldestKey) {
			oldestKey, oldest = k, v.LastSeen
		}
	}
	if oldestKey != "" {
		delete(s.transitions, oldestKey)
		s.evictedTransitions.Add(1)
	}
}

func (s *sequenceStore) publishSnapshot() {
	if s == nil {
		return
	}
	// Serialize clone/sort/store so an older concurrent clone can never replace
	// a newer published model after spending longer in sorting.
	s.publishMu.Lock()
	defer s.publishMu.Unlock()
	now := s.now()
	s.stateMu.Lock()
	s.pruneLocked(now)
	s.refreshLearningDerivedLocked(now)
	model := SequenceModel{
		Version: sequenceStateVersion, GeneratedAt: now,
		SessionTTLSeconds: int64(s.settings.SessionTTL / time.Second), SessionAbsoluteSeconds: int64(s.settings.SessionAbsoluteTTL / time.Second),
		TransitionTTLSeconds: int64(s.settings.TransitionTTL / time.Second), WorkflowTTLSeconds: int64(s.settings.WorkflowTTL / time.Second),
		StaleAfterSeconds: int64(s.settings.StaleAfter / time.Second),
		MaxSessions:       s.settings.MaxSessions, MaxTransitions: s.settings.MaxTransitions, MaxWorkflows: s.settings.MaxWorkflows,
		MaxWorkflowOperations: s.settings.MaxWorkflowOperations, MaxRecentOperations: s.settings.MaxRecentOps,
		MinObservations: s.settings.MinObservations, MinSessions: s.settings.MinSessions,
		MinLearningSeconds: int64(s.settings.MinLearningDuration / time.Second), ConfidenceThreshold: s.settings.ConfidenceThreshold,
		QueueCapacity: cap(s.queue), QueueDepth: len(s.queue), DroppedObservations: s.dropped.Load(),
		EvictedSessions: s.evictedSessions.Load(), EvictedTransitions: s.evictedTransitions.Load(), EvictedWorkflows: s.evictedWorkflows.Load(), EvictedViolations: s.evictedViolations.Load(), EvictedRecentSessions: s.evictedRecentSessions.Load(),
		ViolationTTLSeconds: int64(s.settings.ViolationTTL / time.Second), MaxViolations: s.settings.MaxViolations, MaxExceptions: s.settings.MaxExceptions,
		EntryConfidenceThreshold: s.settings.EntryConfidenceThreshold, DivergenceExpectedMinConfidence: s.settings.DivergenceExpectedMinConfidence,
		DivergenceObservedMaxConfidence: s.settings.DivergenceObservedMaxConfidence, RepetitionTailProbability: s.settings.RepetitionTailProbability, RepetitionMinimumRun: s.settings.RepetitionMinimumRun,
		RecentSessionTTLSeconds: int64(s.settings.RecentSessionTTL / time.Second), MaxRecentSessions: s.settings.MaxRecentSessions,
		Sessions: make([]SequenceSession, 0, len(s.sessions)), Transitions: make([]SequenceTransition, 0, len(s.transitions)),
		Workflows: make([]SequenceWorkflowModel, 0, len(s.workflows)), Violations: cloneSequenceViolations(s.violations), Exceptions: make([]SequenceException, 0, len(s.exceptions)),
		RecentSessions: cloneSequenceRecentSessions(s.recentSessions), Controls: make([]SequenceSiteControl, 0, len(s.controls)),
	}
	for _, v := range s.sessions {
		v.RecentOperationIDs = append([]string(nil), v.RecentOperationIDs...)
		v.SeenTransitionIDs = append([]string(nil), v.SeenTransitionIDs...)
		model.Sessions = append(model.Sessions, v)
	}
	for _, v := range s.transitions {
		model.Transitions = append(model.Transitions, v)
	}
	for _, v := range s.workflows {
		if v != nil {
			workflow := v.snapshot(s.settings, now)
			workflow.DetectionState = sequenceDetectionLearn
			if s.sequenceModeLocked(v.Site) == sequenceDetectionDetect && workflow.Maturity == sequenceMaturityMature {
				workflow.DetectionState = sequenceDetectionDetect
			}
			model.Workflows = append(model.Workflows, workflow)
		}
	}
	for _, v := range s.exceptions {
		model.Exceptions = append(model.Exceptions, v)
	}
	for _, v := range s.controls {
		model.Controls = append(model.Controls, v)
	}
	s.stateMu.Unlock()
	sort.Slice(model.Sessions, func(i, j int) bool { return model.Sessions[i].Key < model.Sessions[j].Key })
	sort.Slice(model.Transitions, func(i, j int) bool { return model.Transitions[i].ID < model.Transitions[j].ID })
	sort.Slice(model.Workflows, func(i, j int) bool { return model.Workflows[i].ID < model.Workflows[j].ID })
	sort.Slice(model.Violations, func(i, j int) bool {
		if model.Violations[i].Time.Equal(model.Violations[j].Time) {
			return model.Violations[i].ID < model.Violations[j].ID
		}
		return model.Violations[i].Time.Before(model.Violations[j].Time)
	})
	sort.Slice(model.Exceptions, func(i, j int) bool { return model.Exceptions[i].ID < model.Exceptions[j].ID })
	sort.Slice(model.RecentSessions, func(i, j int) bool {
		if model.RecentSessions[i].EndedAt.Equal(model.RecentSessions[j].EndedAt) {
			return model.RecentSessions[i].SessionFingerprint < model.RecentSessions[j].SessionFingerprint
		}
		return model.RecentSessions[i].EndedAt.Before(model.RecentSessions[j].EndedAt)
	})
	sort.Slice(model.Controls, func(i, j int) bool { return model.Controls[i].Site < model.Controls[j].Site })
	s.runtime.Store(&model)
}

func (s *sequenceStore) snapshot() SequenceModel {
	if s == nil {
		return SequenceModel{}
	}
	s.publishSnapshot()
	p := s.runtime.Load()
	if p == nil {
		return SequenceModel{}
	}
	out := *p
	out.Sessions = append([]SequenceSession(nil), p.Sessions...)
	for i := range out.Sessions {
		out.Sessions[i].RecentOperationIDs = append([]string(nil), p.Sessions[i].RecentOperationIDs...)
		out.Sessions[i].SeenTransitionIDs = append([]string(nil), p.Sessions[i].SeenTransitionIDs...)
	}
	out.Transitions = append([]SequenceTransition(nil), p.Transitions...)
	out.Workflows = cloneSequenceWorkflowModels(p.Workflows)
	out.Violations = cloneSequenceViolations(p.Violations)
	out.Exceptions = cloneSequenceExceptions(p.Exceptions)
	out.RecentSessions = cloneSequenceRecentSessions(p.RecentSessions)
	out.Controls = cloneSequenceControls(p.Controls)
	return out
}

func (s *sequenceStore) save(configPath string) error {
	if s == nil {
		return nil
	}
	if err := s.ensureCorrelationKey(configPath); err != nil {
		return err
	}
	model := s.snapshot()
	state := sequenceStateFile{Version: sequenceStateVersion, Saved: time.Now().UTC(), Sessions: model.Sessions, Transitions: model.Transitions, Workflows: model.Workflows, Violations: model.Violations, Exceptions: model.Exceptions, RecentSessions: model.RecentSessions, Controls: model.Controls}
	return atomicWriteJSON(apiSecurityStatePath(configPath, "api-sequence.json"), state)
}

func (s *sequenceStore) load(configPath string) error {
	if s == nil {
		return nil
	}
	if err := s.ensureCorrelationKey(configPath); err != nil {
		return err
	}
	var state sequenceStateFile
	if err := readJSONIfExists(apiSecurityStatePath(configPath, "api-sequence.json"), &state); err != nil {
		return err
	}
	if state.Version != 0 && state.Version != 1 && state.Version != 2 && state.Version != 3 && state.Version != sequenceStateVersion {
		return fmt.Errorf("unsupported API sequence state version %d", state.Version)
	}
	now := s.now()
	s.stateMu.Lock()
	if state.Version == 1 {
		s.migrateV1StateLocked(&state, now)
	}
	for _, v := range state.Workflows {
		if len(s.workflows) >= s.settings.MaxWorkflows || !validSequenceWorkflow(v, now, s.settings) {
			continue
		}
		s.workflows[v.ID] = workflowRuntimeFromModel(v, s.settings)
	}
	for _, v := range state.Sessions {
		if !validSequenceSession(v, now, s.settings.MaxRecentOps, s.settings.MaxSeenTransitions) || len(s.sessions) >= s.settings.MaxSessions {
			continue
		}
		v.RecentOperationIDs = append([]string(nil), v.RecentOperationIDs...)
		v.SeenTransitionIDs = append([]string(nil), v.SeenTransitionIDs...)
		s.sessions[v.Key] = v
	}
	for _, v := range state.Transitions {
		if !validSequenceTransition(v, now) || len(s.transitions) >= s.settings.MaxTransitions {
			continue
		}
		if v.ObservationCount == 0 {
			v.ObservationCount = v.Count
		}
		v.Count = v.ObservationCount
		s.transitions[v.ID] = v
	}
	for _, v := range state.Violations {
		if len(s.violations) >= s.settings.MaxViolations || !validSequenceViolation(v, now) {
			continue
		}
		s.violations = append(s.violations, v)
	}
	for _, v := range state.Exceptions {
		if len(s.exceptions) >= s.settings.MaxExceptions || !validSequenceException(v, now) {
			continue
		}
		s.exceptions[v.ID] = v
	}
	for _, v := range state.RecentSessions {
		if len(s.recentSessions) >= s.settings.MaxRecentSessions || !validSequenceRecentSession(v, now) {
			continue
		}
		s.recentSessions = append(s.recentSessions, v)
	}
	for _, v := range state.Controls {
		if !validSequenceSite(v.Site) || !validSequenceMode(v.Mode) || len(s.controls) >= s.settings.MaxWorkflows {
			continue
		}
		s.controls[v.Site] = v
	}
	s.stateMu.Unlock()
	s.publishSnapshot()
	return nil
}

func validSequenceOperationID(v string) bool {
	if len(v) != 32 {
		return false
	}
	_, err := hex.DecodeString(v)
	return err == nil
}

func validSequenceSession(v SequenceSession, now time.Time, recentLimit, seenLimit int) bool {
	if len(v.Key) != 40 || (v.CorrelationKind != "anonymous" && v.CorrelationKind != "verified_identity") ||
		strings.TrimSpace(v.Site) == "" || !validSequenceOperationID(v.CurrentOperationID) || !v.ExpiresAt.After(now) ||
		(!v.AbsoluteExpiresAt.IsZero() && !v.AbsoluteExpiresAt.After(now)) || len(v.RecentOperationIDs) > recentLimit || len(v.SeenTransitionIDs) > seenLimit {
		return false
	}
	if _, err := hex.DecodeString(v.Key); err != nil {
		return false
	}
	for _, op := range v.RecentOperationIDs {
		if !validSequenceOperationID(op) {
			return false
		}
	}
	return true
}

func validSequenceTransition(v SequenceTransition, now time.Time) bool {
	if len(v.ID) != 40 || strings.TrimSpace(v.Site) == "" || !validSequenceOperationID(v.FromOperationID) ||
		!validSequenceOperationID(v.ToOperationID) || !v.ExpiresAt.After(now) || v.Confidence < 0 || v.Confidence > 1 {
		return false
	}
	_, err := hex.DecodeString(v.ID)
	return err == nil
}

func (s *sequenceStore) ensureCorrelationKey(configPath string) error {
	path := apiSecurityStatePath(configPath, "api-sequence.key")
	info, err := os.Lstat(path)
	if err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return errors.New("api-sequence.key must be a regular file")
		}
		b, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if len(b) != 32 {
			return errors.New("api-sequence.key must contain exactly 32 bytes")
		}
		s.keyMu.Lock()
		s.correlationKey = append(s.correlationKey[:0], b...)
		s.keyMu.Unlock()
		return nil
	}
	if !os.IsNotExist(err) {
		return err
	}
	s.keyMu.RLock()
	key := append([]byte(nil), s.correlationKey...)
	s.keyMu.RUnlock()
	if len(key) != 32 {
		return errors.New("sequence correlation key unavailable")
	}
	dir := filepath.Dir(path)
	f, err := os.CreateTemp(dir, ".api-sequence.key.tmp-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	ok := false
	defer func() {
		_ = f.Close()
		if !ok {
			_ = os.Remove(tmp)
		}
	}()
	if err := f.Chmod(0o600); err != nil {
		return err
	}
	if _, err := f.Write(key); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		if existing, readErr := os.ReadFile(path); readErr == nil && len(existing) == 32 {
			return s.ensureCorrelationKey(configPath)
		}
		return err
	}
	ok = true
	return nil
}

func (a *adminServer) handleSequenceModel(w http.ResponseWriter, r *http.Request) {
	model := a.srv.sequence.snapshot()
	writeJSON(w, map[string]any{
		"version": model.Version, "generated_at": model.GeneratedAt,
		"session_count": len(model.Sessions), "transition_count": len(model.Transitions), "workflow_count": len(model.Workflows),
		"session_ttl_seconds": model.SessionTTLSeconds, "session_absolute_seconds": model.SessionAbsoluteSeconds,
		"transition_ttl_seconds": model.TransitionTTLSeconds, "workflow_ttl_seconds": model.WorkflowTTLSeconds, "stale_after_seconds": model.StaleAfterSeconds,
		"max_sessions": model.MaxSessions, "max_transitions": model.MaxTransitions, "max_workflows": model.MaxWorkflows,
		"max_workflow_operations": model.MaxWorkflowOperations, "max_recent_operations": model.MaxRecentOperations,
		"min_observations": model.MinObservations, "min_sessions": model.MinSessions, "min_learning_duration_seconds": model.MinLearningSeconds,
		"confidence_threshold": model.ConfidenceThreshold, "queue_capacity": model.QueueCapacity,
		"queue_depth": model.QueueDepth, "dropped_observations": model.DroppedObservations,
		"evicted_sessions": model.EvictedSessions, "evicted_transitions": model.EvictedTransitions, "evicted_workflows": model.EvictedWorkflows, "evicted_violations": model.EvictedViolations,
		"violation_count": len(model.Violations), "exception_count": len(model.Exceptions), "violation_ttl_seconds": model.ViolationTTLSeconds, "max_violations": model.MaxViolations, "max_exceptions": model.MaxExceptions,
		"entry_confidence_threshold": model.EntryConfidenceThreshold, "divergence_expected_min_confidence": model.DivergenceExpectedMinConfidence, "divergence_observed_max_confidence": model.DivergenceObservedMaxConfidence,
		"repetition_tail_probability": model.RepetitionTailProbability, "repetition_minimum_run": model.RepetitionMinimumRun,
		"recent_session_count": len(model.RecentSessions), "recent_session_ttl_seconds": model.RecentSessionTTLSeconds, "max_recent_sessions": model.MaxRecentSessions, "evicted_recent_sessions": model.EvictedRecentSessions, "control_count": len(model.Controls),
	})
}

func (a *adminServer) handleSequenceSessions(w http.ResponseWriter, r *http.Request) {
	model := a.srv.sequence.snapshot()
	a.audit.add(who(r).user, "api_sequence.sessions_view", fmt.Sprintf("count=%d", len(model.Sessions)))
	writeJSON(w, model.Sessions)
}

func (a *adminServer) handleSequenceTransitions(w http.ResponseWriter, r *http.Request) {
	model := a.srv.sequence.snapshot()
	a.audit.add(who(r).user, "api_sequence.transitions_view", fmt.Sprintf("count=%d", len(model.Transitions)))
	writeJSON(w, model.Transitions)
}
