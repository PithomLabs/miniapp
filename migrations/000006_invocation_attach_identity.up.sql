-- WS-A: invocation attachment identity (R1) + nonempty run-ID
-- uniqueness (X-22). attached_run_id carries an adopted winner
-- run for attached invocations (many observers, no unique
-- constraint); run_id stays the row's own submitted run (unique
-- when nonempty).
--
-- Preflight (CR-07): the X-03 defect may have left duplicate
-- nonempty run IDs in development databases. Detect and fail with
-- an actionable diagnostic rather than letting the unique index
-- fail opaquely or silently picking an owner.
DO $$
DECLARE
	dups TEXT;
BEGIN
	SELECT string_agg(format('%s x%s', run_id, c), ', ')
	INTO dups
	FROM (SELECT run_id, count(*) c FROM app.invocations
		WHERE run_id <> '' GROUP BY run_id HAVING count(*) > 1) d;
	IF dups IS NOT NULL THEN
		RAISE EXCEPTION 'migration 000006 refused: duplicate nonempty run_id values exist (%). Each owner invocation must resolve to one run before the uniqueness constraint can apply: re-run the owning invocations to converge on canonical run IDs (or, on a disposable test database, TRUNCATE app.invocations), then re-run this migration. This migration made no changes.', dups;
	END IF;
END $$;

ALTER TABLE app.invocations ADD COLUMN IF NOT EXISTS attached_run_id TEXT NOT NULL DEFAULT '';

-- NOTE: index names cannot be schema-qualified in CREATE INDEX;
-- the index inherits its parent table's (app) schema. The down
-- file addresses it qualified, which DROP INDEX accepts.
CREATE UNIQUE INDEX IF NOT EXISTS invocations_run_id_uidx
	ON app.invocations (run_id) WHERE run_id <> '';
