package infra

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestSpikeSkipsDirectToLocalFirst requires every missing-DSN skip
// string to direct developers to test:integration (local Docker
// PostgreSQL) before mentioning Neon, and forbids any wording that
// presents Neon access as a prerequisite for running spikes.
func TestSpikeSkipsDirectToLocalFirst(t *testing.T) {
	files, err := filepath.Glob(filepath.Join(repoRoot(t), "spikes", "*_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("no spike test files found")
	}
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		base := filepath.Base(f)
		for _, line := range strings.Split(string(data), "\n") {
			trim := strings.TrimSpace(line)
			if strings.Contains(trim, "needs a Neon branch") {
				t.Errorf("%s presents Neon as mandatory: %q", base, trim)
			}
			if strings.Contains(trim, "t.Skip(") && strings.Contains(trim, "DATABASE_URL unset") {
				if !strings.Contains(trim, "test:integration") {
					t.Errorf("%s skip omits the local path: %q", base, trim)
				}
				if !strings.Contains(trim, "local Docker") {
					t.Errorf("%s skip omits local Docker wording: %q", base, trim)
				}
			}
		}
	}
	doc, err := os.ReadFile(filepath.Join(repoRoot(t), "spikes", "doc.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(doc), "test:integration") {
		t.Error("spikes package doc omits the local test:integration path")
	}
	if strings.Contains(string(doc), "needs a Neon branch") {
		t.Error("spikes package doc presents Neon as mandatory")
	}
}

// spikesDir locates the spikes package directory from this test file's
// own path, independent of the test runner's working directory.
func spikesDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate test file")
	}
	return filepath.Join(filepath.Dir(thisFile), "..", "spikes")
}

// TestSpikesDirHygiene enforces the spikes directory allowlist: Go
// sources plus explicitly permitted companions. Crash-probe runtime
// files must live in OS temp dirs, never beside the tests.
func TestSpikesDirHygiene(t *testing.T) {
	entries, err := os.ReadDir(spikesDir(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		name := e.Name()
		if strings.HasSuffix(name, ".go") {
			continue
		}
		t.Errorf("spikes/ contains non-allowlisted file %q (remove it or extend the allowlist deliberately)", name)
	}
}
