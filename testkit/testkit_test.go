package testkit

import (
	"context"
	"testing"
	"time"
)

// TestMigrationsApplyCleanly migrates a scratch database and asserts
// every framework table exists with RLS enforced. It is the only
// test that speaks to migration mechanics; all other DB tests assume
// a migrated database from NewDB.
func TestMigrationsApplyCleanly(t *testing.T) {
	db := NewDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	for _, table := range []string{"invocations", "audit_events", "approvals", "effect_records", "crm_contacts"} {
		var enforced bool
		err := db.Pool.QueryRow(ctx,
			`SELECT relforcerowsecurity FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
			 WHERE n.nspname = 'app' AND c.relname = $1`, table).Scan(&enforced)
		if err != nil {
			t.Fatalf("table app.%s missing or unreadable: %v", table, err)
		}
		if !enforced {
			t.Fatalf("table app.%s lacks FORCE ROW LEVEL SECURITY", table)
		}
	}
}
