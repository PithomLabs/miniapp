package effects_test

// Prompt28 workstream B — six-boundary crash matrix (prompt22
// §10.3), genuine SIGKILL plus persisted PostgreSQL state. Each
// test positions a worker process at its boundary with observable
// state (row locks, ATTEMPTING rows, provider-log markers),
// kills it, and proves recovery: durable state, provider
// accounting, allowed retry, epoch retention, and evidence.
// No boundary relies on a returned error where the contract
// calls for crash behavior.

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/effects"
	"github.com/PithomLabs/miniapp/internal/store"
	"github.com/PithomLabs/miniapp/testkit"
	"github.com/PithomLabs/miniapp/testkit/mockcap"
)

// p28LockingCanceller mirrors production's locked liveness read
// (SELECT status ... FOR UPDATE): claims queue behind a held
// invocation row exactly as they do against the real canceller.
type p28LockingCanceller struct {
	pool *pgxpool.Pool
}

func (c p28LockingCanceller) CheckCancelled(ctx context.Context, tenantID, invocationID string) error {
	live, err := store.WithTenant(ctx, c.pool, tenantID, func(tx pgx.Tx) (bool, error) {
		var status string
		err := tx.QueryRow(ctx, `SELECT status FROM app.invocations
			WHERE id = $1::uuid AND tenant_id = $2 FOR UPDATE`, invocationID, tenantID).Scan(&status)
		if err != nil {
			return false, err
		}
		switch contract.RunStatus(status) {
		case contract.RunRunning, contract.RunWaitingEvent, contract.RunWaitingApproval:
			return true, nil
		default:
			return false, nil
		}
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return effects.ErrInvocationCancelled
		}
		return err
	}
	if !live {
		return effects.ErrInvocationCancelled
	}
	return nil
}

func TestEffectsCrashWorkerLockingCancel(t *testing.T) {
	if !testkit.WorkerMode(workerEnvKey) {
		t.Skip("crash worker entrypoint (parent-spawned only)")
	}
	db := testkit.NewDB(t)
	mockcap.EnsureLog(t, db.Pool)
	svc := newAuthService(t, db, map[string]effects.Channel{"mock": mockcap.New(db.Pool)},
		p28LockingCanceller{pool: db.Pool}, allowPolicy{})
	op := os.Getenv("MINIAPP_EFFECTS_OP")
	_, _ = svc.Perform(context.Background(), wsTenant, effects.Intent{
		BusinessOpID: wsTenant + ":" + op, AttemptID: "a-worker-lock",
		InvocationID: authzInvID(wsTenant, os.Getenv("MINIAPP_EFFECTS_INV")),
		EffectToken:  authzToken,
		Operation: effects.Operation{CapabilityID: "mock", Operation: "send",
			Payload: mockPayload(mockcap.ModeAccept, op)},
	})
}

func TestEffectsCrashWorkerGated(t *testing.T) {
	if !testkit.WorkerMode(workerEnvKey) {
		t.Skip("crash worker entrypoint (parent-spawned only)")
	}
	db := testkit.NewDB(t)
	mockcap.EnsureLog(t, db.Pool)
	gate := &gateChannel{entered: make(chan struct{}), release: make(chan struct{})}
	svc := newAuthService(t, db, map[string]effects.Channel{"mock": gate}, allowCancellation{}, allowPolicy{})
	op := os.Getenv("MINIAPP_EFFECTS_OP")
	_, _ = svc.Perform(context.Background(), wsTenant, effects.Intent{
		BusinessOpID: wsTenant + ":" + op, AttemptID: "a-worker-gated",
		InvocationID: authzInvID(wsTenant, os.Getenv("MINIAPP_EFFECTS_INV")),
		EffectToken:  authzToken,
		Operation: effects.Operation{CapabilityID: "mock", Operation: "send",
			Payload: mockPayload(mockcap.ModeAccept, op)},
	})
}

