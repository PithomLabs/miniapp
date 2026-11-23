package effects_test

// Prompt28 workstream E — audit-backfill serialization. The
// idempotent-repeat path re-reads the settled row FOR UPDATE
// before the backfill check, so concurrent identical repeats
// serialize and backfill at most once. Red shape (pre-fix):
// duplicate effect-settled audits under 32-way repeats.

import (
	"context"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/internal/store"
)

func TestP28SettleBackfillNoDuplicates(t *testing.T) {
	svc, db := setupMock(t)
	seedLiveInvocation(t, db, wsTenant, "inv-p28bfd")
	p28SeedUnknown(t, db, "p28bfd", authzInvID(wsTenant, "inv-p28bfd"))
	ctx := context.Background()
	op := wsTenant + ":p28bfd"
	if _, err := svc.Settle(ctx, wsTenant, op, contract.EffectConfirmed, "note", "p28-op"); err != nil {
		t.Fatal(err)
	}
	for round := 0; round < 20; round++ {
		_, err := store.WithTenant(ctx, db.Pool, wsTenant, func(tx pgx.Tx) (bool, error) {
			_, err := tx.Exec(ctx, `DELETE FROM app.audit_events
				WHERE tenant_id = $1 AND action = 'effect-settled'
				AND details->>'business_op_id' = $2`, wsTenant, op)
			return true, err
		})
		if err != nil {
			t.Fatal(err)
		}
		var wg sync.WaitGroup
		for i := 0; i < 32; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, _ = svc.Settle(ctx, wsTenant, op, contract.EffectConfirmed, "note", "p28-op")
			}()
		}
		wg.Wait()
		if n := p28SettledAudits(t, db, "p28bfd"); n != 1 {
			t.Fatalf("round %d: settled audits = %d, want exactly 1", round, n)
		}
	}
}
