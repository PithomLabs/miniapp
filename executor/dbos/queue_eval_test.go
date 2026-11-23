package dbosexec_test

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dbos-inc/dbos-transact-golang/dbos"
	"github.com/jackc/pgx/v5/pgxpool"
)

// This file is the WS-A queue evaluation: source-verified answers to
// the six prompt13 §7 queue questions, with the two load-bearing
// behaviors (sleep-slot occupancy, cross-partition concurrency)
// pinned by a live probe on the pinned SDK. It wires no queue into
// the executor; queue use stays an explicitly-evaluated option until
// a conformance need requires it.

type evalInput struct {
	Marker    string
	SleepMs   int
	Partition string
}

type evalTimes struct {
	start  atomic.Int64
	finish atomic.Int64
}

var evalEvents sync.Map // marker -> *evalTimes

// evalWorkflow records wall-clock start/finish around a durable
// sleep. Recording runs inside steps (timestamps are step-local
// observations, never workflow control flow); the in-memory rendezvous
// is crash-free by design — this probe never kills its worker, so no
// replay re-executes the recording. Kill tests use durable markers.
func evalWorkflow(ctx dbos.Context, in evalInput) (string, error) {
	if _, err := dbos.RunAsStep(ctx, func(ctx context.Context) (string, error) {
		tm := &evalTimes{}
		tm.start.Store(time.Now().UnixNano())
		evalEvents.Store(in.Marker, tm)
		return in.Marker, nil
	}); err != nil {
		return "", err
	}
	if in.SleepMs > 0 {
		if _, err := dbos.Sleep(ctx, time.Duration(in.SleepMs)*time.Millisecond); err != nil {
			return "", err
		}
	}
	_, err := dbos.RunAsStep(ctx, func(ctx context.Context) (string, error) {
		if v, ok := evalEvents.Load(in.Marker); ok {
			v.(*evalTimes).finish.Store(time.Now().UnixNano())
		}
		return in.Marker, nil
	})
	return in.Marker, err
}

func evalTimesFor(t *testing.T, marker string) *evalTimes {
	t.Helper()
	v, ok := evalEvents.Load(marker)
	if !ok {
		t.Fatalf("no events for %s", marker)
	}
	return v.(*evalTimes)
}

// TestPartitionedQueueEvaluation proves, against v1.9.0 behavior:
// (a) a sleeping workflow holds its partition slot — the second
// same-partition workflow starts only after the first finishes;
// (b) a different partition proceeds concurrently while the first
// sleeps. Dequeue order within a partition is FIFO (priority,
// created_at) per the SDK dequeue query.
func TestPartitionedQueueEvaluation(t *testing.T) {
	dsn := os.Getenv("TEST_DBOS_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DBOS_DATABASE_URL unset")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	app := fmt.Sprintf("probe-qeval-%d", time.Now().UnixNano())
	dctx, err := dbos.NewContext(ctx, dbos.Config{AppName: app, SystemDBPool: pool})
	if err != nil {
		t.Fatal(err)
	}
	// LIFO cleanup: Shutdown (registered second) runs before pool
	// close (registered first). Reversing this deadlocks.
	t.Cleanup(func() {
		if err := dbos.Shutdown(dctx, 30*time.Second); err != nil {
			t.Errorf("shutdown: %v", err)
		}
	})
	stamp := fmt.Sprintf("%d", time.Now().UnixNano())
	// Queue names are unique across applications sharing one system
	// database, so every probe run mints its own.
	q, err := dbos.RegisterQueue(dctx, "eval-"+stamp, dbos.WithPartitionConcurrency(1),
		dbos.WithQueueBasePollingInterval(100*time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}
	dbos.RegisterWorkflow(dctx, evalWorkflow)
	if err := dctx.Launch(); err != nil {
		t.Fatal(err)
	}
	run := func(marker, partition string, sleepMs int) {
		t.Helper()
		_, err := dbos.RunWorkflow(dctx, evalWorkflow,
			evalInput{Marker: marker, SleepMs: sleepMs, Partition: partition},
			dbos.WithQueue(q), dbos.WithQueuePartitionKey(partition),
			dbos.WithWorkflowID("qeval-"+marker+"-"+stamp))
		if err != nil {
			t.Fatalf("enqueue %s: %v", marker, err)
		}
	}
	run("aqeval-a", "part-1", 3000)
	run("aqeval-b", "part-1", 0)
	run("aqeval-c", "part-2", 0)

	deadline := time.Now().Add(2 * time.Minute)
	for _, m := range []string{"aqeval-a", "aqeval-b", "aqeval-c"} {
		for {
			if v, ok := evalEvents.Load(m); ok && v.(*evalTimes).finish.Load() != 0 {
				break
			}
			if time.Now().After(deadline) {
				t.Fatalf("%s never finished", m)
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
	a, b, c := evalTimesFor(t, "aqeval-a"), evalTimesFor(t, "aqeval-b"), evalTimesFor(t, "aqeval-c")
	if b.start.Load() < a.finish.Load() {
		t.Fatalf("same-partition workflow started before the sleeper finished (slot not held)")
	}
	if c.start.Load() > a.finish.Load() {
		t.Fatalf("different-partition workflow waited for the sleeper (partitions not independent)")
	}
}
