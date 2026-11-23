// Package transport_test proves the REST, MCP, and CLI adapters are
// thin: every adapter produces outcomes equivalent to direct
// canonical-service calls across success, refusal, and lifecycle
// cases, adding no authorization, policy, idempotency, or execution
// decision of its own.
package transport_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/PithomLabs/miniapp/auth"
	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/effects"
	crmread "github.com/PithomLabs/miniapp/miniapps/crm-read"
	classd "github.com/PithomLabs/miniapp/probe/classd"
	"github.com/PithomLabs/miniapp/registry"
	"github.com/PithomLabs/miniapp/runtime"
	"github.com/PithomLabs/miniapp/testkit"
	"github.com/PithomLabs/miniapp/testkit/mockcap"
	"github.com/PithomLabs/miniapp/testkit/stack"
	"github.com/PithomLabs/miniapp/transport/cli"
	"github.com/PithomLabs/miniapp/transport/mcp"
	"github.com/PithomLabs/miniapp/transport/rest"
)

const (
	tCred   = "key-transport"
	tDecide = "key-transport-decide"
	tTenant = "tenant-tr"
)

func txPolicy() *auth.Policy {
	binding := func(agent string, roles ...string) auth.Binding {
		return auth.Binding{AgentID: agent, Authority: contract.Authority{
			Kind: contract.AuthorityUser, ID: agent, TenantID: tTenant,
		}, Roles: roles}
	}
	return auth.NewPolicy(
		map[string]auth.Binding{
			tCred:   binding("agent-tx"),
			tDecide: binding("human-tx", "tenant-ops"),
		},
		map[string]map[string]contract.TrustMode{
			tTenant: {
				crmread.ID:    contract.TrustAutonomous,
				"gated-write": contract.TrustConfirm,
				classd.ID:     contract.TrustAutonomous,
			},
		},
	)
}

func seedContact(t *testing.T, fx *stack.Assembled) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	tx, err := fx.DB.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SELECT set_config('app.tenant_id', $1, true)`, tTenant); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO app.crm_contacts (id, tenant_id, name, email)
		VALUES ('tx-1', $1, 'Tx Ada', 'tx@example.com')
		ON CONFLICT (id) DO UPDATE SET name = 'Tx Ada'`, tTenant); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
}

func newFixture(t *testing.T) *stack.Assembled {
	t.Helper()
	db := testkit.NewDB(t)
	classd.EnsureTables(t, db.Pool)
	mockcap.EnsureLog(t, db.Pool)
	db.Truncate(t, "app.invocations", "app.approvals", "app.audit_events",
		"app.crm_contacts", "app.effect_records", "app.kill_switches",
		"probe_sequences", "probe_observations", "probe_wait_marks", "mock_provider_log")
	fx := stack.Assemble(t, stack.Config{DB: db,
		Register: func(t *testing.T, reg *registry.Registry) {
			if err := crmread.Register(reg); err != nil {
				t.Fatal(err)
			}
			if err := classd.Register(reg); err != nil {
				t.Fatal(err)
			}
			mf := testkit.ClassAManifest("gated-write", "probe.gated_write", "", []string{"mock.send"})
			mf.Tools[0].NoEntityWhy = ""
			mf.Tools[0].EntityKey = "subject_id"
			mf.Tools[0].Approval = contract.ApprovalRequirement{Required: true, ApproverScope: "tenant-ops"}
			impl := testkit.ClassAImpl("test", "probe.gated_write",
				func(ctx context.Context, call registry.ToolCall) (json.RawMessage, error) {
					return json.RawMessage(`{"wrote":true}`), nil
				})
			if err := reg.Register(mf, impl, testkit.Envelope(t, mf, "test")); err != nil {
				t.Fatal(err)
			}
		},
		Policy:   txPolicy(),
		Channels: map[string]effects.Channel{"mock": mockcap.New(db.Pool)},
	})
	classd.Effects = fx.Effects
	seedContact(t, fx)
	return fx
}

// adapters drives one scenario through every surface and asserts
// equivalent outcomes.
type adapters struct {
	t   *testing.T
	fx  *stack.Assembled
	srv *httptest.Server
	mcp *sdk.ClientSession
}

func setupAdapters(t *testing.T) *adapters {
	t.Helper()
	fx := newFixture(t)
	srv := httptest.NewServer(rest.New(fx.Svc).Handler())
	t.Cleanup(srv.Close)
	ctx := context.Background()
	clientT, serverT := sdk.NewInMemoryTransports()
	msrv := mcp.NewServer(fx.Svc)
	ssess, err := msrv.Connect(ctx, serverT, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ssess.Close() })
	cli := sdk.NewClient(&sdk.Implementation{Name: "contract-test", Version: "0.0.0"}, nil)
	csess, err := cli.Connect(ctx, clientT, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { csess.Close() })
	return &adapters{t: t, fx: fx, srv: srv, mcp: csess}
}