func p28HoldEffectRow(t *testing.T, db *testkit.DB, op string) (release func()) {
	t.Helper()
	ctx := context.Background()
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, store.TenantContextSQL(), wsTenant); err != nil {
		t.Fatal(err)
	}
	var held string
	if err := tx.QueryRow(ctx, `SELECT business_op_id FROM app.effect_records
		WHERE business_op_id = $1 FOR UPDATE`, wsTenant+":"+op).Scan(&held); err != nil {
		t.Fatal(err)
	}
	return func() { _ = tx.Rollback(ctx) }
}

func p28HoldInvocationRow(t *testing.T, db *testkit.DB, inv string) (release func()) {
	t.Helper()
	ctx := context.Background()
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, store.TenantContextSQL(), wsTenant); err != nil {
		t.Fatal(err)
	}
	var held string
	if err := tx.QueryRow(ctx, `SELECT id::text FROM app.invocations
		WHERE id = $1::uuid FOR UPDATE`, inv).Scan(&held); err != nil {
		t.Fatal(err)
	}
	return func() { _ = tx.Rollback(ctx) }
}

func p28ReadRowOK(t *testing.T, db *testkit.DB, op string) (p28RowView, bool) {
	t.Helper()
	v, err := store.WithTenant(context.Background(), db.Pool, wsTenant, func(tx pgx.Tx) (p28RowView, error) {
		var v p28RowView
		err := tx.QueryRow(context.Background(), `SELECT state, attempt_id,
			provider_request, provider_reply FROM app.effect_records
			WHERE business_op_id = $1 AND tenant_id = $2`, op, wsTenant).Scan(
			&v.state, &v.attempt, &v.req, &v.reply)
		return v, err
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return p28RowView{}, false
		}
		t.Fatal(err)
	}
	return v, true
}

// p28Svc builds the standard authorized service WITHOUT
// truncating: post-kill recovery must observe survivor state.
func p28Svc(t *testing.T, db *testkit.DB) *effects.Service {
	t.Helper()
	mockcap.EnsureLog(t, db.Pool)
	return newAuthService(t, db, map[string]effects.Channel{"mock": mockcap.New(db.Pool)}, allowCancellation{}, allowPolicy{})
}

