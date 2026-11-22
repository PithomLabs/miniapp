package spikes

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/dbos-inc/dbos-transact-golang/dbos"
	"github.com/jackc/pgx/v5/pgxpool"
)

// spikeDS and spikeDir are process-scoped spike fixtures. They are set
// once during setup and never mutated mid-workflow, so workflow replay
// observes stable values.
var (
	spikeDS  *dbos.DataSource
	spikeDSB *dbos.DataSource
	spikeDir string
)

type spikeInput struct {
	Marker string
	Fail   bool
	Block  bool
}

// spikeWorkflow writes a started marker transactionally, optionally
// blocks mid-execution (crash probe), then writes a finished marker.
func spikeWorkflow(ctx dbos.Context, in spikeInput) (string, error) {
	if _, err := dbos.RunAsTransaction(ctx, spikeDS, func(ctx context.Context, tx dbos.Tx) (string, error) {
		if _, err := tx.Exec(ctx, `INSERT INTO spike_markers(marker, stage) VALUES ($1, 'started')`, in.Marker); err != nil {
			return "", err
		}
		if in.Fail {
			return "", fmt.Errorf("spike forced failure")
		}
		return in.Marker, nil
	}); err != nil {
		return "", err
	}
	if in.Block {
		if _, err := dbos.RunAsStep(ctx, func(ctx context.Context) (string, error) {
			f, err := os.Create(filepath.Join(spikeDir, "started"))
			if err != nil {
				return "", err
			}
			f.Close()
			// Block until the orchestrator releases us. A recovered
			// step re-executes from scratch, so a fixed long sleep
			// would outlast any bounded result wait; the release
			// file makes completion deterministic on every path.
			deadline := time.Now().Add(5 * time.Minute)
			for {
				if _, err := os.Stat(filepath.Join(spikeDir, "release")); err == nil {
					return "released", nil
				}
				select {
				case <-ctx.Done():
					return "", ctx.Err()
				case <-time.After(200 * time.Millisecond):
				}
				if time.Now().After(deadline) {
					return "", fmt.Errorf("spike release file never arrived")
				}
			}
		}); err != nil {
			return "", err
		}
	}
	return dbos.RunAsTransaction(ctx, spikeDS, func(ctx context.Context, tx dbos.Tx) (string, error) {
		_, err := tx.Exec(ctx, `INSERT INTO spike_markers(marker, stage) VALUES ($1, 'finished')`, in.Marker)
		return in.Marker, err
	})
}

// spikeWorkflowB mirrors spikeWorkflow against the second family
// datasource so the isolation test uses no closures.
func spikeWorkflowB(ctx dbos.Context, in spikeInput) (string, error) {
	return dbos.RunAsTransaction(ctx, spikeDSB, func(ctx context.Context, tx dbos.Tx) (string, error) {
		_, err := tx.Exec(ctx, `INSERT INTO spike_markers(marker, stage) VALUES ($1, 'finished')`, in.Marker)
		return in.Marker, err
	})
}

