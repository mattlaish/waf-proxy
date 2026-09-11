package main

import (
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"
)

func TestPhase4PersistentSecurityStateRoundTripAndNoRawSessionToken(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	emptyBlocks := blockSnapshot{}
	ai := &aiEngine{}
	ai.block.Store(&emptyBlocks)
	s := &server{ai: ai, learn: newLearnStore(), notify: newNotifier(log), security: newSecurityManager(), log: log}
	admin := &adminServer{sessions: newSessionStore(), audit: newAuditLog(), log: log}

	token := admin.sessions.create("alice", roleOperator)
	admin.audit.add("alice", "test.action", "detail")
	s.notify.push(notifyInfo, "info", "title", "body", "", "", nil)
	s.learn.noteRequest("site-a", "/login", httpStatusOKForStateTest)
	s.learn.noteMatch("site-a", "/login", 942100, "192.0.2.1", "WARNING")
	blocks := blockSnapshot{blockKey("site-a", "198.51.100.4"): {IP: "198.51.100.4", Site: "site-a", Score: 90, Reason: "test", Expires: time.Now().Add(time.Hour)}}
	s.ai.block.Store(&blocks)
	s.security.allowed.Store(7)

	path := t.TempDir() + "/security-state.json"
	if err := savePersistentSecurityState(path, s, admin); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), token) {
		t.Fatal("raw bearer token persisted to disk")
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Fatalf("state file mode=%v", fi.Mode().Perm())
	}

	empty2 := blockSnapshot{}
	ai2 := &aiEngine{}
	ai2.block.Store(&empty2)
	s2 := &server{ai: ai2, learn: newLearnStore(), notify: newNotifier(log), security: newSecurityManager(), log: log}
	admin2 := &adminServer{sessions: newSessionStore(), audit: newAuditLog(), log: log}
	if err := loadPersistentSecurityState(path, s2, admin2); err != nil {
		t.Fatal(err)
	}
	if _, ok := admin2.sessions.lookup(token); !ok {
		t.Fatal("hashed persisted session did not survive restart")
	}
	if len(admin2.audit.list(10)) != 1 || s2.notify.unreadCount() != 1 {
		t.Fatal("audit/notification state did not round trip")
	}
	if len(s2.ai.blocklist()) != 1 || s2.security.counters().Allowed != 7 {
		t.Fatal("AI block/security counters did not round trip")
	}
	rec := s2.learn.recommend("site-a")
	if len(rec.Pages) != 1 || rec.Pages[0].Path != "/login" {
		t.Fatalf("learner state did not round trip: %#v", rec)
	}
}

const httpStatusOKForStateTest = 200
