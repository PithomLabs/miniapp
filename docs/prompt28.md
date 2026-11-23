**Prompt27 closeout accepted. Stage 3A is now authorized to resume.**

The closeout preserves the approved P26 decisions, records the empty-mode behavior accurately, and restores C1–C7 traceability. The new characterization test also exposes an important distinction: authorization refusal does not automatically terminalize the invocation. Keep that behavior visible in subsequent testing.&#x20;

prompt27-p26-low-findings-closeout.md

prompt27-p26-low-findings-closeout.md



The independent review explicitly released Gate 1 for the remaining authorization/crash matrix, C6 fixture, revocation tooling, Settle wiring and operator authentication, and bypass suite.&#x20;

adv_review_prompt26.md



I authorize the following work order.

# Prompt28 — Complete Remaining Stage 3A Work (Phase 2)

## Role

Act as the senior architect and implementation owner of `github.com/PithomLabs/miniapp`.

Gate 1 is CLEAR. Prompt27 has closed P26-001/002/003 and established the new evidence baseline. You are authorized to resume the remaining Stage 3A work.

Your objective is to complete the outstanding Phase 2 authorization, recovery, settlement, revocation, and provider-bypass work, prove its invariants, and prepare the complete Gate 2 independent-review package.

This is an implementation work order. Investigate actual source state before changing anything. Do not assume that an item remains missing merely because an earlier report said it was missing.

## 1. Governing architecture and constraints

Preserve the governing principle:

**The framework owns reliability; miniapps own business logic.**

The binding constraints remain:

- C1: `authorizeAttempt` acquires no advisory lock.
- C2: Global lock order is preserved: grant snapshot RLock → invocation → approval → sequence when activated → policy controls/grants → effect.
- C3: Authorization uses its own transaction and pool checkout, never the Class A attempt's `pgx.Tx`.
- C4: No new advisory-lock acquisition paths.
- C5: Pending lifecycle semantics remain explicit and unchanged.
- C6: Class A handlers that call `Perform` require proven bounded recovery, stable BusinessOpID, and no indefinite transaction hold.
- C7: O-01 closes at the effect boundary; no provider send without committed authorization.

Preserve Prompt21, Prompt22, Prompt24, Prompt26, and Prompt27 decisions and accepted residuals.

Do not redesign the architecture, introduce a general workflow engine, or weaken any established transaction or ownership invariant.

## 2. Baseline and evidence freeze

Start with a fresh source audit.

Verify:

- Repository, branch, HEAD, and worktree.
- Prompt27 source digest: `664558bed1390c6d3d982809eabb588a52fee3a03164e2b09ea478d2abf80b2f`.
- Prompt27 tree digest: `2ee4c9eaf038c289bfd74dacb06aa6808a0c65fc805464735456de6a3abd8827`.
- `docs/evidence/digests/p27-closeout-2026-11-23.manifest.txt`.
- Lifecycle contract, authorization design, Prompt25 closeout, Prompt26 review, and Prompt27 closeout.
- Actual status of every outstanding Stage 3A item.

Resolve any digest mismatch or unexpected worktree modification before implementation. Do not rewrite historical evidence or manifests.

Use frozen pre-fix source snapshots and identical test bytes for genuine red/green comparisons. Do not claim red/green where the test is merely characterization.

## 3. Workstream A — Full authorization matrix

Complete the authorization matrix defined by the approved authorization design.

Cover the relevant combinations of:

**Invocation and identity**

- Correct and incorrect tenant.
- Correct and incorrect invocation binding.
- Correct and incorrect operation/effect identity.
- Persisted tool and trust binding.
- Missing invocation and phantom binding.
- Pending, running, completed, failed, cancelled, and other terminal states.
- Stale, mismatched, quarantined, and superseded attempts.

**Approval**

- Confirm with valid approval.
- Confirm with missing, pending, denied, expired, cancelled, or mismatched approval.
- Persisted Autonomous with explicit manifest permission.
- Persisted Autonomous without explicit manifest permission.
- Manifest-disallowed persisted trust.
- Attempted caller-side trust spoofing.

Preserve the Prompt26/P27 owner decision exactly. Preserve the empty-TrustModes characterization and its activation trigger. Do not change authorization semantics to make a test pass.

**Grant and policy**

