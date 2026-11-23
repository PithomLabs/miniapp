-- Phase 1 framework tables. All names are schema-qualified: the
-- application never relies on search_path. Tenant tables carry RLS
-- policies as defense in depth; mandatory application authorization
-- remains primary (see internal/store.WithTenant).
CREATE SCHEMA IF NOT EXISTS app;

CREATE TABLE IF NOT EXISTS app.invocations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id TEXT NOT NULL,
  miniapp_id TEXT NOT NULL,
  miniapp_version TEXT NOT NULL,
  tool TEXT NOT NULL,
  idempotency_key TEXT NOT NULL,
  input JSONB NOT NULL,
  status TEXT NOT NULL DEFAULT 'running',
  result JSONB,
  run_id TEXT NOT NULL DEFAULT '',
  error TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, miniapp_id, miniapp_version, tool, idempotency_key)
);

CREATE TABLE IF NOT EXISTS app.audit_events (
  id BIGSERIAL PRIMARY KEY,
  at TIMESTAMPTZ NOT NULL DEFAULT now(),
  tenant_id TEXT NOT NULL,
  agent_id TEXT NOT NULL,
  authority_kind TEXT NOT NULL,
  authority_id TEXT NOT NULL,
  miniapp_id TEXT NOT NULL,
  miniapp_version TEXT NOT NULL,
  tool TEXT NOT NULL,
  invocation_id UUID,
  run_id TEXT NOT NULL DEFAULT '',
  action TEXT NOT NULL,
  outcome TEXT NOT NULL,
  details JSONB
);
CREATE INDEX IF NOT EXISTS audit_events_tenant_at ON app.audit_events (tenant_id, at);

CREATE TABLE IF NOT EXISTS app.approvals (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id TEXT NOT NULL,
  miniapp_id TEXT NOT NULL,
  miniapp_version TEXT NOT NULL,
  invocation_id TEXT NOT NULL,
  operation TEXT NOT NULL,
  resource TEXT NOT NULL DEFAULT '',
  intent_hash TEXT NOT NULL,
  approver_scope TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'pending',
  requested_by TEXT NOT NULL,
  decided_by TEXT NOT NULL DEFAULT '',
  decision TEXT NOT NULL DEFAULT '',
  expires_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  decided_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS app.effect_records (
  business_op_id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  attempt_id TEXT NOT NULL,
  idempotency_key TEXT NOT NULL DEFAULT '',
  capability_id TEXT NOT NULL,
  operation TEXT NOT NULL,
  state TEXT NOT NULL,
  provider_request TEXT NOT NULL DEFAULT '',
  provider_reply TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE app.invocations ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.invocations FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON app.invocations
  USING (tenant_id = current_setting('app.tenant_id', true))
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true));

ALTER TABLE app.approvals ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.approvals FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON app.approvals
  USING (tenant_id = current_setting('app.tenant_id', true))
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true));

ALTER TABLE app.effect_records ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.effect_records FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON app.effect_records
  USING (tenant_id = current_setting('app.tenant_id', true))
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true));

-- Audit uses USING-only: audit writes must never fail closed on
-- context, but reads stay tenant-scoped for non-bypass roles.
ALTER TABLE app.audit_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.audit_events FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON app.audit_events
  USING (tenant_id = current_setting('app.tenant_id', true));
