package dbosexec_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dbos-inc/dbos-transact-golang/dbos"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PithomLabs/miniapp/approvals"
	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/executor"
	dbosexec "github.com/PithomLabs/miniapp/executor/dbos"
	"github.com/PithomLabs/miniapp/registry"
	"github.com/PithomLabs/miniapp/testkit"
)

// probeDS is the probe miniapp's datasource, bound once at executor
// setup and never mutated mid-workflow.
var probeDS *dbos.DataSource

type probeInput struct {
	TenantID string `json:"tenantId"`
	Marker   string `json:"marker"`
	SleepMs  int    `json:"sleepMs,omitempty"`
}

func translateProbe(req registry.WorkflowRequest) (probeInput, error) {
	var body struct {
		Marker string `json:"marker"`
	}
	if err := json.Unmarshal(req.Input, &body); err != nil {
		return probeInput{}, err
	}
	return probeInput{TenantID: req.TenantID, Marker: body.Marker}, nil
}

// recordWorkflow durably records one marker under the invoking
// tenant. It is the substrate probe: no business semantics.
func recordWorkflow(ctx dbos.Context, in probeInput) (string, error) {
	return dbos.RunAsTransaction(ctx, probeDS, func(ctx context.Context, tx dbos.Tx) (string, error) {
		if err := dbosexec.ArmTenant(ctx, tx, in.TenantID); err != nil {
			return "", err
		}
		_, err := tx.Exec(ctx, `INSERT INTO probe_markers(marker, tenant_id, stage)
			VALUES ($1, $2, 'done')`, in.Marker, in.TenantID)
		return in.Marker, err
	})
}

// awaitWorkflow blocks for a probe.go signal, then records the
// marker annotated with the received payload.
func awaitWorkflow(ctx dbos.Context, in probeInput) (string, error) {
	payload, err := dbos.Recv[string](ctx, "probe.go", 2*time.Minute)
	if err != nil {
		return "", err
	}
	return dbos.RunAsTransaction(ctx, probeDS, func(ctx context.Context, tx dbos.Tx) (string, error) {
		if err := dbosexec.ArmTenant(ctx, tx, in.TenantID); err != nil {
			return "", err
		}
		_, err := tx.Exec(ctx, `INSERT INTO probe_markers(marker, tenant_id, stage)
			VALUES ($1, $2, $3)`, in.Marker, in.TenantID, "signaled:"+payload)
		return payload, err
	})
}

func recordRegister(env registry.WorkflowEnv) error {
	probeDS = env.Data
	dbos.RegisterWorkflow(env.Context, recordWorkflow)
	return nil
}

func awaitRegister(env registry.WorkflowEnv) error {
	probeDS = env.Data
	dbos.RegisterWorkflow(env.Context, awaitWorkflow)
	return nil
}

func recordStart(ctx dbos.Context, req registry.WorkflowRequest, runID string) error {
	in, err := translateProbe(req)
	if err != nil {
		return err
	}
	_, err = dbos.RunWorkflow(ctx, recordWorkflow, in, dbos.WithWorkflowID(runID))
	return err
}

func awaitStart(ctx dbos.Context, req registry.WorkflowRequest, runID string) error {
	in, err := translateProbe(req)
	if err != nil {
		return err
	}
	_, err = dbos.RunWorkflow(ctx, awaitWorkflow, in, dbos.WithWorkflowID(runID))
	return err
}

type approvalReadInput struct {
	TenantID   string `json:"tenantId"`
	ApprovalID string `json:"approvalId"`
	Marker     string `json:"marker"`
}

// approvalReadWorkflow models the framework's approval-wait pattern:
// wake, then re-read the authoritative table inside a transaction
// step and record what it said.
func approvalReadWorkflow(ctx dbos.Context, in approvalReadInput) (string, error) {
	return dbos.RunAsTransaction(ctx, probeDS, func(ctx context.Context, tx dbos.Tx) (string, error) {
		if err := dbosexec.ArmTenant(ctx, tx, in.TenantID); err != nil {
			return "", err
		}
		status, decision, err := dbosexec.ReadApprovalDecision(ctx, tx, in.TenantID, in.ApprovalID)
		if err != nil {
			return "", err
		}
		_, err = tx.Exec(ctx, `INSERT INTO probe_markers(marker, tenant_id, stage)
			VALUES ($1, $2, $3)`, in.Marker, in.TenantID, status+":"+string(decision))
		return status, err
	})
}

func approvalReadRegister(env registry.WorkflowEnv) error {
	probeDS = env.Data
	dbos.RegisterWorkflow(env.Context, approvalReadWorkflow)
	return nil
}

