-- Backfill: the payload column joined 000004 after that version had
-- already applied to the development database. Forward-only repair:
-- no-op where 000004 applied complete, adds the missing column
-- where it applied early.
ALTER TABLE app.effect_records ADD COLUMN IF NOT EXISTS payload TEXT NOT NULL DEFAULT '';