- Allowed and denied grants.
- Grant snapshot replacement and revocation races.
- Each existing kill-switch scope and policy-control level.
- Token binding, credential resolution, and stale or mismatched policy conditions.

**Sequence**

- Preserve the approved M5 deferral. No production sequence subsystem, generic sequence table, verifier implementation, or probe sequence-wiring work is authorized in this work order.
- A sequence-bound non-mock provider must remain unavailable until the framework-declared requirement, trusted identity propagation, verifier, supersede validation, and X-08 barrier test all exist.
- Do not infer sequence requirements from `Intent.Sequence` presence.

For every refusal, assert:

- Exact terminal or retryable error classification.
- No committed unauthorized `ATTEMPTING`.
- No provider invocation.
- Correct invocation/effect state.
- Correct audit behavior, subject to the explicitly accepted attribution limitations.

The matrix must distinguish security refusal from retryable contention and ambiguous provider outcome.

## 4. Workstream B — Deterministic authorization concurrency and crash matrix

Implement the outstanding deterministic race suite on real PostgreSQL 19 sessions.

### Required concurrency cases

Cover:

1. Both cancellation-versus-authorization orders.
2. Worker versus worker on the same effect.
3. Worker versus worker across attempt generations.
4. Lease turnover versus authorization.
5. Restoration versus dispatch.
6. Grant revocation or snapshot replacement versus authorization.
7. Supersede versus authorization.
8. Settle versus effect ownership and authorization.
9. Shared-row contention and competing transitions.

For each test, use explicit synchronization and observable PostgreSQL state:

- Row-lock waits.
- Query/session identity.
- Persisted state transitions.
- Entered/released provider gates.
- Attempt IDs and operation IDs.

Never rely on sleeps, goroutine counts, or repetition as proof of ordering.

Assert the intended winner, loser, persisted state, audit, provider count, and retry classification.

### Six-boundary crash matrix

Complete the six crash boundaries specified in the approved authorization design's §10.3.

Do not invent replacement boundaries or quietly reduce the required matrix.

For every boundary, record:

- Last durable database state.
- Whether a provider call may have occurred.
- What a retry is allowed to do.
- Whether adoption, reconciliation, quarantine, or UNKNOWN is required.
- Whether the same BusinessOpID and attempt epoch are retained or replaced.
- Which evidence establishes the outcome.

Use actual process termination/restart where the contract calls for crash behavior. A panic or returned error is not a substitute for a process crash.

Preserve the core rule: never blindly resend an operation whose provider outcome is ambiguous.

The full crash matrix must prove that a late worker cannot overwrite a newer attempt and that any reconciliation creates a fresh authorized epoch.

## 5. Workstream C — C6 Class A recovery fixture

C6 remains the largest unproven integration requirement.

Build the missing Class A fixture in which a real Class A handler invokes `Perform` and encounters `ErrAttemptInProgress` under deterministic contention.

Prove:

- The effect contention is reachable through the real Class A execution path.
- The enclosing Class A transaction rolls back completely when the handler cannot proceed.
- No partially committed business writes survive the failed attempt.
- The invocation is not incorrectly marked as a successful or terminal business result.
- Recovery does not wait while holding the Class A transaction, authorization transaction, grant RLock, or a database lock.
- The caller receives correct state-specific retry guidance.
- The BusinessOpID is stable across retries of the same business operation.
- Attempt IDs remain framework-generated and represent distinct ownership epochs.
- The recovery procedure is bounded by the existing lease semantics.
- The two-minute lease is not bypassed or misrepresented.
- The eventual retry either adopts/reconciles safely or returns an explicit terminal result.

Use controlled database timestamps or an equivalent deterministic mechanism to exercise lease expiry. Do not sleep for two minutes.

Preserve existing idempotency-key semantics. In particular, do not tell callers that same-key retry will converge when a running Class A record requires the documented new-key/interrupted recovery path.

Add the required operation-ID guidance to the lifecycle contract and relevant developer-facing documentation.

The fixture must be a genuine integration test, not a mocked simulation of the runtime's transaction boundary.

## 6. Workstream D — Grant hash, health, and revocation tooling

Complete the remaining grant-hash consumers and tests.

The existing configuration hash is:

- SHA-256 of canonical JSON for the grants map.
- Deterministic and independent of map iteration order.
- Credential-free.
- Sensitive to grant-map changes.

Preserve its documented scope: it is a grant-configuration hash, not a hash of all key roles or authorities. Do not make claims that it detects role-only drift.

