package effects_test

// Prompt28 workstream A — grant/policy matrix legs (full Perform
// path). Each refusal asserts typed classification, untouched
// PENDING state, zero sends, and exact audit counts.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/PithomLabs/miniapp/approvals"
	"github.com/PithomLabs/miniapp/audit"
	"github.com/PithomLabs/miniapp/auth"
	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/effects"
	"github.com/PithomLabs/miniapp/registry"
	"github.com/PithomLabs/miniapp/testkit"
	"github.com/PithomLabs/miniapp/testkit/mockcap"

	"github.com/PithomLabs/miniapp/internal/store"
	"github.com/jackc/pgx/v5"
)

// p28PolicySvc mirrors newAuthService with a caller-owned policy
// handle, so tests can swap grants mid-flight.
func p28PolicySvc(t *testing.T, db *testkit.DB, policy *auth.Policy) *effects.Service {
	t.Helper()
	reg := registry.New()
	testkit.RegisterClassATool(t, reg, authzMiniapp, authzTool, "", nil,
		func(ctx context.Context, call registry.ToolCall) (json.RawMessage, error) {
			return json.RawMessage(`{}`), nil
		})
	return effects.New(effects.Config{Pool: db.Pool,
		Channels:  map[string]effects.Channel{"mock": mockcap.New(db.Pool)},
		Canceller: allowCancellation{},
		Policy:    allowPolicy{},
		Registry:  reg,
		Grants:    policy,
		Approver:  approvals.New(db.Pool),
		Auditor:   audit.New(db.Pool, nil),
	})
}

func p28GrantsWith() map[string]map[string]contract.TrustMode {
	return map[string]map[string]contract.TrustMode{
		wsTenant: {authzMiniapp: contract.TrustAutonomous},
	}
}

// p28AssertRefusalEvidence pins the refusal contract P29-001
// requires: the claimed row stays PENDING with no provider
// outcome recorded, and the attributable refusal audits exactly
// one effect-refused event.
func p28AssertRefusalEvidence(t *testing.T, db *testkit.DB, op string) {
	t.Helper()
	v := p28ReadRow(t, db, op)
	if v.state != "PENDING" {
		t.Errorf("row state = %q, want PENDING (refusal must not transition)", v.state)
	}
	if v.req != "" || v.reply != "" {
		t.Errorf("provider outcome = (%q, %q), want empty (refusal must not record)", v.req, v.reply)
	}
	if n := p28RefusedAudits(t, db, op); n != 1 {
		t.Errorf("effect-refused audits = %d, want 1", n)
	}
}

// Kill-switch scopes: global, tenant, miniapp, capability. Each
// engaged scope refuses with typed kill-switch classification;
// releasing restores success. (Tenant scope additionally
// covered e2e by TestKillSwitchBlocks.)
func TestP28KillSwitchScopes(t *testing.T) {
	scopes := []struct {
		name string
		eng  string
	}{
		{"global", "global"},
		{"tenant", "tenant:" + wsTenant},
		{"miniapp", "miniapp:" + authzMiniapp},
		{"capability", "capability:mock"},
	}
	for _, tc := range scopes {
		t.Run(tc.name, func(t *testing.T) {
			svc, db := setupMock(t)
			seedLiveInvocation(t, db, wsTenant, "inv-p28sw-"+tc.name)
			ctx := context.Background()
			intent := mockIntent("p28sw-"+tc.name, mockcap.ModeAccept)
			if err := svc.SetKillSwitch(ctx, tc.eng, true); err != nil {
				t.Fatal(err)
			}
			_, err := svc.Perform(ctx, wsTenant, intent)
			var refusal *effects.RefusalError
			if !errors.As(err, &refusal) {
				t.Fatalf("err = %v, want terminal *RefusalError", err)
			}
			if refusal.Reason != effects.RefusalKillSwitch {
				t.Errorf("reason = %s, want kill-switch", refusal.Reason)
			}
			if n := mockcap.SentCount(t, db.Pool, "p28sw-"+tc.name); n != 0 {
				t.Errorf("sends = %d, want 0", n)
			}
			p28AssertRefusalEvidence(t, db, intent.BusinessOpID)
			if err := svc.SetKillSwitch(ctx, tc.eng, false); err != nil {
				t.Fatal(err)
			}
			// The refused claim left a fresh PENDING row; age it
			// so the retry takes over instead of contending.
			p28AgeRow(t, db, intent.BusinessOpID)
			rec, err := svc.Perform(ctx, wsTenant, intent)
			if err != nil || rec.State != contract.EffectConfirmed {
				t.Fatalf("released = %+v %v, want CONFIRMED", rec, err)
			}
		})
	}
}

