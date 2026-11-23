package classd_test

// Prompt28 workstream F — DBOS workflow provider-path proof.
// The probe workflow's effect step calls Perform; this test
// proves the ledger outcome carries authorized lineage: the
// row is CONFIRMED with a framework-minted attempt epoch, the
// authorizing invocation's binding, and the provider receipt.
// ATTEMPTING-before-send ordering for the shared Perform path
// is proven by the effects-suite commit proofs (POC-1
// authorize-first, live-race entered markers); this test pins
// the workflow-called outcome to the same lineage.

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/internal/store"
	classd "github.com/PithomLabs/miniapp/probe/classd"
)

func TestP28ProbeEffectLedgerBound(t *testing.T) {
	fx := newStack(t)
	out, err := invokeProbe(t, fx, classd.ToolOpen, "p28ledger", "key-p28ledger", credProber, map[string]any{
		"sleep_ms": 100, "wait_ms": 30000,
	})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(500 * time.Millisecond)
	sig := contract.Signal{Type: classd.SignalProceed, Payload: json.RawMessage(`{"go":true}`)}
	if err := fx.Svc.Signal(context.Background(), credProber, out.Handle.RunID, sig); err != nil {
		t.Fatalf("signal: %v", err)
	}
	done := pollTerminal(t, fx, credProber, out.InvocationID)
	if done.Status != contract.RunSucceeded {
		t.Fatalf("status = %s", done.Status)
	}
	row := readSequence(t, fx, tenantPT, "p28ledger")
	opID := contract.ScopeID(tenantPT, fmt.Sprintf("p28ledger:%d:effect1", row.generation))
	var st, attempt, inv, req, reply string
	_, err = store.WithTenant(context.Background(), fx.DB.Pool, tenantPT, func(tx pgx.Tx) (bool, error) {
		return true, tx.QueryRow(context.Background(), `SELECT state, attempt_id,
			invocation_id, provider_request, provider_reply FROM app.effect_records
			WHERE business_op_id = $1 AND tenant_id = $2`,
			opID, tenantPT).Scan(&st, &attempt, &inv, &req, &reply)
	})
	if err != nil {
		t.Fatal(err)
	}
	if st != string(contract.EffectConfirmed) {
		t.Errorf("effect state = %s, want CONFIRMED", st)
	}
	if _, err := uuid.Parse(attempt); err != nil {
		t.Errorf("attempt %q is not a framework UUID: %v", attempt, err)
	}
	if inv != out.InvocationID {
		t.Errorf("invocation binding = %s, want authorizing %s", inv, out.InvocationID)
	}
	if req == "" || reply == "" {
		t.Errorf("receipt = (%q, %q), want provider-issued identifiers", req, reply)
	}
	if n := providerSends(t, fx, fmt.Sprintf("p28ledger:%d", row.generation)); n != 1 {
		t.Errorf("provider sends = %d, want exactly 1", n)
	}
}
