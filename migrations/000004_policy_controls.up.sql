-- WS-G: kill switches plus invocation trust/token for effect-time
-- policy revalidation.
CREATE TABLE IF NOT EXISTS app.kill_switches (
  name TEXT PRIMARY KEY,
  engaged BOOLEAN NOT NULL DEFAULT FALSE,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE app.invocations ADD COLUMN IF NOT EXISTS trust_mode TEXT NOT NULL DEFAULT '';
ALTER TABLE app.invocations ADD COLUMN IF NOT EXISTS effect_token TEXT NOT NULL DEFAULT '';
ALTER TABLE app.effect_records ADD COLUMN IF NOT EXISTS payload TEXT NOT NULL DEFAULT '';

-- Kill switches are operator-global and service-mediated (no RLS):
-- tenant scope is encoded in switch names (tenant:<id>), and only
-- the framework service reads or flips them.
