-- WS-D: link effect records to their invocation so cancellation can
-- report in-flight effects and gate new claims.
ALTER TABLE app.effect_records ADD COLUMN IF NOT EXISTS invocation_id TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS effect_records_invocation ON app.effect_records (tenant_id, invocation_id);
