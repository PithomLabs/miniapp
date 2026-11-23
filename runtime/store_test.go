package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/internal/store"
	"github.com/PithomLabs/miniapp/testkit"
)

// completeTxWon drives the live completion op inside a tenant
// transaction (P21-001: completeIfLive deleted; completeTx is the
// sole completion).
func completeTxWon(t *testing.T, s *invocations, tenant, id string, status contract.RunStatus, result json.RawMessage, errMsg string) (bool, error) {
	t.Helper()
	return store.WithTenant(context.Background(), s.pool, tenant, func(tx pgx.Tx) (bool, error) {
		return s.completeTx(context.Background(), tx, tenant, id, status, result, errMsg)
	})
}

func claimRow(t *testing.T, s *invocations, tenant, key, initial string) invocation {
	t.Helper()
	rec, created, err := s.claim(context.Background(), tenant, InvokeRequest{
		MiniappID: "m", Version: "1.0.0", Tool: "t.t",
		Input: json.RawMessage(`{}`), IdempotencyKey: key,
	}, "00000000-0000-0000-0000-"+keySuffix(key), "tok-"+key, contract.RunStatus(initial))
	if err != nil || !created {
		t.Fatalf("claim %s = %v, %v", key, rec, err)
	}
	return rec
}

// keySuffix derives a deterministic UUID suffix from a short test
// key (hex-padded, 12 chars).
func keySuffix(key string) string {
	var b strings.Builder
	for _, c := range key {
		if len(b.String()) >= 12 {
			break
		}
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') {
			b.WriteRune(c)
		} else {
			b.WriteRune('0')
		}
	}
	for len(b.String()) < 12 {
		b.WriteByte('0')
	}
	return b.String()
}

// TestPredicatedOpsRejectCrossTenant ports the tenant-predicate
// proof to the M-01 predicated ops: a write scoped to another
// tenant matches nothing on EVERY op, even where RLS is bypassed.
func TestPredicatedOpsRejectCrossTenant(t *testing.T) {
	db := testkit.NewDB(t)
	db.Truncate(t, "app.invocations")
	s := &invocations{pool: db.Pool}
	ctx := context.Background()
	running := claimRow(t, s, "tenant-a", "k1", string(contract.RunRunning))
	waiting := claimRow(t, s, "tenant-a", "k2", string(contract.RunWaitingApproval))

	// Cross-tenant attempts on the running row: all lose.
	if won, err := completeTxWon(t, s, "tenant-b", running.ID, contract.RunSucceeded, json.RawMessage(`{}`), ""); err != nil || won {
		t.Fatalf("cross-tenant completeTx = %v, %v; want false, nil", won, err)
	}
	if won, err := s.syncIfLive(ctx, "tenant-b", running.ID, contract.RunSucceeded); err != nil || won {
		t.Fatalf("cross-tenant syncIfLive = %v, %v; want false, nil", won, err)
	}
	if won, err := s.cancelIfLive(ctx, "tenant-b", running.ID, "", "x"); err != nil || won {
		t.Fatalf("cross-tenant cancelIfLive = %v, %v; want false, nil", won, err)
	}
	if won, err := s.failIfWaiting(ctx, "tenant-b", running.ID, "", "x"); err != nil || won {
		t.Fatalf("cross-tenant failIfWaiting = %v, %v; want false, nil", won, err)
	}
	// Cross-tenant attempts on the waiting row: failIfWaiting's
	// from-state WOULD allow the write, so losing proves the
	// tenant predicate specifically.
	if won, err := s.failIfWaiting(ctx, "tenant-b", waiting.ID, "", "x"); err != nil || won {
		t.Fatalf("cross-tenant failIfWaiting(waiting) = %v, %v; want false, nil", won, err)
	}
	if won, err := completeTxWon(t, s, "tenant-b", waiting.ID, contract.RunSucceeded, json.RawMessage(`{}`), ""); err != nil || won {
		t.Fatalf("cross-tenant completeTx(waiting) = %v, %v; want false, nil", won, err)
	}
	for _, tc := range []struct {
		id   string
		want contract.RunStatus
	}{{running.ID, contract.RunRunning}, {waiting.ID, contract.RunWaitingApproval}} {
		got, err := s.getByID(ctx, "tenant-a", tc.id)
		if err != nil {
			t.Fatal(err)
		}
		if got.Status != tc.want {
			t.Fatalf("cross-tenant write changed status to %s, want %s", got.Status, tc.want)
		}
	}
	// Own-tenant controls: both ops win from their live states.
	if won, err := completeTxWon(t, s, "tenant-a", running.ID, contract.RunSucceeded, json.RawMessage(`{"ok":true}`), ""); err != nil || !won {
		t.Fatalf("own-tenant completeTx = %v, %v; want true, nil", won, err)
	}
	if won, err := s.failIfWaiting(ctx, "tenant-a", waiting.ID, "", "denied"); err != nil || !won {
		t.Fatalf("own-tenant failIfWaiting = %v, %v; want true, nil", won, err)
	}
	got, err := s.getByID(ctx, "tenant-a", running.ID)
	if err != nil {
		t.Fatal(err)
	}
	var res map[string]any
	if err := json.Unmarshal(got.Result, &res); err != nil {
		t.Fatalf("own-tenant complete result not JSON: %v", err)
	}
	if got.Status != contract.RunSucceeded || res["ok"] != true {
		t.Fatalf("own-tenant complete left %s %s, want succeeded result", got.Status, got.Result)
	}
}