// Tool-declared kill switch: a manifest-declared switch name
// gates the tool's effects like any other scope.
func TestP28ToolDeclaredKillSwitch(t *testing.T) {
	db := testkit.NewDB(t)
	truncateLedger(t, db)
	mockcap.EnsureLog(t, db.Pool)
	mockcap.TruncateLog(t, db.Pool)
	reg := registry.New()
	mf := testkit.ClassAManifest("t-p28sw", "t.p28sw", "", nil)
	mf.Tools[0].Limits.KillSwitch = "tool:p28-declared"
	impl := testkit.ClassAImpl("test", "t.p28sw",
		func(ctx context.Context, call registry.ToolCall) (json.RawMessage, error) {
			return json.RawMessage(`{}`), nil
		})
	if err := reg.Register(mf, impl, testkit.Envelope(t, mf, "test")); err != nil {
		t.Fatal(err)
	}
	policy := auth.NewPolicy(nil, map[string]map[string]contract.TrustMode{
		wsTenant: {"t-p28sw": contract.TrustAutonomous},
	})
	svc := effects.New(effects.Config{Pool: db.Pool,
		Channels:  map[string]effects.Channel{"mock": mockcap.New(db.Pool)},
		Canceller: allowCancellation{}, Policy: allowPolicy{},
		Registry: reg, Grants: policy,
		Approver: approvals.New(db.Pool), Auditor: audit.New(db.Pool, nil),
	})
	ctx := context.Background()
	inv := authzInvID(wsTenant, "inv-p28swdecl")
	seedInvocationFor(t, db, wsTenant, inv, "t-p28sw", "t.p28sw")
	intent := effects.Intent{BusinessOpID: wsTenant + ":p28swdecl", AttemptID: "a1",
		InvocationID: inv, EffectToken: authzToken,
		Operation: effects.Operation{CapabilityID: "mock", Operation: "send",
			Payload: mockPayload(mockcap.ModeAccept, "p28swdecl")}}
	if err := svc.SetKillSwitch(ctx, "tool:p28-declared", true); err != nil {
		t.Fatal(err)
	}
	_, err := svc.Perform(ctx, wsTenant, intent)
	var refusal *effects.RefusalError
	if !errors.As(err, &refusal) || refusal.Reason != effects.RefusalKillSwitch {
		t.Fatalf("err = %v, want kill-switch refusal", err)
	}
	if n := mockcap.SentCount(t, db.Pool, "p28swdecl"); n != 0 {
		t.Fatalf("sends at refusal = %d, want 0", n)
	}
	p28AssertRefusalEvidence(t, db, intent.BusinessOpID)
	if err := svc.SetKillSwitch(ctx, "tool:p28-declared", false); err != nil {
		t.Fatal(err)
	}
	p28AgeRow(t, db, intent.BusinessOpID)
	if rec, err := svc.Perform(ctx, wsTenant, intent); err != nil || rec.State != contract.EffectConfirmed {
		t.Fatalf("released = %+v %v, want CONFIRMED", rec, err)
	}
}

func p28AgeRow(t *testing.T, db *testkit.DB, op string) {
	t.Helper()
	_, err := store.WithTenant(context.Background(), db.Pool, wsTenant, func(tx pgx.Tx) (bool, error) {
		_, err := tx.Exec(context.Background(), `UPDATE app.effect_records
			SET updated_at = now() - interval '1 hour'
			WHERE business_op_id = $1 AND tenant_id = $2`, op, wsTenant)
		return true, err
	})
	if err != nil {
		t.Fatal(err)
	}
}

