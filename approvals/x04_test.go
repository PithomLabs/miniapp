// X-04 discriminating tests (concurrent conflicting decisions).
//
// RED/GREEN-COMPATIBLE SOURCE: uses only the stable Approver API
// plus shared fixtures from approvals_test.go (present in both
// trees). The red run overlays exactly this file onto the
// snapshot. testRequest/requester/approver identities come from
// the existing helpers.
package approvals

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/internal/store"
	"github.com/PithomLabs/miniapp/testkit"
)

// rowGateHolder locks one approval row in an uncommitted
// transaction on a dedicated connection. Callers must invoke the
// returned release exactly once (rollback; nothing was written).
// Duplicated per red-compatible test file on purpose: shared
// helpers would need overlaying into the snapshot's non-test
// files, which red/green forbids.
func rowGateHolder(t *testing.T, db *testkit.DB, tenant, id string) (holderPID int32, release func()) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	pool, err := pgxpool.New(ctx, db.DSN)
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		cancel()
		pool.Close()
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, store.TenantContextSQL(), tenant); err != nil {
		cancel()
		pool.Close()
		t.Fatal(err)
	}
	var got string
	if err := tx.QueryRow(ctx, `SELECT id::text FROM app.approvals
		WHERE id = $1::uuid FOR UPDATE`, id).Scan(&got); err != nil {
		cancel()
		pool.Close()
		t.Fatal(err)
	}
	if err := tx.QueryRow(ctx, `SELECT pg_backend_pid()`).Scan(&holderPID); err != nil {
		cancel()
		pool.Close()
		t.Fatal(err)
	}
	return holderPID, func() {
		defer cancel()
		defer pool.Close()
		_ = tx.Rollback(ctx)
	}
}

