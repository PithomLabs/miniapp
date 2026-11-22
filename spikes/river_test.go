package spikes

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
	"github.com/riverqueue/river/rivertype"
)

// spikeArgs is a minimal job payload with a caller-chosen marker.
type spikeArgs struct {
	Marker string `json:"marker"`
}

// Kind implements river.JobArgs.
func (spikeArgs) Kind() string { return "spike" }

// spikeWorker is a no-op worker. River requires every inserted kind to
// be registered in the client's Workers bundle, even for insert-only
// tests that never start the client.
type spikeWorker struct{}

func (spikeWorker) Middleware(job *rivertype.JobRow) []rivertype.WorkerMiddleware {
	return nil
}

func (spikeWorker) NextRetry(job *river.Job[spikeArgs]) time.Time { return time.Time{} }

func (spikeWorker) Timeout(job *river.Job[spikeArgs]) time.Duration { return 0 }

func (spikeWorker) Work(ctx context.Context, job *river.Job[spikeArgs]) error { return nil }

func riverTestSetup(t *testing.T) (*river.Client[pgx.Tx], *pgxpool.Pool) {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL unset: run task --taskfile Taskfile.dev.yml test:integration for local Docker PostgreSQL, or export a Neon branch URL (Q4)")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	// Cancel first, then close: Close waits for checked-out resources
	// that may be bound to this context.
	t.Cleanup(func() { cancel(); pool.Close() })
	if _, err := pool.Exec(ctx, "CREATE SCHEMA IF NOT EXISTS river"); err != nil {
		t.Fatal(err)
	}
	driver := riverpgxv5.New(pool)
	migrator, err := rivermigrate.New(driver, &rivermigrate.Config{Schema: "river"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := migrator.Migrate(ctx, rivermigrate.DirectionUp, nil); err != nil {
		t.Fatal(err)
	}
	workers := river.NewWorkers()
	river.AddWorker(workers, spikeWorker{})
	client, err := river.NewClient(driver, &river.Config{
		Queues:  map[string]river.QueueConfig{river.QueueDefault: {MaxWorkers: 1}},
		Workers: workers,
		Schema:  "river",
	})
	if err != nil {
		t.Fatal(err)
	}
	return client, pool
}

// TestRiverTransactionalEnqueue proves InsertTx participates in the
// caller's transaction: rollback removes the job, commit keeps it.
// The rolled-back insert must leave no trace that later trips unique
// deduplication.
func TestRiverTransactionalEnqueue(t *testing.T) {
	client, pool := riverTestSetup(t)
	ctx := context.Background()
	// Unique per run: the committed probe row outlives the test, and a
	// fixed marker would trip ByPeriod dedup on re-runs.
	marker := fmt.Sprintf("tx-probe-%d", time.Now().UnixNano())

	unique := &river.InsertOpts{UniqueOpts: river.UniqueOpts{ByArgs: true, ByPeriod: time.Minute}}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.InsertTx(ctx, tx, spikeArgs{Marker: marker}, unique); err != nil {
		t.Fatal(err)
	}
	if err := tx.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	res, err := client.Insert(ctx, spikeArgs{Marker: marker}, unique)
	if err != nil {
		t.Fatal(err)
	}
	if res.UniqueSkippedAsDuplicate {
		t.Fatal("rolled-back insert left a unique trace: atomicity violated")
	}

	// A committed insert must dedplicate a repeat while the period holds.
	res2, err := client.Insert(ctx, spikeArgs{Marker: marker}, unique)
	if err != nil {
		t.Fatal(err)
	}
	if !res2.UniqueSkippedAsDuplicate {
		t.Fatal("expected duplicate suppression for committed unique job")
	}
	if res2.Job.ID != res.Job.ID {
		t.Fatalf("duplicate returned job %d, want %d", res2.Job.ID, res.Job.ID)
	}
}