func seedInvocationFor(t *testing.T, db *testkit.DB, tenant, inv, miniapp, tool string) {
	t.Helper()
	_, err := store.WithTenant(context.Background(), db.Pool, tenant, func(tx pgx.Tx) (bool, error) {
		_, err := tx.Exec(context.Background(), `INSERT INTO app.invocations
			(id, tenant_id, miniapp_id, miniapp_version, tool, idempotency_key, input,
			 status, trust_mode, effect_token)
			VALUES ($1, $2, $3, '1.0.0', $4, $5, '{}', 'running', 'autonomous', $6)`,
			inv, tenant, miniapp, tool, "key-"+inv, authzToken)
		return true, err
	})
	if err != nil {
		t.Fatal(err)
	}
}

// Typed grant allow/deny pair at the authorize boundary.
func TestP28GrantAllowDenyTyped(t *testing.T) {
	db := testkit.NewDB(t)
	truncateLedger(t, db)
	mockcap.EnsureLog(t, db.Pool)
	mockcap.TruncateLog(t, db.Pool)
	ctx := context.Background()

	allowed := p28PolicySvc(t, db, auth.NewPolicy(nil, p28GrantsWith()))
	seedLiveInvocation(t, db, wsTenant, "inv-p28grant-allow")
	rec, err := allowed.Perform(ctx, wsTenant, mockIntent("p28grant-allow", mockcap.ModeAccept))
	if err != nil || rec.State != contract.EffectConfirmed {
		t.Fatalf("allowed = %+v %v, want CONFIRMED", rec, err)
	}

	denied := p28PolicySvc(t, db, auth.NewPolicy(nil, map[string]map[string]contract.TrustMode{}))
	seedLiveInvocation(t, db, wsTenant, "inv-p28grant-deny")
	_, err = denied.Perform(ctx, wsTenant, mockIntent("p28grant-deny", mockcap.ModeAccept))
	var refusal *effects.RefusalError
	if !errors.As(err, &refusal) {
		t.Fatalf("err = %v, want terminal *RefusalError", err)
	}
	if refusal.Reason != effects.RefusalGrantDenied {
		t.Errorf("reason = %s, want grant-denied", refusal.Reason)
	}
	if n := mockcap.SentCount(t, db.Pool, "p28grant-deny"); n != 0 {
		t.Errorf("sends = %d, want 0", n)
	}
	p28AssertRefusalEvidence(t, db, wsTenant+":p28grant-deny")
}

// Grant snapshot replacement racing authorization: every
// outcome is either success under a held snapshot or a typed
// grant-denied refusal — never partial, never another error.
// Run under -race; the detector plus exact accounting is the
// proof (in-memory swaps have no DB order to force).
func TestP28GrantSwapRace(t *testing.T) {
	db := testkit.NewDB(t)
	truncateLedger(t, db)
	mockcap.EnsureLog(t, db.Pool)
	mockcap.TruncateLog(t, db.Pool)
	ctx := context.Background()
	policy := auth.NewPolicy(nil, p28GrantsWith())
	svc := p28PolicySvc(t, db, policy)
	const workers = 8
	for i := 0; i < workers; i++ {
		seedLiveInvocation(t, db, wsTenant, fmt.Sprintf("inv-p28swap-%d", i))
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for j := 0; j < 40; j++ {
			if j%2 == 0 {
				policy.SetGrants(map[string]map[string]contract.TrustMode{})
			} else {
				policy.SetGrants(p28GrantsWith())
			}
		}
	}()
	type outcome struct {
		i   int
		rec contract.EffectRecord
		err error
	}
	out := make(chan outcome, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			op := fmt.Sprintf("p28swap-%d", i)
			rec, err := svc.Perform(ctx, wsTenant, mockIntent(op, mockcap.ModeAccept))
			out <- outcome{i, rec, err}
		}(i)
	}
	wg.Wait()
	close(out)
	confirmed := 0
	for o := range out {
		if o.err == nil {
			if o.rec.State != contract.EffectConfirmed {
				t.Errorf("worker %d success state = %s, want CONFIRMED", o.i, o.rec.State)
			}
			confirmed++
			continue
		}
		var refusal *effects.RefusalError
		if !errors.As(o.err, &refusal) || refusal.Reason != effects.RefusalGrantDenied {
			t.Errorf("worker %d err = %v, want nil or grant-denied", o.i, o.err)
		}
	}
	sends := 0
	for i := 0; i < workers; i++ {
		sends += mockcap.SentCount(t, db.Pool, fmt.Sprintf("p28swap-%d", i))
	}
	if sends != confirmed {
		t.Errorf("sends = %d, confirmed = %d; every send needs its success", sends, confirmed)
	}
}

