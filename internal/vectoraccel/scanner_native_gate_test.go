//go:build vectorscan && cgo

package vectoraccel

import (
	"strings"
	"testing"
)

// TestNativeVectorScanCompileAndScanGate is a release-host correctness gate.
// It must only be counted as real VectorScan evidence when the vectorscan tag
// is linked against a verified real libvectorscan/libhs installation. The test
// exercises hs_compile_multi through nativeFactory.Compile and hs_scan through
// compiledScanner.Scan; ABI-only shims are not acceptable release provenance.
func TestNativeVectorScanCompileAndScanGate(t *testing.T) {
	factory := newNativeFactory()
	if !factory.Available() {
		t.Fatal("native VectorScan factory is unavailable under vectorscan+cgo build")
	}
	if version := strings.TrimSpace(factory.Version()); version == "" || version == "unavailable" {
		t.Fatalf("native VectorScan version is not usable: %q", version)
	}

	rules := []RuleSpec{
		{ID: 187101, Pattern: `vector-native-sentinel`},
		{ID: 187102, Pattern: `second-vector-sentinel`},
	}
	db, err := factory.Compile(rules)
	if err != nil {
		t.Fatalf("native multi-pattern compile failed: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Errorf("native database close failed: %v", err)
		}
	}()

	assertIDs := func(name string, input string, want ...int) {
		t.Helper()
		got, err := db.Scan([]byte(input))
		if err != nil {
			t.Fatalf("%s native scan failed: %v", name, err)
		}
		seen := make(map[int]struct{}, len(got))
		for _, id := range got {
			seen[id] = struct{}{}
		}
		if len(seen) != len(want) {
			t.Fatalf("%s native scan IDs=%v want=%v", name, got, want)
		}
		for _, id := range want {
			if _, ok := seen[id]; !ok {
				t.Fatalf("%s native scan IDs=%v missing=%d", name, got, id)
			}
		}
	}

	assertIDs("first-pattern", "prefix vector-native-sentinel suffix", 187101)
	assertIDs("second-pattern", "prefix second-vector-sentinel suffix", 187102)
	assertIDs("clean-input", "no native sentinel should match here")
}
