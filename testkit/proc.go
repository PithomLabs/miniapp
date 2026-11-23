package testkit

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"
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

// RequireDB skips the test unless every named environment variable
// is set. DB-gated tests that spawn workers must call it before
// spawning: a worker that skips for lack of configuration would
// otherwise surface as a timeout instead of a skip. This matches
// the suite-wide convention (NewDB skips when TEST_DATABASE_URL is
// unset); CI provides a real database so gated tests genuinely
// execute there.
func RequireDB(t *testing.T, vars ...string) {
	t.Helper()
	for _, v := range vars {
		if os.Getenv(v) == "" {
			t.Skipf("%s unset: run task --taskfile Taskfile.dev.yml test:integration for local Docker PostgreSQL, or export a Neon branch URL (Q4)", v)
		}
	}
}

// maxWorkerLog bounds captured worker output per stream. Workers
// under test run DBOS with debug logging; only the tail matters
// for failure diagnosis.
const maxWorkerLog = 64 << 10

// tailWriter keeps the last n bytes written. It is safe for
// concurrent use by the child process pipes.
type tailWriter struct {
	mu  sync.Mutex
	buf []byte
	n   int
}

func (w *tailWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.n <= 0 {
		return len(p), nil
	}
	w.buf = append(w.buf, p...)
	if len(w.buf) > w.n {
		w.buf = append([]byte(nil), w.buf[len(w.buf)-w.n:]...)
	}
	return len(p), nil
}

func (w *tailWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return string(w.buf)
}

// Worker is a spawned crash-probe worker with captured output and
// exactly-once reaping. The zero value is not usable; construct
// with SpawnWorker.
type Worker struct {
	cmd  *exec.Cmd
	out  *tailWriter
	err  *tailWriter
	done chan struct{}

	mu      sync.Mutex
	waitErr error
	waited  bool
}

// SpawnWorker re-executes the current test binary with -test.run
// selecting the named worker entrypoint. extraEnv is added to the
// process environment (database URLs and rendezvous paths travel
// this way). Worker stdout and stderr are captured (bounded tail)
// for failure diagnosis, and a cleanup is registered so a worker
// still running at test end is killed and reaped on every exit
// path — a Fatalf between spawn and kill must never leak a worker.
func SpawnWorker(t *testing.T, testName string, extraEnv map[string]string) *Worker {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run", testName, "-test.v")
	env := os.Environ()
	for k, v := range extraEnv {
		env = append(env, k+"="+v)
	}
	cmd.Env = env
	w, err := startWorker(cmd)
	if err != nil {
		t.Fatalf("spawn %s: %v", testName, err)
	}
	t.Cleanup(func() {
		_ = w.cmd.Process.Signal(syscall.SIGKILL)
		<-w.done
	})
	return w
}

// startWorker launches cmd with bounded output capture and exactly-once
// reaping. The done channel closes when the process has been reaped.
func startWorker(cmd *exec.Cmd) (*Worker, error) {
	w := &Worker{
		cmd:  cmd,
		out:  &tailWriter{n: maxWorkerLog},
		err:  &tailWriter{n: maxWorkerLog},
		done: make(chan struct{}),
	}
	cmd.Stdout = w.out
	cmd.Stderr = w.err
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	go func() {
		err := cmd.Wait()
		w.mu.Lock()
		w.waitErr = err
		w.waited = true
		w.mu.Unlock()
		close(w.done)
	}()
	return w, nil
}

// exited reports whether the worker process has been reaped, and if
// so its wait error (nil for exit status 0).
func (w *Worker) exited() (done bool, waitErr error) {
	select {
	case <-w.done:
		w.mu.Lock()
		defer w.mu.Unlock()
		return true, w.waitErr
	default:
		return false, nil
	}
}

// diagnosis renders worker state for failure messages: exit status
// when reaped, plus the captured output tails that distinguish a
// skip, a crash, and a startup hang.
func (w *Worker) diagnosis() string {
	var b bytes.Buffer
	if done, err := w.exited(); done {
		fmt.Fprintf(&b, "\nworker exited: %v", err)
	} else {
		b.WriteString("\nworker still running")
	}
	if out := w.out.String(); out != "" {
		fmt.Fprintf(&b, "\n--- worker stdout (tail) ---\n%s", out)
	}
	if errOut := w.err.String(); errOut != "" {
		fmt.Fprintf(&b, "\n--- worker stderr (tail) ---\n%s", errOut)
	}
	return b.String()
}

// WaitForFile polls for path until timeout, failing the test on
// expiry. If the worker exits before producing the marker, it fails
// fast with the worker's exit status and captured output instead of
// burning the full timeout: an early exit is never readiness, and
// the marker file remains the only readiness signal.
func WaitForFile(t *testing.T, w *Worker, path string, timeout time.Duration) {
	t.Helper()
	if err := waitForFile(path, w, timeout); err != nil {
		t.Fatal(err)
	}
}

func waitForFile(path string, w *Worker, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	tick := time.NewTicker(200 * time.Millisecond)
	defer tick.Stop()
	for {
		if _, err := os.Stat(path); err == nil {
			return nil
		}
		if done, waitErr := w.exited(); done {
			return fmt.Errorf("worker exited before producing %s (wait: %v)%s", path, waitErr, w.diagnosis())
		}
		if time.Now().After(deadline) {
			_ = w.cmd.Process.Signal(syscall.SIGKILL)
			<-w.done
			return fmt.Errorf("worker never produced %s within %s%s", path, timeout, w.diagnosis())
		}
		<-tick.C
	}
}

// Kill9 SIGKILLs a spawned worker and reaps it. SIGKILL (not SIGTERM)
// models abrupt worker loss: no graceful shutdown, no cleanup. A
// worker that already exited is reaped without error; callers that
// need liveness proof assert on rendezvous markers, not here.
func Kill9(t *testing.T, w *Worker) {
	t.Helper()
	if err := w.cmd.Process.Signal(syscall.SIGKILL); err != nil && !errors.Is(err, os.ErrProcessDone) {
		t.Fatalf("kill worker: %v%s", err, w.diagnosis())
	}
	<-w.done
}