// sleepWorkflow sleeps durably, then records completion. Kill tests
// interrupt the sleep; envelope tests run many concurrently.
func sleepWorkflow(ctx dbos.Context, in probeInput) (string, error) {
	if _, err := dbos.Sleep(ctx, time.Duration(in.SleepMs)*time.Millisecond); err != nil {
		return "", err
	}
	return dbos.RunAsTransaction(ctx, probeDS, func(ctx context.Context, tx dbos.Tx) (string, error) {
		if err := dbosexec.ArmTenant(ctx, tx, in.TenantID); err != nil {
			return "", err
		}
		_, err := tx.Exec(ctx, `INSERT INTO probe_markers(marker, tenant_id, stage)
			VALUES ($1, $2, 'slept')`, in.Marker, in.TenantID)
		return in.Marker, err
	})
}

func sleepRegister(env registry.WorkflowEnv) error {
	probeDS = env.Data
	dbos.RegisterWorkflow(env.Context, sleepWorkflow)
	return nil
}

func sleepStart(ctx dbos.Context, req registry.WorkflowRequest, runID string) error {
	var body struct {
		Marker  string `json:"marker"`
		SleepMs int    `json:"sleep_ms"`
	}
	if err := json.Unmarshal(req.Input, &body); err != nil {
		return err
	}
	in := probeInput{TenantID: req.TenantID, Marker: body.Marker, SleepMs: body.SleepMs}
	_, err := dbos.RunWorkflow(ctx, sleepWorkflow, in, dbos.WithWorkflowID(runID))
	return err
}

// suspendWorkflow records how a suspend-mode sleep resolves without
// Enterprise configuration. The error text (if any) lands in the
// marker stage for the edition test to inspect.
func suspendWorkflow(ctx dbos.Context, in probeInput) (string, error) {
	_, serr := dbos.Sleep(ctx, time.Second, dbos.WithSleepSuspend())
	msg := "no-error"
	if serr != nil {
		msg = serr.Error()
		if len(msg) > 120 {
			msg = msg[:120]
		}
	}
	return dbos.RunAsTransaction(ctx, probeDS, func(ctx context.Context, tx dbos.Tx) (string, error) {
		if err := dbosexec.ArmTenant(ctx, tx, in.TenantID); err != nil {
			return "", err
		}
		_, err := tx.Exec(ctx, `INSERT INTO probe_markers(marker, tenant_id, stage)
			VALUES ($1, $2, $3)`, in.Marker, in.TenantID, "suspend:"+msg)
		return msg, err
	})
}

func suspendRegister(env registry.WorkflowEnv) error {
	probeDS = env.Data
	dbos.RegisterWorkflow(env.Context, suspendWorkflow)
	return nil
}

func suspendStart(ctx dbos.Context, req registry.WorkflowRequest, runID string) error {
	in, err := translateProbe(req)
	if err != nil {
		return err
	}
	_, err = dbos.RunWorkflow(ctx, suspendWorkflow, in, dbos.WithWorkflowID(runID))
	return err
}

func approvalReadStart(ctx dbos.Context, req registry.WorkflowRequest, runID string) error {
	var body struct {
		ApprovalID string `json:"approvalId"`
		Marker     string `json:"marker"`
	}
	if err := json.Unmarshal(req.Input, &body); err != nil {
		return err
	}
	in := approvalReadInput{TenantID: req.TenantID, ApprovalID: body.ApprovalID, Marker: body.Marker}
	_, err := dbos.RunWorkflow(ctx, approvalReadWorkflow, in, dbos.WithWorkflowID(runID))
	return err
}

const (
	probeID        = "dbos-probe"
	probeIDB       = "dbos-probe-b"
	probeVer       = "1.0.0"
	toolRecord     = "probe.record"
	toolAwait      = "probe.await"
	toolApprRead   = "probe.approval_read"
	toolSleep      = "probe.sleep"
	toolSuspend    = "probe.suspend_check"
	testIntentHash = "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
)

// probeDSB is family B's datasource. Production miniapps keep one
// datasource global per package; the two-test-family split models
// that with two globals.
var probeDSB *dbos.DataSource

func recordWorkflowB(ctx dbos.Context, in probeInput) (string, error) {
	return dbos.RunAsTransaction(ctx, probeDSB, func(ctx context.Context, tx dbos.Tx) (string, error) {
		if err := dbosexec.ArmTenant(ctx, tx, in.TenantID); err != nil {
			return "", err
		}
		_, err := tx.Exec(ctx, `INSERT INTO probe_markers(marker, tenant_id, stage)
			VALUES ($1, $2, 'done-b')`, in.Marker, in.TenantID)
		return in.Marker, err
	})
}