Implement the approved `/healthz` exposure and `verify-revocation` CLI behavior according to the existing authorization design and operational contract.

Requirements:

- Expose only the non-secret configuration hash and approved health metadata.
- Never expose grants, credentials, API tokens, or secret material.
- Verify the hash consumers use the same canonicalization as `auth.Policy`.
- Test map-order invariance, grant-change sensitivity, and credential non-disclosure.
- Make the CLI output actionable and deterministic.
- Distinguish configuration drift from an authorization failure.
- Preserve the documented ten-minute staleness baseline and cross-replica restart requirements.
- Do not claim immediate global revocation or stronger consistency than the architecture provides.

If the existing design does not define an important CLI or health response semantic, identify the gap and stop for owner direction rather than inventing a new revocation model.

## 7. Workstream E — Settle crash safety and production wiring

Complete the outstanding Settle tests before wiring the operation into a production-facing control path.

### Atomic settlement

Preserve the existing single-transaction architecture:

- Only eligible UNKNOWN effects can be settled.
- State transition and audit remain atomic.
- Settle takes only the effect-row lock; P24-003's lock-free invocation attribution remains intact.
- No provider network call occurs in the Settle transaction.
- Conflicting second decisions preserve the first committed decision.
- NOT_APPLIED reconciliation always proceeds through fresh effect-time authorization before a new provider call.

Complete all four Settle crash scenarios:

1. Crash before commit: no partial state or audit survives.
2. Commit succeeds but the caller loses the response: retry discovers the committed outcome.
3. Identical repeat under crash/retry: operation is idempotent and audit behavior remains truthful.
4. Conflicting second settlement: the first committed decision wins and the conflict is reported deterministically.

The existing operator-commit fault test counts as the existing starting point. Do not claim the Settle crash matrix complete until all four cases are covered.

Re-examine the concurrent identical-repeat audit-backfill race. Preserve the previously accepted residual unless a small, schema-compatible correction is clearly justified. Do not introduce a migration merely to eliminate a cosmetic audit-only race without owner approval.

### Operator authentication and wiring

Wire Settle through the existing authenticated framework/control-plane boundary.

Requirements:

- Operator identity must be derived from a trusted authenticated principal, never accepted as an arbitrary caller-supplied string.
- Authorization must establish that the principal has the existing required administrative authority before settlement.
- Audit attribution must use the server-derived operator identity.
- Tenant and effect ownership must be checked.
- Invalid or unauthenticated callers must not reach Settle.
- Direct package-level Settle remains inaccessible to untrusted miniapp code.
- The authenticated boundary must preserve idempotency and conflict semantics.
- Tests must cover allowed, denied, missing, and mismatched operator identity, tenant, and effect cases.

Inspect the existing authentication and authority model first. Reuse the existing principal and permission mechanisms. Do not invent a hardcoded header, shared secret, bypass flag, or unauthenticated internal route.

If no suitable trusted operator authority exists, stop and present the smallest architecture decision required. Do not weaken the boundary to complete wiring.

No real provider operations are permitted.

## 8. Workstream F — Provider-path bypass assurance

Repeat the complete production provider-path inventory.

Include:

- Fresh `Perform`.
- Takeover.
- Adoption.
- Reconciliation.
- Retry/resume.
- DBOS workflow call sites.
- All `Channel` implementations.
- Every provider `Send` call site.
- Any internal administrative or settlement operation that could lead to provider work.

For every path, prove:

1. Authorization occurs before provider work.
2. `ATTEMPTING` is durably committed before the send.
3. The attempt ID and invocation/effect ownership are the authoritative persisted values.
4. An authorization refusal produces zero provider calls.
5. A stale worker cannot record an outcome against a newer attempt.
6. UNKNOWN is never blindly resent.

Add a discriminating test for each distinct initiation path. Do not rely solely on exhaustive grep.

Keep fault wrappers and mock-only test wiring clearly separated from production entrypoints.

### Deferred capability and sequence requirements

P24-008 remains trigger-bound: capability membership must be enforced before any first non-mock provider is enabled.

Do not add a real provider as part of this task. Preserve the documented capability prerequisite and ensure the current mock-only provider inventory does not accidentally imply that the requirement is already implemented.

Likewise, preserve the P24-004 M5 sequence deferral. No sequence-bound non-mock send may become reachable without the full framework-owned sequence mechanism.

