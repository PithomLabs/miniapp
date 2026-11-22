package spikes

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dbos-inc/dbos-transact-golang/dbos"
	"github.com/jackc/pgx/v5/pgxpool"
)

// spikeDSShared is the datasource bound to the shared system pool. Set
// once during setup, never mutated mid-workflow.
var spikeDSShared *dbos.DataSource

// sharedWorkflow mirrors spikeWorkflow against the shared datasource:
// a started marker, then a finished marker, with a forced-failure mode
// for the rollback case.
func sharedWorkflow(ctx dbos.Context, in spikeInput) (string, error) {
	if _, err := dbos.RunAsTransaction(ctx, spikeDSShared, func(ctx context.Context, tx dbos.Tx) (string, error) {
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
	return dbos.RunAsTransaction(ctx, spikeDSShared, func(ctx context.Context, tx dbos.Tx) (string, error) {
		_, err := tx.Exec(ctx, `INSERT INTO spike_markers(marker, stage) VALUES ($1, 'finished')`, in.Marker)
		return in.Marker, err
	})
}

// logCapture is a concurrency-safe slog sink for asserting which
// DBOS durability path executed. The SDK logs engine-sharing
// detection at Debug level; capturing it proves the test ran the
// configuration it claims instead of silently falling back.
type logCapture struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (c *logCapture) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.buf.Write(p)
}

func (c *logCapture) String() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.buf.String()
}

func captureLogger(c *logCapture) *slog.Logger {
	return slog.New(slog.NewTextHandler(c, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

const sharedDetection = "shares the system database"

// TestDBOSSharedHandle proves the shared-pool-handle configuration:
// one engine serves as both the DBOS system pool and the application
// datasource, so application writes and step checkpoints commit in a
// single transaction. It asserts commit and rollback through durable
// database state plus DBOS step metadata, and the captured log must
// show the SDK selected single-transaction durability.
func TestDBOSSharedHandle(t *testing.T) {
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
	// Deliberately no pool.Close: DBOS background workers share this
	// pool after Launch, so closing it here would deadlock waiting for
	// connections the running executor holds. Shutdown ordering
	// (executor stop before pool close) belongs to the Phase 1
	// executor; the existing spikes likewise never shut DBOS down.
	if _, err := pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS spike_markers (marker text, stage text)`); err != nil {
		t.Fatal(err)
	}
	logs := &logCapture{}
	dctx, err := dbos.NewContext(ctx, dbos.Config{
		AppName:      "spike-shared",
		SystemDBPool: pool,
		Logger:       captureLogger(logs),
	})
	if err != nil {
		t.Fatal(err)
	}
	ds, err := dbos.NewDataSource(dctx, pool, dbos.WithDataSourceName("app-shared"))
	if err != nil {
		t.Fatal(err)
	}
	spikeDSShared = ds
	dbos.RegisterWorkflow(dctx, sharedWorkflow)
	if err := dctx.Launch(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(logs.String(), sharedDetection) {
		t.Fatalf("shared-handle configuration was not selected (detection log absent)")
	}

	// Commit case: both markers durable, steps completed without error.
	marker := fmt.Sprintf("shared-%d", time.Now().UnixNano())
	wfid := "spike-shared-" + marker
	h, err := dbos.RunWorkflow(dctx, sharedWorkflow, spikeInput{Marker: marker}, dbos.WithWorkflowID(wfid))
	if err != nil {
		t.Fatal(err)
	}
	if got, err := awaitResult(t, h, 2*time.Minute); err != nil || got != marker {
		t.Fatalf("shared workflow result = %q, err = %v", got, err)
	}
	if n := markerCount(t, ctx, pool, marker, "started"); n != 1 {
		t.Fatalf("started rows = %d, want 1", n)
	}
	if n := markerCount(t, ctx, pool, marker, "finished"); n != 1 {
		t.Fatalf("finished rows = %d, want 1", n)
	}
	steps, err := dbos.GetWorkflowSteps(dctx, wfid)
	if err != nil {
		t.Fatal(err)
	}
	if len(steps) < 2 {
		t.Fatalf("expected at least 2 recorded steps, got %d", len(steps))
	}
	for _, s := range steps {
		if s.Error != nil {
			t.Fatalf("step %q recorded error: %v", s.StepName, s.Error)
		}
		if s.CompletedAt.IsZero() {
			t.Fatalf("step %q has no completion timestamp", s.StepName)
		}
	}

	// Rollback case: the failing transaction leaves no application row.
	rollbackMarker := fmt.Sprintf("shared-rb-%d", time.Now().UnixNano())
	h2, err := dbos.RunWorkflow(dctx, sharedWorkflow,
		spikeInput{Marker: rollbackMarker, Fail: true}, dbos.WithWorkflowID("spike-shared-rb-"+rollbackMarker))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := awaitResult(t, h2, 2*time.Minute); err == nil {
		t.Fatal("expected workflow failure, got success")
	}
	if n := markerCount(t, ctx, pool, rollbackMarker, "started"); n != 0 {
		t.Fatalf("started rows = %d, want 0 (rollback violated)", n)
	}
}

// TestDBOSSharedDetectionNegative proves the shared-handle assertion
// distinguishes configurations: a separate-pool datasource must NOT
// emit the single-transaction detection message. If it ever does, the
// positive test above would be meaningless.
func TestDBOSSharedDetectionNegative(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	dsn := os.Getenv("TEST_DBOS_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DBOS_DATABASE_URL unset: run task --taskfile Taskfile.dev.yml test:integration for local Docker PostgreSQL, or export a Neon branch URL (Q4)")
	}
	logs := &logCapture{}
	dctx, err := dbos.NewContext(ctx, dbos.Config{
		AppName:     "spike-shared-neg",
		DatabaseURL: dsn,
		Logger:      captureLogger(logs),
	})
	if err != nil {
		t.Fatal(err)
	}
	appPool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer appPool.Close()
	if _, err := dbos.NewDataSource(dctx, appPool, dbos.WithDataSourceName("app-separate")); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(logs.String(), sharedDetection) {
		t.Fatal("separate-pool datasource falsely reported shared-handle durability")
	}
}
