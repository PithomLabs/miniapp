DROP INDEX IF EXISTS app.effect_records_invocation;
ALTER TABLE app.effect_records DROP COLUMN IF EXISTS invocation_id;