func dbosTestSetup(t *testing.T, ctx context.Context) (dbos.Context, *pgxpool.Pool) {
	t.Helper()
	dsn := os.Getenv("TEST_DBOS_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DBOS_DATABASE_URL unset: run task --taskfile Taskfile.dev.yml test:integration for local Docker PostgreSQL, or export a Neon branch URL (Q4)")
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if _, err := pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS spike_markers (marker text, stage text)`); err != nil {
		t.Fatal(err)
	}
	dctx, err := dbos.NewContext(ctx, dbos.Config{AppName: "spike", DatabaseURL: dsn})
	if err != nil {
		t.Fatal(err)
	}
	ds, err := dbos.NewDataSource(dctx, pool, dbos.WithDataSourceName("app"))
	if err != nil {
		t.Fatal(err)
	}
	spikeDS = ds
	dbos.RegisterWorkflow(dctx, spikeWorkflow)
	if err := dctx.Launch(); err != nil {
		t.Fatal(err)
	}
	return dctx, pool
}

func markerCount(t *testing.T, ctx context.Context, pool *pgxpool.Pool, marker, stage string) int {
	t.Helper()
	var n int
	err := pool.QueryRow(ctx, `SELECT count(*) FROM spike_markers WHERE marker = $1 AND stage = $2`, marker, stage).Scan(&n)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func awaitResult[R any](t *testing.T, h dbos.WorkflowHandle[R], timeout time.Duration) (R, error) {
	t.Helper()
	type out struct {
		v   R
		err error
	}
	ch := make(chan out, 1)
	go func() {
		v, err := h.GetResult()
		ch <- out{v, err}
	}()
	select {
	case r := <-ch:
		return r.v, r.err
	case <-time.After(timeout):
		var zero R
		t.Fatalf("timed out waiting for workflow result")
		return zero, fmt.Errorf("timeout")
	}
}

// TestDBOSAtomicCommit proves RunAsTransaction commits the application
// write together with workflow progress on the same-database pool
// handle: both marker rows exist exactly once after success.
func TestDBOSAtomicCommit(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	dctx, pool := dbosTestSetup(t, ctx)
	marker := fmt.Sprintf("commit-%d", time.Now().UnixNano())
	h, err := dbos.RunWorkflow(dctx, spikeWorkflow, spikeInput{Marker: marker},
		dbos.WithWorkflowID("spike-commit-"+marker))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := awaitResult(t, h, 2*time.Minute); err != nil {
		t.Fatal(err)
	}
	if n := markerCount(t, ctx, pool, marker, "started"); n != 1 {
		t.Fatalf("started rows = %d, want 1", n)
	}
	if n := markerCount(t, ctx, pool, marker, "finished"); n != 1 {
		t.Fatalf("finished rows = %d, want 1", n)
	}
}

// TestDBOSAtomicRollback proves a failing transaction function leaves
// no application write behind: the app row and the checkpoint abort
// together.
func TestDBOSAtomicRollback(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	dctx, pool := dbosTestSetup(t, ctx)
	marker := fmt.Sprintf("rollback-%d", time.Now().UnixNano())
	h, err := dbos.RunWorkflow(dctx, spikeWorkflow, spikeInput{Marker: marker, Fail: true},
		dbos.WithWorkflowID("spike-rollback-"+marker))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := awaitResult(t, h, 2*time.Minute); err == nil {
		t.Fatal("expected workflow failure, got success")
	}
	if n := markerCount(t, ctx, pool, marker, "started"); n != 0 {
		t.Fatalf("started rows = %d, want 0 (rollback violated)", n)
	}
}

// TestDBOSFamilyIsolation proves two DBOS applications (distinct
// AppName) share one database without interfering: the same
// workflow shape completes independently under each family with
// correct results. This is the G0-Core behavioral half of the
// family/version plan; Fly process-group retention is proven at G6.
func TestDBOSFamilyIsolation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	dsn := os.Getenv("TEST_DBOS_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DBOS_DATABASE_URL unset: run task --taskfile Taskfile.dev.yml test:integration for local Docker PostgreSQL, or export a Neon branch URL (Q4)")
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if _, err := pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS spike_markers (marker text, stage text)`); err != nil {
		t.Fatal(err)
	}
	setup := func(app string, fn dbos.Workflow[spikeInput, string], ds **dbos.DataSource) dbos.Context {
		dctx, err := dbos.NewContext(ctx, dbos.Config{AppName: app, DatabaseURL: dsn})
		if err != nil {
			t.Fatal(err)
		}
		var err2 error
		*ds, err2 = dbos.NewDataSource(dctx, pool, dbos.WithDataSourceName("app-"+app))
		if err2 != nil {
			t.Fatal(err2)
		}
		dbos.RegisterWorkflow(dctx, fn)
		if err := dctx.Launch(); err != nil {
			t.Fatal(err)
		}
		return dctx
	}
	dctxA := setup("spike-fam-a", spikeWorkflow, &spikeDS)
	dctxB := setup("spike-fam-b", spikeWorkflowB, &spikeDSB)
	stamp := fmt.Sprintf("%d", time.Now().UnixNano())
	ha, err := dbos.RunWorkflow(dctxA, spikeWorkflow,
		spikeInput{Marker: "fam-a-" + stamp}, dbos.WithWorkflowID("spike-fam-a-"+stamp))
	if err != nil {
		t.Fatal(err)
	}
	hb, err := dbos.RunWorkflow(dctxB, spikeWorkflowB,
		spikeInput{Marker: "fam-b-" + stamp}, dbos.WithWorkflowID("spike-fam-b-"+stamp))
	if err != nil {
		t.Fatal(err)
	}
	if got, err := awaitResult(t, ha, 2*time.Minute); err != nil || got != "fam-a-"+stamp {
		t.Fatalf("family A result = %q, err = %v", got, err)
	}
	if got, err := awaitResult(t, hb, 2*time.Minute); err != nil || got != "fam-b-"+stamp {
		t.Fatalf("family B result = %q, err = %v", got, err)
	}
	if n := markerCount(t, ctx, pool, "fam-a-"+stamp, "finished"); n != 1 {
		t.Fatalf("family A finished rows = %d, want 1", n)
	}
	if n := markerCount(t, ctx, pool, "fam-b-"+stamp, "finished"); n != 1 {
		t.Fatalf("family B finished rows = %d, want 1", n)
	}
}

