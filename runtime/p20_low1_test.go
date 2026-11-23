// LOW-1 discriminating tests: the duplicate audit marker fires only
// when the claim actually identified an existing invocation —
// lost-race reloads answer without mislabeling the caller.
//
// RED/GREEN-COMPATIBLE SOURCE: reuses the HIGH-1 staging helpers
// (present identically pre-fix) plus public APIs and SQL audit
// counts. The red run overlays exactly this file onto the pre-fix
// working copy.
package runtime_test

import (
	"strings"
	"sync"
	"testing"

	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/testkit"
)

// TestP20NoDuplicateOnCompletionLoss runs the settlement-wins
// interleaving: three invokes each find the row via claim (three
// markers), then the losing completion reloads silently. Red:
// the loss replay emits a fourth marker.
func TestP20NoDuplicateOnCompletionLoss(t *testing.T) {
	db := newDB(t)
	p20Low1Truncate(t, db)
	gate := newP20Gate()
	fx := newP20Fixture(t, db, gate)
	key := "key-p20l1-A"
	seedPendingRow(t, db, key)
	var wg sync.WaitGroup
	bOut := make(chan p20Out, 1)
	holder, rd := p20Stage(t, db, fx, gate, key, bOut, &wg)
	aOut := make(chan p20Out, 1)
	wg.Add(1)
	go func() {
		defer wg.Done()
		res, err := invoke(fx.svc, p20Miniapp, p20Tool, `{}`, key, credAgentA, contract.TrustConfirm)
		aOut <- p20Out{res, err}
	}()
	if _, ok := waitBlockedByHolder(t, db, holder, "app.invocations", 1); !ok {
		rd.failf("settle gate never formed")
	}
	rd.once.Do(func() { rd.release() })
	a := rd.recv(aOut, "settle return")
	gate.shutdown()
	b := rd.recv(bOut, "completion return")
	wg.Wait()
	if a.err == nil || !strings.Contains(a.err.Error(), "approval-expired") {
		t.Fatalf("settle err = %+v, want approval-expired (interleaving check)", a.err)
	}
	if b.err == nil || !strings.Contains(b.err.Error(), "recorded-failure") {
		t.Fatalf("completion err = %+v, want recorded-failure (interleaving check)", b.err)
	}
	if n := p20AuditCount(t, db, "invoke", "duplicate"); n != 3 {
		t.Fatalf("duplicate audits = %d, want exactly 3 (loss reload must not mark)", n)
	}
}

// TestP20NoDuplicateOnSettleLoss runs the completion-wins
// interleaving: three claim-found markers, then the losing
// settlement reloads silently. Red: a fourth marker.
func TestP20NoDuplicateOnSettleLoss(t *testing.T) {
	db := newDB(t)
	p20Low1Truncate(t, db)
	gate := newP20Gate()
	fx := newP20Fixture(t, db, gate)
	key := "key-p20l1-B"
	seedPendingRow(t, db, key)
	var wg sync.WaitGroup
	bOut := make(chan p20Out, 1)
	holder, rd := p20Stage(t, db, fx, gate, key, bOut, &wg)
	gate.shutdown()
	if _, ok := waitBlockedByHolder(t, db, holder, "app.invocations", 1); !ok {
		rd.failf("completion gate never formed")
	}
	aOut := make(chan p20Out, 1)
	wg.Add(1)
	go func() {
		defer wg.Done()
		res, err := invoke(fx.svc, p20Miniapp, p20Tool, `{}`, key, credAgentA, contract.TrustConfirm)
		aOut <- p20Out{res, err}
	}()
	if _, ok := waitBlockedByHolder(t, db, holder, "app.invocations", 2); !ok {
		rd.failf("settle gate never formed behind completion")
	}
	rd.once.Do(func() { rd.release() })
	b := rd.recv(bOut, "completion return")
	a := rd.recv(aOut, "settle return")
	wg.Wait()
	if b.err != nil || b.res.Status != contract.RunSucceeded {
		t.Fatalf("completion = %+v, %v; want success (interleaving check)", b.res, b.err)
	}
	if a.err != nil || a.res.Status != contract.RunSucceeded {
		t.Fatalf("settle = %+v, %v; want replayed success (interleaving check)", a.res, a.err)
	}
	if n := p20AuditCount(t, db, "invoke", "duplicate"); n != 3 {
		t.Fatalf("duplicate audits = %d, want exactly 3 (loss reload must not mark)", n)
	}
}

func p20Low1Truncate(t *testing.T, db *testkit.DB) {
	t.Helper()
	db.Truncate(t, "app.invocations", "app.approvals", "app.audit_events", "app.crm_contacts", "app.effect_records", "app.kill_switches", "mock_provider_log")
}
