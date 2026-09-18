package vectorscan

import "testing"

func TestZeroFalseNegative(t *testing.T) {
	results := []DifferentialResult{
		Compare(1001, true, true),
		Compare(1002, false, true),
	}
	if !ZeroFalseNegative(results) {
		t.Fatal("expected no false negatives")
	}
}

func TestDetectFalseNegative(t *testing.T) {
	results := []DifferentialResult{Compare(1001, true, false)}
	if ZeroFalseNegative(results) {
		t.Fatal("expected false negative")
	}
}

func TestLifecycle(t *testing.T) {
	if !ValidTransition(Learning, Validated) {
		t.Fatal("expected lifecycle transition")
	}
}
