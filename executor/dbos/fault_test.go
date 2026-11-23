// Fault-injection tests for executor boundaries: production
// construction leaves injection disabled, and a configured
// shutdown fault reaches the caller without stopping workers.
package dbosexec_test

import (
	"errors"
	"log/slog"
	"testing"
	"time"

	dbosexec "github.com/PithomLabs/miniapp/executor/dbos"
	"github.com/PithomLabs/miniapp/registry"
	"github.com/PithomLabs/miniapp/testkit/fault"
)

var errFault = errors.New("injected fault")

func emptyExecutor(t *testing.T) *dbosexec.Executor {
	t.Helper()
	pool := newPool(t)
	ex, err := dbosexec.New(dbosexec.Config{Pool: pool, Registry: registry.New(), Logger: slog.Default()})
	if err != nil {
		t.Fatal(err)
	}
	// Shutdown before pool close (LIFO): pool close registered
	// first so it runs last.
	t.Cleanup(pool.Close)
	t.Cleanup(func() {
		_ = ex.Shutdown(10 * time.Second)
	})
	return ex
}

// TestExecutorNilTestFault proves production construction leaves
// injection disabled.
func TestExecutorNilTestFault(t *testing.T) {
	if ex := emptyExecutor(t); ex.TestFault != nil {
		t.Fatal("TestFault != nil after production New")
	}
}

// TestFaultExecutorShutdown: a configured shutdown fault reaches
// the caller as the shutdown error (boundary + propagation). Pool
// lifecycle under shutdown failure is X-14 (WS-D).
func TestFaultExecutorShutdown(t *testing.T) {
	ex := emptyExecutor(t)
	inj := fault.New()
	inj.FailAt("executor-shutdown", 1, errFault)
	ex.TestFault = inj

	if err := ex.Shutdown(10 * time.Second); !errors.Is(err, errFault) {
		t.Fatalf("Shutdown err = %v, want injected fault", err)
	}
	if n := inj.Count("executor-shutdown"); n != 1 {
		t.Fatalf("fired %d times, want 1", n)
	}
	// Fault consumed: a clean shutdown converges.
	ex.TestFault = nil
	if err := ex.Shutdown(10 * time.Second); err != nil {
		t.Fatalf("retry shutdown: %v", err)
	}
}
