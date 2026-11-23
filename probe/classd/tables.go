package classd

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// tablesDDL owns the probe's domain tables. They are test fixtures
// (spike-table precedent), not framework migrations: production
// databases never need them, and no framework package imports them.
const tablesDDL = `
CREATE TABLE IF NOT EXISTS probe_sequences (
  tenant_id TEXT NOT NULL,
  subject_id TEXT NOT NULL,
  generation INT NOT NULL,
  status TEXT NOT NULL DEFAULT 'active',
  outcome TEXT NOT NULL DEFAULT '',
  stage INT NOT NULL DEFAULT 0,
  run_id TEXT NOT NULL DEFAULT '',
  wake_at TIMESTAMPTZ,
  eligible BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, subject_id, generation)
);
DROP INDEX IF EXISTS probe_sequences_active;
CREATE UNIQUE INDEX IF NOT EXISTS probe_sequences_active
  ON probe_sequences (tenant_id, subject_id) WHERE status = 'active';
ALTER TABLE probe_sequences ENABLE ROW LEVEL SECURITY;
ALTER TABLE probe_sequences FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON probe_sequences;
CREATE POLICY tenant_isolation ON probe_sequences
  USING (tenant_id = current_setting('app.tenant_id', true))
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true));
CREATE SEQUENCE IF NOT EXISTS probe_generation_seq;
CREATE TABLE IF NOT EXISTS probe_observations (
  invocation_id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  detail TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
ALTER TABLE probe_observations ENABLE ROW LEVEL SECURITY;
ALTER TABLE probe_observations FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON probe_observations;
CREATE POLICY tenant_isolation ON probe_observations
  USING (tenant_id = current_setting('app.tenant_id', true))
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true));
CREATE TABLE IF NOT EXISTS probe_wait_marks (
  invocation_id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  approval_id TEXT NOT NULL DEFAULT '',
  marked_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
ALTER TABLE probe_wait_marks ENABLE ROW LEVEL SECURITY;
ALTER TABLE probe_wait_marks FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation ON probe_wait_marks;
CREATE POLICY tenant_isolation ON probe_wait_marks
  USING (tenant_id = current_setting('app.tenant_id', true))
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true));
`

// EnsureTables creates the probe fixture tables.
func EnsureTables(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if _, err := pool.Exec(ctx, tablesDDL); err != nil {
		t.Fatal(err)
	}
}
