package runtime_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dbos-inc/dbos-transact-golang/dbos"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/PithomLabs/miniapp/approvals"
	"github.com/PithomLabs/miniapp/audit"
	"github.com/PithomLabs/miniapp/auth"
	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/effects"
	"github.com/PithomLabs/miniapp/executor"
	"github.com/PithomLabs/miniapp/executor/classa"
	"github.com/PithomLabs/miniapp/internal/store"
	"github.com/PithomLabs/miniapp/miniapps/crm-read"
	"github.com/PithomLabs/miniapp/registry"
	"github.com/PithomLabs/miniapp/runtime"
	"github.com/PithomLabs/miniapp/testkit"
	"github.com/PithomLabs/miniapp/testkit/mockcap"
)

const (
	credAgentA     = "key-agent-a"
	credAgentA2    = "key-agent-a2"
	credAgentB     = "key-agent-b"
	credDecide     = "key-decider"
	tenantA        = "tenant-a"
	tenantB        = "tenant-b"
	gatedID        = "crm-gated"
	probeID        = "test-probe"
	probeTool      = "probe.noop"
	probeGatedID   = "test-probe-gated"
	probeGatedTool = "probe.gated"
)

func testPolicy() auth.Policy {
	binding := func(agent, tenant string, roles ...string) auth.Binding {
		return auth.Binding{AgentID: agent, Authority: contract.Authority{
			Kind: contract.AuthorityUser, ID: agent, TenantID: tenant,
		}, Roles: roles}
	}
	return auth.Policy{
		Keys: map[string]auth.Binding{
			credAgentA:  binding("agent-a", tenantA),
			credAgentA2: binding("agent-a2", tenantA),
			credAgentB:  binding("agent-b", tenantB),
			credDecide:  binding("human-op", tenantA, "tenant-ops"),
		},
		Grants: map[string]map[string]contract.TrustMode{
			tenantA: {
				crmread.ID:     contract.TrustAutonomous,
				gatedID:        contract.TrustConfirm,
				probeID:        contract.TrustAutonomous,
				probeGatedID:   contract.TrustConfirm,
				"entity-probe": contract.TrustAutonomous,
			},
			tenantB: {crmread.ID: contract.TrustAutonomous, "entity-probe": contract.TrustAutonomous},
		},
	}
}

// fakeAsync is a deterministic stand-in for the DBOS executor. It
// proves the runtime's Class D dispatch and lifecycle paths; durable
// execution itself is covered by the executor's own tests.
type signalRec struct {
	run     string
	typ     string
	payload string
}

type fakeAsync struct {
	mu             sync.Mutex
	status         map[string]contract.RunStatus
	signal         []signalRec
	submits        []executor.Request
	failNextSubmit bool
	failNextCancel bool
	failNextSignal bool
	failNextStatus bool
	submitCalls    int
	cancelCalls    int
}

func (f *fakeAsync) Submit(_ context.Context, req executor.Request) (contract.RunHandle, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.submitCalls++
	if f.failNextSubmit {
		f.failNextSubmit = false
		return contract.RunHandle{}, fmt.Errorf("injected submit failure")
	}
	f.submits = append(f.submits, req)
	id := req.RunID
	if id == "" {
		id = "run-" + uuid.NewString()
	}
	if st, ok := f.status[id]; ok {
		return contract.RunHandle{RunID: id, MiniappID: req.MiniappID,
			Version: contract.Version(req.Version), Status: st}, nil
	}
	f.status[id] = contract.RunRunning
	return contract.RunHandle{RunID: id, MiniappID: req.MiniappID,
		Version: contract.Version(req.Version), Status: contract.RunRunning}, nil
}

func (f *fakeAsync) Status(_ context.Context, runID string) (contract.RunStatus, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failNextStatus {
		f.failNextStatus = false
		return "", fmt.Errorf("injected status failure")
	}
	st, ok := f.status[runID]
	if !ok {
		return "", fmt.Errorf("unknown run %s: %w", runID, executor.ErrUnknownRun)
	}
	return st, nil
}

func (f *fakeAsync) Signal(_ context.Context, runID string, sig contract.Signal) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failNextSignal {
		f.failNextSignal = false
		return fmt.Errorf("injected signal failure")
	}
	if _, ok := f.status[runID]; !ok {
		return fmt.Errorf("unknown run %s: %w", runID, executor.ErrUnknownRun)
	}
	f.signal = append(f.signal, signalRec{run: runID, typ: sig.Type, payload: string(sig.Payload)})
	return nil
}

func (f *fakeAsync) Cancel(_ context.Context, runID string, _ string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.cancelCalls++
	if f.failNextCancel {
		f.failNextCancel = false
		return fmt.Errorf("injected cancel failure")
	}
	if _, ok := f.status[runID]; !ok {
		return fmt.Errorf("unknown run %s: %w", runID, executor.ErrUnknownRun)
	}
	f.status[runID] = contract.RunCancelled
	return nil
}

type fixture struct {
	svc      *runtime.Service
	aud      *audit.Service
	async    *fakeAsync
	approver *approvals.Service
	effects  *effects.Service
	policy   auth.Policy
	reg      *registry.Registry
	captured *captureStore
}

type capturedCall struct {
	tenant    string
	agent     string
	authority contract.Authority
	entity    string
}

type captureStore struct {
	mu    sync.Mutex
	calls map[string]capturedCall
}

func (c *captureStore) handler() registry.ToolFunc {
	return func(ctx context.Context, call registry.ToolCall) (json.RawMessage, error) {
		c.mu.Lock()
		defer c.mu.Unlock()
		c.calls[call.InvocationID] = capturedCall{
			tenant: call.TenantID, agent: call.Principal.AgentID,
			authority: call.Principal.Authority, entity: call.EntityKey,
		}
		return json.RawMessage(`{"captured":true}`), nil
	}
}

func (c *captureStore) get(inv string) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.calls[inv].entity
}

func (c *captureStore) getCall(inv string) capturedCall {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.calls[inv]
}

