package effects_test

// P24-007 (Prompt 25): fresh attempt IDs are framework-minted
// ownership epochs. claim() mints unconditionally — a
// caller-provided Intent.AttemptID (empty, malformed, colliding,
// or attacker-chosen) must never become the persisted ID — and
// the first attempt authorizes against the persisted epoch.

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/effects"
	"github.com/PithomLabs/miniapp/internal/store"
	"github.com/PithomLabs/miniapp/testkit"
	"github.com/PithomLabs/miniapp/testkit/mockcap"
)

func p25MintSvc(t *testing.T, db *testkit.DB) *effects.Service {
	t.Helper()
	mockcap.EnsureLog(t, db.Pool)
	return newAuthService(t, db,
		map[string]effects.Channel{"mock": mockcap.New(db.Pool)},
		allowCancellation{}, allowPolicy{})
}

func p25MintIntent(inv, op, attempt string) effects.Intent {
	return effects.Intent{BusinessOpID: wsTenant + ":" + op, AttemptID: attempt,
		InvocationID: inv, EffectToken: authzToken,
		Operation: effects.Operation{CapabilityID: "mock", Operation: "send",
			Payload: []byte(`{"mock":{"mode":"accept","opId":"` + op + `"}}`)}}
}

func p25MintedAttempt(t *testing.T, db *testkit.DB, op string) string {
	t.Helper()
	id, err := store.WithTenant(context.Background(), db.Pool, wsTenant, func(tx pgx.Tx) (string, error) {
		var s string
		err := tx.QueryRow(context.Background(), `SELECT attempt_id FROM app.effect_records
			WHERE business_op_id = $1 AND tenant_id = $2`, wsTenant+":"+op, wsTenant).Scan(&s)
		return s, err
	})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func p25AssertMinted(t *testing.T, persisted, caller string) {
	t.Helper()
	if persisted == "" {
		t.Fatalf("persisted attempt ID is empty (caller %q)", caller)
	}
	if persisted == caller {
		t.Fatalf("persisted attempt ID %q equals caller input (not framework-minted)", persisted)
	}
	if _, err := uuid.Parse(persisted); err != nil {
		t.Fatalf("persisted attempt ID %q is not a UUID: %v", persisted, err)
	}
}

func TestP25MintEmptyCallerID(t *testing.T) {
	db := testkit.NewDB(t)
	truncateLedger(t, db)
	svc := p25MintSvc(t, db)
	seedLiveInvocation(t, db, wsTenant, "p25mint1")
	inv := authzInvID(wsTenant, "p25mint1")

	rec, err := svc.Perform(context.Background(), wsTenant, p25MintIntent(inv, "p25mint1", ""))
	if err != nil {
		t.Fatalf("perform err = %v, want nil (empty caller ID is minted, not an error)", err)
	}
	if rec.State != contract.EffectConfirmed {
		t.Errorf("state = %s, want CONFIRMED", rec.State)
	}
	p25AssertMinted(t, p25MintedAttempt(t, db, "p25mint1"), "")
}

func TestP25MintMalformedCallerID(t *testing.T) {
	db := testkit.NewDB(t)
	truncateLedger(t, db)
	svc := p25MintSvc(t, db)
	seedLiveInvocation(t, db, wsTenant, "p25mint2")
	inv := authzInvID(wsTenant, "p25mint2")

	rec, err := svc.Perform(context.Background(), wsTenant, p25MintIntent(inv, "p25mint2", "not-a-uuid!!!"))
	if err != nil {
		t.Fatalf("perform err = %v, want nil", err)
	}
	if rec.State != contract.EffectConfirmed {
		t.Errorf("state = %s, want CONFIRMED", rec.State)
	}
	p25AssertMinted(t, p25MintedAttempt(t, db, "p25mint2"), "not-a-uuid!!!")
}

func TestP25MintAttackerChosen(t *testing.T) {
	db := testkit.NewDB(t)
	truncateLedger(t, db)
	svc := p25MintSvc(t, db)
	seedLiveInvocation(t, db, wsTenant, "p25mint3")
	inv := authzInvID(wsTenant, "p25mint3")

	if _, err := svc.Perform(context.Background(), wsTenant, p25MintIntent(inv, "p25mint3", "attacker-chosen")); err != nil {
		t.Fatalf("perform err = %v, want nil", err)
	}
	p25AssertMinted(t, p25MintedAttempt(t, db, "p25mint3"), "attacker-chosen")
}

func TestP25MintDistinctAcrossOps(t *testing.T) {
	db := testkit.NewDB(t)
	truncateLedger(t, db)
	svc := p25MintSvc(t, db)
	ctx := context.Background()
	seedLiveInvocation(t, db, wsTenant, "p25mint4")
	inv := authzInvID(wsTenant, "p25mint4")

	// Colliding caller IDs must still mint distinct epochs.
	if _, err := svc.Perform(ctx, wsTenant, p25MintIntent(inv, "p25mint4a", "same")); err != nil {
		t.Fatalf("perform A err = %v", err)
	}
	if _, err := svc.Perform(ctx, wsTenant, p25MintIntent(inv, "p25mint4b", "same")); err != nil {
		t.Fatalf("perform B err = %v", err)
	}
	a := p25MintedAttempt(t, db, "p25mint4a")
	b := p25MintedAttempt(t, db, "p25mint4b")
	p25AssertMinted(t, a, "same")
	p25AssertMinted(t, b, "same")
	if a == b {
		t.Errorf("persisted IDs collide: %q (caller collision leaked into epochs)", a)
	}
}

func TestP25MintRetryPreserves(t *testing.T) {
	db := testkit.NewDB(t)
	truncateLedger(t, db)
	svc := p25MintSvc(t, db)
	ctx := context.Background()
	seedLiveInvocation(t, db, wsTenant, "p25mint5")
	inv := authzInvID(wsTenant, "p25mint5")

	first, err := svc.Perform(ctx, wsTenant, p25MintIntent(inv, "p25mint5", "caller-req"))
	if err != nil {
		t.Fatalf("first perform err = %v", err)
	}
	second, err := svc.Perform(ctx, wsTenant, p25MintIntent(inv, "p25mint5", "caller-req"))
	if err != nil {
		t.Fatalf("retry perform err = %v", err)
	}
	if second.State != contract.EffectConfirmed {
		t.Errorf("retry state = %s, want CONFIRMED (recorded outcome)", second.State)
	}
	if second.AttemptID != first.AttemptID {
		t.Errorf("retry attempt = %s, want preserved %s", second.AttemptID, first.AttemptID)
	}
	p25AssertMinted(t, p25MintedAttempt(t, db, "p25mint5"), "caller-req")
}
