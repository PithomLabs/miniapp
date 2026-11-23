-- crm-read reference data. Per-miniapp tables ship in their own
-- migrations; the framework never owns domain tables.
CREATE TABLE IF NOT EXISTS app.crm_contacts (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  name TEXT NOT NULL,
  email TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE app.crm_contacts ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.crm_contacts FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation ON app.crm_contacts
  USING (tenant_id = current_setting('app.tenant_id', true))
  WITH CHECK (tenant_id = current_setting('app.tenant_id', true));
