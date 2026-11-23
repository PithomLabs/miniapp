# Prompt30 residual closeout — P29-001 through P29-005

Date: 2026-11-23. Work order: `docs/prompt30.md`. Findings:
`docs/adv_review_prompt29.md` §4 (Gate 2, CONDITIONAL GO).
Plan: `docs/prompt30_plan.md`. Classification: all new tests
green-first characterization; no red/green claims (no defect
found, none fixed).

## 1. Baseline

- HEAD `a09b7ff2c639f4bdcd61e7e7192c1597187b8037`, worktree
  dirty at freeze: 61 status entries (Prompt28/28.1 work +
  outside activity), snapshotted to `/tmp/p30_baseline/`.
- Key hashes at freeze:
  - `effects/effects.go` `77f4052c…25686`
  - `effects/p28_crash_matrix_test.go` `13ebd995…004a2`
  - `effects/p28_matrix_policy_test.go` `2455b068…55e96`
  - `effects/p28_bypass_guard_test.go` `094e035f…bf89`
- PostgreSQL image re-resolved via `docker inspect`:
  `postgres@sha256:bb6e70add742cd868fb62cb3b2f67d1dd2bdd2b65a521e778420748753687c60`
  — matches the mandated digest. All DB tests ran on
  disposable containers off this digest; both removed after.

## 2. Dispositions

### P29-001 — Refusal evidence hardened (6 legs)

Added `p28AssertRefusalEvidence` (row PENDING + empty
provider outcome + exactly one `effect-refused` audit, via
the existing `p28ReadRow`/`p28RefusedAudits` helpers) and
called it from all six legs: the four kill-switch scopes,
tool-declared, and grant-deny. Mechanism verified first:
`Perform` claims the PENDING row, `attempt` → `authorizeAttempt`
refuses after invocation load (attributable) and before the
Lock 5 transition, so ×1 audit and PENDING-untouched are the
contract truth. All six pass — the register claim is
confirmed, no narrowing needed. The file header's five-point
promise is now literally true.

### P29-002 — Guard negative control added

Extracted the inventory test's per-file AST walk into shared
`p28SendSitesInFile` (test-only refactor; production
untouched) and refactored `TestP28ProviderSendInventory`
onto it with identical accept/reject behavior. Added fixture
`effects/testdata/bypass_guard_violations.go` (planted Send
impl + call site with `// want:` markers, plus unmarked
`dbos.Send` and bare-selector negatives) and
`TestP28GuardNegativeControl`, which runs the SAME matcher
the inventory test uses and requires exact flagged-line-set
equality. Temporarily neutering the matcher fails both
planted lines loudly; the file was restored byte-identical
(`sha256sum -c` OK) and both tests pass. The register's
"negative-controlled" claim is now true.

### P29-003 — Trust-name detection strengthened

Added `isTrustBearingName` (lowercase, strip `_`/`-`/space,
substring `trust`) and rewired `TestP28IntentCarriesNoTrust`
to it. Added `TestP28TrustBearingNameDetection`: 11
positives (the 4 required names + adversarial `trust_level`
+ legacy/casing/embedding extras) and 12 negatives (the 7
real `Intent` fields pinned literally + ordinary and
near-miss lookalikes including `RustLevel`/`TrussBridge`).
Zero false positives; `Intent` API unchanged (7 fields
verified trust-free). Note: `Autonomous` is negative under
the approved substring semantics (it contains no "trust") —
documented in the test, not a regression.

### P29-004 — Supersession record verified complete

Verify-only; nothing changed. All five prompt30 §5 items
hold: closeout STOPPED statements intact (L135/234);
Option A recorded in the decision-memo addendum; GO
traceable (`docs/prompt28.1_plan.md`, "GO — approved to
start" + guardrails); memo addendum + register rows chain
the supersession; `SettleEffect` is the singular Settle
boundary in all referencing docs. No historical verdict
rewritten.

### P29-005 — Abandoned-PENDING boundary documented

Three insertions: the 6-step scenario + 4 explanations
(trade-off rationale, no silent adoption, recovery path,
developer limitations) in
`docs/development/effect-authoring.md`, with explicit
analyzed-only/no-execution language per Prompt29; a
contract pointer at the C6 rule; one hash-sourcing line in
the ops revocation section (`grant_config_hash` from the
startup log or live `/healthz` — both code-verified at
`cmd/miniapp/main.go:240` and
`transport/rest/server.go:78`). Ten-minute semantics
untouched.

## 3. Exact tests and results

DB tests (disposable PG19, digest above):

- `go test ./effects/ -run 'TestP28KillSwitchScopes|TestP28ToolDeclaredKillSwitch|TestP28GrantAllowDenyTyped' -count=1 -v`
  → PASS (6/6 legs incl. all 4 scopes).
- `go test -p 1 ./effects/ ./runtime/ -count=1` → ok both.
- `go test -p 1 ./... -count=1` → zero failures tree-wide.
- `go test ./effects/ -run 'TestP28KillSwitchScopes|TestP28GrantSwapRace|TestP28GrantAllowDenyTyped' -race -count=1`
  → ok.

DB-free:

- `go test ./effects/ -run 'TestP28ProviderSendInventory|TestP28GuardNegativeControl' -count=1` → PASS.
- `go test ./effects/ -run 'TestP28IntentCarriesNoTrust|TestP28TrustBearingNameDetection' -count=1` → PASS.
- `gofmt -l .` → clean. `go vet ./...` → clean.
- `bash scripts/checks.sh` → ALL-CHECKS-PASS.

Byte-identity: `effects/effects.go` and
`effects/p28_crash_matrix_test.go` verify OK against
pre-recorded hashes; the Settle crash-test sources were
never opened for edit. Change footprint vs the frozen
baseline: exactly the intended edits (2 test files
strengthened, 1 fixture added, 3 doc insertions, register
rows, this report).

## 4. Operational revocation drill — NOT EXECUTED

Inspected, not assumed. Findings: `compose.yaml` is DB-only
(single `db` service, no app replicas); no `fly.toml` in
repo; no deployed Machines (prompt4 record: neither app
deployed, none authorized); no `sprite-env`; no Neon branch
or drill credentials authorized in this work order. A
`flyctl status` probe was attempted but is sandbox-blocked
(read-only fs for its config) — inconclusive, not evidence.
Precise missing prerequisite: an explicitly authorized,
isolated multi-replica runtime with named credentials.
Per prompt30 §7 ("do not create a deployment"), no
environment was constructed. Recorded as NOT EXECUTED in
the register; no credentials touched, no prod contact.

## 5. Residuals that remain

- The drill prerequisite above (owner-authorized isolated
  replicas + credentials).
- The P29-005 wedge stays analyzed-only: no crash test
  covers the claim-to-authorization window with a
  cross-invocation retry, as documented.
- One outside arrival during this session,
  `docs/adv_review_prompt31.md` (prompt31 track), preserved
  untouched.

## 6. Stage 3A guarantees unchanged

No production file modified (`effects.go` byte-identical;
no advisory-lock, lock-order, authorization, effect, audit,
or recovery path touched). No migration, dependency, or API
change. No weakened tests — every touched test asserts
strictly more than before. Historical closeouts, the memo,
and outside activity preserved. Gate 2 CLEAR verdict
stands; its five LOW residuals are now closed with
evidence, except the drill, which is honestly blocked.