func awaitWorkflowB(ctx dbos.Context, in probeInput) (string, error) {
	payload, err := dbos.Recv[string](ctx, "probe.go", 2*time.Minute)
	if err != nil {
		return "", err
	}
	return dbos.RunAsTransaction(ctx, probeDSB, func(ctx context.Context, tx dbos.Tx) (string, error) {
		if err := dbosexec.ArmTenant(ctx, tx, in.TenantID); err != nil {
			return "", err
		}
		_, err := tx.Exec(ctx, `INSERT INTO probe_markers(marker, tenant_id, stage)
			VALUES ($1, $2, $3)`, in.Marker, in.TenantID, "signaled-b:"+payload)
		return payload, err
	})
}

func recordRegisterB(env registry.WorkflowEnv) error {
	probeDSB = env.Data
	dbos.RegisterWorkflow(env.Context, recordWorkflowB)
	return nil
}

func awaitRegisterB(env registry.WorkflowEnv) error {
	probeDSB = env.Data
	dbos.RegisterWorkflow(env.Context, awaitWorkflowB)
	return nil
}

func recordStartB(ctx dbos.Context, req registry.WorkflowRequest, runID string) error {
	in, err := translateProbe(req)
	if err != nil {
		return err
	}
	_, err = dbos.RunWorkflow(ctx, recordWorkflowB, in, dbos.WithWorkflowID(runID))
	return err
}

func awaitStartB(ctx dbos.Context, req registry.WorkflowRequest, runID string) error {
	in, err := translateProbe(req)
	if err != nil {
		return err
	}
	_, err = dbos.RunWorkflow(ctx, awaitWorkflowB, in, dbos.WithWorkflowID(runID))
	return err
}

func probeManifestB() *contract.MiniappManifest {
	mf := probeManifest()
	mf.ID = contract.Name(probeIDB)
	kept := mf.Tools[:0]
	for _, t := range mf.Tools {
		name := string(t.Name)
		if name == toolRecord || name == toolAwait {
			kept = append(kept, t)
		} else {
			delete(mf.Deployment.Tools, name)
		}
	}
	mf.Tools = kept
	return mf
}

func probeImplB(version string) *registry.Implementation {
	return &registry.Implementation{Build: "test", Workflows: map[string]registry.WorkflowDef{
		toolRecord: {Name: toolRecord, Version: version, Register: recordRegisterB, Start: recordStartB},
		toolAwait:  {Name: toolAwait, Version: version, Register: awaitRegisterB, Start: awaitStartB},
	}}
}

func probeManifest() *contract.MiniappManifest {
	sum := sha256.Sum256([]byte(`{"name":"default","placement":"machine","substrate":"dbos"}`))
	profile := contract.Digest("sha256:" + hex.EncodeToString(sum[:]))
	schema := json.RawMessage(`{"type":"object"}`)
	tool := func(name, desc string) contract.ToolManifest {
		return contract.ToolManifest{
			Name: contract.ToolName(name), Description: desc,
			Class: contract.ClassD, Substrate: contract.SubstrateDBOS,
			InputSchema: schema, OutputSchema: schema,
			NoEntityWhy: "Substrate probe: writes one test-owned marker row.",
		}
	}
	dep := contract.ToolDeployment{Placement: contract.PlacementMachine,
		Profile: contract.ProfileRef{Name: "default", Digest: profile}}
	return &contract.MiniappManifest{
		ID: contract.Name(probeID), Version: contract.Version(probeVer),
		Description: "DBOS substrate probe: no business semantics.",
		Tools: []contract.ToolManifest{
			tool(toolRecord, "Record a marker."),
			tool(toolAwait, "Await a signal, then record."),
			tool(toolApprRead, "Re-read an approval row."),
			tool(toolSleep, "Sleep durably, then record."),
			tool(toolSuspend, "Probe suspend-mode sleep."),
		},
		Deployment: contract.DeploymentSection{Tools: map[string]contract.ToolDeployment{
			toolRecord: dep, toolAwait: dep, toolApprRead: dep, toolSleep: dep, toolSuspend: dep,
		}},
	}
}

