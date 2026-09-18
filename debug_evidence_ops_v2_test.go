package main

import (
	"testing"
	"time"
)

func TestDebugEvidenceOpsV2ListUsesCurrentBundleModel(t *testing.T) {
	store := NewDebugEvidenceStore(10, time.Hour)
	store.Put(DebugBundle{Tenant: "site-a", TransactionID: "tx-a"})
	store.Put(DebugBundle{Tenant: "site-b", TransactionID: "tx-b"})

	got := (DebugEvidenceOpsV2{Store: store}).List("site-a")
	if len(got) != 1 {
		t.Fatalf("List(site-a) returned %d entries, want 1", len(got))
	}
	if got[0].ID != "tx-a" || got[0].Tenant != "site-a" || got[0].CreatedAt.IsZero() {
		t.Fatalf("unexpected summary: %#v", got[0])
	}
}

func TestTLSVersionName(t *testing.T) {
	tests := []struct {
		version uint16
		want    string
	}{
		{0x0301, "TLS1.0"},
		{0x0302, "TLS1.1"},
		{0x0303, "TLS1.2"},
		{0x0304, "TLS1.3"},
		{0x9999, "0x9999"},
	}
	for _, tc := range tests {
		if got := tlsVersionName(tc.version); got != tc.want {
			t.Fatalf("tlsVersionName(%#x)=%q, want %q", tc.version, got, tc.want)
		}
	}
}
