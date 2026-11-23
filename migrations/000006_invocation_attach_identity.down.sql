DROP INDEX IF EXISTS app.invocations_run_id_uidx;
ALTER TABLE app.invocations DROP COLUMN IF EXISTS attached_run_id;