func (a *adapters) restInvoke(t *testing.T, body map[string]any, cred string) (int, map[string]any) {
	t.Helper()
	raw, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", a.srv.URL+"/v1/invoke", strings.NewReader(string(raw)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(rest.CredentialHeader, cred)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode rest body: %v", err)
	}
	return resp.StatusCode, out
}

func (a *adapters) mcpInvokeRaw(t *testing.T, args map[string]any) (string, error) {
	t.Helper()
	res, err := a.mcp.CallTool(context.Background(), &sdk.CallToolParams{
		Name: mcp.ToolInvoke, Arguments: args,
	})
	if err != nil {
		return "", err
	}
	if len(res.Content) != 1 {
		t.Fatalf("mcp content blocks = %d", len(res.Content))
	}
	text, ok := res.Content[0].(*sdk.TextContent)
	if !ok {
		t.Fatalf("mcp content type %T", res.Content[0])
	}
	return text.Text, nil
}

func (a *adapters) mcpInvoke(t *testing.T, args map[string]any) (map[string]any, error) {
	t.Helper()
	text, err := a.mcpInvokeRaw(t, args)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("decode mcp body %q: %v", text, err)
	}
	return out, nil
}

func (a *adapters) cliInvoke(t *testing.T, args ...string) (int, string) {
	t.Helper()
	var stdout, stderr strings.Builder
	r := &cli.Runner{Svc: a.fx.Svc, Stdout: &stdout, Stderr: &stderr}
	code := r.Run(context.Background(), args)
	return code, stdout.String() + stderr.String()
}

func searchBody(key string) map[string]any {
	return map[string]any{
		"miniapp_id": crmread.ID, "version": "1.0.0", "tool": crmread.ToolSearch,
		"input": map[string]any{"query": "ada"}, "idempotency_key": key,
		"requested_trust": "autonomous",
	}
}

func TestAdaptersHappyPath(t *testing.T) {
	a := setupAdapters(t)

	out, err := a.fx.Svc.Invoke(context.Background(), runtime.InvokeRequest{
		MiniappID: crmread.ID, Version: "1.0.0", Tool: crmread.ToolSearch,
		Input: json.RawMessage(`{"query":"ada"}`), IdempotencyKey: "k-direct",
		Credential: tCred, RequestedTrust: contract.TrustAutonomous,
	})
	if err != nil || out.Status != contract.RunSucceeded {
		t.Fatalf("direct = %+v, %v", out, err)
	}

	code, rout := a.restInvoke(t, searchBody("k-rest"), tCred)
	if code != 200 || rout["status"] != string(contract.RunSucceeded) {
		t.Fatalf("rest = %d %+v", code, rout)
	}

	mout, err := a.mcpInvoke(t, toAnyMap(t, searchBody("k-mcp"), tCred))
	if err != nil {
		t.Fatalf("mcp: %v", err)
	}
	if mout["status"] != string(contract.RunSucceeded) {
		t.Fatalf("mcp = %+v", mout)
	}

	code, cout := a.cliInvoke(t, "invoke", "--miniapp", crmread.ID, "--version", "1.0.0",
		"--tool", crmread.ToolSearch, "--input", `{"query":"ada"}`, "--key", "k-cli",
		"--trust", "autonomous", "--credential", tCred)
	if code != 0 || !strings.Contains(cout, `"status": "succeeded"`) {
		t.Fatalf("cli = %d %s", code, cout)
	}
}

func toAnyMap(t *testing.T, body map[string]any, cred string) map[string]any {
	t.Helper()
	body["credential"] = cred
	return body
}

func TestAdaptersRefusals(t *testing.T) {
	a := setupAdapters(t)
	ctx := context.Background()

	_, err := a.fx.Svc.Invoke(ctx, runtime.InvokeRequest{
		MiniappID: crmread.ID, Version: crmread.Version, Tool: crmread.ToolSearch,
		Input: json.RawMessage(`{}`), Credential: "wrong",
	})
	if err == nil || !strings.Contains(err.Error(), "auth-required") {
		t.Fatalf("direct bad-cred err = %v", err)
	}
	code, _ := a.restInvoke(t, searchBody("k-r1"), "wrong")
	if code != 401 {
		t.Fatalf("rest bad-cred = %d, want 401", code)
	}
	text, err := a.mcpInvokeRaw(t, toAnyMap(t, searchBody("k-r2"), "wrong"))
	if err == nil && !strings.Contains(text, "auth-required") {
		t.Fatalf("mcp bad-cred = %q, %v; want auth-required refusal", text, err)
	}
	if err != nil && !strings.Contains(err.Error(), "auth-required") {
		t.Fatalf("mcp bad-cred err = %v; want auth-required refusal", err)
	}
	if code, _ := a.cliInvoke(t, "invoke", "--miniapp", crmread.ID, "--version", "1.0.0",
		"--tool", crmread.ToolSearch, "--credential", "wrong"); code == 0 {
		t.Fatal("cli bad-cred succeeded")
	}

	badTool := searchBody("k-r3")
	badTool["tool"] = "crm.nope"
	if code, _ := a.restInvoke(t, badTool, tCred); code != 404 {
		t.Fatalf("rest unknown-tool = %d, want 404", code)
	}
	badInput := searchBody("k-r4")
	badInput["input"] = map[string]any{"limit": 999}
	if code, _ := a.restInvoke(t, badInput, tCred); code != 400 {
		t.Fatalf("rest invalid-input = %d, want 400", code)
	}
}

