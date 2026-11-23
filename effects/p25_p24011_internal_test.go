package effects

// P24-011 (Prompt 25): a missing invocation row is a deterministic
// permanent condition (forged ID or phantom-bound op), not
// infrastructure noise. Authorization maps it to a terminal
// binding refusal routed through the refusal-audit path;
// settlement maps it to a terminal settle error. Genuine
// connection errors still propagate retryable (only isNoRows is
// mapped). Unattributable refusals (nothing loaded) skip the
// audit per the standing attribution rule — miniapp and tool
// are unknowable for a phantom invocation.

import (
	"context"
	"errors"
	"testing"

	"github.com/PithomLabs/miniapp/testkit"
)

const p25PhantomInv = "99999999-9999-9999-9999-999999999999"

func TestP25NoRowsAuthorizeTerminal(t *testing.T) {
	db := testkit.NewDB(t)
	db.Truncate(t, "app.invocations", "app.audit_events", "app.effect_records")
	svc := p25AuthService(t, db)
	op := p25Tenant + ":p25-phantom"
	p25SeedEffect(t, db, op, p25PhantomInv, "PENDING")

	err := p25Authorize(t, svc, op, p25PhantomInv)
	var refusal *RefusalError
	if !errors.As(err, &refusal) {
		t.Fatalf("authorize err = %v, want terminal *RefusalError (not infra noise)", err)
	}
	if refusal.Reason != RefusalBindingMismatch {
		t.Errorf("reason = %s, want binding-mismatch", refusal.Reason)
	}
	if st := p25State(t, db, op); st != "PENDING" {
		t.Errorf("row state = %s, want PENDING (untouched)", st)
	}
	// Unattributable: routed through the refusal-audit path,
	// which skips rather than fabricating attribution.
	if n := p25Audits(t, db, op, "effect-refused"); n != 0 {
		t.Errorf("effect-refused audits = %d, want 0 (unattributable skip)", n)
	}
	if n := p25Audits(t, db, op, "effect-withdrawn"); n != 0 {
		t.Errorf("effect-withdrawn audits = %d, want 0", n)
	}
}

func TestP25NoRowsSettleTerminal(t *testing.T) {
	db := testkit.NewDB(t)
	db.Truncate(t, "app.invocations", "app.audit_events", "app.effect_records")
	svc := p25AuthService(t, db)
	op := p25Tenant + ":p25-phantom-s"
	p25SeedEffect(t, db, op, p25PhantomInv, "UNKNOWN")

	_, err := svc.Settle(context.Background(), p25Tenant, op, "NOT_APPLIED", "p25-note", "p25-op")
	if !errors.Is(err, ErrPhantomBinding) {
		t.Fatalf("settle err = %v, want terminal ErrPhantomBinding", err)
	}
	if st := p25State(t, db, op); st != "UNKNOWN" {
		t.Errorf("row state = %s, want UNKNOWN (untouched)", st)
	}
}
