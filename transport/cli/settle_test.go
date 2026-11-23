package cli_test

// Prompt28.1 — settle verb tests. Flag misuse is exercised
// without a service (parse failures never reach it); one
// end-to-end run through Runner proves the verb wires flags to
// the shared SettleEffect boundary with no independent
// authorization logic.

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/PithomLabs/miniapp/auth"
	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/effects"
	"github.com/PithomLabs/miniapp/internal/store"
	"github.com/PithomLabs/miniapp/registry"
	"github.com/PithomLabs/miniapp/testkit"
	"github.com/PithomLabs/miniapp/testkit/mockcap"
	"github.com/PithomLabs/miniapp/testkit/stack"
	"github.com/PithomLabs/miniapp/transport/cli"
)

func TestP281SettleVerbFlags(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		{"missing-op", []string{"settle", "--to", "confirmed", "--note", "e", "--credential", "k"}},
		{"missing-to", []string{"settle", "--op", "t:o", "--note", "e", "--credential", "k"}},
		{"missing-note", []string{"settle", "--op", "t:o", "--to", "confirmed", "--credential", "k"}},
		{"bad-outcome", []string{"settle", "--op", "t:o", "--to", "bogus", "--note", "e", "--credential", "k"}},
		// No missing-credential row here: the CLI passes credentials
		// through untouched (no independent logic) and the shared
		// boundary refuses empty/unknown credentials — covered by
		// the service matrix (auth-required legs).
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			r := &cli.Runner{Stdout: &stdout, Stderr: &stderr}
			if code := r.Run(context.Background(), tc.args); code == 0 {
				t.Errorf("exit = 0, want nonzero for %v", tc.args)
			}
			if stderr.Len() == 0 {
				t.Errorf("empty stderr for %v, want usage/error text", tc.args)
			}
		})
	}
}

func TestP281SettleVerbEndToEnd(t *testing.T) {
	db := testkit.NewDB(t)
	db.Truncate(t, "app.invocations", "app.audit_events", "app.effect_records")
	policy := auth.NewPolicy(
		map[string]auth.Binding{
			"key-cli-op": {AgentID: "cli-op", Authority: contract.Authority{
				Kind: contract.AuthorityUser, ID: "cli-op", TenantID: "tenant-cli",
			}, Roles: []string{"miniapp.settle.operator"}},
			"key-cli-agent": {AgentID: "cli-agent", Authority: contract.Authority{
				Kind: contract.AuthorityUser, ID: "cli-agent", TenantID: "tenant-cli",
			}},
		},
		map[string]map[string]contract.TrustMode{},
	)
	fx := stack.Assemble(t, stack.Config{DB: db,
		Register: func(t *testing.T, reg *registry.Registry) {},
		Policy:   policy,
		Channels: map[string]effects.Channel{"mock": mockcap.New(db.Pool)},
	})
	ctx := context.Background()
	op := "tenant-cli:cli-settle-1"
	_, err := store.WithTenant(ctx, db.Pool, "tenant-cli", func(tx pgx.Tx) (bool, error) {
		var inv string
		err := tx.QueryRow(ctx, `INSERT INTO app.invocations
			(id, tenant_id, miniapp_id, miniapp_version, tool, idempotency_key, input,
			 status, trust_mode, effect_token)
			VALUES (gen_random_uuid(), 'tenant-cli', 'cli-mini', '1.0.0', 'cli.tool',
			 'key-cli-1', '{}', 'running', 'autonomous', 'tok-cli')
			RETURNING id::text`).Scan(&inv)
		if err != nil {
			return false, err
		}
		_, err = tx.Exec(ctx, `INSERT INTO app.effect_records
			(business_op_id, tenant_id, attempt_id, idempotency_key, invocation_id,
			 capability_id, operation, payload, state)
			VALUES ($1, 'tenant-cli', 'a-cli', '', $2, 'mock', 'send', '{}', 'UNKNOWN')`,
			op, inv)
		return true, err
	})
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	r := &cli.Runner{Svc: fx.Svc, Stdout: &stdout, Stderr: &stderr}
	code := r.Run(ctx, []string{"settle", "--op", op, "--to", "confirmed",
		"--note", "cli evidence", "--credential", "key-cli-op"})
	if code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr: %s)", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "CONFIRMED") {
		t.Errorf("stdout lacks CONFIRMED record: %q", stdout.String())
	}
	var st, agent string
	_, err = store.WithTenant(ctx, db.Pool, "tenant-cli", func(tx pgx.Tx) (bool, error) {
		err := tx.QueryRow(ctx, `SELECT state FROM app.effect_records
			WHERE business_op_id = $1`, op).Scan(&st)
		if err != nil {
			return false, err
		}
		return true, tx.QueryRow(ctx, `SELECT agent_id FROM app.audit_events
			WHERE tenant_id = 'tenant-cli' AND action = 'effect-settled'
			AND details->>'business_op_id' = $1`, op).Scan(&agent)
	})
	if err != nil {
		t.Fatal(err)
	}
	if st != "CONFIRMED" || agent != "cli-op" {
		t.Errorf("persisted = (%s, %s), want (CONFIRMED, cli-op)", st, agent)
	}
	// A non-operator through the same verb is refused by the
	// shared boundary (exit 1, no independent CLI logic).
	var stdout2, stderr2 bytes.Buffer
	r2 := &cli.Runner{Svc: fx.Svc, Stdout: &stdout2, Stderr: &stderr2}
	if code := r2.Run(ctx, []string{"settle", "--op", op, "--to", "failed",
		"--note", "x", "--credential", "key-cli-agent"}); code == 0 {
		t.Errorf("non-operator exit = 0, want nonzero")
	}
}