func newFixture(t *testing.T, db *testkit.DB) fixture {
	t.Helper()
	reg := registry.New()
	if err := crmread.Register(reg); err != nil {
		t.Fatal(err)
	}
	registerGated(t, reg)
	registerProbe(t, reg)
	registerProbeWith(t, reg, probeGatedID, probeGatedTool, true)
	captured := &captureStore{calls: map[string]capturedCall{}}
	testkit.RegisterClassATool(t, reg, "entity-probe", "probe.capture", "subject_id", []string{"mock.send"}, captured.handler())
	async := &fakeAsync{status: map[string]contract.RunStatus{}}
	aud := audit.New(db.Pool, nil)
	approver := approvals.New(db.Pool)
	policy := testPolicy()
	effSvc := effects.New(effects.Config{Pool: db.Pool,
		Channels:  map[string]effects.Channel{"mock": mockcap.New(db.Pool)},
		Canceller: runtime.EffectCanceller(db.Pool),
		Policy:    effects.StrictPolicy{Pool: db.Pool, Registry: reg, Grants: policy},
	})
	svc, err := runtime.New(runtime.Config{
		Registry: reg,
		Policy:   policy,
		Pool:     db.Pool,
		Sync:     classa.New(db.Pool, reg),
		Async:    async,
		Approver: approver,
		Auditor:  aud,
		Effects:  effSvc,
	})
	if err != nil {
		t.Fatal(err)
	}
	return fixture{svc: svc, aud: aud, async: async, approver: approver, effects: effSvc, policy: policy, reg: reg, captured: captured}
}

// registerGated clones the crm-read manifest under a new ID with
// approval required, so the approval paths run against a real tool.
func registerGated(t *testing.T, reg *registry.Registry) {
	t.Helper()
	mf := crmread.Manifest()
	mf.ID = contract.Name(gatedID)
	mf.Tools[0].Approval = contract.ApprovalRequirement{Required: true, ApproverScope: "tenant-ops"}
	digest, err := contract.ProfileDigest(mf)
	if err != nil {
		t.Fatal(err)
	}
	env := contract.ReleaseEnvelope{
		Manifest: digest, Implementation: crmread.Build, Build: crmread.Build,
		SourceRevision: "test", Artifact: digest,
	}
	if err := reg.Register(mf, crmread.Implementation(), env); err != nil {
		t.Fatal(err)
	}
}

// registerProbe admits a minimal Class D tool. It carries no business
// semantics; it exists only to exercise runtime lifecycle dispatch.
// (ADR-001: this is not lead-followup.)
func registerProbe(t *testing.T, reg *registry.Registry) {
	t.Helper()
	registerProbeWith(t, reg, probeID, probeTool, false)
}

func registerProbeWith(t *testing.T, reg *registry.Registry, id, tool string, approval bool) {
	t.Helper()
	sum := sha256.Sum256([]byte(`{"name":"default","placement":"machine","substrate":"dbos"}`))
	profile := contract.Digest("sha256:" + hex.EncodeToString(sum[:]))
	schema := json.RawMessage(`{"type":"object"}`)
	tm := contract.ToolManifest{
		Name: contract.ToolName(tool), Description: "No-op lifecycle probe.",
		Class: contract.ClassD, Substrate: contract.SubstrateDBOS,
		InputSchema: schema, OutputSchema: schema,
		NoEntityWhy: "Probe: no effects.",
	}
	if approval {
		tm.Approval = contract.ApprovalRequirement{Required: true, ApproverScope: "tenant-ops", Expiry: "1h"}
	}
	mf := &contract.MiniappManifest{
		ID: contract.Name(id), Version: contract.Version("1.0.0"),
		Description: "Lifecycle probe: no business semantics.",
		Tools:       []contract.ToolManifest{tm},
		Deployment: contract.DeploymentSection{Tools: map[string]contract.ToolDeployment{
			tool: {Placement: contract.PlacementMachine,
				Profile: contract.ProfileRef{Name: "default", Digest: profile}},
		}},
	}
	if err := mf.Validate(); err != nil {
		t.Fatalf("probe manifest: %v", err)
	}
	digest, err := contract.ProfileDigest(mf)
	if err != nil {
		t.Fatal(err)
	}
	impl := &registry.Implementation{Build: "test", Workflows: map[string]registry.WorkflowDef{
		tool: {Name: tool, Version: "1",
			Register: func(env registry.WorkflowEnv) error { return nil },
			Start: func(ctx dbos.Context, req registry.WorkflowRequest, runID string) error {
				return nil
			}},
	}}
	env := contract.ReleaseEnvelope{Manifest: digest, Implementation: "test",
		Build: "test", SourceRevision: "test", Artifact: digest}
	if err := reg.Register(mf, impl, env); err != nil {
		t.Fatal(err)
	}
}

func newDB(t *testing.T) *testkit.DB {
	t.Helper()
	db := testkit.NewDB(t)
	mockcap.EnsureLog(t, db.Pool)
	db.Truncate(t, "app.invocations", "app.approvals", "app.audit_events", "app.crm_contacts", "app.effect_records", "app.kill_switches", "mock_provider_log")
	return db
}

func seedContact(t *testing.T, db *testkit.DB, tenant, id, name, email string) {
	t.Helper()
	_, err := store.WithTenant(context.Background(), db.Pool, tenant, func(tx pgx.Tx) (bool, error) {
		_, err := tx.Exec(context.Background(), `INSERT INTO app.crm_contacts (id, tenant_id, name, email)
			VALUES ($1, $2, $3, $4)`, id, tenant, name, email)
		return true, err
	})
	if err != nil {
		t.Fatal(err)
	}
}

func invoke(svc *runtime.Service, miniapp, tool, input, key, cred string, trust contract.TrustMode) (runtime.InvokeResult, error) {
	return svc.Invoke(context.Background(), runtime.InvokeRequest{
		MiniappID: miniapp, Version: "1.0.0", Tool: tool,
		Input: json.RawMessage(input), IdempotencyKey: key,
		Credential: cred, RequestedTrust: trust,
	})
}

