package testkit

import (
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

// WorkerMode reports whether the current test process is a spawned
// crash-probe worker. Worker entrypoints gate on this and skip when
// unset, so parent-spawned tests never run standalone.
func WorkerMode(envKey string) bool {
	return os.Getenv(envKey) != ""
}

// SpawnWorker re-executes the current test binary with -test.run
// selecting the named worker entrypoint. extraEnv is added to the
// process environment (database URLs and rendezvous paths travel
// this way). The caller owns rendezvous, kill, and wait.
func SpawnWorker(t *testing.T, testName string, extraEnv map[string]string) *exec.Cmd {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run", testName, "-test.v")
	env := os.Environ()
	for k, v := range extraEnv {
		env = append(env, k+"="+v)
	}
	cmd.Env = env
	if err := cmd.Start(); err != nil {
		t.Fatalf("spawn %s: %v", testName, err)
	}
	return cmd
}

// WaitForFile polls for path until timeout, killing cmd and failing
// the test on expiry.
func WaitForFile(t *testing.T, cmd *exec.Cmd, path string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		if _, err := os.Stat(path); err == nil {
			return
		}
		if time.Now().After(deadline) {
			_ = cmd.Process.Kill()
			t.Fatalf("worker never produced %s", path)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// Kill9 SIGKILLs a spawned worker and reaps it. SIGKILL (not SIGTERM)
// models abrupt worker loss: no graceful shutdown, no cleanup.
func Kill9(t *testing.T, cmd *exec.Cmd) {
	t.Helper()
	if err := cmd.Process.Signal(syscall.SIGKILL); err != nil {
		t.Fatalf("kill worker: %v", err)
	}
	_ = cmd.Wait()
}