## 9. PostgreSQL 19 and CI parity

PostgreSQL 19 is mandatory for:

- Local development.
- Integration tests.
- Authorization race tests.
- Crash-recovery tests.
- DBOS tests.
- CI.

Required image:

`postgres@sha256:bb6e70add742cd868fb62cb3b2f67d1dd2bdd2b65a521e778420748753687c60`

Re-resolve the image digest using Docker inspection before the evidence snapshot.

Use the same immutable digest in local configuration, Compose, and CI. No floating `postgres:19.0` or `latest` tags.

Record:

- Docker image reference and full digest.
- `SELECT version()`.
- Go version and architecture.
- Database identities.
- Redacted environment.
- Exact commands, exit codes, durations, package counts, and skips.

If CI or the environment cannot establish the exact PostgreSQL 19 digest, report BLOCKED. Never report PASS on another major version.

## 10. Evidence discipline

For each implemented invariant:

- Identify the governing contract row.
- Identify the source implementation.
- Identify the discriminating test.
- Record the exact command and test result.
- Record the source/test digest.
- Classify as execution-proven, source-proven, composition-proven, characterization-only, accepted residual, or blocked/unverified.

For behavioral defects, use identical test bytes on frozen red and green trees.

Do not call tests red/green when both sides pass. Do not weaken or delete an existing test merely to obtain a green result.

Update the evidence register for:

- Authorization matrix.
- Authorization concurrency.
- Crash/recovery.
- C6.
- Ownership/adoption.
- Revocation and grant snapshot.
- Settle.
- Operator authorization.
- Provider bypass.

Every guarantee must have a traceable source/test/evidence chain.

## 11. Final verification and closeout

Run:

- `gofmt -l .`
- `go vet ./...`
- `go test -p 1 ./... -count=1`
- `go test -race -p 1 ./runtime/ ./approvals/ ./effects/`
- `bash scripts/checks.sh`
- DBOS and Class D suites with the required PostgreSQL 19 URLs.
- All focused authorization, concurrency, crash, C6, Settle, operator, and bypass tests.

No hidden skips or silently omitted packages.

Update the lifecycle contract, authorization design, revocation/operations documentation, provider-path inventory, and evidence register.

Create:

`docs/phase2/prompt28-stage3a-closeout.md`

The report must contain:

1. Exact baseline and final HEAD/worktree status.
2. Source and tree digests.
3. PostgreSQL 19 immutable image and server identity.
4. Authorization matrix results.
5. Concurrency/race matrix results.
6. Six-boundary crash matrix results.
7. C6 integration result and stable BusinessOpID semantics.
8. Grant hash, health, and revocation tooling result.
9. Settle crash matrix and authenticated operator result.
10. Provider bypass inventory and test results.
11. C1–C7 true traceability table.
12. Every remaining residual and activation trigger.
13. Exact commands and results.
14. Evidence-register references.
15. Explicit Gate 2 readiness verdict.

## 12. Stop conditions and non-goals

Stop and report if:

- A C1–C7 constraint cannot be preserved.
- Correctness requires an unapproved semantic change.
- A crash boundary permits a blind duplicate provider send.
- An attempt fencing failure reappears.
- Settle cannot be authenticated using the existing trusted identity model.
- A non-mock sequence-bound path becomes reachable without the deferred sequence mechanism.
- Exact PG19 parity cannot be established.
- A schema migration or dependency change becomes necessary for a load-bearing invariant without owner approval.

Do not:

- Implement River/Class B or Class C.
- Implement X-12 or X-10.
- Implement Sprites, Fly rollout, production deployment, UI, or generators.
- Add a real external provider or perform real provider writes.
- Implement the deferred generic sequence subsystem.
- Commit, stage, push, release, or deploy.
- Begin Phase 3.

Stop after the Gate 2 reviewer package and closeout report are complete. Gate 2 must be independently reviewed before Phase 2 final acceptance. Phase 3 requires a separate owner instruction.

**Execution recommendation:** implement in the order A → B → C → D → E → F → final verification. This gets the authorization invariants and recovery behavior proven before the newly exposed Settle control path is wired. Do not try to close everything with a single large green test run; each workstream needs its own evidence.

The next independent review is **Gate 2**, after Prompt28 closeout. Phase 3 remains unreleased.