package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func newAPI72Stores() (*objectLocatorStore, *objectRelationshipStore) {
	locators := newObjectLocatorStoreWithKey([]byte("0123456789abcdef0123456789abcdef"))
	relations := newObjectRelationshipStoreWithKey([]byte("abcdef0123456789abcdef0123456789"))
	relations.locators = locators
	return locators, relations
}

func api72IncludeLocator(t *testing.T, s *objectLocatorStore, opID, location, field string) ObjectLocator {
	t.Helper()
	if _, err := s.upsertOverride(ObjectLocatorOverride{OperationID: opID, Location: location, Field: field, SemanticName: field, SchemaType: "string", Action: objectLocatorOverrideInclude}); err != nil {
		t.Fatal(err)
	}
	loc, ok := s.activeLocator(opID, location, field)
	if !ok {
		t.Fatalf("locator not active: %s %s", location, field)
	}
	return loc
}

func TestAPI72VerifiedIdentityAndKeyedObjectCreateRelationship(t *testing.T) {
	locators, relations := newAPI72Stores()
	opID := apiOperationID("shop", http.MethodGet, "/orders/{id}")
	api72IncludeLocator(t, locators, opID, "path", "id")
	id := VerifiedAPIIdentity{Issuer: "https://issuer.example/", Subject: "alice-secret", TenantID: "tenant-secret", ClientID: "client-secret"}
	r := &http.Request{Method: http.MethodGet, URL: &url.URL{Path: "/orders/12345"}}
	r = r.WithContext(context.WithValue(r.Context(), apiIdentityContextKey{}, id))
	ev := relations.observation(r, "shop")
	if ev.IdentityFingerprint == "" || strings.Contains(ev.IdentityFingerprint, "alice") {
		t.Fatalf("verified identity was not pseudonymized: %#v", ev)
	}
	relations.process(ev)
	rows := relations.snapshot()
	if len(rows) != 1 || rows[0].OperationID != opID || rows[0].ObjectFingerprint == "" {
		t.Fatalf("unexpected relationships: %#v", rows)
	}
	if rows[0].TenantFingerprint == "" || rows[0].ClientFingerprint == "" {
		t.Fatalf("verified tenant/client evidence not pseudonymized: %#v", rows[0])
	}
}

func TestAPI72RejectsUnverifiedAndCallerSuppliedOwnershipMetadata(t *testing.T) {
	_, relations := newAPI72Stores()
	r := &http.Request{Method: http.MethodGet, URL: &url.URL{Path: "/orders/123", RawQuery: "owner_id=alice"}, Header: http.Header{}}
	r.Header.Set("X-Tenant-ID", "attacker-tenant")
	r.Header.Set("X-Owner-ID", "attacker-owner")
	if ev := relations.observation(r, "shop"); ev.IdentityFingerprint != "" || ev.OperationID != "" {
		t.Fatalf("unverified request created relationship observation: %#v", ev)
	}
}

func TestAPI72SubjectAndClientIdentityPseudonymsAreSeparated(t *testing.T) {
	_, relations := newAPI72Stores()
	subject := relations.digestIdentity("subject", "issuer", "same-value")
	client := relations.digestIdentity("client", "issuer", "same-value")
	if subject == client || subject == "" || client == "" {
		t.Fatalf("identity domain separation failed: subject=%q client=%q", subject, client)
	}
}

func TestAPI72NormalizedOperationAndObjectFingerprintAvoidRawCardinality(t *testing.T) {
	locators, relations := newAPI72Stores()
	opID := apiOperationID("shop", http.MethodGet, "/orders/{id}")
	api72IncludeLocator(t, locators, opID, "path", "id")
	identity := relations.digestIdentity("subject", "issuer", "alice")
	for _, raw := range []string{"/orders/1", "/orders/999999"} {
		relations.process(objectRelationshipObservation{OperationID: opID, Path: raw, IdentityKind: "subject", IdentityFingerprint: identity, At: time.Now().UTC()})
	}
	rows := relations.snapshot()
	if len(rows) != 2 {
		t.Fatalf("expected two object relationships under one normalized operation, got %d", len(rows))
	}
	for _, row := range rows {
		if row.OperationID != opID || strings.Contains(row.ObjectFingerprint, "999999") {
			t.Fatalf("raw object leaked/cardinality changed: %#v", row)
		}
	}
}

func TestAPI72SuppressedLocatorCannotCreateRelationship(t *testing.T) {
	locators, relations := newAPI72Stores()
	opID := apiOperationID("shop", http.MethodGet, "/orders/{id}")
	api72IncludeLocator(t, locators, opID, "path", "id")
	if _, err := locators.upsertOverride(ObjectLocatorOverride{OperationID: opID, Location: "path", Field: "id", Action: objectLocatorOverrideSuppress}); err != nil {
		t.Fatal(err)
	}
	relations.process(objectRelationshipObservation{OperationID: opID, Path: "/orders/1", IdentityKind: "subject", IdentityFingerprint: relations.digestIdentity("subject", "issuer", "alice"), At: time.Now().UTC()})
	if got := len(relations.snapshot()); got != 0 {
		t.Fatalf("suppressed locator created %d relationships", got)
	}
}