// TestPredicatedOpsFromStates pins each op's permitted source
// states: wrong-from-state attempts lose without touching the row,
// terminal records are immutable, and quarantined rows refuse
// cancellation (CR-05) at the op level.
func TestPredicatedOpsFromStates(t *testing.T) {
	db := testkit.NewDB(t)
	db.Truncate(t, "app.invocations")
	s := &invocations{pool: db.Pool}
	ctx := context.Background()

	running := claimRow(t, s, "tenant-a", "f1", string(contract.RunRunning))
	if won, err := s.failIfWaiting(ctx, "tenant-a", running.ID, "", "x"); err != nil || won {
		t.Fatalf("failIfWaiting(running) = %v, %v; want false, nil", won, err)
	}
	if won, err := s.syncIfLive(ctx, "tenant-a", running.ID, contract.RunSucceeded); err != nil || !won {
		t.Fatalf("syncIfLive(running) = %v, %v; want true, nil", won, err)
	}
	got, err := s.getByID(ctx, "tenant-a", running.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != contract.RunSucceeded {
		t.Fatalf("syncIfLive left %s, want succeeded", got.Status)
	}
	// Terminal record: every op loses, nothing changes.
	for name, op := range map[string]func() (bool, error){
		"completeTx": func() (bool, error) {
			return completeTxWon(t, s, "tenant-a", running.ID, contract.RunFailed, json.RawMessage(`{}`), "late")
		},
		"syncIfLive": func() (bool, error) {
			return s.syncIfLive(ctx, "tenant-a", running.ID, contract.RunCancelled)
		},
		"cancelIfLive": func() (bool, error) {
			return s.cancelIfLive(ctx, "tenant-a", running.ID, "", "late")
		},
		"failIfWaiting": func() (bool, error) {
			return s.failIfWaiting(ctx, "tenant-a", running.ID, "", "late")
		},
	} {
		if won, err := op(); err != nil || won {
			t.Fatalf("%s(terminal) = %v, %v; want false, nil", name, won, err)
		}
	}
	got, err = s.getByID(ctx, "tenant-a", running.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != contract.RunSucceeded {
		t.Fatalf("terminal record moved to %s", got.Status)
	}

	// Cancel happy path from waiting, then terminal immutability.
	waiting := claimRow(t, s, "tenant-a", "f2", string(contract.RunWaitingApproval))
	if won, err := s.cancelIfLive(ctx, "tenant-a", waiting.ID, "", "operator"); err != nil || !won {
		t.Fatalf("cancelIfLive(waiting) = %v, %v; want true, nil", won, err)
	}
	got, err = s.getByID(ctx, "tenant-a", waiting.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != contract.RunCancelled || got.Err != "operator" {
		t.Fatalf("cancel left %s err %q, want cancelled/operator", got.Status, got.Err)
	}
	if won, err := s.syncIfLive(ctx, "tenant-a", waiting.ID, contract.RunSucceeded); err != nil || won {
		t.Fatalf("syncIfLive(cancelled) = %v, %v; want false, nil", won, err)
	}

	// Quarantined rows: cancellation refused (CR-05), no op moves
	// them; exit is via operator resume/fail only.
	quar := claimRow(t, s, "tenant-a", "f3", string(contract.RunQuarantined))
	for name, op := range map[string]func() (bool, error){
		"cancelIfLive": func() (bool, error) {
			return s.cancelIfLive(ctx, "tenant-a", quar.ID, "", "x")
		},
		"completeTx": func() (bool, error) {
			return completeTxWon(t, s, "tenant-a", quar.ID, contract.RunSucceeded, json.RawMessage(`{}`), "")
		},
		"syncIfLive": func() (bool, error) {
			return s.syncIfLive(ctx, "tenant-a", quar.ID, contract.RunSucceeded)
		},
	} {
		if won, err := op(); err != nil || won {
			t.Fatalf("%s(quarantined) = %v, %v; want false, nil", name, won, err)
		}
	}
	got, err = s.getByID(ctx, "tenant-a", quar.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != contract.RunQuarantined {
		t.Fatalf("quarantined record moved to %s", got.Status)
	}
}

// TestPredicatedOpsRejectCrossTenantBypassRole proves the tenant
// predicate — not RLS — rejects cross-tenant writes on every M-01
// op: the writes run as a BYPASSRLS role for which policies do not
// apply.
func TestPredicatedOpsRejectCrossTenantBypassRole(t *testing.T) {
	db := testkit.NewDB(t)
	db.Truncate(t, "app.invocations")
	s := &invocations{pool: db.Pool}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	running := claimRow(t, s, "tenant-a", "b1", string(contract.RunRunning))
	waiting := claimRow(t, s, "tenant-a", "b2", string(contract.RunWaitingApproval))
	// Neon enforces password strength on role creation; this value
	// satisfies length and character-class rules on both engines.
	const role, pw = "ws_bypass", "WsBypass#2026!Test$09"
	u, err := url.Parse(db.DSN)
	if err != nil {
		t.Fatal(err)
	}
	dbName := strings.TrimPrefix(u.Path, "/")
	dropRole := func() error {
		// Fresh context: cleanup runs after the test's own context
		// is canceled, and must still report honestly.
		cctx, ccancel := context.WithTimeout(context.Background(), time.Minute)
		defer ccancel()
		// DROP OWNED works on superuser engines; managed engines
		// without it need explicit revokes before the drop.
		_, _ = db.Pool.Exec(cctx, `DROP OWNED BY `+role)
		for _, revoke := range []string{
			`REVOKE ALL ON app.invocations FROM ` + role,
			`REVOKE USAGE ON SCHEMA app FROM ` + role,
			`REVOKE CONNECT ON DATABASE ` + dbName + ` FROM ` + role,
		} {
			_, _ = db.Pool.Exec(cctx, revoke)
		}
		_, err := db.Pool.Exec(cctx, `DROP ROLE IF EXISTS `+role)
		return err
	}
	if err := dropRole(); err != nil {
		t.Fatalf("drop leftover role: %v", err)
	}
	t.Cleanup(func() {
		if err := dropRole(); err != nil {
			t.Errorf("drop test role: %v", err)
		}
	})
	if _, err := db.Pool.Exec(ctx, `CREATE ROLE `+role+` BYPASSRLS LOGIN PASSWORD '`+pw+`'`); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "42501" {
			t.Skipf("bypass role unavailable here (managed PG without superuser): %v", err)
		}
		t.Fatal(err)
	}
	for _, grant := range []string{
		`GRANT CONNECT ON DATABASE ` + dbName + ` TO ` + role,
		`GRANT USAGE ON SCHEMA app TO ` + role,
		`GRANT SELECT, UPDATE ON app.invocations TO ` + role,
	} {
		if _, err := db.Pool.Exec(ctx, grant); err != nil {
			t.Fatal(err)
		}
	}
	u.User = url.UserPassword(role, pw)
	if u.Hostname() == "" {
		t.Fatal("DSN has no hostname")
	}
	bypassPool, err := pgxpool.New(ctx, u.String())
	if err != nil {
		t.Fatal(err)
	}
	defer bypassPool.Close()
	// Sanity: the bypass role really bypasses RLS (sees the row
	// without tenant context).
	var seen string
	if err := bypassPool.QueryRow(ctx, `SELECT tenant_id FROM app.invocations WHERE id = $1::uuid`, running.ID).Scan(&seen); err != nil {
		t.Fatalf("bypass role cannot read row (RLS not bypassed?): %v", err)
	}
	bs := &invocations{pool: bypassPool}
	if won, err := completeTxWon(t, bs, "tenant-b", running.ID, contract.RunSucceeded, json.RawMessage(`{}`), ""); err != nil || won {
		t.Fatalf("bypass completeTx = %v, %v; want false, nil", won, err)
	}
	if won, err := bs.syncIfLive(ctx, "tenant-b", running.ID, contract.RunSucceeded); err != nil || won {
		t.Fatalf("bypass syncIfLive = %v, %v; want false, nil", won, err)
	}
	if won, err := bs.cancelIfLive(ctx, "tenant-b", running.ID, "", "x"); err != nil || won {
		t.Fatalf("bypass cancelIfLive = %v, %v; want false, nil", won, err)
	}
	if won, err := bs.failIfWaiting(ctx, "tenant-b", waiting.ID, "", "x"); err != nil || won {
		t.Fatalf("bypass failIfWaiting(waiting) = %v, %v; want false, nil", won, err)
	}
	for _, tc := range []struct {
		id   string
		want contract.RunStatus
	}{{running.ID, contract.RunRunning}, {waiting.ID, contract.RunWaitingApproval}} {
		got, err := s.getByID(ctx, "tenant-a", tc.id)
		if err != nil {
			t.Fatal(err)
		}
		if got.Status != tc.want {
			t.Fatalf("bypass-role cross-tenant write changed status to %s, want %s (predicate failed)", got.Status, tc.want)
		}
	}
}
