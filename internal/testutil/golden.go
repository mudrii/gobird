package testutil

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
)

// UpdateGolden is set via -update flag to regenerate golden files.
var UpdateGolden = flag.Bool("update", false, "regenerate golden test files")

// AssertGolden checks that got matches the content of the golden file at path.
// When -update is passed, it writes got to the golden file instead.
func AssertGolden(t *testing.T, path string, got []byte) {
	t.Helper()
	if *UpdateGolden {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir golden dir: %v", err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatalf("write golden file: %v", err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("golden file missing: %s (run with -update to create)", path)
	}
	if string(got) != string(want) {
		t.Errorf("output mismatch for %s\n--- want ---\n%s\n--- got ---\n%s", path, want, got)
	}
}
