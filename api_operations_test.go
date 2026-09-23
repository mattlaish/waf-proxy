package main

import "testing"

func TestAPIOperationNormalizationHardening(t *testing.T) {
	cases := map[string]string{
		"/api/users/12345": "/api/users/{id}",
		"/api/payment/550e8400-e29b-41d4-a716-446655440000": "/api/payment/{id}",
		"/api/report/20260921":                              "/api/report/{date}",
		"/api/item/01ARZ3NDEKTSV4RRFFQ69G5FAV":              "/api/item/{id}",
	}
	for in, want := range cases {
		if got := normalizeAPIOperationPath(in); got != want {
			t.Fatalf("%s: got %s want %s", in, got, want)
		}
	}
}

func TestAPIOperationFingerprintStable(t *testing.T) {
	if operationFingerprint("get", "/api/a") != operationFingerprint("GET", "/api/a") {
		t.Fatal("fingerprint must normalize method")
	}
}