func TestInvokeClassAHappyPath(t *testing.T) {
	db := newDB(t)
	fx := newFixture(t, db)
	seedContact(t, db, tenantA, "a1", "Ada Lovelace", "ada@example.com")

	out, err := invoke(fx.svc, crmread.ID, crmread.ToolSearch, `{"query":"ada"}`, "key-1", credAgentA, contract.TrustAutonomous)
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	if out.Status != contract.RunSucceeded {
		t.Fatalf("status = %s, want succeeded", out.Status)
	}
	var decoded struct {
		Contacts []struct {
			ID string `json:"id"`
		} `json:"contacts"`
	}
	if err := json.Unmarshal(out.Result, &decoded); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(decoded.Contacts) != 1 || decoded.Contacts[0].ID != "a1" {
		t.Fatalf("result = %+v, want [a1]", decoded.Contacts)
	}
	events, err := fx.aud.Query(context.Background(), tenantA, 100)
	if err != nil {
		t.Fatalf("audit query: %v", err)
	}
	saw := map[string]bool{}
	for _, e := range events {
		if e.InvocationID == out.InvocationID {
			saw[e.Action+"/"+e.Outcome] = true
		}
	}
	for _, want := range []string{"invoke/dispatched", "invoke/succeeded"} {
		if !saw[want] {
			t.Fatalf("audit for %s missing %q (have %v)", out.InvocationID, want, saw)
		}
	}
}

func TestInvokeIdempotentReplay(t *testing.T) {
	db := newDB(t)
	fx := newFixture(t, db)
	seedContact(t, db, tenantA, "a1", "Ada Lovelace", "ada@example.com")

	first, err := invoke(fx.svc, crmread.ID, crmread.ToolSearch, `{"query":""}`, "key-dup", credAgentA, contract.TrustAutonomous)
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	second, err := invoke(fx.svc, crmread.ID, crmread.ToolSearch, `{"query":""}`, "key-dup", credAgentA, contract.TrustAutonomous)
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if second.InvocationID != first.InvocationID {
		t.Fatalf("replay invocation %s != %s", second.InvocationID, first.InvocationID)
	}
	// Compare semantically: the record round-trips through jsonb,
	// which normalizes whitespace relative to the handler's encoding.
	var firstDecoded, secondDecoded any
	if err := json.Unmarshal(first.Result, &firstDecoded); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(second.Result, &secondDecoded); err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%#v", secondDecoded) != fmt.Sprintf("%#v", firstDecoded) {
		t.Fatal("replay result differs from first execution")
	}
}

func TestInvokeRejections(t *testing.T) {
	db := newDB(t)
	fx := newFixture(t, db)
	cases := []struct {
		name       string
		miniapp    string
		tool       string
		input      string
		cred       string
		trust      contract.TrustMode
		wantSubstr string
	}{
		{"bad credential", crmread.ID, crmread.ToolSearch, `{}`, "nope", contract.TrustAutonomous, "auth-required"},
		{"unknown miniapp", "missing", crmread.ToolSearch, `{}`, credAgentA, contract.TrustAutonomous, "unknown-miniapp"},
		{"unknown tool", crmread.ID, "crm.nope", `{}`, credAgentA, contract.TrustAutonomous, "unknown-tool"},
		{"invalid input", crmread.ID, crmread.ToolSearch, `{"limit":999}`, credAgentA, contract.TrustAutonomous, "invalid-input"},
		{"trust exceeds grant", gatedID, crmread.ToolSearch, `{}`, credAgentA, contract.TrustAutonomous, "forbidden"},
		{"ungranted tenant", gatedID, crmread.ToolSearch, `{}`, credAgentB, contract.TrustConfirm, "forbidden"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := invoke(fx.svc, c.miniapp, c.tool, c.input, "key-"+c.name, c.cred, c.trust)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", c.wantSubstr)
			}
			if got := err.Error(); !strings.Contains(got, c.wantSubstr) {
				t.Fatalf("error = %q, want substring %q", got, c.wantSubstr)
			}
		})
	}
}

func TestApprovalGrantReleases(t *testing.T) {
	db := newDB(t)
	fx := newFixture(t, db)
	seedContact(t, db, tenantA, "a1", "Ada Lovelace", "ada@example.com")
	input := `{"query":"ada"}`

	waiting, err := invoke(fx.svc, gatedID, crmread.ToolSearch, input, "key-appr", credAgentA, contract.TrustConfirm)
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	if waiting.Status != contract.RunWaitingApproval || waiting.ApprovalID == "" {
		t.Fatalf("invoke = %+v, want waiting_approval with approval ID", waiting)
	}
	hash := runtime.IntentHash(gatedID, "1.0.0", crmread.ToolSearch, json.RawMessage(input))
	if err := fx.svc.DecideApproval(context.Background(), credDecide, waiting.ApprovalID, contract.DecisionApprove, hash); err != nil {
		t.Fatalf("decide: %v", err)
	}
	done, err := invoke(fx.svc, gatedID, crmread.ToolSearch, input, "key-appr", credAgentA, contract.TrustConfirm)
	if err != nil {
		t.Fatalf("re-invoke: %v", err)
	}
	if done.Status != contract.RunSucceeded || done.InvocationID != waiting.InvocationID {
		t.Fatalf("re-invoke = %+v, want succeeded on %s", done, waiting.InvocationID)
	}
}

func TestApprovalRejectFails(t *testing.T) {
	db := newDB(t)
	fx := newFixture(t, db)
	input := `{"query":""}`

	waiting, err := invoke(fx.svc, gatedID, crmread.ToolSearch, input, "key-rej", credAgentA, contract.TrustConfirm)
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	hash := runtime.IntentHash(gatedID, "1.0.0", crmread.ToolSearch, json.RawMessage(input))
	if err := fx.svc.DecideApproval(context.Background(), credDecide, waiting.ApprovalID, contract.DecisionReject, hash); err != nil {
		t.Fatalf("decide: %v", err)
	}
	_, err = invoke(fx.svc, gatedID, crmread.ToolSearch, input, "key-rej", credAgentA, contract.TrustConfirm)
	if err == nil || !strings.Contains(err.Error(), "approval-denied") {
		t.Fatalf("re-invoke err = %v, want approval-denied", err)
	}
}

