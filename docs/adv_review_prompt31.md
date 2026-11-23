# Prompt31 — Independent Adversarial Review: Prompt28.1 Settle Operator Boundary

**Verdict: CLEAR** — all material guarantees independently verified. One LOW test-completeness
finding; no security, authorization, tenant-isolation, attribution, or correctness violation.

## 1. Review baseline

- HEAD `a09b7ff` (main); no commits made; worktree read-only (this report is the only file added).
- Scope files re-read in full: `runtime/lifecycle.go` (`SettleEffect`, `hasSettleRole`),
  `transport/cli/cli.go` (settle verb), `cmd/miniapp/main.go` (dispatch), `runtime/p281_settleop_test.go`,
  `transport/cli/settle_test.go`, contract §16 P28.1 row + §12 F-03 bullet, decision-memo original
  audit + addendum, evidence-register rows 166–167, `docs/prompt28.1.md`, `docs/prompt28.1_plan.md`.
- Environment: Go 1.27.3, `GOWORK=off`; PostgreSQL `19.0 (Debian 19.0-1.pgdg13+1)` disposable
  container, image digest verified equal to required `postgres@sha256:bb6e70ad…c60`; fresh
  `p31test`/`miniapp_dbos` databases. Long-lived dev resources untouched; review container removed.

## 2. Evidence inspected

- All scope sources (line-level reads above) plus adversarial call-graph walks: `Settle` callers
  (exactly one non-test: `SettleEffect`), `effect-settled` writers (only `insertSettleAudit` +
  backfill check), `SettleEffect` callers (CLI verb only + tests), role-const references (single
  comparison), `Authenticate` failure shape, `InsertTx` parameterization + attribution validation.
- Both test files executed on PG19 (all green; `-count=2` idempotency confirmed via truncating
  fixture): 4 service tests (authorized, 9-leg refusals, isolation, 4-leg validation), 2 CLI tests
  (4-case flags matrix, authorized e2e + non-operator refusal).
- Regression evidence: `effects.go`, all settle crash/fault/concurrency tests, `go.mod`/`go.sum`
  byte-identical to the p28 manifest; HEAD unchanged; migrations end at 000006 with no status
  changes; no REST/MCP settle surface (grep-verified); decision-memo original audit intact with
  appended addendum.
- Full battery: gofmt/vet clean, checks ALL-CHECKS-PASS, full serial suite 22 ok / 0 fail
  (dbos 37.6s, classd 40.2s), affected suites re-run. First-run dbos/spikes failures were a
  fresh-container DB-ordering artifact (missing `miniapp_dbos`, pre-existing convention);
  identical bytes green on re-run and in isolation.

## 3. Findings

### P31-001 — Boundary test completeness (LOW, non-blocking)

- **Invariant:** defense-in-depth coverage of the operator boundary (prompt28.1 §6 spirit).
- **References:** `runtime/lifecycle.go` degenerate-binding guard
  (`binding.AgentID == "" || tenantID == ""`); conflicting-repeat path through `SettleEffect`;
  `runtime/p281_settleop_test.go` (absent legs).
