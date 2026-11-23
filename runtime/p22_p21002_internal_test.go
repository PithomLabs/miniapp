// P21-002 structural enforcement: run adoption atomically marks
// a pending record running, so pending implies unsubmitted by
// construction (not merely by reachability argument).
//
// INTERNAL TEST SOURCE (package runtime): drives adoptRunID
// directly — API-level adoption onto pending is refused by the
// approval gate (pinned by TestP22PendingSubmitRequiresLinkage),
// so the boundary op is tested at the layer that owns it.
// Red/green-compatible: the op signature is unchanged.
package runtime

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/internal/store"
	"github.com/PithomLabs/miniapp/testkit"
)

// TestP22AdoptionMarksPendingRunning proves adoption and the
// pending→running transition commit atomically: after a won
// adoption the record is running with the adopted run. Red: the
// adoption wins but the record stays pending.
func TestP22AdoptionMarksPendingRunning(t *testing.T) {
	db := testkit.NewDB(t)
	db.Truncate(t, "app.invocations")
	s := &invocations{pool: db.Pool}
	ctx := context.Background()
	seed := func(key, status string) string {
		t.Helper()
		id := uuid.NewString()
		_, err := store.WithTenant(ctx, db.Pool, "tenant-a", func(tx pgx.Tx) (bool, error) {
			_, err := tx.Exec(ctx, `INSERT INTO app.invocations
				(id, tenant_id, miniapp_id, miniapp_version, tool, idempotency_key, input,
				 status, trust_mode, effect_token, run_id)
				VALUES ($1, 'tenant-a', 'm', '1.0.0', 't.t', $2, '{}', $3, 'confirm', 'tok-p22', '')`,
				id, key, status)
			return true, err
		})
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	id := seed("p22-flip", string(contract.RunPending))
	adopted, err := s.adoptRunID(ctx, "tenant-a", id, "run-p22flip")
	if err != nil || !adopted {
		t.Fatalf("adoptRunID(pending) = %v, %v; want true, nil (pending stays adoptable)", adopted, err)
	}
	got, err := s.getByID(ctx, "tenant-a", id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != contract.RunRunning {
		t.Fatalf("status after adoption = %s, want running (pending implies unsubmitted)", got.Status)
	}
	if got.RunID != "run-p22flip" {
		t.Fatalf("run_id = %q, want the adopted run", got.RunID)
	}
	// Controls (both trees): adoption on already-live states
	// changes no status.
	for _, tc := range []struct {
		key    string
		status contract.RunStatus
		run    string
	}{
		{"p22-live-w", contract.RunWaitingApproval, "run-p22cw"},
		{"p22-live-r", contract.RunRunning, "run-p22cr"},
	} {
		cid := seed(tc.key, string(tc.status))
		if adopted, err := s.adoptRunID(ctx, "tenant-a", cid, tc.run); err != nil || !adopted {
			t.Fatalf("adoptRunID(%s) = %v, %v; want true, nil", tc.status, adopted, err)
		}
		got, err := s.getByID(ctx, "tenant-a", cid)
		if err != nil {
			t.Fatal(err)
		}
		if got.Status != tc.status {
			t.Fatalf("adoptRunID(%s) moved status to %s; want unchanged", tc.status, got.Status)
		}
	}
}