func TestApprovalSelfDecisionRejected(t *testing.T) {
	db := newDB(t)
	fx := newFixture(t, db)

	waiting, err := invoke(fx.svc, gatedID, crmread.ToolSearch, `{}`, "key-self", credAgentA, contract.TrustConfirm)
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	hash := runtime.IntentHash(gatedID, "1.0.0", crmread.ToolSearch, json.RawMessage(`{}`))
	err = fx.svc.DecideApproval(context.Background(), credAgentA, waiting.ApprovalID, contract.DecisionApprove, hash)
	if err == nil || !strings.Contains(err.Error(), "decision-rejected") {
		t.Fatalf("self-decide err = %v, want decision-rejected", err)
	}
}

func TestCrossTenantIsolation(t *testing.T) {
	db := newDB(t)
	fx := newFixture(t, db)
	seedContact(t, db, tenantA, "a1", "Ada Lovelace", "ada@example.com")

	out, err := invoke(fx.svc, crmread.ID, crmread.ToolSearch, `{}`, "key-x", credAgentA, contract.TrustAutonomous)
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	if _, err := fx.svc.Status(context.Background(), credAgentB, out.InvocationID); err == nil {
		t.Fatal("cross-tenant status succeeded")
	}
	st, err := fx.svc.Status(context.Background(), credAgentA, out.InvocationID)
	if err != nil {
		t.Fatalf("own status: %v", err)
	}
	if st.Status != contract.RunSucceeded {
		t.Fatalf("status = %s, want succeeded", st.Status)
	}
}

func TestClassDLifecycle(t *testing.T) {
	db := newDB(t)
	fx := newFixture(t, db)

	out, err := invoke(fx.svc, probeID, probeTool, `{}`, "key-d", credAgentA, contract.TrustAutonomous)
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	if out.Status != contract.RunRunning || out.Handle == nil {
		t.Fatalf("invoke = %+v, want running with handle", out)
	}
	runID := out.Handle.RunID
	st, err := fx.svc.Status(context.Background(), credAgentA, out.InvocationID)
	if err != nil || st.Status != contract.RunRunning {
		t.Fatalf("status = %+v, err = %v", st, err)
	}
	sig := contract.Signal{Type: "probe.ping", Payload: json.RawMessage(`{"n":1}`)}
	if err := fx.svc.Signal(context.Background(), credAgentA, runID, sig); err != nil {
		t.Fatalf("signal: %v", err)
	}
	if len(fx.async.signal) != 1 {
		t.Fatalf("signals recorded = %v, want 1", fx.async.signal)
	}
	if _, err := fx.svc.Cancel(context.Background(), credAgentA, runID, "test done"); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	st, err = fx.svc.Status(context.Background(), credAgentA, out.InvocationID)
	if err != nil || st.Status != contract.RunCancelled {
		t.Fatalf("post-cancel status = %+v, err = %v", st, err)
	}
	if err := fx.svc.Signal(context.Background(), credAgentA, runID, sig); err == nil {
		t.Fatal("signal to cancelled run succeeded")
	}
}

func TestIntentHashStability(t *testing.T) {
	a := runtime.IntentHash("m", "1.0.0", "t", json.RawMessage(`{"x":1}`))
	b := runtime.IntentHash("m", "1.0.0", "t", json.RawMessage(`{"x":1}`))
	c := runtime.IntentHash("m", "1.0.0", "t", json.RawMessage(`{"x":2}`))
	if a != b {
		t.Fatal("same intent hashed differently")
	}
	if a == c {
		t.Fatal("different intents hashed identically")
	}
	d := runtime.IntentHash("m", "1.0.0", "t", json.RawMessage("{ \"x\" : 1 }"))
	e := runtime.IntentHash("m", "1.0.0", "t", json.RawMessage(`{"y":0,"x":1}`))
	f := runtime.IntentHash("m", "1.0.0", "t", json.RawMessage(`{"x":1,"y":0}`))
	if a != d {
		t.Fatal("whitespace variant hashed differently")
	}
	if e != f {
		t.Fatal("key-order variants hashed differently")
	}
}

func TestInvokeEntityDerived(t *testing.T) {
	db := newDB(t)
	fx := newFixture(t, db)

	out, err := invoke(fx.svc, "entity-probe", "probe.capture", `{"subject_id":"s1"}`, "key-ent", credAgentA, contract.TrustAutonomous)
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	if out.Status != contract.RunSucceeded {
		t.Fatalf("status = %s, want succeeded", out.Status)
	}
	if got := fx.captured.get(out.InvocationID); got != "tenant-a:s1" {
		t.Fatalf("entity key = %q, want %q", got, "tenant-a:s1")
	}
}

func TestInvokeEntityMissingFails(t *testing.T) {
	db := newDB(t)
	fx := newFixture(t, db)

	_, err := invoke(fx.svc, "entity-probe", "probe.capture", `{}`, "key-ent-miss", credAgentA, contract.TrustAutonomous)
	if err == nil || !strings.Contains(err.Error(), "invalid-entity") {
		t.Fatalf("err = %v, want invalid-entity", err)
	}
}

func TestIdempotencyKeyConflict(t *testing.T) {
	db := newDB(t)
	fx := newFixture(t, db)
	seedContact(t, db, tenantA, "a1", "Ada Lovelace", "ada@example.com")

	if _, err := invoke(fx.svc, crmread.ID, crmread.ToolSearch, `{"query":"ada"}`, "key-conflict", credAgentA, contract.TrustAutonomous); err != nil {
		t.Fatalf("first: %v", err)
	}
	_, err := invoke(fx.svc, crmread.ID, crmread.ToolSearch, `{"query":"grace"}`, "key-conflict", credAgentA, contract.TrustAutonomous)
	if err == nil || !strings.Contains(err.Error(), "idempotency-key-reuse") {
		t.Fatalf("err = %v, want idempotency-key-reuse", err)
	}
	// Serialization-only differences still replay.
	out, err := invoke(fx.svc, crmread.ID, crmread.ToolSearch, `{ "query" : "ada" }`, "key-conflict", credAgentA, contract.TrustAutonomous)
	if err != nil || out.Status != contract.RunSucceeded {
		t.Fatalf("whitespace-variant replay = %+v, %v", out, err)
	}
}

