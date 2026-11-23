package classd_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/PithomLabs/miniapp/auth"
	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/effects"
	"github.com/PithomLabs/miniapp/internal/store"
	classd "github.com/PithomLabs/miniapp/probe/classd"
	"github.com/PithomLabs/miniapp/registry"
	"github.com/PithomLabs/miniapp/runtime"
	"github.com/PithomLabs/miniapp/testkit"
	"github.com/PithomLabs/miniapp/testkit/mockcap"
	"github.com/PithomLabs/miniapp/testkit/stack"
)

const (
	credProber  = "key-prober"
	credProberB = "key-prober-b"
	credDecide  = "key-decider"
	tenantPT    = "tenant-pt"
	tenantPB    = "tenant-pb"
)

func probePolicy() auth.Policy {
	binding := func(agent, tenant string, roles ...string) auth.Binding {
		return auth.Binding{AgentID: agent, Authority: contract.Authority{
			Kind: contract.AuthorityUser, ID: agent, TenantID: tenant,
		}, Roles: roles}
	}
	return auth.Policy{
		Keys: map[string]auth.Binding{
			credProber:  binding("agent-probe", tenantPT),
			credProberB: binding("agent-probe-b", tenantPB),
			credDecide:  binding("human-op", tenantPT, "tenant-ops"),
		},
		Grants: map[string]map[string]contract.TrustMode{
			tenantPT: {classd.ID: contract.TrustConfirm},
			tenantPB: {classd.ID: contract.TrustConfirm},
		},
	}
}

func newStack(t *testing.T) *stack.Assembled {
	t.Helper()
	return newStackInner(t, true)
}

// reStack rebuilds the stack over existing database state (crash
// recovery must observe, never wipe, the pre-crash rows).
func reStack(t *testing.T) *stack.Assembled {
	t.Helper()
	return newStackInner(t, false)
}

func newStackInner(t *testing.T, truncate bool) *stack.Assembled {
	t.Helper()
	db := testkit.NewDB(t)
	classd.EnsureTables(t, db.Pool)
	mockcap.EnsureLog(t, db.Pool)
	if truncate {
		db.Truncate(t, "app.invocations", "app.approvals", "app.audit_events",
			"app.effect_records", "app.kill_switches", "probe_sequences",
			"probe_observations", "probe_wait_marks", "mock_provider_log")
	}
	fx := stack.Assemble(t, stack.Config{DB: db,
		Register: func(t *testing.T, reg *registry.Registry) {
			if err := classd.Register(reg); err != nil {
				t.Fatal(err)
			}
		},
		Policy:   probePolicy(),
		Channels: map[string]effects.Channel{"mock": mockcap.New(db.Pool)},
	})
	classd.Effects = fx.Effects
	return fx
}

func seqInput(subject string, opts map[string]any) json.RawMessage {
	m := map[string]any{"subject_id": subject}
	for k, v := range opts {
		m[k] = v
	}
	raw, _ := json.Marshal(m)
	return raw
}

