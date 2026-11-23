DROP TABLE IF EXISTS app.kill_switches;
ALTER TABLE app.invocations DROP COLUMN IF EXISTS trust_mode;
ALTER TABLE app.invocations DROP COLUMN IF EXISTS effect_token;
ALTER TABLE app.effect_records DROP COLUMN IF EXISTS payload;