func TestSubmitFailureRetryAttaches(t *testing.T) {
	db := newDB(t)
	fx := newFixture(t, db)
	fx.async.failNextSubmit = true

	_, err := invoke(fx.svc, probeID, probeTool, `{}`, "key-flaky", credAgentA, contract.TrustAutonomous)
	if err == nil || !strings.Contains(err.Error(), "submit-failed") {
		t.Fatalf("first err = %v, want submit-failed", err)
	}
	out, err := invoke(fx.svc, probeID, probeTool, `{}`, "key-flaky", credAgentA, contract.TrustAutonomous)
	if err != nil {
		t.Fatalf("retry: %v", err)
	}
	if out.Status != contract.RunRunning || out.Handle == nil || out.Handle.RunID == "" {
		t.Fatalf("retry = %+v, want running with valid handle", out)
	}
	fx.async.mu.Lock()
	defer fx.async.mu.Unlock()
	if len(fx.async.status) != 1 {
		t.Fatalf("executor runs = %d, want 1 (no orphan, no duplicate)", len(fx.async.status))
	}
	if _, ok := fx.async.status[out.Handle.RunID]; !ok {
		t.Fatalf("handle %s names no live run", out.Handle.RunID)
	}
}

func TestConfirmSubmitStableHandle(t *testing.T) {
	db := newDB(t)
	fx := newFixture(t, db)

	first, err := invoke(fx.svc, probeID, probeTool, `{}`, "key-stable", credAgentA, contract.TrustAutonomous)
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	second, err := invoke(fx.svc, probeID, probeTool, `{}`, "key-stable", credAgentA, contract.TrustAutonomous)
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if second.Handle == nil || second.Handle.RunID != first.Handle.RunID {
		t.Fatalf("handles %v vs %v: duplicate poll must return the same run", first.Handle, second.Handle)
	}
}

func effIntent(inv, token, op, mode string) effects.Intent {
	return effects.Intent{
		BusinessOpID: tenantA + ":" + op, AttemptID: "a-" + op, InvocationID: inv,
		EffectToken: token,
		Operation: effects.Operation{CapabilityID: "mock", Operation: "send",
			Payload: json.RawMessage(`{"mock":{"mode":"` + mode + `","opId":"` + op + `"}}`)},
	}
}

func effectToken(t *testing.T, db *testkit.DB, tenant, invID string) string {
	t.Helper()
	token, err := store.WithTenant(context.Background(), db.Pool, tenant, func(tx pgx.Tx) (string, error) {
		var tok string
		err := tx.QueryRow(context.Background(),
			`SELECT effect_token FROM app.invocations WHERE id = $1::uuid`, invID).Scan(&tok)
		return tok, err
	})
	if err != nil {
		t.Fatal(err)
	}
	if token == "" {
		t.Fatal("empty effect token")
	}
	return token
}

func TestCancelBeforeClaim(t *testing.T) {
	db := newDB(t)
	fx := newFixture(t, db)

	out, err := invoke(fx.svc, probeID, probeTool, `{}`, "key-cancel-clean", credAgentA, contract.TrustAutonomous)
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	res, err := fx.svc.Cancel(context.Background(), credAgentA, out.Handle.RunID, "stop")
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if len(res.InFlight) != 0 {
		t.Fatalf("in-flight = %v, want clean cancel", res.InFlight)
	}
	tok := effectToken(t, db, tenantA, out.InvocationID)
	_, err = fx.effects.Perform(context.Background(), tenantA, effIntent(out.InvocationID, tok, "op-after-cancel", mockcap.ModeAccept))
	if !errors.Is(err, effects.ErrInvocationCancelled) {
		t.Fatalf("post-cancel claim err = %v, want ErrInvocationCancelled", err)
	}
}

func TestCancelPendingApproval(t *testing.T) {
	db := newDB(t)
	fx := newFixture(t, db)

	waiting, err := invoke(fx.svc, probeGatedID, probeGatedTool, `{}`, "key-cancel-appr", credAgentA, contract.TrustConfirm)
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	if waiting.Status != contract.RunWaitingApproval || waiting.Handle == nil {
		t.Fatalf("invoke = %+v, want waiting with handle", waiting)
	}
	if _, err := fx.svc.Cancel(context.Background(), credAgentA, waiting.Handle.RunID, "stop"); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	fx.async.mu.Lock()
	st := fx.async.status[waiting.Handle.RunID]
	fx.async.mu.Unlock()
	if st != contract.RunCancelled {
		t.Fatalf("run status = %s, want cancelled", st)
	}
	appr, err := fx.approver.ByInvocation(context.Background(), tenantA, waiting.InvocationID)
	if err != nil {
		t.Fatal(err)
	}
	if appr.Status != "cancelled" {
		t.Fatalf("approval status = %s, want cancelled", appr.Status)
	}
}

// blockChannel holds a send open until released.
type blockChannel struct {
	entered chan struct{}
	release chan struct{}
}

func (c *blockChannel) Send(ctx context.Context, op effects.Operation) (effects.Receipt, error) {
	close(c.entered)
	select {
	case <-c.release:
		return effects.Receipt{ProviderRequestID: "r1"}, nil
	case <-ctx.Done():
		return effects.Receipt{}, ctx.Err()
	}
}

