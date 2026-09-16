package main

import (
	"testing"
	"time"
)

func TestValidateCRLFetchURLBlocksLocalTargets(t *testing.T) {
	for _, u := range []string{
		"http://127.0.0.1/crl",
		"https://localhost/crl",
		"http://169.254.169.254/latest",
	} {
		if err := validateCRLFetchURL(u, crlFetchPolicy{AllowHTTP: true}); err == nil {
			t.Fatalf("expected blocked URL: %s", u)
		}
	}
}

func TestLastKnownGoodCRLRetention(t *testing.T) {
	old := &lastKnownGoodCRL{Data: []byte("old")}
	got := retainLastKnownGood(old, []byte("bad"), timeNow(), false)
	if string(got.Data) != "old" {
		t.Fatal("previous CRL was not retained")
	}
}

func timeNow() time.Time { return time.Now() }