// waitBlockedByHolder waits until exactly want backends are
// lock-blocked behind the holder on statements touching tableLike,
// with no other lock waiters database-wide. Blockage is proven
// transitively: each waiter's pg_blocking_pids chain must root at
// the holder pid (tuple-lock queues chain waiter-behind-waiter, so
// direct linkage alone is insufficient). Reports success; the
// caller releases the holder and joins racers on every path.
func waitBlockedByHolder(t *testing.T, db *testkit.DB, holderPID int32, tableLike string, want int) bool {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	deadline := time.Now().Add(30 * time.Second)
	for {
		matched, total, err := blockedBehind(t, ctx, db, holderPID, tableLike)
		if err != nil {
			t.Fatal(err)
		}
		if len(matched) == want && total == want {
			return true
		}
		if time.Now().After(deadline) {
			dumpActivity(t, ctx, db)
			t.Logf("blocked-behind-holder waiters = %d (pids %v), total lock waiters = %d; want %d on %s (holder %d)",
				len(matched), matched, total, want, tableLike, holderPID)
			return false
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// blockedBehind returns the pids whose statements touch tableLike
// and whose blocking chain roots at the holder, plus the total
// lock-waiter count database-wide.
func blockedBehind(t *testing.T, ctx context.Context, db *testkit.DB, holderPID int32, tableLike string) ([]int32, int, error) {
	t.Helper()
	type backend struct {
		pid      int32
		query    string
		blocking []int32
	}
	rows, err := db.Pool.Query(ctx, `SELECT pid, query, pg_blocking_pids(pid) FROM pg_stat_activity
		WHERE datname = current_database() AND pid <> pg_backend_pid()
		AND wait_event_type = 'Lock'`)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	byPID := map[int32]backend{}
	var waiters []int32
	for rows.Next() {
		var b backend
		if err := rows.Scan(&b.pid, &b.query, &b.blocking); err != nil {
			return nil, 0, err
		}
		byPID[b.pid] = b
		waiters = append(waiters, b.pid)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	rootsAt := func(pid int32) bool {
		seen := map[int32]bool{pid: true}
		queue := []int32{pid}
		for len(queue) > 0 {
			cur := queue[0]
			queue = queue[1:]
			if cur == holderPID {
				return true
			}
			b, ok := byPID[cur]
			if !ok {
				continue
			}
			for _, next := range b.blocking {
				if !seen[next] {
					seen[next] = true
					queue = append(queue, next)
				}
			}
		}
		return false
	}
	var matched []int32
	for _, pid := range waiters {
		q := byPID[pid].query
		if strings.Contains(strings.ToUpper(q), strings.ToUpper(tableLike)) && rootsAt(pid) {
			matched = append(matched, pid)
		}
	}
	return matched, len(waiters), nil
}

func dumpActivity(t *testing.T, ctx context.Context, db *testkit.DB) {
	t.Helper()
	diag, err := db.Pool.Query(ctx, `SELECT pid, state, wait_event_type, wait_event,
		pg_blocking_pids(pid), left(query, 120) FROM pg_stat_activity
		WHERE datname = current_database() AND pid <> pg_backend_pid()
		ORDER BY pid`)
	if err != nil {
		return
	}
	defer diag.Close()
	for diag.Next() {
		var pid int32
		var state, q string
		var wet, we *string
		var blocking []int32
		_ = diag.Scan(&pid, &state, &wet, &we, &blocking, &q)
		wetS, weS := "-", "-"
		if wet != nil {
			wetS = *wet
		}
		if we != nil {
			weS = *we
		}
		t.Logf("backend pid=%d state=%s wait=%s:%s blocking=%v query=%.100s",
			pid, state, wetS, weS, blocking, q)
	}
}

// TestConcurrentApproveReject gates two conflicting deciders
// behind a row lock so both must observe pending before either
// commits: exactly one wins, the loser gets a defined conflict,
// and the final row matches the winner. Green passes
// deterministically (lock + predicates). Red fails
// deterministically (both unpredicated overwrites succeed).
func TestConcurrentApproveReject(t *testing.T) {
	const rounds = 3
	db := testkit.NewDB(t)
	svc := New(db.Pool)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	decisions := []contract.ApprovalDecision{contract.DecisionApprove, contract.DecisionReject}
	deciders := []string{"agent-racer-a", "agent-racer-b"}
	for round := 0; round < rounds; round++ {
		db.Truncate(t, "app.approvals")
		id, err := svc.Submit(ctx, testRequest())
		if err != nil {
			t.Fatal(err)
		}
		holder, release := rowGateHolder(t, db, "t-appr", id)
		start := make(chan struct{})
		var wg sync.WaitGroup
		errs := make([]error, 2)
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				<-start
				d := reviewer()
				d.AgentID = deciders[i]
				errs[i] = svc.Decide(ctx, "t-appr", id, d, decisions[i], testHash)
			}(i)
		}
		close(start)
		// Both racers must be blocked behind OUR holder on the
		// approvals write path (green: locked read; red:
		// blind update) before either proceeds: at this
		// point both have observed pending and neither has
		// committed. Release and join on every path so a gate
		// failure can never wedge teardown.
		if !waitBlockedByHolder(t, db, holder, "app.approvals", 2) {
			release()
			wg.Wait()
			t.Fatalf("round %d: gate never formed; see activity dump above", round)
		}
		release()
		wg.Wait()
		nok := 0
		for _, err := range errs {
			if err == nil {
				nok++
			}
		}
		if nok != 1 {
			t.Fatalf("round %d: successes = %d, want exactly 1 (silent overwrites)", round, nok)
		}
		rec, err := svc.Get(ctx, "t-appr", id)
		if err != nil {
			t.Fatal(err)
		}
		for i, err := range errs {
			if err == nil && (rec.Decision != decisions[i] || rec.DecidedBy != deciders[i]) {
				t.Fatalf("round %d: final = %+v, winner decided %s as %s", round, rec, decisions[i], deciders[i])
			}
		}
	}
}

// TestConcurrentDecideAndCancel races a decision against
// CancelPending: the pair must be consistent — either the decision
// wins (cancel settles nothing) or cancellation wins (the decision
// is refused). No silent overwrite in either order. The losing
// order on red is cancel-first (unpredicated Decide overwrites);
// rounds make observing it near-certain, while green is consistent
// on every round by construction.
func TestConcurrentDecideAndCancel(t *testing.T) {
	db := testkit.NewDB(t)
	svc := New(db.Pool)
	for round := 0; round < 20; round++ {
		db.Truncate(t, "app.approvals")
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		id, err := svc.Submit(ctx, testRequest())
		if err != nil {
			t.Fatal(err)
		}
		holder, release := rowGateHolder(t, db, "t-appr", id)
		start := make(chan struct{})
		var wg sync.WaitGroup
		var decideErr error
		var cancelled bool
		var cancelErr error
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			decideErr = svc.Decide(ctx, "t-appr", id, reviewer(), contract.DecisionApprove, testHash)
		}()
		go func() {
			defer wg.Done()
			<-start
			cancelled, cancelErr = svc.CancelPending(ctx, "t-appr", "inv-1")
		}()
		close(start)
		// Both sides observe pending before either commits (the
		// gate forces true overlap every round); post-release
		// order is genuinely racy, so rounds cover both orders
		// and every round must be internally consistent.
		// Release and join on every path so a gate failure can
		// never wedge teardown.
		if !waitBlockedByHolder(t, db, holder, "app.approvals", 2) {
			release()
			wg.Wait()
			t.Fatalf("round %d: gate never formed; see activity dump above", round)
		}
		release()
		wg.Wait()
		if cancelErr != nil {
			t.Fatalf("round %d: cancel err = %v", round, cancelErr)
		}
		rec, err := svc.Get(ctx, "t-appr", id)
		if err != nil {
			t.Fatal(err)
		}
		decideWon := decideErr == nil
		cancelWon := cancelled
		if decideWon == cancelWon {
			t.Fatalf("round %d: decideErr=%v cancelled=%v final=%s (exactly one must win)",
				round, decideErr, cancelled, rec.Status)
		}
		if decideWon && (rec.Status != "decided" || rec.Decision != contract.DecisionApprove) {
			t.Fatalf("round %d: decide won but row = %+v", round, rec)
		}
		if cancelWon && (rec.Status != "cancelled" || decideErr == nil ||
			!strings.Contains(decideErr.Error(), "already decided")) {
			t.Fatalf("round %d: cancel won but decideErr=%v row=%+v (decision must be refused)", round, decideErr, rec)
		}
	}
}

// TestConcurrentSelfApprovalRefused pins that the requester can
// never land a self-approval through timing: the self-check reads
// immutable fields (requester identity, decider identity), so the
// requester's call fails deterministically on both trees. This is
// a guard test, not a red discriminator: no red failure is
// expected, and none would indicate the X-04 defect.
func TestConcurrentSelfApprovalRefused(t *testing.T) {
	db := testkit.NewDB(t)
	db.Truncate(t, "app.approvals")
	svc := New(db.Pool)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	id, err := svc.Submit(ctx, testRequest())
	if err != nil {
		t.Fatal(err)
	}
	self := reviewer()
	self.AgentID = "agent-1" // the requesting agent
	start := make(chan struct{})
	var wg sync.WaitGroup
	var selfErr, apprErr error
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		selfErr = svc.Decide(ctx, "t-appr", id, self, contract.DecisionApprove, testHash)
	}()
	go func() {
		defer wg.Done()
		<-start
		apprErr = svc.Decide(ctx, "t-appr", id, reviewer(), contract.DecisionApprove, testHash)
	}()
	close(start)
	wg.Wait()
	// Either refusal is correct (self-check if read while pending,
	// already-decided if the approver won first); what must never
	// happen is the self-approval landing.
	if selfErr == nil {
		t.Fatal("self-decide succeeded; self-approval must never land")
	}
	if !strings.Contains(selfErr.Error(), "cannot approve its own request") &&
		!strings.Contains(selfErr.Error(), "already decided") {
		t.Fatalf("self-decide err = %v, want self-refusal or already-decided", selfErr)
	}
	if apprErr != nil {
		t.Fatalf("approver decide err = %v, want success", apprErr)
	}
	rec, err := svc.Get(ctx, "t-appr", id)
	if err != nil || rec.DecidedBy != "agent-2" || rec.Decision != contract.DecisionApprove {
		t.Fatalf("final = %+v, %v; want the approver's decision", rec, err)
	}
}

// TestConcurrentExpiryAndDecision pins expiry behavior under
// concurrency: an expired-but-unmarked row reports expired to all
// racers with a single marking. Pre-sweeper this is consistent on
// both trees (guard test); it gains teeth when the WS-D sweeper
// marks expiry concurrently.
func TestConcurrentExpiryAndDecision(t *testing.T) {
	db := testkit.NewDB(t)
	db.Truncate(t, "app.approvals")
	svc := New(db.Pool)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	req := testRequest()
	req.Expiry = time.Now().UTC().Add(-time.Hour)
	id, err := svc.Submit(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			errs[i] = svc.Decide(ctx, "t-appr", id, reviewer(), contract.DecisionApprove, testHash)
		}(i)
	}
	close(start)
	wg.Wait()
	for i, err := range errs {
		if err == nil || !strings.Contains(err.Error(), "expired") {
			t.Fatalf("racer %d err = %v, want expiry report", i, err)
		}
	}
	rec, err := svc.Get(ctx, "t-appr", id)
	if err != nil || rec.Status != "expired" {
		t.Fatalf("final = %+v, %v; want expired", rec, err)
	}
}