func TestCancelInFlightReported(t *testing.T) {
	db := newDB(t)
	fx := newFixture(t, db)

	out, err := invoke(fx.svc, probeID, probeTool, `{}`, "key-cancel-flight", credAgentA, contract.TrustAutonomous)
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	gate := &blockChannel{entered: make(chan struct{}), release: make(chan struct{})}
	effSvc := effects.New(effects.Config{Pool: db.Pool,
		Channels:  map[string]effects.Channel{"mock": gate},
		Canceller: runtime.EffectCanceller(db.Pool),
		Policy:    effects.StrictPolicy{Pool: db.Pool, Registry: fx.reg, Grants: fx.policy},
	})
	flightTok := effectToken(t, db, tenantA, out.InvocationID)
	done := make(chan error, 1)
	go func() {
		_, err := effSvc.Perform(context.Background(), tenantA, effects.Intent{
			BusinessOpID: tenantA + ":op-flight", AttemptID: "a1", InvocationID: out.InvocationID,
			EffectToken: flightTok,
			Operation:   effects.Operation{CapabilityID: "mock", Operation: "send"},
		})
		done <- err
	}()
	select {
	case <-gate.entered:
	case err := <-done:
		t.Fatalf("perform failed before send: %v", err)
	case <-time.After(30 * time.Second):
		t.Fatal("worker never reached Send")
	}
	res, err := fx.svc.Cancel(context.Background(), credAgentA, out.Handle.RunID, "stop")
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if len(res.InFlight) != 1 || res.InFlight[0] != tenantA+":op-flight" {
		t.Fatalf("in-flight = %v, want the open attempt", res.InFlight)
	}
	close(gate.release)
	if err := <-done; err != nil {
		t.Fatalf("in-flight attempt: %v", err)
	}
	rec, err := effSvc.Perform(context.Background(), tenantA, effects.Intent{
		BusinessOpID: tenantA + ":op-flight", AttemptID: "a2", InvocationID: out.InvocationID,
		EffectToken: flightTok,
		Operation:   effects.Operation{CapabilityID: "mock", Operation: "send"},
	})
	if err != nil {
		t.Fatalf("repeat: %v", err)
	}
	if rec.State != contract.EffectConfirmed {
		t.Fatalf("in-flight settled as %s, want CONFIRMED by evidence", rec.State)
	}
}

func TestCancelDuplicateSubmission(t *testing.T) {
	db := newDB(t)
	fx := newFixture(t, db)

	out, err := invoke(fx.svc, probeID, probeTool, `{}`, "key-cancel-dup", credAgentA, contract.TrustAutonomous)
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	if _, err := fx.svc.Cancel(context.Background(), credAgentA, out.Handle.RunID, "stop"); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	_, err = invoke(fx.svc, probeID, probeTool, `{}`, "key-cancel-dup", credAgentA, contract.TrustAutonomous)
	if err == nil || !strings.Contains(err.Error(), "recorded-failure") {
		t.Fatalf("re-invoke err = %v, want recorded-failure", err)
	}
}

func TestCancelExecutorDownRedelivers(t *testing.T) {
	db := newDB(t)
	fx := newFixture(t, db)

	out, err := invoke(fx.svc, probeID, probeTool, `{}`, "key-cancel-down", credAgentA, contract.TrustAutonomous)
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	runID := out.Handle.RunID
	fx.async.failNextCancel = true
	if _, err := fx.svc.Cancel(context.Background(), credAgentA, runID, "stop"); err == nil ||
		!strings.Contains(err.Error(), "cancel-failed") {
		t.Fatalf("cancel err = %v, want retryable cancel-failed", err)
	}
	st, err := fx.svc.Status(context.Background(), credAgentA, out.InvocationID)
	if err != nil || st.Status != contract.RunCancelled {
		t.Fatalf("status = %+v, %v; record must show durable cancel", st, err)
	}
	fx.async.mu.Lock()
	es := fx.async.status[runID]
	fx.async.mu.Unlock()
	if es != contract.RunCancelled {
		t.Fatalf("executor run = %s, want cancelled via Status redelivery", es)
	}
	if _, err := fx.svc.Cancel(context.Background(), credAgentA, runID, "again"); err != nil {
		t.Fatalf("second cancel (redeliver path): %v", err)
	}
}

