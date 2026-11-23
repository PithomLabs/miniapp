package testkit

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestWaitFailsFastOnWorkerExit proves a worker that dies before
// producing its marker fails the wait immediately — with the exit
// status and captured output attached — instead of burning the full
// timeout. This is the CI failure made diagnostic: the old code
// waited 120s and reported only the missing path.
func TestWaitFailsFastOnWorkerExit(t *testing.T) {
	w, err := startWorker(exec.Command("sh", "-c", "echo simulated-startup-boom >&2; exit 3"))
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	err = waitForFile(filepath.Join(t.TempDir(), "never"), w, 30*time.Second)
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("wait succeeded despite worker exit without marker")
	}
	if elapsed > 15*time.Second {
		t.Fatalf("wait took %s, want fail-fast well under the timeout", elapsed)
	}
	msg := err.Error()
	for _, want := range []string{"exited", "exit status 3", "simulated-startup-boom"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("error %q missing %q", msg, want)
		}
	}
}

// TestWaitFailsFastOnCleanExit covers the exact CI shape: a worker
// that exits 0 early (e.g. a skipped test binary) must fail the
// wait with its output visible — the skip reason included — rather
// than timing out opaquely.
func TestWaitFailsFastOnCleanExit(t *testing.T) {
	w, err := startWorker(exec.Command("sh", "-c", "echo '--- SKIP: TEST_X unset'; exit 0"))
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	err = waitForFile(filepath.Join(t.TempDir(), "never"), w, 30*time.Second)
	if err == nil {
		t.Fatal("wait succeeded despite clean worker exit without marker")
	}
	if time.Since(start) > 15*time.Second {
		t.Fatal("clean early exit did not fail fast")
	}
	if !strings.Contains(err.Error(), "SKIP: TEST_X unset") {
		t.Fatalf("error %q hides the worker skip reason", err)
	}
}

// TestWaitSucceedsOnMarker proves the marker path is unchanged: a
// present marker returns success even while the worker runs on.
func TestWaitSucceedsOnMarker(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "submitted")
	if err := os.WriteFile(marker, []byte("go"), 0o600); err != nil {
		t.Fatal(err)
	}
	w, err := startWorker(exec.Command("sleep", "30"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = w.cmd.Process.Signal(os.Kill)
		<-w.done
	}()
	if err := waitForFile(marker, w, 30*time.Second); err != nil {
		t.Fatalf("wait on present marker: %v", err)
	}
}

// TestWaitTimeoutReaps proves the timeout path kills and reaps the
// worker: no lingering process, and the error carries the timeout.
func TestWaitTimeoutReaps(t *testing.T) {
	w, err := startWorker(exec.Command("sleep", "60"))
	if err != nil {
		t.Fatal(err)
	}
	err = waitForFile(filepath.Join(t.TempDir(), "never"), w, 2*time.Second)
	if err == nil || !strings.Contains(err.Error(), "never produced") {
		t.Fatalf("timeout err = %v, want missing-marker failure", err)
	}
	select {
	case <-w.done:
	default:
		t.Fatal("worker not reaped after timeout kill")
	}
	if err := w.cmd.Process.Signal(syscall.Signal(0)); err == nil {
		t.Fatal("worker process still alive after timeout kill")
	}
}

// TestKill9Reaps proves Kill9 terminates a live worker and reaps it.
func TestKill9Reaps(t *testing.T) {
	w, err := startWorker(exec.Command("sleep", "60"))
	if err != nil {
		t.Fatal(err)
	}
	Kill9(t, w)
	select {
	case <-w.done:
	default:
		t.Fatal("worker not reaped after Kill9")
	}
}

// TestKill9OnExitedWorker proves killing an already-exited worker
// reaps cleanly instead of failing: callers assert liveness via
// rendezvous markers, not via kill errors.
func TestKill9OnExitedWorker(t *testing.T) {
	w, err := startWorker(exec.Command("sh", "-c", "exit 0"))
	if err != nil {
		t.Fatal(err)
	}
	<-w.done
	Kill9(t, w) // must not fail the test
}

// TestTailWriterBounded proves captured output is capped to the tail.
func TestTailWriterBounded(t *testing.T) {
	w := &tailWriter{n: 8}
	if _, err := w.Write([]byte("0123456789abcdef")); err != nil {
		t.Fatal(err)
	}
	if got := w.String(); got != "89abcdef" {
		t.Fatalf("tail = %q, want %q", got, "89abcdef")
	}
}