func TestAPI72RelationshipTTLAndCardinalityAreBounded(t *testing.T) {
	locators, relations := newAPI72Stores()
	base := time.Unix(1_700_000_000, 0).UTC()
	relations.now = func() time.Time { return base }
	opID := apiOperationID("shop", http.MethodGet, "/objects/{id}")
	loc := api72IncludeLocator(t, locators, opID, "path", "id")
	for i := 0; i < objectRelationshipMaxPerObject+20; i++ {
		identity := relations.digestIdentity("subject", "issuer", fmt.Sprintf("user-%d", i))
		relations.merge(objectRelationshipObservation{OperationID: opID, IdentityKind: "subject", IdentityFingerprint: identity, At: base}, loc, locators.digestValue(opID, "path", "id", "object-1"))
	}
	if got := len(relations.snapshot()); got != objectRelationshipMaxPerObject {
		t.Fatalf("per-object cap not enforced: got=%d want=%d", got, objectRelationshipMaxPerObject)
	}
	relations.now = func() time.Time { return base.Add(objectRelationshipDefaultTTL + time.Second) }
	if got := len(relations.snapshot()); got != 0 {
		t.Fatalf("expired relationships retained: %d", got)
	}
}

func TestAPI72PersistenceRestoresKeyedEvidenceWithoutRawClaimsOrObjects(t *testing.T) {
	locators, relations := newAPI72Stores()
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	opID := apiOperationID("shop", http.MethodGet, "/orders/{id}")
	loc := api72IncludeLocator(t, locators, opID, "path", "id")
	identity := relations.digestIdentity("subject", "https://issuer.example/", "alice-raw-secret")
	relations.merge(objectRelationshipObservation{OperationID: opID, IdentityKind: "subject", IdentityFingerprint: identity, TenantFingerprint: relations.digestIdentity("tenant", "issuer", "tenant-raw-secret"), At: time.Now().UTC()}, loc, locators.digestValue(opID, "path", "id", "order-raw-secret"))
	if err := relations.save(configPath); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "api-object-relationships.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"alice-raw-secret", "tenant-raw-secret", "order-raw-secret", "Authorization", "Bearer "} {
		if strings.Contains(string(b), forbidden) {
			t.Fatalf("raw sensitive material persisted: %q", forbidden)
		}
	}
	keyInfo, err := os.Stat(filepath.Join(dir, "api-object-relationship.key"))
	if err != nil || keyInfo.Mode().Perm() != 0o600 {
		t.Fatalf("relationship key mode invalid: info=%v err=%v", keyInfo, err)
	}
	restored := newObjectRelationshipStoreWithKey([]byte("00000000000000000000000000000000"))
	restored.locators = locators
	if err := restored.load(configPath); err != nil {
		t.Fatal(err)
	}
	if got := len(restored.snapshot()); got != 1 {
		t.Fatalf("restart did not restore relationship evidence: %d", got)
	}
}

func TestAPI72RepeatedObservationIsEvidenceNotOwnershipVerdict(t *testing.T) {
	locators, relations := newAPI72Stores()
	opID := apiOperationID("shop", http.MethodGet, "/orders/{id}")
	loc := api72IncludeLocator(t, locators, opID, "path", "id")
	ev := objectRelationshipObservation{OperationID: opID, IdentityKind: "subject", IdentityFingerprint: relations.digestIdentity("subject", "issuer", "alice"), At: time.Now().UTC()}
	fp := locators.digestValue(opID, "path", "id", "42")
	relations.merge(ev, loc, fp)
	ev.At = ev.At.Add(time.Second)
	relations.merge(ev, loc, fp)
	row := relations.snapshot()[0]
	if row.EvidenceLevel != objectRelationshipEvidenceRepeated || row.ObservationCount != 2 {
		t.Fatalf("repeat evidence incorrect: %#v", row)
	}
}

func TestAPI72ConcurrentRelationshipUpdatesRemainBounded(t *testing.T) {
	locators, relations := newAPI72Stores()
	opID := apiOperationID("shop", http.MethodGet, "/orders/{id}")
	loc := api72IncludeLocator(t, locators, opID, "path", "id")
	fp := locators.digestValue(opID, "path", "id", "42")
	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				relations.merge(objectRelationshipObservation{OperationID: opID, IdentityKind: "subject", IdentityFingerprint: relations.digestIdentity("subject", "issuer", fmt.Sprintf("user-%d", i)), At: time.Now().UTC()}, loc, fp)
			}
		}(i)
	}
	wg.Wait()
	if got := len(relations.snapshot()); got > objectRelationshipMaxPerObject || got > objectRelationshipMaxRelations {
		t.Fatalf("relationship bounds exceeded: %d", got)
	}
}
