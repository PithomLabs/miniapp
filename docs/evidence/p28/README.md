# Prompt28 Gate 2 reviewer package (docs/evidence/p28)

Closeout report: `docs/phase2/prompt28-stage3a-closeout.md`.
Evidence register: `docs/evidence/README.md` (Prompt 28 rows).
Tree manifest:
`docs/evidence/digests/p28-closeout-2026-11-23.manifest.txt`.

## What is new

- Tests: `effects/p28_*` (matrix, races, crash matrix,
  backfill, settle crash, bypass paths/guard, wait-revoke),
  `runtime/p28_restore_race_test.go`,
  `runtime/p28_c6_fixture_test.go`,
  `transport/cli/revoke_test.go`,
  `transport/p28_healthz_test.go`,
  `probe/classd/p28_ledger_proof_test.go`,
  `auth/policy_snapshot_test.go` (+1 test),
  `infra/compose_test.go` (pin expectation).
- Production: `/healthz` route+handler,
  `Service.GrantConfigHash`, startup hash log,
  `miniapp auth verify-revocation`, Settle backfill row
  lock, digest pins (Compose/CI/Taskfile/script).
- Docs: C6 op-ID rule (contract + effect-authoring guide),
  revocation ops section, settle-operator decision memo
  (WS-E wiring STOPPED, owner decision required).
  (P26-002 trigger notes were written in Prompt27 and are
  unchanged here.)

## Genuine red/green pairs

Exactly one behavioral defect was found and fixed:

- Settle audit-backfill race: identical test bytes
  (`effects/p28_settle_backfill_test.go`,
  `1d48f257…`) failed pre-fix (up to 16 duplicate
  `effect-settled` audits over 640 32-way races) and
  pass 3× post-fix (FOR UPDATE serialization, no
  migration). Reproduce red by reverting the
  `loadTxLocked` re-read in `Settle`'s repeat branch
  (`effects/effects.go`) on a scratch copy.

All other Prompt28 tests are green-first coverage of
specified behavior (characterization where both sides
would pass — labeled as such in code and register).

## Reproduction

Disposable pinned PG19 (`postgres@sha256:bb6e70ad…`,
re-resolve via `docker inspect`), fresh database,
`TEST_DATABASE_URL`/`TEST_DBOS_DATABASE_URL` pointed at
it, `GOCACHE` set to a writable dir, then the §11
commands from the closeout report. The long-lived dev
database is never touched.