func p28EffectRowCount(t *testing.T, db *testkit.DB, op string) int {
	t.Helper()
	var n int
	err := db.Pool.QueryRow(context.Background(), `SELECT count(*) FROM app.effect_records
		WHERE business_op_id = $1`, wsTenant+":"+op).Scan(&n)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func p28MockCount(t *testing.T, db *testkit.DB, op string) int {
	t.Helper()
	var n int
	err := db.Pool.QueryRow(context.Background(), `SELECT count(*) FROM mock_provider_log
		WHERE op_id = $1`, op).Scan(&n)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func p28SeedInvocationStatus(t *testing.T, db *testkit.DB, local, status string) string {
	t.Helper()
	inv := authzInvID(wsTenant, local)
	_, err := store.WithTenant(context.Background(), db.Pool, wsTenant, func(tx pgx.Tx) (bool, error) {
		_, err := tx.Exec(context.Background(), `INSERT INTO app.invocations
			(id, tenant_id, miniapp_id, miniapp_version, tool, idempotency_key, input,
			 status, trust_mode, effect_token)
			VALUES ($1, $2, 't-auth-fix', '1.0.0', 't.auth', $3, '{}', $4, 'autonomous', $5)`,
			inv, wsTenant, "key-"+local, status, authzToken)
		return true, err
	})
	if err != nil {
		t.Fatal(err)
	}
	return inv
}

// B1a — kill queued before the authorization transaction (and
// before claim): nothing persisted for the op; later liveness
// converges cleanly.
func TestP28CrashBeforeClaimQueue(t *testing.T) {
	db := testkit.NewDB(t)
	truncateLedger(t, db)
	mockcap.EnsureLog(t, db.Pool)
	mockcap.TruncateLog(t, db.Pool)
	inv := p28SeedInvocationStatus(t, db, "inv-p28b1a", "pending")
	release := p28HoldInvocationRow(t, db, inv)
	defer release()
	w := testkit.SpawnWorker(t, "TestEffectsCrashWorkerLockingCancel$", map[string]string{
		workerEnvKey: "1", "MINIAPP_EFFECTS_OP": "p28b1a", "MINIAPP_EFFECTS_INV": "inv-p28b1a",
	})
	p28WaitLockWaiter(t, db.Pool, "SELECT status FROM app.invocations", "pre-claim liveness queue")
	testkit.Kill9(t, w)
	if n := p28EffectRowCount(t, db, "p28b1a"); n != 0 {
		t.Errorf("effect rows = %d, want 0 (killed before claim)", n)
	}
	release()
	// Convergence once live (adoption proven in B5; simulate the
	// live transition): full success, exactly one send.
	_, err := store.WithTenant(context.Background(), db.Pool, wsTenant, func(tx pgx.Tx) (bool, error) {
		_, err := tx.Exec(context.Background(), `UPDATE app.invocations SET status = 'running'
			WHERE id = $1::uuid`, inv)
		return true, err
	})
	if err != nil {
		t.Fatal(err)
	}
	svc := p28Svc(t, db)
	intent := mockIntent("p28b1a", mockcap.ModeAccept)
	intent.InvocationID = inv
	rec, err := svc.Perform(context.Background(), wsTenant, intent)
	if err != nil || rec.State != contract.EffectConfirmed {
		t.Fatalf("converged = %+v %v, want CONFIRMED", rec, err)
	}
	if n := mockcap.SentCount(t, db.Pool, "p28b1a"); n != 1 {
		t.Errorf("sends = %d, want exactly 1", n)
	}
}

// B1b — kill queued mid-authorization-transaction (claim
// committed, auth uncommitted): PENDING survives, nothing else
// does; takeover converges with a fresh epoch.
func TestP28CrashMidAuthorizeQueue(t *testing.T) {
	db := testkit.NewDB(t)
	truncateLedger(t, db)
	mockcap.EnsureLog(t, db.Pool)
	mockcap.TruncateLog(t, db.Pool)
	seedLiveInvocation(t, db, wsTenant, "inv-kill")
	inv := authzInvID(wsTenant, "inv-kill")
	release := p28HoldInvocationRow(t, db, inv)
	defer release()
	w := testkit.SpawnWorker(t, "TestEffectsCrashWorkerGated$", map[string]string{
		workerEnvKey: "1", "MINIAPP_EFFECTS_OP": "p28b1b", "MINIAPP_EFFECTS_INV": "inv-kill",
	})
	p28WaitLockWaiter(t, db.Pool, "SELECT id::text", "mid-authorize queue")
	if v := p28ReadRow(t, db, wsTenant+":p28b1b"); v.state != "PENDING" {
		testkit.Kill9(t, w)
		t.Fatalf("pre-kill row = %s, want PENDING (claim committed, auth open)", v.state)
	}
	claimed := p28ReadRow(t, db, wsTenant+":p28b1b")
	testkit.Kill9(t, w)
	release()
	final := p28ReadRow(t, db, wsTenant+":p28b1b")
	if final.state != "PENDING" || final.attempt != claimed.attempt {
		t.Errorf("post-kill = (%s, %s), want (PENDING, %s)", final.state, final.attempt, claimed.attempt)
	}
	if n := p28RefusedAudits(t, db, wsTenant+":p28b1b"); n != 0 {
		t.Errorf("refused audits = %d, want 0 (nothing decided)", n)
	}
	p28AgeRow(t, db, wsTenant+":p28b1b")
	svc := p28Svc(t, db)
	intent := mockIntent("p28b1b", mockcap.ModeAccept)
	intent.InvocationID = inv
	rec, err := svc.Perform(context.Background(), wsTenant, intent)
	if err != nil || rec.State != contract.EffectConfirmed {
		t.Fatalf("takeover = %+v %v, want CONFIRMED", rec, err)
	}
	if rec.AttemptID == claimed.attempt {
		t.Errorf("takeover epoch = %s, want fresh (not the dead claim's)", rec.AttemptID)
	}
	if n := mockcap.SentCount(t, db.Pool, "p28b1b"); n != 1 {
		t.Errorf("sends = %d, want exactly 1", n)
	}
}

// B2 — kill after authorization commit, before dispatch: the
// ATTEMPTING row is durable with zero provider contact;
// recovery quarantines (never resends the ambiguous epoch).
func TestP28CrashAfterAuthorizeBeforeSend(t *testing.T) {
	db := testkit.NewDB(t)
	truncateLedger(t, db)
	mockcap.EnsureLog(t, db.Pool)
	mockcap.TruncateLog(t, db.Pool)
	seedLiveInvocation(t, db, wsTenant, "inv-kill")
	w := testkit.SpawnWorker(t, "TestEffectsCrashWorkerGated$", map[string]string{
		workerEnvKey: "1", "MINIAPP_EFFECTS_OP": "p28b2", "MINIAPP_EFFECTS_INV": "inv-kill",
	})
	deadline := time.Now().Add(60 * time.Second)
	var row p28RowView
	for {
		var ok bool
		if row, ok = p28ReadRowOK(t, db, wsTenant+":p28b2"); ok && row.state == "ATTEMPTING" {
			break
		}
		if time.Now().After(deadline) {
			testkit.Kill9(t, w)
			t.Fatal("worker never committed ATTEMPTING")
		}
		time.Sleep(20 * time.Millisecond)
	}
	if n := p28MockCount(t, db, "p28b2"); n != 0 {
		testkit.Kill9(t, w)
		t.Fatalf("provider log = %d rows, want 0 (kill must precede dispatch)", n)
	}
	testkit.Kill9(t, w)
	kept := row.attempt
	svc := p28Svc(t, db)
	intent := mockIntent("p28b2", mockcap.ModeAccept)
	intent.InvocationID = authzInvID(wsTenant, "inv-kill")
	if _, err := svc.Perform(context.Background(), wsTenant, intent); !errors.Is(err, effects.ErrAttemptInProgress) {
		t.Fatalf("immediate retry err = %v, want contention", err)
	}
	p28AgeRow(t, db, wsTenant+":p28b2")
	rec, err := svc.Perform(context.Background(), wsTenant, intent)
	if err != nil || rec.State != contract.EffectUnknown {
		t.Fatalf("aged retry = %+v %v, want UNKNOWN quarantine", rec, err)
	}
	final := p28ReadRow(t, db, wsTenant+":p28b2")
	if final.attempt != kept {
		t.Errorf("epoch = %s, want retained %s (quarantine keeps the dead epoch)", final.attempt, kept)
	}
	if n := p28MockCount(t, db, "p28b2"); n != 0 {
		t.Errorf("provider log = %d rows, want 0 (ambiguous epoch never sent)", n)
	}
}

// B4/B5 — kill between provider response and outcome
// persistence, forced by stalling the settlement UPDATE behind
// a held row lock after the DelayedAck outcome is durably
// recorded. B4 (response received, persistence unattempted)
// and B5 (mid-persistence) are externally identical at every
// observable: one deterministic test covers both, recorded
// explicitly here rather than split by sub-statement timing
// no test could separate. Recovery quarantines without
// applying the recorded outcome; an operator settles from
// the provider-log evidence.
func TestP28CrashBetweenResponseAndPersistence(t *testing.T) {
	db := testkit.NewDB(t)
	truncateLedger(t, db)
	mockcap.EnsureLog(t, db.Pool)
	mockcap.TruncateLog(t, db.Pool)
	seedLiveInvocation(t, db, wsTenant, "inv-kill")
	op := wsTenant + ":p28b4"
	w := testkit.SpawnWorker(t, "TestEffectsCrashWorker$", map[string]string{
		workerEnvKey: "1", "MINIAPP_EFFECTS_OP": "p28b4",
	})
	deadline := time.Now().Add(60 * time.Second)
	for {
		if v, ok := p28ReadRowOK(t, db, op); ok && v.state == "ATTEMPTING" {
			break
		}
		if time.Now().After(deadline) {
			testkit.Kill9(t, w)
			t.Fatal("worker never committed ATTEMPTING")
		}
		time.Sleep(20 * time.Millisecond)
	}
	// The worker holds no lock during its 2s DelayedAck send:
	// seize the row so its settlement UPDATE must queue.
	release := p28HoldEffectRow(t, db, "p28b4")
	p28WaitLockWaiter(t, db.Pool, "UPDATE app.effect_records", "settlement UPDATE queue")
	if n := p28MockCount(t, db, "p28b4"); n != 1 {
		testkit.Kill9(t, w)
		release()
		t.Fatalf("provider log = %d rows, want 1 (response recorded pre-kill)", n)
	}
	kept := p28ReadRow(t, db, op)
	testkit.Kill9(t, w)
	release()
	if v := p28ReadRow(t, db, op); v.state != "ATTEMPTING" || v.attempt != kept.attempt {
		t.Fatalf("post-kill = (%s, %s), want (ATTEMPTING, %s)", v.state, v.attempt, kept.attempt)
	}
	p28AgeRow(t, db, wsTenant+":p28b4")
	svc := p28Svc(t, db)
	intent := mockIntent("p28b4", mockcap.ModeDelayedAck) // match the worker's claimed payload
	intent.InvocationID = authzInvID(wsTenant, "inv-kill")
	rec, err := svc.Perform(context.Background(), wsTenant, intent)
	if err != nil || rec.State != contract.EffectUnknown {
		t.Fatalf("recovery = %+v %v, want UNKNOWN quarantine", rec, err)
	}
	final := p28ReadRow(t, db, op)
	if final.reply == "" || final.reply == "ok:p28b4" {
		t.Errorf("quarantine reply = %q, want the adopt marker (recorded outcome never applied)", final.reply)
	}
	if final.attempt != kept.attempt {
		t.Errorf("epoch = %s, want retained %s", final.attempt, kept.attempt)
	}
	// The operator settles from the provider-log evidence.
	settled, err := svc.Settle(context.Background(), wsTenant, op, contract.EffectConfirmed,
		"evidence: provider accepted delayed-ack (mock log row present)", "p28-op")
	if err != nil || settled.State != contract.EffectConfirmed {
		t.Fatalf("evidence settle = %+v %v, want CONFIRMED", settled, err)
	}
}

// B6a — kill queued inside takeover authorization: the stale
// PENDING row survives untouched; the next touch takes over
// with a fresh epoch and sends once.
func TestP28CrashDuringTakeoverAuthorize(t *testing.T) {
	db := testkit.NewDB(t)
	truncateLedger(t, db)
	mockcap.EnsureLog(t, db.Pool)
	mockcap.TruncateLog(t, db.Pool)
	seedLiveInvocation(t, db, wsTenant, "inv-kill")
	op := wsTenant + ":p28b6a"
	_, err := store.WithTenant(context.Background(), db.Pool, wsTenant, func(tx pgx.Tx) (bool, error) {
		_, err := tx.Exec(context.Background(), `INSERT INTO app.effect_records
			(business_op_id, tenant_id, attempt_id, idempotency_key, invocation_id,
			 capability_id, operation, payload, state, updated_at)
			VALUES ($1, $2, 'a-p28b6a', '', $3, 'mock', 'send',
			 '{"mock":{"mode":"accept","opId":"p28b6a"}}', 'PENDING', now() - interval '1 hour')`,
			op, wsTenant, authzInvID(wsTenant, "inv-kill"))
		return true, err
	})
	if err != nil {
		t.Fatal(err)
	}
	release := p28HoldEffectRow(t, db, "p28b6a")
	defer release()
	w := testkit.SpawnWorker(t, "TestEffectsCrashWorkerGated$", map[string]string{
		workerEnvKey: "1", "MINIAPP_EFFECTS_OP": "p28b6a", "MINIAPP_EFFECTS_INV": "inv-kill",
	})
	p28WaitLockWaiter(t, db.Pool, "UPDATE app.effect_records", "takeover step-5 queue")
	testkit.Kill9(t, w)
	release()
	if v := p28ReadRow(t, db, op); v.state != "PENDING" || v.attempt != "a-p28b6a" {
		t.Fatalf("post-kill = (%s, %s), want (PENDING, a-p28b6a) untouched", v.state, v.attempt)
	}
	svc := p28Svc(t, db)
	intent := mockIntent("p28b6a", mockcap.ModeAccept)
	intent.InvocationID = authzInvID(wsTenant, "inv-kill")
	rec, err := svc.Perform(context.Background(), wsTenant, intent)
	if err != nil || rec.State != contract.EffectConfirmed {
		t.Fatalf("retry = %+v %v, want CONFIRMED", rec, err)
	}
	if rec.AttemptID == "a-p28b6a" {
		t.Errorf("epoch = %s, want a fresh takeover epoch", rec.AttemptID)
	}
	if n := mockcap.SentCount(t, db.Pool, "p28b6a"); n != 1 {
		t.Errorf("sends = %d, want exactly 1", n)
	}
}

// B6b — kill queued inside quarantine adoption: the stale
// ATTEMPTING row survives untouched; the next touch
// quarantines with zero new sends.
func TestP28CrashDuringQuarantineAdopt(t *testing.T) {
	db := testkit.NewDB(t)
	truncateLedger(t, db)
	mockcap.EnsureLog(t, db.Pool)
	mockcap.TruncateLog(t, db.Pool)
	seedLiveInvocation(t, db, wsTenant, "inv-kill")
	op := wsTenant + ":p28b6b"
	_, err := store.WithTenant(context.Background(), db.Pool, wsTenant, func(tx pgx.Tx) (bool, error) {
		_, err := tx.Exec(context.Background(), `INSERT INTO app.effect_records
			(business_op_id, tenant_id, attempt_id, idempotency_key, invocation_id,
			 capability_id, operation, payload, state, updated_at)
			VALUES ($1, $2, 'a-p28b6b', '', $3, 'mock', 'send',
			 '{"mock":{"mode":"accept","opId":"p28b6b"}}', 'ATTEMPTING', now() - interval '1 hour')`,
			op, wsTenant, authzInvID(wsTenant, "inv-kill"))
		return true, err
	})
	if err != nil {
		t.Fatal(err)
	}
	release := p28HoldEffectRow(t, db, "p28b6b")
	defer release()
	w := testkit.SpawnWorker(t, "TestEffectsCrashWorkerGated$", map[string]string{
		workerEnvKey: "1", "MINIAPP_EFFECTS_OP": "p28b6b", "MINIAPP_EFFECTS_INV": "inv-kill",
	})
	p28WaitLockWaiter(t, db.Pool, "UPDATE app.effect_records", "adopt quarantine queue")
	testkit.Kill9(t, w)
	release()
	if v := p28ReadRow(t, db, op); v.state != "ATTEMPTING" || v.attempt != "a-p28b6b" {
		t.Fatalf("post-kill = (%s, %s), want (ATTEMPTING, a-p28b6b) untouched", v.state, v.attempt)
	}
	svc := p28Svc(t, db)
	intent := mockIntent("p28b6b", mockcap.ModeAccept)
	intent.InvocationID = authzInvID(wsTenant, "inv-kill")
	rec, err := svc.Perform(context.Background(), wsTenant, intent)
	if err != nil || rec.State != contract.EffectUnknown {
		t.Fatalf("retry = %+v %v, want UNKNOWN quarantine", rec, err)
	}
	if n := mockcap.SentCount(t, db.Pool, "p28b6b"); n != 0 {
		t.Errorf("sends = %d, want 0 (quarantine never sends)", n)
	}
}