func pollTerminal(t *testing.T, fx *stack.Assembled, cred, invID string) runtime.InvokeResult {
	t.Helper()
	deadline := time.Now().Add(2 * time.Minute)
	for {
		out, err := fx.Svc.Status(context.Background(), cred, invID)
		if err != nil {
			t.Fatalf("status: %v", err)
		}
		switch out.Status {
		case contract.RunSucceeded, contract.RunFailed, contract.RunCancelled:
			return out
		}
		if time.Now().After(deadline) {
			t.Fatalf("invocation %s never settled (last %s)", invID, out.Status)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

type seqRow struct {
	status     string
	outcome    string
	stage      int
	generation int
}

func readSequence(t *testing.T, fx *stack.Assembled, tenant, subject string) seqRow {
	t.Helper()
	row, found := tryReadSequence(t, fx, tenant, subject)
	if !found {
		t.Fatalf("no sequence row for %s/%s", tenant, subject)
	}
	return row
}

func tryReadSequence(t *testing.T, fx *stack.Assembled, tenant, subject string) (seqRow, bool) {
	t.Helper()
	row, err := store.WithTenant(context.Background(), fx.DB.Pool, tenant, func(tx pgx.Tx) (seqRow, error) {
		var r seqRow
		err := tx.QueryRow(context.Background(), `SELECT status, outcome, stage, generation
			FROM probe_sequences WHERE tenant_id = $1 AND subject_id = $2
			ORDER BY generation DESC LIMIT 1`, tenant, subject).Scan(
			&r.status, &r.outcome, &r.stage, &r.generation)
		if err != nil {
			return seqRow{}, err
		}
		return r, nil
	})
	if err != nil {
		return seqRow{}, false
	}
	return row, true
}

func setEligible(t *testing.T, fx *stack.Assembled, tenant, subject string, eligible bool) {
	t.Helper()
	deadline := time.Now().Add(time.Minute)
	for {
		updated, err := store.WithTenant(context.Background(), fx.DB.Pool, tenant, func(tx pgx.Tx) (bool, error) {
			tag, err := tx.Exec(context.Background(), `UPDATE probe_sequences SET eligible = $1
				WHERE tenant_id = $2 AND subject_id = $3 AND status = 'active'`,
				eligible, tenant, subject)
			if err != nil {
				return false, err
			}
			return tag.RowsAffected() > 0, nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if updated {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("sequence row never appeared for eligibility flip")
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func providerSends(t *testing.T, fx *stack.Assembled, opID string) int {
	t.Helper()
	return mockcap.SentCount(t, fx.DB.Pool, opID)
}

// providerSendsSubject counts actual provider sends for a subject
// across all generations (one log row per send). Generations come
// from a global sequence, so tests must never assume generation 1.
// Subjects stay unique per test; the log's tenant column carries
// attribution for cross-tenant assertions elsewhere.
func providerSendsSubject(t *testing.T, fx *stack.Assembled, subject string) int {
	t.Helper()
	n, err := store.WithTenant(context.Background(), fx.DB.Pool, tenantPT, func(tx pgx.Tx) (int, error) {
		var c int
		err := tx.QueryRow(context.Background(),
			`SELECT count(*) FROM mock_provider_log WHERE op_id LIKE $1`, subject+":%").Scan(&c)
		return c, err
	})
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func invokeProbe(t *testing.T, fx *stack.Assembled, tool, subject, key, cred string, opts map[string]any) (runtime.InvokeResult, error) {
	t.Helper()
	return fx.Svc.Invoke(context.Background(), runtime.InvokeRequest{
		MiniappID: classd.ID, Version: classd.Version, Tool: tool,
		Input: seqInput(subject, opts), IdempotencyKey: key,
		Credential: cred, RequestedTrust: contract.TrustConfirm,
	})
}

func decide(t *testing.T, fx *stack.Assembled, tool, approvalID string, d contract.ApprovalDecision, input json.RawMessage) {
	t.Helper()
	hash := runtime.IntentHash(classd.ID, classd.Version, tool, input)
	if err := fx.Svc.DecideApproval(context.Background(), credDecide, approvalID, d, hash); err != nil {
		t.Fatalf("decide: %v", err)
	}
}

func TestProbeHappyPath(t *testing.T) {
	fx := newStack(t)
	out, err := invokeProbe(t, fx, classd.ToolOpen, "happy-1", "key-happy", credProber, map[string]any{
		"sleep_ms": 1000, "wait_ms": 30000,
	})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(3 * time.Second)
	sig := contract.Signal{Type: classd.SignalProceed, Payload: json.RawMessage(`{"go":true}`)}
	if err := fx.Svc.Signal(context.Background(), credProber, out.Handle.RunID, sig); err != nil {
		t.Fatalf("signal: %v", err)
	}
	done := pollTerminal(t, fx, credProber, out.InvocationID)
	if done.Status != contract.RunSucceeded {
		t.Fatalf("status = %s", done.Status)
	}
	row := readSequence(t, fx, tenantPT, "happy-1")
	if row.status != "done" || row.outcome != "completed-signaled" || row.stage != 3 {
		t.Fatalf("sequence = %+v", row)
	}
	if n := providerSends(t, fx, fmt.Sprintf("happy-1:%d", row.generation)); n != 1 {
		t.Fatalf("provider sends = %d, want 1", n)
	}
	events, err := fx.Auditor.Query(context.Background(), tenantPT, 100)
	if err != nil {
		t.Fatal(err)
	}
	saw := map[string]bool{}
	for _, e := range events {
		if e.InvocationID == out.InvocationID {
			saw[e.Action+"/"+e.Outcome] = true
		}
	}
	for _, want := range []string{"invoke/dispatched", "signal/delivered", "run-settled/succeeded"} {
		if !saw[want] {
			t.Fatalf("audit missing %q (have %v)", want, saw)
		}
	}
}

func TestProbeApprovalFlow(t *testing.T) {
	fx := newStack(t)
	input := seqInput("appr-1", map[string]any{"sleep_ms": 500, "wait_ms": 30000, "approval_timeout_ms": 60000})
	out, err := fx.Svc.Invoke(context.Background(), runtime.InvokeRequest{
		MiniappID: classd.ID, Version: classd.Version, Tool: classd.ToolRun,
		Input: input, IdempotencyKey: "key-appr-flow",
		Credential: credProber, RequestedTrust: contract.TrustConfirm,
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Status != contract.RunWaitingApproval {
		t.Fatalf("status = %s, want waiting_approval", out.Status)
	}
	decide(t, fx, classd.ToolRun, out.ApprovalID, contract.DecisionApprove, input)
	time.Sleep(3 * time.Second)
	sig := contract.Signal{Type: classd.SignalProceed, Payload: json.RawMessage(`{}`)}
	if err := fx.Svc.Signal(context.Background(), credProber, out.Handle.RunID, sig); err != nil {
		t.Fatalf("signal: %v", err)
	}
	done := pollTerminal(t, fx, credProber, out.InvocationID)
	if done.Status != contract.RunSucceeded {
		t.Fatalf("status = %s", done.Status)
	}
	row := readSequence(t, fx, tenantPT, "appr-1")
	if row.status != "done" || row.outcome != "completed-signaled" {
		t.Fatalf("sequence = %+v", row)
	}
}

func TestProbeApprovalTimeout(t *testing.T) {
	fx := newStack(t)
	out, err := invokeProbe(t, fx, classd.ToolRun, "appr-to", "key-appr-to", credProber, map[string]any{
		"approval_timeout_ms": 2000, "sleep_ms": 100, "wait_ms": 1000,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := pollTerminal(t, fx, credProber, out.InvocationID)
	if done.Status != contract.RunSucceeded {
		t.Fatalf("status = %s", done.Status)
	}
	row := readSequence(t, fx, tenantPT, "appr-to")
	if row.status != "done" || row.outcome != "timeout" {
		t.Fatalf("sequence = %+v, want done/timeout", row)
	}
	if n := providerSendsSubject(t, fx, "appr-to"); n != 0 {
		t.Fatalf("provider sends = %d, want 0 (nothing consequential before approval)", n)
	}
}

func TestProbeRejectExits(t *testing.T) {
	fx := newStack(t)
	input := seqInput("appr-rej", map[string]any{"sleep_ms": 100, "wait_ms": 1000, "approval_timeout_ms": 60000})
	out, err := fx.Svc.Invoke(context.Background(), runtime.InvokeRequest{
		MiniappID: classd.ID, Version: classd.Version, Tool: classd.ToolRun,
		Input: input, IdempotencyKey: "key-appr-rej",
		Credential: credProber, RequestedTrust: contract.TrustConfirm,
	})
	if err != nil {
		t.Fatal(err)
	}
	decide(t, fx, classd.ToolRun, out.ApprovalID, contract.DecisionReject, input)
	_, err = fx.Svc.Invoke(context.Background(), runtime.InvokeRequest{
		MiniappID: classd.ID, Version: classd.Version, Tool: classd.ToolRun,
		Input: input, IdempotencyKey: "key-appr-rej",
		Credential: credProber, RequestedTrust: contract.TrustConfirm,
	})
	if err == nil || !strings.Contains(err.Error(), "approval-denied") {
		t.Fatalf("poll err = %v, want approval-denied", err)
	}
	if n := providerSendsSubject(t, fx, "appr-rej"); n != 0 {
		t.Fatalf("provider sends = %d, want 0 after rejection", n)
	}
}

func TestProbePoisonQuarantine(t *testing.T) {
	fx := newStack(t)
	out, err := invokeProbe(t, fx, classd.ToolOpen, "poison-1", "key-poison", credProber, map[string]any{
		"sleep_ms": 100, "wait_ms": 1000, "poison_stage": 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	done := pollTerminal(t, fx, credProber, out.InvocationID)
	if done.Status != contract.RunSucceeded {
		t.Fatalf("status = %s", done.Status)
	}
	row := readSequence(t, fx, tenantPT, "poison-1")
	if row.status != "done" || row.outcome != "quarantined" {
		t.Fatalf("sequence = %+v, want done/quarantined", row)
	}
	if n := providerSends(t, fx, fmt.Sprintf("poison-1:%d", row.generation)); n != 1 {
		t.Fatalf("provider sends = %d, want exactly 1 (no blind retry)", n)
	}
	found, err := fx.Effects.ListUnknown(context.Background(), tenantPT, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 {
		t.Fatalf("quarantine = %d records, want 1", len(found))
	}
	if _, err := fx.Effects.Settle(context.Background(), tenantPT, found[0].BusinessOpID, contract.EffectFailed, "test reconcile: provider has no record"); err != nil {
		t.Fatalf("settle: %v", err)
	}
}

func TestProbeCancelMidWait(t *testing.T) {
	fx := newStack(t)
	out, err := invokeProbe(t, fx, classd.ToolOpen, "cancel-1", "key-cancel-wait", credProber, map[string]any{
		"sleep_ms": 30000, "wait_ms": 1000,
	})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Second)
	res, err := fx.Svc.Cancel(context.Background(), credProber, out.Handle.RunID, "stop")
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if len(res.InFlight) != 0 {
		t.Fatalf("in-flight = %v, want clean pre-effect cancel", res.InFlight)
	}
	done := pollTerminal(t, fx, credProber, out.InvocationID)
	if done.Status != contract.RunCancelled {
		t.Fatalf("status = %s, want cancelled", done.Status)
	}
	if n := providerSendsSubject(t, fx, "cancel-1"); n != 0 {
		t.Fatalf("provider sends = %d, want 0", n)
	}
}

func TestProbeConcurrency(t *testing.T) {
	fx := newStack(t)
	// The winner holds a long sleep so the second claim necessarily
	// lands while a sequence is active; asserting exact-race
	// outcomes instead would be timing-flaky by nature.
	winner, err := invokeProbe(t, fx, classd.ToolOpen, "conc-1", "key-conc-a", credProber, map[string]any{
		"sleep_ms": 30000, "wait_ms": 1000,
	})
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Minute)
	for {
		row, found := tryReadSequence(t, fx, tenantPT, "conc-1")
		if found && row.status == "active" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("winner never claimed: %+v", row)
		}
		time.Sleep(200 * time.Millisecond)
	}
	loser, err := invokeProbe(t, fx, classd.ToolOpen, "conc-1", "key-conc-b", credProber, map[string]any{
		"sleep_ms": 100, "wait_ms": 1000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if done := pollTerminal(t, fx, credProber, loser.InvocationID); done.Status != contract.RunSucceeded {
		t.Fatalf("attached run status = %s", done.Status)
	}
	if _, err := fx.Svc.Cancel(context.Background(), credProber, winner.Handle.RunID, "test done"); err != nil {
		t.Fatalf("cancel winner: %v", err)
	}
	if done := pollTerminal(t, fx, credProber, winner.InvocationID); done.Status != contract.RunCancelled {
		t.Fatalf("winner status = %s", done.Status)
	}
	n, err := store.WithTenant(context.Background(), fx.DB.Pool, tenantPT, func(tx pgx.Tx) (int, error) {
		var c int
		err := tx.QueryRow(context.Background(),
			`SELECT count(*) FROM probe_observations WHERE tenant_id = $1`, tenantPT).Scan(&c)
		return c, err
	})
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("attach observations = %d, want exactly 1 (one winner, one attached)", n)
	}
}

func TestProbeCrossTenant(t *testing.T) {
	fx := newStack(t)
	a, err := invokeProbe(t, fx, classd.ToolOpen, "shared-subject", "key-xt-a", credProber, map[string]any{
		"sleep_ms": 100, "wait_ms": 1000,
	})
	if err != nil {
		t.Fatal(err)
	}
	b, err := invokeProbe(t, fx, classd.ToolOpen, "shared-subject", "key-xt-b", credProberB, map[string]any{
		"sleep_ms": 100, "wait_ms": 1000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if done := pollTerminal(t, fx, credProber, a.InvocationID); done.Status != contract.RunSucceeded {
		t.Fatalf("tenant A status = %s", done.Status)
	}
	if done := pollTerminal(t, fx, credProberB, b.InvocationID); done.Status != contract.RunSucceeded {
		t.Fatalf("tenant B status = %s", done.Status)
	}
	ra := readSequence(t, fx, tenantPT, "shared-subject")
	rb := readSequence(t, fx, tenantPB, "shared-subject")
	if ra.status != "done" || rb.status != "done" {
		t.Fatalf("rows = %+v / %+v; identical subject IDs must run independently", ra, rb)
	}
}

func TestProbeEligibilityExit(t *testing.T) {
	fx := newStack(t)
	out, err := invokeProbe(t, fx, classd.ToolOpen, "elig-1", "key-elig", credProber, map[string]any{
		"sleep_ms": 3000, "wait_ms": 1000,
	})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(1 * time.Second)
	setEligible(t, fx, tenantPT, "elig-1", false)
	done := pollTerminal(t, fx, credProber, out.InvocationID)
	if done.Status != contract.RunSucceeded {
		t.Fatalf("status = %s", done.Status)
	}
	row := readSequence(t, fx, tenantPT, "elig-1")
	if row.status != "done" || row.outcome != "ineligible" {
		t.Fatalf("sequence = %+v, want done/ineligible", row)
	}
	if n := providerSendsSubject(t, fx, "elig-1"); n != 0 {
		t.Fatalf("provider sends = %d, want 0 (gate precedes effect)", n)
	}
}

func TestProbeReplayDeterminism(t *testing.T) {
	fx := newStack(t)
	opts := map[string]any{"sleep_ms": 200, "wait_ms": 1000}
	a, err := invokeProbe(t, fx, classd.ToolOpen, "replay-a", "key-replay-a", credProber, opts)
	if err != nil {
		t.Fatal(err)
	}
	b, err := invokeProbe(t, fx, classd.ToolOpen, "replay-b", "key-replay-b", credProber, opts)
	if err != nil {
		t.Fatal(err)
	}
	pollTerminal(t, fx, credProber, a.InvocationID)
	pollTerminal(t, fx, credProber, b.InvocationID)
	ra := readSequence(t, fx, tenantPT, "replay-a")
	rb := readSequence(t, fx, tenantPT, "replay-b")
	if ra.stage != 3 || rb.stage != 3 || ra.outcome != rb.outcome {
		t.Fatalf("rows = %+v / %+v; identical inputs must progress identically", ra, rb)
	}
	ra = readSequence(t, fx, tenantPT, "replay-a")
	rb = readSequence(t, fx, tenantPT, "replay-b")
	if n := providerSends(t, fx, fmt.Sprintf("replay-a:%d", ra.generation)); n != 1 {
		t.Fatalf("run A sends = %d, want 1", n)
	}
	if n := providerSends(t, fx, fmt.Sprintf("replay-b:%d", rb.generation)); n != 1 {
		t.Fatalf("run B sends = %d, want 1", n)
	}
}

func TestProbeAuditReconstruction(t *testing.T) {
	fx := newStack(t)
	input := seqInput("audit-1", map[string]any{"sleep_ms": 500, "wait_ms": 30000, "approval_timeout_ms": 60000})
	out, err := fx.Svc.Invoke(context.Background(), runtime.InvokeRequest{
		MiniappID: classd.ID, Version: classd.Version, Tool: classd.ToolRun,
		Input: input, IdempotencyKey: "key-audit",
		Credential: credProber, RequestedTrust: contract.TrustConfirm,
	})
	if err != nil {
		t.Fatal(err)
	}
	decide(t, fx, classd.ToolRun, out.ApprovalID, contract.DecisionApprove, input)
	time.Sleep(3 * time.Second)
	sig := contract.Signal{Type: classd.SignalProceed, Payload: json.RawMessage(`{}`)}
	if err := fx.Svc.Signal(context.Background(), credProber, out.Handle.RunID, sig); err != nil {
		t.Fatalf("signal: %v", err)
	}
	if done := pollTerminal(t, fx, credProber, out.InvocationID); done.Status != contract.RunSucceeded {
		t.Fatalf("status = %s", done.Status)
	}
	events, err := fx.Auditor.Query(context.Background(), tenantPT, 100)
	if err != nil {
		t.Fatal(err)
	}
	saw := map[string]bool{}
	for _, e := range events {
		if e.InvocationID != out.InvocationID {
			continue
		}
		if e.TenantID == "" || e.AgentID == "" || e.AuthorityKind == "" || e.MiniappID == "" || e.Tool == "" {
			t.Fatalf("unattributed event: %+v", e)
		}
		saw[e.Action+"/"+e.Outcome] = true
	}
	for _, want := range []string{"approval-requested/pending", "approval-decided/approve", "signal/delivered", "run-settled/succeeded"} {
		if !saw[want] {
			t.Fatalf("audit chain missing %q (have %v)", want, saw)
		}
	}
}

const probeWorkerEnv = "MINIAPP_PROBE_WORKER"

func TestProbeCrashWorker(t *testing.T) {
	if !testkit.WorkerMode(probeWorkerEnv) {
		t.Skip("crash worker entrypoint (parent-spawned only)")
	}
	fx := newStack(t)
	dir := os.Getenv("MINIAPP_PROBE_DIR")
	subject := os.Getenv("MINIAPP_PROBE_SUBJECT")
	key := os.Getenv("MINIAPP_PROBE_KEY")
	out, err := invokeProbe(t, fx, classd.ToolOpen, subject, key, credProber, map[string]any{
		"sleep_ms": 15000, "wait_ms": 60000,
	})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "runid"), []byte(out.Handle.RunID), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "submitted"), []byte(out.InvocationID), 0o600); err != nil {
		t.Fatal(err)
	}
	select {}
}

func TestProbeRestart(t *testing.T) {
	testkit.RequireDB(t, "TEST_DATABASE_URL")
	dir := t.TempDir()
	subject := fmt.Sprintf("restart-%d", time.Now().UnixNano())
	key := "key-probe-restart"
	w := testkit.SpawnWorker(t, "TestProbeCrashWorker", map[string]string{
		probeWorkerEnv: "1", "MINIAPP_PROBE_DIR": dir,
		"MINIAPP_PROBE_SUBJECT": subject, "MINIAPP_PROBE_KEY": key,
	})
	testkit.WaitForFile(t, w, filepath.Join(dir, "submitted"), 2*time.Minute)
	time.Sleep(2 * time.Second)
	testkit.Kill9(t, w)

	fx := reStack(t)
	out, err := invokeProbe(t, fx, classd.ToolOpen, subject, key, credProber, map[string]any{
		"sleep_ms": 15000, "wait_ms": 60000,
	})
	if err != nil {
		t.Fatalf("re-invoke after kill: %v", err)
	}
	time.Sleep(2 * time.Second)
	sig := contract.Signal{Type: classd.SignalProceed, Payload: json.RawMessage(`{}`)}
	if err := fx.Svc.Signal(context.Background(), credProber, out.Handle.RunID, sig); err != nil {
		t.Fatalf("signal: %v", err)
	}
	done := pollTerminal(t, fx, credProber, out.InvocationID)
	if done.Status != contract.RunSucceeded {
		t.Fatalf("status = %s", done.Status)
	}
	row := readSequence(t, fx, tenantPT, subject)
	if row.status != "done" || row.stage != 3 {
		t.Fatalf("sequence = %+v, want completed run", row)
	}
	if sends := providerSendsSubject(t, fx, subject); sends != 1 {
		t.Fatalf("provider sends across generations = %d, want exactly 1", sends)
	}
}

// waitForWaitMark blocks until the workflow records approval-wait
// entry for one invocation (bounded, state-based polling).
func waitForWaitMark(t *testing.T, fx *stack.Assembled, invID string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Minute)
	for {
		marked, err := store.WithTenant(context.Background(), fx.DB.Pool, tenantPT, func(tx pgx.Tx) (bool, error) {
			var c int
			err := tx.QueryRow(context.Background(),
				`SELECT count(*) FROM probe_wait_marks WHERE invocation_id = $1`, invID).Scan(&c)
			return c == 1, err
		})
		if err != nil {
			t.Fatal(err)
		}
		if marked {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("workflow never reached the approval wait for %s", invID)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// TestProbeApprovalWaitNoBypass proves the workflow-start
// permission (Rule 3): a gated run submitted on a pending
// approval reaches its durable wait live and performs no
// consequential work; after the grant it proceeds exactly once.
func TestProbeApprovalWaitNoBypass(t *testing.T) {
	fx := newStack(t)
	input := seqInput("wait-1", map[string]any{"sleep_ms": 100, "wait_ms": 1000, "approval_timeout_ms": 120000})
	out, err := invokeProbe(t, fx, classd.ToolRun, "wait-1", "key-wait", credProber, map[string]any{
		"sleep_ms": 100, "wait_ms": 1000, "approval_timeout_ms": 120000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Status != contract.RunWaitingApproval || out.ApprovalID == "" || out.Handle == nil {
		t.Fatalf("invoke = %+v; want parked with run handle", out)
	}
	// The workflow reached its wait branch (marker), is still live
	// (authoritative DBOS status), and sent nothing consequential.
	waitForWaitMark(t, fx, out.InvocationID)
	st, err := fx.Exec.Status(context.Background(), out.Handle.RunID)
	if err != nil {
		t.Fatalf("executor status: %v", err)
	}
	if st == contract.RunSucceeded || st == contract.RunFailed || st == contract.RunCancelled {
		t.Fatalf("workflow status = %s while approval pending, want live", st)
	}
	if n := providerSendsSubject(t, fx, "wait-1"); n != 0 {
		t.Fatalf("provider sends while pending = %d, want 0 (wait not bypassed)", n)
	}
	// After the grant the same run proceeds exactly once.
	decide(t, fx, classd.ToolRun, out.ApprovalID, contract.DecisionApprove, input)
	done := pollTerminal(t, fx, credProber, out.InvocationID)
	if done.Status != contract.RunSucceeded {
		t.Fatalf("status = %s", done.Status)
	}
	if n := providerSendsSubject(t, fx, "wait-1"); n != 1 {
		t.Fatalf("provider sends after grant = %d, want exactly 1", n)
	}
}

// TestProbeReplayWaitsForApproval proves the replay entry path
// obeys the same permission: a legacy Running record without a
// run re-enters approval handling, submits bound to the parked
// linkage, waits live without sending, and proceeds on grant.
func TestProbeReplayWaitsForApproval(t *testing.T) {
	fx := newStack(t)
	// The fixture input must equal the replay input byte-for-byte;
	// anything else is a KeyConflict, not the replay path.
	input := seqInput("wait-replay", map[string]any{"sleep_ms": 100, "wait_ms": 1000, "approval_timeout_ms": 120000})
	_, err := store.WithTenant(context.Background(), fx.DB.Pool, tenantPT, func(tx pgx.Tx) (bool, error) {
		_, err := tx.Exec(context.Background(), `INSERT INTO app.invocations
			(id, tenant_id, miniapp_id, miniapp_version, tool, idempotency_key, input,
			 status, trust_mode, effect_token)
			VALUES (gen_random_uuid(), $1, $2, $3, $4, 'key-wait-replay', $5, 'running', 'confirm', 'tok-wr')`,
			tenantPT, classd.ID, classd.Version, classd.ToolRun, string(input))
		return true, err
	})
	if err != nil {
		t.Fatal(err)
	}
	out, err := invokeProbe(t, fx, classd.ToolRun, "wait-replay", "key-wait-replay", credProber, map[string]any{
		"sleep_ms": 100, "wait_ms": 1000, "approval_timeout_ms": 120000,
	})
	if err != nil {
		t.Fatalf("replay err = %v, want parked result", err)
	}
	if out.Status != contract.RunWaitingApproval || out.ApprovalID == "" || out.Handle == nil {
		t.Fatalf("replay = %+v; want parked with run handle", out)
	}
	waitForWaitMark(t, fx, out.InvocationID)
	st, err := fx.Exec.Status(context.Background(), out.Handle.RunID)
	if err != nil {
		t.Fatalf("executor status: %v", err)
	}
	if st == contract.RunSucceeded || st == contract.RunFailed || st == contract.RunCancelled {
		t.Fatalf("workflow status = %s while approval pending, want live", st)
	}
	if n := providerSendsSubject(t, fx, "wait-replay"); n != 0 {
		t.Fatalf("provider sends while pending = %d, want 0", n)
	}
	decide(t, fx, classd.ToolRun, out.ApprovalID, contract.DecisionApprove,
		seqInput("wait-replay", map[string]any{"sleep_ms": 100, "wait_ms": 1000, "approval_timeout_ms": 120000}))
	done := pollTerminal(t, fx, credProber, out.InvocationID)
	if done.Status != contract.RunSucceeded {
		t.Fatalf("status = %s", done.Status)
	}
	if n := providerSendsSubject(t, fx, "wait-replay"); n != 1 {
		t.Fatalf("provider sends after grant = %d, want exactly 1", n)
	}
}
