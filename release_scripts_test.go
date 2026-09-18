package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
)

func TestReleaseScriptsAreLFAndBashSyntaxClean(t *testing.T) {
	paths, err := filepath.Glob("*.sh")
	if err != nil {
		t.Fatal(err)
	}
	paths = append(paths,
		filepath.Join("benchmark", "build.sh"),
		filepath.Join("qualification", "hsm", "run-softhsm-qualification.sh"),
		filepath.Join("qualification", "hsm", "run-vendor-hsm-qualification.sh"),
		filepath.Join("packaging", "deb", "build-deb.sh"),
		filepath.Join("packaging", "deb", "build-release-deb.sh"),
		filepath.Join("packaging", "deb", "verify-deb.sh"),
		filepath.Join("packaging", "deb", "tests", "test-deb-packaging.sh"),
		filepath.Join("packaging", "rpm", "build-rpm.sh"),
		filepath.Join("packaging", "rpm", "build-release-rpm.sh"),
		filepath.Join("packaging", "rpm", "verify-rpm.sh"),
		filepath.Join("packaging", "rpm", "tests", "test-rpm-source.sh"),
		filepath.Join("packaging", "rpm", "tests", "test-rpm-packaging.sh"),
		filepath.Join("packaging", "qualification", "build-lifecycle-fixtures.sh"),
		filepath.Join("packaging", "qualification", "run-package-lifecycle-qualification.sh"),
		filepath.Join("packaging", "qualification", "tests", "test-qualification-source.sh"),
		filepath.Join("packaging", "cleanhost", "run-clean-host-qualification.sh"),
		filepath.Join("packaging", "cleanhost", "tests", "test-clean-host-source.sh"),
		filepath.Join("tools", "tests", "test-openai-integration-source.sh"),
	)
	sort.Strings(paths)
	if len(paths) == 0 {
		t.Fatal("no release shell scripts found")
	}

	bash, bashErr := exec.LookPath("bash")
	for _, path := range paths {
		path := path
		t.Run(filepath.ToSlash(path), func(t *testing.T) {
			body, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if bytes.ContainsRune(body, '\r') {
				t.Fatal("carriage return found; release scripts must be LF-normalized")
			}
			info, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			if info.Mode().Perm()&0o111 == 0 {
				t.Fatal("release script is not executable")
			}
			if runtime.GOOS == "windows" || bashErr != nil {
				t.Skip("bash not available for syntax validation")
			}
			cmd := exec.Command(bash, "-n", path)
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("bash -n failed: %v\n%s", err, out)
			}
		})
	}
}
