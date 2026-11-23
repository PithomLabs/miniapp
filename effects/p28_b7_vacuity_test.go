package effects

// Prompt28 workstream B, case B7 — sequence/subsequence vacuity
// proof. The M5 production sequence subsystem is deferred: no
// verifier is installed anywhere, and no production code sets
// Intent.Sequence (verified: only the type definition and the
// authorize check site reference it). Any sequence-bearing
// operation therefore refuses fail-closed. This leg pins that
// refusal; supersede-vs-authorize racing is untestable without
// the deferred subsystem and stays an accepted residual.

import (
	"context"
	"errors"
	"testing"

	"github.com/PithomLabs/miniapp/testkit"
)

func TestP28SequenceBoundRefusesWithoutVerifier(t *testing.T) {
	db := testkit.NewDB(t)
	db.Truncate(t, "app.invocations", "app.approvals", "app.audit_events", "app.effect_records")
	svc := p25mService(t, db) // Config.Sequences is nil: no verifier installed
	op := p25mTenant + ":p28-seq"
	inv := p25mSeedInvocation(t, db, "000000000021", "running", "autonomous", p25mGated, p25mGatedTl)
	p25mSeedEffect(t, db, op, inv)

	_, err := svc.authorizeAttempt(context.Background(), p25mTenant, Intent{
		BusinessOpID: op, InvocationID: inv, EffectToken: p25mToken,
		Operation: Operation{CapabilityID: "mock", Operation: "send"},
		Sequence:  &SequenceRef{Subject: "tenant-p25m:subject-1", Generation: 7},
	}, AuthorizeFresh, "a-p25m")
	var refusal *RefusalError
	if !errors.As(err, &refusal) {
		t.Fatalf("err = %v, want terminal *RefusalError", err)
	}
	if refusal.Reason != RefusalStaleOwnership {
		t.Errorf("reason = %s, want stale-ownership (no verifier installed)", refusal.Reason)
	}
	if st := p25mState(t, db, op); st != "PENDING" {
		t.Errorf("row state = %s, want PENDING (untouched)", st)
	}
	if n := p28Audits(t, db, op, "effect-refused"); n != 1 {
		t.Errorf("effect-refused audits = %d, want exactly 1", n)
	}
}