// Capability-identity mismatch on reuse refuses (operation and
// payload mismatch are pinned by TestOperationReuseConflict).
func TestP28CapabilityReuseConflict(t *testing.T) {
	svc, db := setupMock(t)
	seedLiveInvocation(t, db, wsTenant, "inv-p28cap")
	ctx := context.Background()
	first := mockIntent("p28cap", mockcap.ModeAccept)
	if _, err := svc.Perform(ctx, wsTenant, first); err != nil {
		t.Fatalf("first: %v", err)
	}
	second := mockIntent("p28cap", mockcap.ModeAccept)
	second.Operation.CapabilityID = "other-cap"
	if _, err := svc.Perform(ctx, wsTenant, second); err == nil {
		t.Fatal("same op ID with different capability accepted")
	}
}

// isTrustBearingName reports whether a Go field name plausibly
// carries trust, under normalized case-insensitive detection:
// lowercased with separators stripped, then a "trust" substring
// match. Novel spellings (TrustLevel, trustMode, trust_level)
// cannot evade by casing or separators; ordinary fields stay
// silent (P29-003).
func isTrustBearingName(name string) bool {
	norm := strings.ToLower(name)
	norm = strings.ReplaceAll(norm, "_", "")
	norm = strings.ReplaceAll(norm, "-", "")
	norm = strings.ReplaceAll(norm, " ", "")
	return strings.Contains(norm, "trust")
}

// Structural anti-spoofing guard: effects.Intent must never
// grow a trust-carrying field; persisted row trust is the sole
// trust source (P26-001). Fails loudly on introduction.
func TestP28IntentCarriesNoTrust(t *testing.T) {
	ty := reflect.TypeOf(effects.Intent{})
	for i := 0; i < ty.NumField(); i++ {
		name := ty.Field(i).Name
		if isTrustBearingName(name) {
			t.Fatalf("effects.Intent gained trust field %q; persisted row trust is the sole source", name)
		}
	}
}

// TestP28TrustBearingNameDetection pins the detector's contract:
// the prompt30 §4 required names plus adversarial variants flag,
// while the seven real Intent fields and ordinary lookalikes
// (including "rust", which is not "trust") stay silent.
func TestP28TrustBearingNameDetection(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		// Required prompt30 §4 positives.
		{"TrustLevel", true},
		{"RequestedTrustMode", true},
		{"trustMode", true},
		{"TrustPolicy", true},
		// Adversarial: lowercase with separator — evades naive
		// CamelCase matching, caught by normalization.
		{"trust_level", true},
		// Extra positives: legacy exact names, casing, embedding.
		{"Trust", true},
		{"TrustMode", true},
		{"RequestedTrust", true},
		{"TrustClaim", true},
		{"TRUST", true},
		{"Distrust", true},
		// Negatives: the seven real Intent fields, pinned
		// literally so a rename re-proves zero false positives.
		{"BusinessOpID", false},
		{"AttemptID", false},
		{"IdempotencyKey", false},
		{"InvocationID", false},
		{"EffectToken", false},
		{"Operation", false},
		{"Sequence", false},
		// Ordinary unrelated names and near-miss lookalikes.
		{"Autonomous", false},
		{"Metadata", false},
		{"Payload", false},
		{"RustLevel", false},
		{"TrussBridge", false},
	}
	for _, tc := range cases {
		if got := isTrustBearingName(tc.name); got != tc.want {
			t.Errorf("isTrustBearingName(%q) = %v, want %v", tc.name, got, tc.want)
		}
	}
}
