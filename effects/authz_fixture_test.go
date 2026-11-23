// Shared authorization fixture: every Perform authorizes, so
// ledger-mechanics tests wire real authorization dependencies
// (registry, grants, approver, auditor) and seed live invocations
// matching their intents. The fixture tool is ungated, keeping
// approval linkage out of ledger-focused tests (approval paths
// are covered by the authorization matrix suite).
package effects_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/PithomLabs/miniapp/approvals"
	"github.com/PithomLabs/miniapp/audit"
	"github.com/PithomLabs/miniapp/auth"
	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/effects"
	"github.com/PithomLabs/miniapp/internal/store"
	"github.com/PithomLabs/miniapp/registry"
	"github.com/PithomLabs/miniapp/testkit"
)

const (
	authzMiniapp = "t-auth-fix"
	authzTool    = "t.auth"
	authzToken   = "tok-authz-fixture"
)

// authzPolicy grants every tenant used by effects tests autonomous
// trust on the fixture miniapp.
func authzPolicy() *auth.Policy {
	grants := map[string]map[string]contract.TrustMode{}
	for _, tenant := range []string{"t-a", "t-b", "t-effect", "t-ws", "tenant-a", "tenant-b"} {
		grants[tenant] = map[string]contract.TrustMode{authzMiniapp: contract.TrustAutonomous}
	}
	return auth.NewPolicy(map[string]auth.Binding{}, grants)
}

// newAuthService builds a fully wired effects service with the
// caller's channels/canceller/policy plus registry, grants,
// approver, and auditor for the authorization boundary.
func newAuthService(t *testing.T, db *testkit.DB, channels map[string]effects.Channel, canceller effects.Cancellation, policy effects.PolicyChecker) *effects.Service {
	t.Helper()
	reg := registry.New()
	testkit.RegisterClassATool(t, reg, authzMiniapp, authzTool, "", nil,
		func(ctx context.Context, call registry.ToolCall) (json.RawMessage, error) {
			return json.RawMessage(`{}`), nil
		})
	return effects.New(effects.Config{Pool: db.Pool,
		Channels:  channels,
		Canceller: canceller,
		Policy:    policy,
		Registry:  reg,
		Grants:    authzPolicy(),
		Approver:  approvals.New(db.Pool),
		Auditor:   audit.New(db.Pool, nil),
	})
}

// authzInvID deterministically maps a test-local invocation name to
// the UUID primary key authorization looks up (stable across
// processes, so crash workers agree with parents without coordination).
func authzInvID(tenant, local string) string {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(tenant+"\x00"+local)).String()
}

// seedLiveInvocation inserts a running invocation row the fixture
// tool owns, addressable as authzInvID(tenant, local). Intents
// must carry that ID plus authzToken.
func seedLiveInvocation(t *testing.T, db *testkit.DB, tenant, local string) {
	t.Helper()
	_, err := store.WithTenant(context.Background(), db.Pool, tenant, func(tx pgx.Tx) (bool, error) {
		_, err := tx.Exec(context.Background(), `INSERT INTO app.invocations
			(id, tenant_id, miniapp_id, miniapp_version, tool, idempotency_key, input,
			 status, trust_mode, effect_token)
			VALUES ($1, $2, $3, '1.0.0', $4, $5, '{}', 'running', 'autonomous', $6)`,
			authzInvID(tenant, local), tenant, authzMiniapp, authzTool, "key-"+local, authzToken)
		return true, err
	})
	if err != nil {
		t.Fatal(err)
	}
}

// truncateLedger resets ledger-adjacent tables between tests that
// seed invocations (the shared dev database persists rows across
// tests in one package run).
func truncateLedger(t *testing.T, db *testkit.DB) {
	t.Helper()
	db.Truncate(t, "app.effect_records", "app.invocations", "app.audit_events")
}