func probeImpl(version string) *registry.Implementation {
	return &registry.Implementation{Build: "test", Workflows: map[string]registry.WorkflowDef{
		toolRecord:   {Name: toolRecord, Version: version, Register: recordRegister, Start: recordStart},
		toolAwait:    {Name: toolAwait, Version: version, Register: awaitRegister, Start: awaitStart},
		toolApprRead: {Name: toolApprRead, Version: version, Register: approvalReadRegister, Start: approvalReadStart},
		toolSleep:    {Name: toolSleep, Version: version, Register: sleepRegister, Start: sleepStart},
		toolSuspend:  {Name: toolSuspend, Version: version, Register: suspendRegister, Start: suspendStart},
	}}
}

type logCapture struct {
	mu  sync.Mutex
	buf strings.Builder
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

type harness struct {
	ex   *dbosexec.Executor
	pool *pgxpool.Pool
	logs *logCapture
}

func newPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DBOS_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DBOS_DATABASE_URL unset")
	}
	testkit.MigrateUp(t, dsn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS probe_markers (marker text, tenant_id text, stage text)`); err != nil {
		t.Fatal(err)
	}
	return pool
}

func registerProbeFamily(t *testing.T, reg *registry.Registry, id, version string, mf *contract.MiniappManifest, impl *registry.Implementation) {
	t.Helper()
	digest, err := contract.ProfileDigest(mf)
	if err != nil {
		t.Fatal(err)
	}
	env := contract.ReleaseEnvelope{Manifest: digest, Implementation: "test",
		Build: "test", SourceRevision: "test", Artifact: digest}
	if err := reg.Register(mf, impl, env); err != nil {
		t.Fatalf("register %s: %v", id, err)
	}
}

func finishHarness(t *testing.T, pool *pgxpool.Pool, reg *registry.Registry) harness {
	t.Helper()
	logs := &logCapture{}
	logger := slog.New(slog.NewTextHandler(logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	ex, err := dbosexec.New(dbosexec.Config{Pool: pool, Registry: reg, Logger: logger})
	if err != nil {
		t.Fatal(err)
	}
	// Shutdown before pool close: cleanups run LIFO, so the pool
	// close is registered first and runs last. Reversing this order
	// deadlocks: pool close waits for connections DBOS workers hold.
	t.Cleanup(pool.Close)
	t.Cleanup(func() {
		if err := ex.Shutdown(30 * time.Second); err != nil {
			t.Errorf("shutdown: %v", err)
		}
	})
	return harness{ex: ex, pool: pool, logs: logs}
}

func newHarness(t *testing.T, version string) harness {
	t.Helper()
	pool := newPool(t)
	reg := registry.New()
	registerProbeFamily(t, reg, probeID, version, probeManifest(), probeImpl(version))
	return finishHarness(t, pool, reg)
}

func (h harness) pollStatus(t *testing.T, runID string, want contract.RunStatus) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Minute)
	for {
		st, err := h.ex.Status(context.Background(), runID)
		if err != nil {
			t.Fatalf("status: %v", err)
		}
		if st == want {
			return
		}
		if st == contract.RunSucceeded || st == contract.RunFailed || st == contract.RunCancelled {
			t.Fatalf("run settled at %s, want %s", st, want)
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s (last %s)", want, st)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func (h harness) markerStage(t *testing.T, marker string) (string, string) {
	t.Helper()
	var tenant, stage string
	err := h.pool.QueryRow(context.Background(),
		`SELECT tenant_id, stage FROM probe_markers WHERE marker = $1`, marker).Scan(&tenant, &stage)
	if err != nil {
		t.Fatalf("marker %q: %v", marker, err)
	}
	return tenant, stage
}

func submitReq(tool, marker string) executor.Request {
	return submitReqMini(probeID, tool, marker)
}

func submitReqMini(miniapp, tool, marker string) executor.Request {
	return executor.Request{
		InvocationID: "inv-" + marker, TenantID: "tenant-exec",
		Principal: contract.Caller{AgentID: "agent-e", Authority: contract.Authority{Kind: contract.AuthorityUser, ID: "agent-e", TenantID: "tenant-exec"}},
		MiniappID: miniapp, Version: probeVer, Tool: tool,
		Input:          json.RawMessage(fmt.Sprintf(`{"marker":%q}`, marker)),
		IdempotencyKey: "key-" + marker, TrustMode: contract.TrustAutonomous,
		RunID: "run-" + marker,
	}
}

func TestSubmitStatusRoundTrip(t *testing.T) {
	h := newHarness(t, "1")
	marker := fmt.Sprintf("rt-%d", time.Now().UnixNano())
	handle, err := h.ex.Submit(context.Background(), submitReq(toolRecord, marker))
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if handle.Status != contract.RunRunning || handle.RunID == "" {
		t.Fatalf("handle = %+v, want running with run ID", handle)
	}
	h.pollStatus(t, handle.RunID, contract.RunSucceeded)
	tenant, stage := h.markerStage(t, marker)
	if tenant != "tenant-exec" || stage != "done" {
		t.Fatalf("marker = (%q, %q), want (tenant-exec, done)", tenant, stage)
	}
	if !strings.Contains(h.logs.String(), "shares the system database") {
		t.Fatal("shared-handle single-tx configuration was not selected (detection log absent)")
	}
}

// TestSubmitIdempotent proves Submit attaches on a repeated run ID:
// one workflow executes (one marker row) and both handles name it.
func TestSubmitIdempotent(t *testing.T) {
	h := newHarness(t, "1")
	marker := fmt.Sprintf("idem-%d", time.Now().UnixNano())
	first, err := h.ex.Submit(context.Background(), submitReq(toolRecord, marker))
	if err != nil {
		t.Fatalf("first submit: %v", err)
	}
	second, err := h.ex.Submit(context.Background(), submitReq(toolRecord, marker))
	if err != nil {
		t.Fatalf("second submit: %v", err)
	}
	if second.RunID != first.RunID {
		t.Fatalf("second run %s != first %s", second.RunID, first.RunID)
	}
	h.pollStatus(t, first.RunID, contract.RunSucceeded)
	var n int
	err = h.pool.QueryRow(context.Background(),
		`SELECT count(*) FROM probe_markers WHERE marker = $1`, marker).Scan(&n)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("marker rows = %d, want 1 (repeated submit must not re-execute)", n)
	}
}

func TestSignalReleases(t *testing.T) {
	h := newHarness(t, "1")
	marker := fmt.Sprintf("sig-%d", time.Now().UnixNano())
	handle, err := h.ex.Submit(context.Background(), submitReq(toolAwait, marker))
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	time.Sleep(2 * time.Second)
	sig := contract.Signal{Type: "probe.go", Payload: json.RawMessage(`{"n":1}`)}
	if err := h.ex.Signal(context.Background(), handle.RunID, sig); err != nil {
		t.Fatalf("signal: %v", err)
	}
	h.pollStatus(t, handle.RunID, contract.RunSucceeded)
	_, stage := h.markerStage(t, marker)
	if stage != `signaled:{"n":1}` {
		t.Fatalf("stage = %q, want signaled payload", stage)
	}
}

// TestApprovalReread proves the workflow side of the decision bridge:
// a step re-reads the authoritative approvals row and observes the
// committed decision.
func TestApprovalReread(t *testing.T) {
	h := newHarness(t, "1")
	ctx := context.Background()
	apprSvc := approvals.New(h.pool)
	approvalID, err := apprSvc.Submit(ctx, approvals.Request{
		TenantID: "tenant-exec", MiniappID: probeID, Version: probeVer,
		InvocationID: "inv-appr-read", Operation: toolApprRead,
		IntentHash: contract.Digest(testIntentHash), RequestedBy: "agent-e",
	})
	if err != nil {
		t.Fatalf("submit approval: %v", err)
	}
	decider := approvals.Decider{Caller: contract.Caller{AgentID: "human-op",
		Authority: contract.Authority{Kind: contract.AuthorityUser, ID: "human-op", TenantID: "tenant-exec"}}}
	if err := apprSvc.Decide(ctx, "tenant-exec", approvalID, decider, contract.DecisionApprove, contract.Digest(testIntentHash)); err != nil {
		t.Fatalf("decide: %v", err)
	}
	marker := fmt.Sprintf("appr-%d", time.Now().UnixNano())
	req := submitReq(toolApprRead, marker)
	req.Input = json.RawMessage(fmt.Sprintf(`{"approvalId":%q,"marker":%q}`, approvalID, marker))
	handle, err := h.ex.Submit(ctx, req)
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	h.pollStatus(t, handle.RunID, contract.RunSucceeded)
	_, stage := h.markerStage(t, marker)
	if stage != "decided:approve" {
		t.Fatalf("reread stage = %q, want decided:approve", stage)
	}
}

func TestCancelStops(t *testing.T) {
	h := newHarness(t, "1")
	marker := fmt.Sprintf("cancel-%d", time.Now().UnixNano())
	handle, err := h.ex.Submit(context.Background(), submitReq(toolAwait, marker))
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	time.Sleep(2 * time.Second)
	if err := h.ex.Cancel(context.Background(), handle.RunID, "test done"); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	h.pollStatus(t, handle.RunID, contract.RunCancelled)
}

func TestConfigRejects(t *testing.T) {
	reg := registry.New()
	for name, cfg := range map[string]dbosexec.Config{
		"nil pool": {Registry: reg},
		"nil reg":  {Pool: &pgxpool.Pool{}},
	} {
		if _, err := dbosexec.New(cfg); err == nil {
			t.Fatalf("%s accepted", name)
		}
	}
}

// TestVersionOverlap proves same-image overlap: a v2 worker starts
// while a v1 run waits, reports (not adopts) the foreign pending
// run, serves its own version, and the v1 run drains on v1 exactly
// once.
func TestVersionOverlap(t *testing.T) {
	h1 := newHarness(t, "1")
	ctx := context.Background()
	stamp := fmt.Sprintf("%d", time.Now().UnixNano())
	m1 := "overlap-v1-" + stamp
	w1, err := h1.ex.Submit(ctx, submitReq(toolAwait, m1))
	if err != nil {
		t.Fatalf("v1 submit: %v", err)
	}
	time.Sleep(2 * time.Second)

	h2 := newHarness(t, "2")
	if got := h2.logs.String(); !strings.Contains(got, "another version") {
		t.Fatalf("v2 startup reported no foreign pending run (logs: %q)", got)
	}
	m2 := "overlap-v2-" + stamp
	w2, err := h2.ex.Submit(ctx, submitReq(toolAwait, m2))
	if err != nil {
		t.Fatalf("v2 submit: %v", err)
	}
	time.Sleep(2 * time.Second)
	sig := contract.Signal{Type: "probe.go", Payload: json.RawMessage(`{"n":1}`)}
	if err := h1.ex.Signal(ctx, w1.RunID, sig); err != nil {
		t.Fatalf("signal v1: %v", err)
	}
	if err := h2.ex.Signal(ctx, w2.RunID, sig); err != nil {
		t.Fatalf("signal v2: %v", err)
	}
	h1.pollStatus(t, w1.RunID, contract.RunSucceeded)
	h2.pollStatus(t, w2.RunID, contract.RunSucceeded)
	_, stage := h1.markerStage(t, m1)
	if stage != `signaled:{"n":1}` {
		t.Fatalf("v1 marker = %q, want exactly-once signaled payload", stage)
	}
	var n int
	if err := h2.pool.QueryRow(ctx, `SELECT count(*) FROM probe_markers WHERE marker = $1`, m1).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("v1 marker rows = %d, want 1 (v2 must never touch v1 runs)", n)
	}
}

// TestRestartRecovery proves a pending run resumes on a restarted
// worker of the same version and completes once signalled.
func TestRestartRecovery(t *testing.T) {
	h1 := newHarness(t, "1")
	ctx := context.Background()
	marker := fmt.Sprintf("restart-%d", time.Now().UnixNano())
	w, err := h1.ex.Submit(ctx, submitReq(toolAwait, marker))
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	time.Sleep(2 * time.Second)
	if err := h1.ex.Shutdown(30 * time.Second); err != nil {
		t.Fatalf("shutdown: %v", err)
	}

	h2 := newHarness(t, "1")
	sig := contract.Signal{Type: "probe.go", Payload: json.RawMessage(`{"n":2}`)}
	if err := h2.ex.Signal(ctx, w.RunID, sig); err != nil {
		t.Fatalf("signal after restart: %v", err)
	}
	h2.pollStatus(t, w.RunID, contract.RunSucceeded)
	_, stage := h2.markerStage(t, marker)
	if stage != `signaled:{"n":2}` {
		t.Fatalf("marker = %q, want post-restart payload", stage)
	}
}

// TestTwoFamilyRouting proves one executor serves two families
// independently: submits route by miniapp, and signal/cancel reach
// the owning family's run.
func TestTwoFamilyRouting(t *testing.T) {
	pool := newPool(t)
	reg := registry.New()
	registerProbeFamily(t, reg, probeID, "1", probeManifest(), probeImpl("1"))
	registerProbeFamily(t, reg, probeIDB, "1", probeManifestB(), probeImplB("1"))
	h := finishHarness(t, pool, reg)
	ctx := context.Background()
	stamp := fmt.Sprintf("%d", time.Now().UnixNano())

	ra, err := h.ex.Submit(ctx, submitReqMini(probeID, toolRecord, "fam-a-"+stamp))
	if err != nil {
		t.Fatalf("submit A: %v", err)
	}
	rb, err := h.ex.Submit(ctx, submitReqMini(probeIDB, toolRecord, "fam-b-"+stamp))
	if err != nil {
		t.Fatalf("submit B: %v", err)
	}
	h.pollStatus(t, ra.RunID, contract.RunSucceeded)
	h.pollStatus(t, rb.RunID, contract.RunSucceeded)
	if _, stage := h.markerStage(t, "fam-a-"+stamp); stage != "done" {
		t.Fatalf("family A stage = %q", stage)
	}
	if _, stage := h.markerStage(t, "fam-b-"+stamp); stage != "done-b" {
		t.Fatalf("family B stage = %q", stage)
	}

	wa, err := h.ex.Submit(ctx, submitReqMini(probeID, toolAwait, "fam-aw-"+stamp))
	if err != nil {
		t.Fatalf("submit A await: %v", err)
	}
	wb, err := h.ex.Submit(ctx, submitReqMini(probeIDB, toolAwait, "fam-bw-"+stamp))
	if err != nil {
		t.Fatalf("submit B await: %v", err)
	}
	time.Sleep(2 * time.Second)
	sig := contract.Signal{Type: "probe.go", Payload: json.RawMessage(`{"n":3}`)}
	if err := h.ex.Signal(ctx, wa.RunID, sig); err != nil {
		t.Fatalf("signal A: %v", err)
	}
	if err := h.ex.Cancel(ctx, wb.RunID, "test"); err != nil {
		t.Fatalf("cancel B: %v", err)
	}
	h.pollStatus(t, wa.RunID, contract.RunSucceeded)
	h.pollStatus(t, wb.RunID, contract.RunCancelled)
	if _, stage := h.markerStage(t, "fam-aw-"+stamp); stage != `signaled:{"n":3}` {
		t.Fatalf("family A await stage = %q", stage)
	}
}

const sleepWorkerEnv = "MINIAPP_SLEEP_WORKER"

func sleepReq(marker string, sleepMs int) executor.Request {
	req := submitReq(toolSleep, marker)
	req.Input = json.RawMessage(fmt.Sprintf(`{"marker":%q,"sleep_ms":%d}`, marker, sleepMs))
	return req
}

// TestSleepWorker submits one long sleep, then stays alive running
// it until the parent kills the process mid-sleep.
func TestSleepWorker(t *testing.T) {
	if !testkit.WorkerMode(sleepWorkerEnv) {
		t.Skip("crash worker entrypoint (parent-spawned only)")
	}
	dir := os.Getenv("MINIAPP_SLEEP_DIR")
	marker := os.Getenv("MINIAPP_SLEEP_MARKER")
	pool := newPool(t)
	reg := registry.New()
	registerProbeFamily(t, reg, probeID, "1", probeManifest(), probeImpl("1"))
	h := finishHarness(t, pool, reg)
	handle, err := h.ex.Submit(context.Background(), sleepReq(marker, 15000))
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "runid"), []byte(handle.RunID), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "submitted"), []byte("go"), 0o600); err != nil {
		t.Fatal(err)
	}
	select {}
}

// TestSleepAcrossRestart kills a worker mid-sleep, relaunches, and
// proves the wait resumes and completes exactly once.
func TestSleepAcrossRestart(t *testing.T) {
	testkit.RequireDB(t, "TEST_DBOS_DATABASE_URL")
	dir := t.TempDir()
	marker := fmt.Sprintf("sleepkill-%d", time.Now().UnixNano())
	w := testkit.SpawnWorker(t, "TestSleepWorker", map[string]string{
		sleepWorkerEnv: "1", "MINIAPP_SLEEP_DIR": dir, "MINIAPP_SLEEP_MARKER": marker,
	})
	testkit.WaitForFile(t, w, filepath.Join(dir, "submitted"), 2*time.Minute)
	time.Sleep(2 * time.Second)
	killAt := time.Now()
	testkit.Kill9(t, w)

	h := newHarness(t, "1")
	handle, err := h.ex.Submit(context.Background(), sleepReq(marker, 15000))
	if err != nil {
		t.Fatalf("re-submit after kill: %v", err)
	}
	h.pollStatus(t, handle.RunID, contract.RunSucceeded)
	if time.Since(killAt) > 90*time.Second {
		t.Fatal("resumed sleep took too long; wake may not have fired on recovery")
	}
	var n int
	if err := h.pool.QueryRow(context.Background(),
		`SELECT count(*) FROM probe_markers WHERE marker = $1 AND stage = 'slept'`, marker).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("slept rows = %d, want exactly 1", n)
	}
}

// TestSignalMatrix documents signal timing behavior: a signal sent
// before the wait starts is buffered; duplicates do not double-apply;
// a signal to a completed run is accepted harmlessly with no state
// change and no duplicate effect.
func TestSignalMatrix(t *testing.T) {
	h := newHarness(t, "1")
	ctx := context.Background()
	stamp := fmt.Sprintf("%d", time.Now().UnixNano())

	early, err := h.ex.Submit(ctx, submitReq(toolAwait, "sig-early-"+stamp))
	if err != nil {
		t.Fatal(err)
	}
	if err := h.ex.Signal(ctx, early.RunID, contract.Signal{Type: "probe.go", Payload: json.RawMessage(`{"n":"early"}`)}); err != nil {
		t.Fatalf("pre-wait signal: %v", err)
	}
	h.pollStatus(t, early.RunID, contract.RunSucceeded)
	if _, stage := h.markerStage(t, "sig-early-"+stamp); stage != `signaled:{"n":"early"}` {
		t.Fatalf("early-signal stage = %q", stage)
	}

	dup, err := h.ex.Submit(ctx, submitReq(toolAwait, "sig-dup-"+stamp))
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Second)
	sig := contract.Signal{Type: "probe.go", Payload: json.RawMessage(`{"n":"dup"}`)}
	if err := h.ex.Signal(ctx, dup.RunID, sig); err != nil {
		t.Fatal(err)
	}
	if err := h.ex.Signal(ctx, dup.RunID, sig); err != nil {
		t.Fatalf("duplicate signal: %v", err)
	}
	h.pollStatus(t, dup.RunID, contract.RunSucceeded)
	var n int
	if err := h.pool.QueryRow(ctx, `SELECT count(*) FROM probe_markers WHERE marker = $1`, "sig-dup-"+stamp).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("duplicate-signal marker rows = %d, want 1", n)
	}

	done, err := h.ex.Submit(ctx, submitReq(toolRecord, "sig-done-"+stamp))
	if err != nil {
		t.Fatal(err)
	}
	h.pollStatus(t, done.RunID, contract.RunSucceeded)
	if err := h.ex.Signal(ctx, done.RunID, sig); err != nil {
		t.Fatalf("post-completion signal rejected: %v (sends buffer without state checks)", err)
	}
	h.pollStatus(t, done.RunID, contract.RunSucceeded)
	if err := h.pool.QueryRow(ctx, `SELECT count(*) FROM probe_markers WHERE marker = $1`, "sig-done-"+stamp).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("post-completion marker rows = %d, want 1", n)
	}
}

// TestSleepSuspendRequiresEnterprise proves the edition boundary:
// without SuspendMode configuration, suspend-mode sleep is refused
// and the workflow observes the Enterprise-gating error.
func TestSleepSuspendRequiresEnterprise(t *testing.T) {
	h := newHarness(t, "1")
	marker := fmt.Sprintf("suspend-%d", time.Now().UnixNano())
	w, err := h.ex.Submit(context.Background(), submitReq(toolSuspend, marker))
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	h.pollStatus(t, w.RunID, contract.RunSucceeded)
	_, stage := h.markerStage(t, marker)
	if !strings.Contains(stage, "Enterprise") {
		t.Fatalf("suspend stage = %q, want the Enterprise-gating error recorded", stage)
	}
}

// TestSleeperEnvelope measures (not asserts thresholds on) twenty
// concurrent durable sleepers: wall time, memory delta, and pool
// use. Numbers land in the evidence entry; the test proves joint
// completion.
func TestSleeperEnvelope(t *testing.T) {
	h := newHarness(t, "1")
	ctx := context.Background()
	const sleepers = 20
	var before runtime.MemStats
	runtime.ReadMemStats(&before)
	start := time.Now()
	type done struct {
		runID string
		err   error
	}
	out := make(chan done, sleepers)
	for i := 0; i < sleepers; i++ {
		go func(i int) {
			marker := fmt.Sprintf("env-%d-%d", time.Now().UnixNano(), i)
			w, err := h.ex.Submit(ctx, sleepReq(marker, 2000))
			if err != nil {
				out <- done{"", err}
				return
			}
			h.pollStatus(t, w.RunID, contract.RunSucceeded)
			out <- done{w.RunID, nil}
		}(i)
	}
	for i := 0; i < sleepers; i++ {
		if d := <-out; d.err != nil {
			t.Fatalf("sleeper: %v", d.err)
		}
	}
	elapsed := time.Since(start)
	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	stat := h.pool.Stat()
	t.Logf("sleepers=%d elapsed=%s alloc=%dKB->%dKB goroutines(approx)=%d conns_acquired=%d",
		sleepers, elapsed.Round(time.Millisecond),
		before.Alloc/1024, after.Alloc/1024, runtime.NumGoroutine(), stat.AcquireCount())
	if elapsed > 90*time.Second {
		t.Fatalf("envelope took %s; sleepers did not progress jointly", elapsed)
	}
}