// TestDBOSCrashWorker is the child side of the crash probe. It runs
// only under MINIAPP_SPIKE_WORKER=1 with the parent's env.
func TestDBOSCrashWorker(t *testing.T) {
	if os.Getenv("MINIAPP_SPIKE_WORKER") == "" {
		t.Skip("crash worker entrypoint (parent-spawned only)")
	}
	ctx := context.Background()
	spikeDir = os.Getenv("MINIAPP_SPIKE_DIR")
	dctx, _ := dbosTestSetup(t, ctx)
	wfid := os.Getenv("MINIAPP_SPIKE_WFID")
	marker := os.Getenv("MINIAPP_SPIKE_MARKER")
	_, err := dbos.RunWorkflow(dctx, spikeWorkflow,
		spikeInput{Marker: marker, Block: true}, dbos.WithWorkflowID(wfid))
	if err != nil {
		t.Fatal(err)
	}
	// Stay alive until the parent kills us mid-step.
	select {}
}

// TestDBOSCrashRecovery kills a worker mid-step with SIGKILL, relaunches
// a fresh executor, and proves the workflow resumes from its last
// completed step: the completed started-marker is never re-executed and
// the finished marker appears exactly once.
func TestDBOSCrashRecovery(t *testing.T) {
	if os.Getenv("TEST_DBOS_DATABASE_URL") == "" {
		t.Skip("TEST_DBOS_DATABASE_URL unset: run task --taskfile Taskfile.dev.yml test:integration for local Docker PostgreSQL, or export a Neon branch URL (Q4)")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	dir := t.TempDir()
	marker := fmt.Sprintf("crash-%d", time.Now().UnixNano())
	wfid := "spike-crash-" + marker
	// The recovered step re-executes in THIS process after relaunch,
	// so the orchestrator directory must be set here too — not only
	// in the child. (Production workflows must never depend on
	// process-local state like this; they carry context in inputs.)
	spikeDir = dir

	cmd := exec.Command(os.Args[0], "-test.run", "TestDBOSCrashWorker", "-test.v")
	cmd.Env = append(os.Environ(),
		"MINIAPP_SPIKE_WORKER=1",
		"TEST_DBOS_DATABASE_URL="+os.Getenv("TEST_DBOS_DATABASE_URL"),
		"MINIAPP_SPIKE_DIR="+dir,
		"MINIAPP_SPIKE_WFID="+wfid,
		"MINIAPP_SPIKE_MARKER="+marker,
	)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Minute)
	for {
		if _, err := os.Stat(filepath.Join(dir, "started")); err == nil {
			break
		}
		if time.Now().After(deadline) {
			_ = cmd.Process.Kill()
			t.Fatal("worker never reached the blocking step")
		}
		time.Sleep(200 * time.Millisecond)
	}
	if err := cmd.Process.Signal(syscall.SIGKILL); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()

	dctx, pool := dbosTestSetup(t, ctx)
	h, err := dbos.RunWorkflow(dctx, spikeWorkflow,
		spikeInput{Marker: marker}, dbos.WithWorkflowID(wfid))
	if err != nil {
		t.Fatalf("re-submit after kill failed (attach semantics?): %v", err)
	}
	// Release the recovered step only after re-submission, so the
	// pre-kill execution could never have observed it.
	if err := os.WriteFile(filepath.Join(dir, "release"), []byte("go"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := awaitResult(t, h, 3*time.Minute); err != nil {
		t.Fatal(err)
	}
	if n := markerCount(t, ctx, pool, marker, "started"); n != 1 {
		t.Fatalf("started rows = %d, want exactly 1 (completed step re-executed)", n)
	}
	if n := markerCount(t, ctx, pool, marker, "finished"); n != 1 {
		t.Fatalf("finished rows = %d, want exactly 1", n)
	}
}
