package main

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// API-1 foundation: normalized API operation discovery.
// This is telemetry only. It never participates in request blocking.

type apiOperation struct {
	Site        string           `json:"site"`
	Method      string           `json:"method"`
	Path        string           `json:"path"`
	Fingerprint string           `json:"fingerprint"`
	Confidence  float64          `json:"confidence"`
	Samples     int64            `json:"samples"`
	Status      map[string]int64 `json:"status_distribution,omitempty"`
	FirstSeen   string           `json:"first_seen"`
	LastSeen    string           `json:"last_seen"`
}

type apiOperationStore struct {
	mu  sync.Mutex
	ops map[string]*apiOperation
	cap int
}

func newAPIOperationStore() *apiOperationStore {
	return &apiOperationStore{ops: make(map[string]*apiOperation), cap: 10000}
}

var (
	apiUUID = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	apiHex  = regexp.MustCompile(`(?i)^[0-9a-f]{16,}$`)
	apiNum  = regexp.MustCompile(`^[0-9]+$`)
	apiDate = regexp.MustCompile(`^[0-9]{4}[-]?[0-9]{2}[-]?[0-9]{2}$`)
	apiULID = regexp.MustCompile(`(?i)^[0-9ABCDEFGHJKMNPQRSTVWXYZ]{26}$`)
)

func operationFingerprint(method, path string) string {
	h := sha256.Sum256([]byte(strings.ToUpper(method) + " " + path))
	return hex.EncodeToString(h[:])
}

func normalizeAPIOperationPath(path string) string {
	if path == "" {
		return "/"
	}
	parts := strings.Split(path, "/")
	for i, p := range parts {
		if apiUUID.MatchString(p) || apiHex.MatchString(p) || apiNum.MatchString(p) || apiULID.MatchString(p) {
			parts[i] = "{id}"
		} else if apiDate.MatchString(p) {
			parts[i] = "{date}"
		}
	}
	return strings.Join(parts, "/")
}

func (s *apiOperationStore) note(site, method, path string, code int) {
	if s == nil {
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	key := site + "|" + strings.ToUpper(method) + "|" + normalizeAPIOperationPath(path)
	s.mu.Lock()
	defer s.mu.Unlock()
	op := s.ops[key]
	if op == nil {
		if len(s.ops) >= s.cap {
			return
		}
		normalized := normalizeAPIOperationPath(path)
		op = &apiOperation{Site: site, Method: strings.ToUpper(method), Path: normalized, Fingerprint: operationFingerprint(strings.ToUpper(method), normalized), Confidence: normalizationConfidence(path, normalized), Status: map[string]int64{}, FirstSeen: now}
		s.ops[key] = op
	}
	op.Samples++
	op.Status[formatStatus(code)]++
	op.LastSeen = now
}

func formatStatus(code int) string {
	if code <= 0 {
		return "unknown"
	}
	return strconv.Itoa(code)
}

func (s *apiOperationStore) snapshot() []apiOperation {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]apiOperation, 0, len(s.ops))
	for _, v := range s.ops {
		out = append(out, *v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Samples > out[j].Samples })
	return out
}

func normalizationConfidence(raw, normalized string) float64 {
	if raw == normalized {
		return 0.5
	}
	return 0.95
}