func TestAdaptersApprovalFlow(t *testing.T) {
	a := setupAdapters(t)

	code, rout := a.restInvoke(t, map[string]any{
		"miniapp_id": "gated-write", "version": "1.0.0", "tool": "probe.gated_write",
		"input": map[string]any{"subject_id": "s1"}, "idempotency_key": "k-rest-gated",
		"requested_trust": "confirm",
	}, tCred)
	if code != 200 || rout["status"] != string(contract.RunWaitingApproval) {
		t.Fatalf("rest gated = %d %+v", code, rout)
	}
	approvalID, _ := rout["approval_id"].(string)
	if approvalID == "" {
		t.Fatalf("rest gated response lacks approval id: %+v", rout)
	}

	hash := runtime.IntentHash("gated-write", "1.0.0", "probe.gated_write",
		json.RawMessage(`{"subject_id":"s1"}`))
	decReq, _ := http.NewRequest("POST", a.srv.URL+"/v1/decide",
		strings.NewReader(fmt.Sprintf(`{"approval_id":%q,"decision":"approve","intent_hash":%q}`, approvalID, hash)))
	decReq.Header.Set("Content-Type", "application/json")
	decReq.Header.Set(rest.CredentialHeader, tDecide)
	resp, err := http.DefaultClient.Do(decReq)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("rest decide = %d", resp.StatusCode)
	}
	code, rout2 := a.restInvoke(t, map[string]any{
		"miniapp_id": "gated-write", "version": "1.0.0", "tool": "probe.gated_write",
		"input": map[string]any{"subject_id": "s1"}, "idempotency_key": "k-rest-gated",
		"requested_trust": "confirm",
	}, tCred)
	if code != 200 || rout2["status"] != string(contract.RunSucceeded) {
		t.Fatalf("rest re-invoke = %d %+v", code, rout2)
	}

	mout, err := a.mcpInvoke(t, map[string]any{
		"credential": tCred, "miniapp_id": "gated-write", "version": "1.0.0",
		"tool": "probe.gated_write", "input": map[string]any{"subject_id": "s2"},
		"idempotency_key": "k-mcp-gated", "requested_trust": "confirm",
	})
	if err != nil {
		t.Fatalf("mcp invoke: %v", err)
	}
	if mout["status"] != string(contract.RunWaitingApproval) {
		t.Fatalf("mcp gated = %+v", mout)
	}
}

func TestAdaptersClassDLifecycle(t *testing.T) {
	a := setupAdapters(t)
	ctx := context.Background()

	mout, err := a.mcpInvoke(t, map[string]any{
		"credential": tCred, "miniapp_id": classd.ID, "version": classd.Version,
		"tool":            classd.ToolOpen,
		"input":           map[string]any{"subject_id": "tx-sub", "sleep_ms": 500, "wait_ms": 30000},
		"idempotency_key": "k-mcp-d", "requested_trust": "autonomous",
	})
	if err != nil {
		t.Fatalf("mcp submit: %v", err)
	}
	if mout["status"] != string(contract.RunRunning) {
		t.Fatalf("mcp submit = %+v", mout)
	}
	handle, _ := mout["handle"].(map[string]any)
	runID, _ := handle["run_id"].(string)
	invID, _ := mout["invocation_id"].(string)
	if runID == "" || invID == "" {
		t.Fatalf("mcp submit lacks handle: %+v", mout)
	}

	time.Sleep(2 * time.Second)
	sigRes, err := a.mcp.CallTool(ctx, &sdk.CallToolParams{
		Name: mcp.ToolSignal, Arguments: map[string]any{
			"credential": tCred, "run_id": runID, "type": classd.SignalProceed,
			"payload": map[string]any{},
		},
	})
	if err != nil {
		t.Fatalf("mcp signal: %v", err)
	}
	_ = sigRes

	deadline := time.Now().Add(2 * time.Minute)
	for {
		req, _ := http.NewRequest("GET", a.srv.URL+"/v1/status?invocation_id="+invID, nil)
		req.Header.Set(rest.CredentialHeader, tCred)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		var out map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			resp.Body.Close()
			t.Fatal(err)
		}
		resp.Body.Close()
		if out["status"] == string(contract.RunSucceeded) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("run never succeeded (last %+v)", out)
		}
		time.Sleep(200 * time.Millisecond)
	}
}
