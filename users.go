package main

// Multi-user auth + role-based access + audit trail.
//
// Modeled on the mini-SIEM's RBAC idea, with one deliberate upgrade: passwords
// are stored as PBKDF2-HMAC-SHA256 (iterated, salted) rather than a single
// salted SHA-256 — the correct version of that idea, brute-force resistant,
// and stdlib-only.
//
// The startup admin token remains a break-glass credential (always full admin)
// so you can never lock yourself out. Named users are layered on top for
// day-to-day access and audit attribution. Sessions are in-memory (re-login
// after a restart); a persistence seam exists for later.

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ── roles ───────────────────────────────────────────────────────────────

// Role hierarchy: admin > operator > viewer. A higher role satisfies any lower
// requirement. reviewer sits beside operator: it may apply suggestions/page
// policies but not edit global config or manage users.
const (
	roleAdmin    = "admin"
	roleOperator = "operator"
	roleReviewer = "reviewer"
	roleViewer   = "viewer"
)

func roleRank(r string) int {
	switch r {
	case roleAdmin:
		return 4
	case roleOperator:
		return 3
	case roleReviewer:
		return 2
	case roleViewer:
		return 1
	}
	return 0
}

func validRole(r string) bool { return roleRank(r) > 0 }

// capabilities each identity is allowed. Kept explicit so the UI and server
// agree. "review" = apply page policies / profiles / learned suggestions.
func canManageUsers(role string) bool { return role == roleAdmin }
func canEditConfig(role string) bool  { return roleRank(role) >= roleRank(roleOperator) }
func canReview(role string) bool      { return role == roleReviewer || roleRank(role) >= roleRank(roleOperator) }

// ── user model ──────────────────────────────────────────────────────────

type UserConfig struct {
	Username     string `json:"username"`
	PasswordHash string `json:"password_hash"` // pbkdf2$sha256$iter$salt$hash ; masked on read
	Role         string `json:"role"`
	Disabled     bool   `json:"disabled,omitempty"`
}

func (c Config) validateUsers() error {
	seen := map[string]bool{}
	for i, u := range c.Users {
		if strings.TrimSpace(u.Username) == "" {
			return fmt.Errorf("user %d: username required", i+1)
		}
		if seen[strings.ToLower(u.Username)] {
			return fmt.Errorf("duplicate user %q", u.Username)
		}
		seen[strings.ToLower(u.Username)] = true
		if !validRole(u.Role) {
			return fmt.Errorf("user %q: role must be admin, operator, reviewer, or viewer", u.Username)
		}
		if u.PasswordHash == "" {
			return fmt.Errorf("user %q: password not set", u.Username)
		}
	}
	return nil
}

// ── password hashing (PBKDF2-HMAC-SHA256, stdlib only) ───────────────────

const pbkdf2Iter = 210000

func pbkdf2SHA256(password, salt []byte, iter, keyLen int) []byte {
	prf := func(block int) []byte {
		mac := hmac.New(sha256.New, password)
		mac.Write(salt)
		var b [4]byte
		binary.BigEndian.PutUint32(b[:], uint32(block))
		mac.Write(b[:])
		u := mac.Sum(nil)
		out := make([]byte, len(u))
		copy(out, u)
		for i := 1; i < iter; i++ {
			mac = hmac.New(sha256.New, password)
			mac.Write(u)
			u = mac.Sum(nil)
			for j := range out {
				out[j] ^= u[j]
			}
		}
		return out
	}
	var dk []byte
	blocks := (keyLen + sha256.Size - 1) / sha256.Size
	for i := 1; i <= blocks; i++ {
		dk = append(dk, prf(i)...)
	}
	return dk[:keyLen]
}

func hashPassword(pw string) (string, error) {
	if len(pw) < 8 {
		return "", fmt.Errorf("password must be at least 8 characters")
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	dk := pbkdf2SHA256([]byte(pw), salt, pbkdf2Iter, 32)
	return fmt.Sprintf("pbkdf2$sha256$%d$%s$%s", pbkdf2Iter,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(dk)), nil
}

func verifyPassword(pw, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 5 || parts[0] != "pbkdf2" || parts[1] != "sha256" {
		return false
	}
	iter, err := strconv.Atoi(parts[2])
	if err != nil || iter < 1 {
		return false
	}
	salt, err1 := base64.RawStdEncoding.DecodeString(parts[3])
	want, err2 := base64.RawStdEncoding.DecodeString(parts[4])
	if err1 != nil || err2 != nil {
		return false
	}
	got := pbkdf2SHA256([]byte(pw), salt, iter, len(want))
	return subtle.ConstantTimeCompare(got, want) == 1
}

// ── sessions (in-memory) ────────────────────────────────────────────────

type session struct {
	user    string
	role    string
	expires time.Time
}

type sessionStore struct {
	mu   sync.Mutex
	byID map[string]session
	ttl  time.Duration
}

func newSessionStore() *sessionStore {
	return &sessionStore{byID: map[string]session{}, ttl: 12 * time.Hour}
}

func (s *sessionStore) create(user, role string) string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	id := base64.RawURLEncoding.EncodeToString(b)
	s.mu.Lock()
	s.byID[id] = session{user: user, role: role, expires: time.Now().Add(s.ttl)}
	s.mu.Unlock()
	return id
}

func (s *sessionStore) lookup(id string) (session, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.byID[id]
	if !ok {
		return session{}, false
	}
	if time.Now().After(sess.expires) {
		delete(s.byID, id)
		return session{}, false
	}
	return sess, true
}

func (s *sessionStore) destroy(id string) {
	s.mu.Lock()
	delete(s.byID, id)
	s.mu.Unlock()
}

// revoke drops all sessions for a user (on disable/delete/role change).
func (s *sessionStore) revoke(user string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, sess := range s.byID {
		if strings.EqualFold(sess.user, user) {
			delete(s.byID, id)
		}
	}
}

// ── audit trail (in-memory ring) ─────────────────────────────────────────

type auditEntry struct {
	Time   string `json:"time"`
	User   string `json:"user"`
	Action string `json:"action"`
	Detail string `json:"detail"`
}

type auditLog struct {
	mu   sync.Mutex
	recs []auditEntry
	cap  int
	sink func(user, action, detail string)
}

func newAuditLog() *auditLog { return &auditLog{cap: 500} }

func (a *auditLog) add(user, action, detail string) {
	a.mu.Lock()
	a.recs = append(a.recs, auditEntry{
		Time: time.Now().Format("2006-01-02 15:04:05"), User: user, Action: action, Detail: detail,
	})
	if len(a.recs) > a.cap {
		a.recs = a.recs[len(a.recs)-a.cap:]
	}
	sink := a.sink
	a.mu.Unlock()
	if sink != nil {
		sink(user, action, detail)
	}
}

func (a *auditLog) list(limit int) []auditEntry {
	a.mu.Lock()
	defer a.mu.Unlock()
	n := len(a.recs)
	if limit <= 0 || limit > n {
		limit = n
	}
	out := make([]auditEntry, limit)
	for i := 0; i < limit; i++ {
		out[i] = a.recs[n-1-i]
	}
	return out
}