func backdateExpiry(t *testing.T, db *testkit.DB, approvalID string) {
	t.Helper()
	_, err := store.WithTenant(context.Background(), db.Pool, tenantA, func(tx pgx.Tx) (bool, error) {
		_, err := tx.Exec(context.Background(),
			`UPDATE app.approvals SET expires_at = now() - interval '1 hour' WHERE id = $1::uuid`, approvalID)
		return true, err
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestDenialCancelsRun(t *testing.T) {
	db := newDB(t)
	fx := newFixture(t, db)
	input := `{}`

	waiting, err := invoke(fx.svc, probeGatedID, probeGatedTool, input, "key-deny-run", credAgentA, contract.TrustConfirm)
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	runID := waiting.Handle.RunID
	hash := runtime.IntentHash(probeGatedID, "1.0.0", probeGatedTool, json.RawMessage(input))
	if err := fx.svc.DecideApproval(context.Background(), credDecide, waiting.ApprovalID, contract.DecisionReject, hash); err != nil {
		t.Fatalf("decide: %v", err)
	}
	_, err = invoke(fx.svc, probeGatedID, probeGatedTool, input, "key-deny-run", credAgentA, contract.TrustConfirm)
	if err == nil || !strings.Contains(err.Error(), "approval-denied") {
		t.Fatalf("poll err = %v, want approval-denied", err)
	}
	fx.async.mu.Lock()
	st := fx.async.status[runID]
	fx.async.mu.Unlock()
	if st != contract.RunCancelled {
		t.Fatalf("denied run status = %s, want cancelled (no orphan)", st)
	}
}

func TestExpiryCancelsRun(t *testing.T) {
	db := newDB(t)
	fx := newFixture(t, db)

	waiting, err := invoke(fx.svc, probeGatedID, probeGatedTool, `{}`, "key-exp-run", credAgentA, contract.TrustConfirm)
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	runID := waiting.Handle.RunID
	backdateExpiry(t, db, waiting.ApprovalID)
	_, err = invoke(fx.svc, probeGatedID, probeGatedTool, `{}`, "key-exp-run", credAgentA, contract.TrustConfirm)
	if err == nil || !strings.Contains(err.Error(), "approval-expired") {
		t.Fatalf("poll err = %v, want approval-expired", err)
	}
	fx.async.mu.Lock()
	st := fx.async.status[runID]
	fx.async.mu.Unlock()
	if st != contract.RunCancelled {
		t.Fatalf("expired run status = %s, want cancelled", st)
	}
}

func TestManifestExpiryRecorded(t *testing.T) {
	db := newDB(t)
	fx := newFixture(t, db)

	waiting, err := invoke(fx.svc, probeGatedID, probeGatedTool, `{}`, "key-exp-manifest", credAgentA, contract.TrustConfirm)
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	appr, err := fx.approver.ByInvocation(context.Background(), tenantA, waiting.InvocationID)
	if err != nil {
		t.Fatal(err)
	}
	if appr.ExpiresAt.IsZero() {
		t.Fatal("manifest-declared expiry produced no ExpiresAt")
	}
}

func TestDecideSignalsRun(t *testing.T) {
	db := newDB(t)
	fx := newFixture(t, db)
	input := `{}`

	waiting, err := invoke(fx.svc, probeGatedID, probeGatedTool, input, "key-sig", credAgentA, contract.TrustConfirm)
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	hash := runtime.IntentHash(probeGatedID, "1.0.0", probeGatedTool, json.RawMessage(input))
	if err := fx.svc.DecideApproval(context.Background(), credDecide, waiting.ApprovalID, contract.DecisionApprove, hash); err != nil {
		t.Fatalf("decide: %v", err)
	}
	fx.async.mu.Lock()
	defer fx.async.mu.Unlock()
	if len(fx.async.signal) != 1 {
		t.Fatalf("signals = %d, want 1 decision notification", len(fx.async.signal))
	}
	got := fx.async.signal[0]
	if got.run != waiting.Handle.RunID || got.typ != contract.SignalApprovalDecided {
		t.Fatalf("signal = %+v, want approval.decided to the waiting run", got)
	}
	var payload contract.ApprovalSignal
	if err := json.Unmarshal([]byte(got.payload), &payload); err != nil {
		t.Fatalf("signal payload: %v", err)
	}
	if payload.ApprovalID != waiting.ApprovalID || payload.Decision != contract.DecisionApprove {
		t.Fatalf("payload = %+v, want approval id + approve", payload)
	}
}

func TestDecideIdempotentAndConflicting(t *testing.T) {
	db := newDB(t)
	fx := newFixture(t, db)
	input := `{}`

	waiting, err := invoke(fx.svc, probeGatedID, probeGatedTool, input, "key-redecide", credAgentA, contract.TrustConfirm)
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	hash := runtime.IntentHash(probeGatedID, "1.0.0", probeGatedTool, json.RawMessage(input))
	decide := func(d contract.ApprovalDecision) error {
		return fx.svc.DecideApproval(context.Background(), credDecide, waiting.ApprovalID, d, hash)
	}
	if err := decide(contract.DecisionApprove); err != nil {
		t.Fatalf("decide: %v", err)
	}
	if err := decide(contract.DecisionApprove); err != nil {
		t.Fatalf("identical re-decide: %v (must converge for crashed deciders)", err)
	}
	if err := decide(contract.DecisionReject); err == nil {
		t.Fatal("conflicting re-decide accepted")
	}
}

func TestDecideCrashRedelivers(t *testing.T) {
	db := newDB(t)
	fx := newFixture(t, db)
	input := `{}`

	waiting, err := invoke(fx.svc, probeGatedID, probeGatedTool, input, "key-sig-crash", credAgentA, contract.TrustConfirm)
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	hash := runtime.IntentHash(probeGatedID, "1.0.0", probeGatedTool, json.RawMessage(input))
	fx.async.failNextSignal = true
	if err := fx.svc.DecideApproval(context.Background(), credDecide, waiting.ApprovalID, contract.DecisionApprove, hash); err == nil ||
		!strings.Contains(err.Error(), "signal-failed") {
		t.Fatalf("decide err = %v, want retryable signal-failed", err)
	}
	if err := fx.svc.DecideApproval(context.Background(), credDecide, waiting.ApprovalID, contract.DecisionApprove, hash); err != nil {
		t.Fatalf("retry: %v", err)
	}
	fx.async.mu.Lock()
	defer fx.async.mu.Unlock()
	if len(fx.async.signal) != 1 || fx.async.signal[0].typ != contract.SignalApprovalDecided {
		t.Fatalf("signals = %+v, want exactly one delivered notification", fx.async.signal)
	}
}

func TestScopeEnforced(t *testing.T) {
	db := newDB(t)
	fx := newFixture(t, db)
	input := `{}`

	waiting, err := invoke(fx.svc, probeGatedID, probeGatedTool, input, "key-scope", credAgentA, contract.TrustConfirm)
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	hash := runtime.IntentHash(probeGatedID, "1.0.0", probeGatedTool, json.RawMessage(input))
	if err := fx.svc.DecideApproval(context.Background(), credAgentA2, waiting.ApprovalID, contract.DecisionApprove, hash); err == nil ||
		!strings.Contains(err.Error(), "decision-rejected") {
		t.Fatalf("unscoped decide err = %v, want decision-rejected", err)
	}
	if err := fx.svc.DecideApproval(context.Background(), credDecide, waiting.ApprovalID, contract.DecisionApprove, hash); err != nil {
		t.Fatalf("scoped decide: %v", err)
	}
}

func TestDecideAfterCancelRefused(t *testing.T) {
	db := newDB(t)
	fx := newFixture(t, db)
	input := `{}`

	waiting, err := invoke(fx.svc, probeGatedID, probeGatedTool, input, "key-decide-cancel", credAgentA, contract.TrustConfirm)
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	if _, err := fx.svc.Cancel(context.Background(), credAgentA, waiting.Handle.RunID, "stop"); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	hash := runtime.IntentHash(probeGatedID, "1.0.0", probeGatedTool, json.RawMessage(input))
	if err := fx.svc.DecideApproval(context.Background(), credDecide, waiting.ApprovalID, contract.DecisionApprove, hash); err == nil ||
		!strings.Contains(err.Error(), "approval-cancelled") {
		t.Fatalf("decide-after-cancel err = %v, want approval-cancelled", err)
	}
}

func TestEffectGrantRevoked(t *testing.T) {
	db := newDB(t)
	fx := newFixture(t, db)

	out, err := invoke(fx.svc, probeID, probeTool, `{}`, "key-revoke", credAgentA, contract.TrustAutonomous)
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	tok := effectToken(t, db, tenantA, out.InvocationID)
	delete(fx.policy.Grants[tenantA], probeID)
	_, err = fx.effects.Perform(context.Background(), tenantA, effIntent(out.InvocationID, tok, "op-revoked", mockcap.ModeAccept))
	if err == nil || !strings.Contains(err.Error(), "no live grant") {
		t.Fatalf("revoked-grant claim err = %v, want refusal", err)
	}
}

func TestKillSwitchBlocks(t *testing.T) {
	db := newDB(t)
	fx := newFixture(t, db)

	out, err := invoke(fx.svc, probeID, probeTool, `{}`, "key-killsw", credAgentA, contract.TrustAutonomous)
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	tok := effectToken(t, db, tenantA, out.InvocationID)
	ctx := context.Background()
	if err := fx.effects.SetKillSwitch(ctx, "tenant:"+tenantA, true); err != nil {
		t.Fatal(err)
	}
	_, err = fx.effects.Perform(ctx, tenantA, effIntent(out.InvocationID, tok, "op-blocked", mockcap.ModeAccept))
	if err == nil || !strings.Contains(err.Error(), "kill switch") {
		t.Fatalf("engaged-switch claim err = %v, want refusal", err)
	}
	if err := fx.effects.SetKillSwitch(ctx, "tenant:"+tenantA, false); err != nil {
		t.Fatal(err)
	}
	rec, err := fx.effects.Perform(ctx, tenantA, effIntent(out.InvocationID, tok, "op-blocked", mockcap.ModeAccept))
	if err != nil || rec.State != contract.EffectConfirmed {
		t.Fatalf("released-switch claim = %+v, %v; want CONFIRMED", rec, err)
	}
}

func TestServerDerivedPrincipal(t *testing.T) {
	db := newDB(t)
	fx := newFixture(t, db)

	out, err := invoke(fx.svc, "entity-probe", "probe.capture",
		`{"subject_id":"s9","tenant_id":"evil","agent_id":"evil","authority":{"kind":"user","id":"evil","tenantId":"evil"}}`,
		"key-spoof", credAgentA, contract.TrustAutonomous)
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	got := fx.captured.getCall(out.InvocationID)
	if got.tenant != tenantA || got.agent != "agent-a" {
		t.Fatalf("call principals = %+v; input must not override server-derived identity", got)
	}
	if got.authority.TenantID != tenantA || got.authority.Kind != contract.AuthorityUser {
		t.Fatalf("authority = %+v, want server-derived tenant-a user", got.authority)
	}
}

func TestEffectTokenMismatch(t *testing.T) {
	db := newDB(t)
	fx := newFixture(t, db)

	out, err := invoke(fx.svc, probeID, probeTool, `{}`, "key-token", credAgentA, contract.TrustAutonomous)
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	_, err = fx.effects.Perform(context.Background(), tenantA, effIntent(out.InvocationID, "forged-token", "op-forged", mockcap.ModeAccept))
	if err == nil || !strings.Contains(err.Error(), "token mismatch") {
		t.Fatalf("forged-token claim err = %v, want refusal", err)
	}
}

func TestStatusStaleOnExecutorDown(t *testing.T) {
	db := newDB(t)
	fx := newFixture(t, db)

	out, err := invoke(fx.svc, probeID, probeTool, `{}`, "key-stale", credAgentA, contract.TrustAutonomous)
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	fresh, err := fx.svc.Status(context.Background(), credAgentA, out.InvocationID)
	if err != nil || fresh.Stale {
		t.Fatalf("fresh status = %+v, %v; want live reading", fresh, err)
	}
	fx.async.failNextStatus = true
	stale, err := fx.svc.Status(context.Background(), credAgentA, out.InvocationID)
	if err != nil {
		t.Fatalf("status during outage: %v (must serve persisted record, not fail)", err)
	}
	if !stale.Stale {
		t.Fatal("stale record served without the stale marker")
	}
	if stale.Status != contract.RunRunning || stale.Handle == nil {
		t.Fatalf("stale status = %+v, want persisted running state", stale)
	}
}

func TestEntityCrossTenantIndependent(t *testing.T) {
	db := newDB(t)
	fx := newFixture(t, db)

	a, err := invoke(fx.svc, "entity-probe", "probe.capture", `{"subject_id":"s1"}`, "key-ent-a", credAgentA, contract.TrustAutonomous)
	if err != nil {
		t.Fatalf("tenant-a: %v", err)
	}
	b, err := invoke(fx.svc, "entity-probe", "probe.capture", `{"subject_id":"s1"}`, "key-ent-b", credAgentB, contract.TrustAutonomous)
	if err != nil {
		t.Fatalf("tenant-b: %v", err)
	}
	ka, kb := fx.captured.get(a.InvocationID), fx.captured.get(b.InvocationID)
	if ka != "tenant-a:s1" || kb != "tenant-b:s1" {
		t.Fatalf("keys = %q, %q; identical local IDs must scope per tenant", ka, kb)
	}
}

// TestSignalEmptyRunID proves X-22: an empty run ID is rejected at
// the service boundary even when a run-less invocation row exists
// (previously getByRun("") resolved an arbitrary same-tenant row).
func TestSignalEmptyRunID(t *testing.T) {
	db := newDB(t)
	fx := newFixture(t, db)
	if _, err := invoke(fx.svc, gatedID, crmread.ToolSearch, `{}`, "key-empty-sig", credAgentA, contract.TrustConfirm); err != nil {
		t.Fatalf("park gated invocation: %v", err)
	}
	err := fx.svc.Signal(context.Background(), credAgentA, "", contract.Signal{Type: "x"})
	if err == nil || !strings.Contains(err.Error(), "invalid-request") {
		t.Fatalf("Signal(\"\") err = %v, want terminal invalid-request", err)
	}
}

// TestCancelEmptyRunID proves X-22 for Cancel: empty run IDs never
// resolve, even with a run-less row present.
func TestCancelEmptyRunID(t *testing.T) {
	db := newDB(t)
	fx := newFixture(t, db)
	if _, err := invoke(fx.svc, gatedID, crmread.ToolSearch, `{}`, "key-empty-cancel", credAgentA, contract.TrustConfirm); err != nil {
		t.Fatalf("park gated invocation: %v", err)
	}
	_, err := fx.svc.Cancel(context.Background(), credAgentA, "", "test")
	if err == nil || !strings.Contains(err.Error(), "invalid-request") {
		t.Fatalf("Cancel(\"\") err = %v, want terminal invalid-request", err)
	}
}
