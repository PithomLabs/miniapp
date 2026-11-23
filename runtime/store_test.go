package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PithomLabs/miniapp/contract"
	"github.com/PithomLabs/miniapp/testkit"
)

// TestUpdateRejectsCrossTenant proves the tenant predicate on
// invocation updates: a write scoped to another tenant changes
// nothing, even where RLS might be bypassed.
func TestUpdateRejectsCrossTenant(t *testing.T) {
	db := testkit.NewDB(t)
	db.Truncate(t, "app.invocations")
	s := &invocations{pool: db.Pool}
	ctx := context.Background()
	rec, created, err := s.claim(ctx, "tenant-a", InvokeRequest{
		MiniappID: "m", Version: "1.0.0", Tool: "t.t",
		Input: json.RawMessage(`{}`), IdempotencyKey: "k1",
	}, "00000000-0000-0000-0000-000000000001", "tok-1", contract.RunRunning)
	if err != nil || !created {
		t.Fatalf("claim = %v, %v", rec, err)
	}
	if err := s.update(ctx, "tenant-b", rec.ID, contract.RunSucceeded, json.RawMessage(`{}`), "", ""); err != nil {
		t.Fatalf("cross-tenant update errored: %v", err)
	}
	got, err := s.getByID(ctx, "tenant-a", rec.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != contract.RunRunning {
		t.Fatalf("cross-tenant update changed status to %s", got.Status)
	}
	if err := s.update(ctx, "tenant-a", rec.ID, contract.RunSucceeded, json.RawMessage(`{"ok":true}`), "", ""); err != nil {
		t.Fatal(err)
	}
	got, err = s.getByID(ctx, "tenant-a", rec.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != contract.RunSucceeded {
		t.Fatalf("own-tenant update status = %s, want succeeded", got.Status)
	}
}

// TestUpdateRejectsCrossTenantBypassRole proves the tenant predicate
// — not RLS — rejects cross-tenant updates: the write runs as a
// BYPASSRLS role for which policies do not apply.
func TestUpdateRejectsCrossTenantBypassRole(t *testing.T) {
	db := testkit.NewDB(t)
	db.Truncate(t, "app.invocations")
	s := &invocations{pool: db.Pool}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	rec, created, err := s.claim(ctx, "tenant-a", InvokeRequest{
		MiniappID: "m", Version: "1.0.0", Tool: "t.t",
		Input: json.RawMessage(`{}`), IdempotencyKey: "k-bypass",
	}, "00000000-0000-0000-0000-000000000002", "tok-2", contract.RunRunning)
	if err != nil || !created {
		t.Fatalf("claim = %v, %v", rec, err)
	}
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
	if err := bypassPool.QueryRow(ctx, `SELECT tenant_id FROM app.invocations WHERE id = $1::uuid`, rec.ID).Scan(&seen); err != nil {
		t.Fatalf("bypass role cannot read row (RLS not bypassed?): %v", err)
	}
	bs := &invocations{pool: bypassPool}
	if err := bs.update(ctx, "tenant-b", rec.ID, contract.RunSucceeded, json.RawMessage(`{}`), "", ""); err != nil {
		t.Fatalf("bypass update errored: %v", err)
	}
	got, err := s.getByID(ctx, "tenant-a", rec.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != contract.RunRunning {
		t.Fatalf("bypass-role cross-tenant update changed status to %s (predicate failed)", got.Status)
	}
}
