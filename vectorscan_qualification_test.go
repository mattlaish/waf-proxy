package main

import "testing"

func TestVectorScanQualificationGateZeroFalseNegative(t *testing.T) {
	r := VectorScanQualificationGate([]int{1,2}, []int{1,2})
	if r.Status != "VALIDATED" {
		t.Fatalf("expected validated, got %s", r.Status)
	}
}

func TestVectorScanQualificationGateFailsSafe(t *testing.T) {
	r := VectorScanQualificationGate([]int{1,2}, []int{1})
	if r.Status != "FAILSAFE_CORAZA_ONLY" {
		t.Fatalf("expected failsafe, got %s", r.Status)
	}
}