- **Failure scenario:** none demonstrated — both behaviors verified correct by source read
  (guard fails closed; conflicts delegate to `effects.Settle`'s tested AlreadySettled path).
  The gap is proof, not behavior: a future regression of the 3-line guard or of error
  passthrough would not be caught at the boundary layer.
- **Evidence:** full test-file read (9 refusal legs cover credential/role/note shapes but no
  degenerate binding; idempotent repeat covered but conflicting repeat not at boundary level).
- **Disposition:** add two legs (degenerate-binding refusal; boundary conflicting-repeat
  asserting first-decision-stands + single audit). **Blocks acceptance: No.**

No other findings. Observations (no action required): CLI env-fallback untested (shared
pre-existing helper, flag path covered); identical query shape makes missing/foreign timing
oracles negligible; suite needs pre-created DBs on fresh containers (pre-existing convention).

## 4. Invariant-by-invariant assessment

1. **Authentication — HOLDS.** `Authenticate` is the first statement; empty/unknown/malformed
   credentials fail identically to terminal `auth-required` (verified `Authenticate` + 3 test
   legs). No path reaches authorization without a resolved binding.
2. **Exact authority — HOLDS.** `hasSettleRole` is byte-exact `==` over server-resolved
   `binding.Roles`; unexported const, no RBAC CRUD, no caller role/identity/tenant input
   (Intent plays no part; outcome allowlisted; note is recorded evidence, never authority).
   9 refusal legs pin case/whitespace/substring/superstring/synthetic/spoofed-note shapes.
3. **Tenant isolation — HOLDS.** Tenant comes only from `binding.Authority.TenantID`; no
   pre-read exists (Authenticate → role → validation → delegated Settle); enforcement is the
   tenant-predicated operation itself. Cross-tenant operator settles own row (positive
   control); foreign row untouched with zero audits.
4. **Non-oracle — HOLDS.** Missing and foreign effects produce byte-identical terminal
   `unknown-effect` (asserted string equality, not just code); both die at the same
   tenant-predicated no-rows read with identical query shape; zero state change, zero audits.
5. **Validation — HOLDS.** Outcome allowlist (3 terminals) + non-blank note enforced before
   any state access; 4 legs assert terminal `invalid-request` + persisted UNKNOWN + 0 audits.
   Empty opID degrades to the no-rows terminal (fail-closed, same shape as missing).
6. **Attribution — HOLDS.** Operator/tenant from the authenticated binding, effect from the
   validated lookup key, decision recorded as the operator's evidence claim with identity
   attached, miniapp/tool from the row's immutable identity read. Actual persisted audit
   records inspected in-test (agent/tenant/tool asserted from the table, not the return
   value). Audit insert fully parameterized with non-empty attribution enforced.
7. **Atomicity — HOLDS.** `effects.Settle` byte-identical to the p28 manifest (zero 28.1
   delta); boundary adds no reads/writes (in-memory auth + pure validation + delegation);
   crash/fault suites byte-identical and green (B8 hammer, backfill, commit-fault, kill
   scenarios all pass on final bytes).
8. **Shared boundary — HOLDS.** CLI performs flag parsing + outcome mapping only, with an
   explicit no-authorization comment; service method is the sole non-test `Settle` caller
   and sole CLI target; no REST/MCP endpoint, no alternate production path (exhaustive grep);
   `effect-settled` writable only inside the atomic settle paths.

## 5. Test assessment

Tests genuinely prove their claims: every refusal asserts terminal type + code + kind *plus*
persisted row state *plus* audit counts (never error returns alone); per-leg fresh fixtures
with distinct keys and truncating isolation (no cross-contamination; `-count=2` green);
positive controls (authorized success, tenant-B self-settle, idempotent repeat) guard against
vacuous always-refuse; indistinguishability asserted on full error strings; CLI e2e asserts
persisted state + attribution, not just exit codes. No shared-fixture smuggling (each leg
seeds its own row), no sleeps-as-proof, no TOCTOU (single-threaded boundary tests over an
already race-proven core). Gaps: the two P31-001 legs only.

## 6. Regression and scope verification

- PG19 digest verified (`sha256:bb6e70ad…c60` inspected, server 19.0); test results above
  reproduced on disposable DBs.
- Settle crash-test sources byte-identical (p28 crash/backfill/fault/concurrency vs manifest).
- `effects.Settle` untouched (byte-identical); no migration, no dependency change, no generic
  RBAC, no REST/MCP endpoint, no unrelated production modification (6-file additive/internal
  diff audited).
- No commits (HEAD `a09b7ff` throughout), no deployments, no real provider writes (mock only).
- STOPPED record intact: original memo audit preserved verbatim with appended addendum;
  supersession chain (decision → GO → implementation → evidence → addendum → register)
  complete and accurately worded.

## 7. Residuals

Retained: demo API-key provisioning is not production identity (memo + §16, carried to the
Security Contract); CLI credential process-listing exposure (documented limitation);
tenant-ops remains synthetic vocabulary; operator role membership changes require config
restarts (same model as grants). All explicitly recorded in the memo addendum and §16 row;
no silent residual.

## 8. Final verdict

**CLEAR.** Proven by independent source, test, and execution evidence: an unauthorized caller
cannot settle an effect (deny-by-default at authentication, exact role, and validation
layers); a caller cannot cross tenant boundaries (binding-derived tenant, predicated
operation, indistinguishable denials); attribution cannot be forged (server-resolved identity,
row-resolved tool, parameterized validated audit); every authorized settlement preserves the
existing atomic effect guarantees (untouched `Settle`, byte-identical crash suite, green
battery). The single LOW finding is test-completeness polish. Accept the Prompt28.1 boundary.
