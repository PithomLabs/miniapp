Understood. **Prompt 17 is the controlling WS-B work order.** The GitHub Actions failure is relevant evidence, but it does not replace the four-phase plan. I am resolving both outstanding questions and authorizing Phase 0.

## **1. Decision on scope**

APPROVED — INCLUDE ALL FIVE ITEMS

| Item | Decision | Required treatment                                             |
| ---- | -------- | -------------------------------------------------------------- |
| L-01 | Include  | Implement `claim-response` hook and dedicated fault test       |
| L-02 | Include  | Correct §12 migration/preflight wording                        |
| L-03 | Include  | Correct §12 legacy-ID compatibility wording                    |
| L-04 | Include  | Preserve immutable manifest-time register copies going forward |
| O-05 | Include  | Add the one-line serial-test methodology note to `testkit`     |

Specific requirements:

**L-01:** Implement the named hook and `TestFaultClaimResponse`. It must be test-only, one-shot according to the established fault-injection contract, and must demonstrate replay convergence, one canonical run, and coherent audit evidence. Verify that no production configuration or assignment path has been introduced.

**L-02 and L-03:** These are documentation corrections, not migration redesigns. Record them through the corrections-never-silent mechanism in §13. Explicitly document the legacy-row behavior and the requirement for dual-read or backfill before any future encoding change when production data exists.

**L-04:** Do not attempt to reconstruct the missing historical pre-image. Adopt the immutable-copy practice from WS-B onward. Preserve the exact register bytes used when generating each evidence manifest.

**O-05:** Add the testkit note making clear that packages sharing a database must run serially with `-p 1`, unless each package process has an isolated database. Do not change test-harness semantics merely to hide misuse.

Also adopt O-02's evidence discipline: record the exact digest command, working directory, and path form. Omit non-reproducible digest values rather than presenting them as verified evidence.

The Prompt 16 review explicitly distinguishes these documentation, process, and harness observations from defects in the accepted WS-A behavior.&#x20;

adv_review_prompt16.md



## **2. Decision on L-05**

APPROVED — OPTION A

Classify the trust-mismatch choke refusal as retryable.

I prefer the minimal correction because the existing behavior already provides the required security properties:

- No unauthorized workflow submission.
- No approval bypass.
- No new dispatch-to-park transition.
- Retry using the original confirm trust converges.

One important qualification: **retryable must not mean blindly retry the same request indefinitely.**

The error classification must communicate that the caller needs to retry through the appropriate confirm-trust path. Preserve the distinction between caller-actionable retry and automatic transient retries.

Add tests for both the classification and the actual state behavior. A classification test alone is insufficient.

Do not choose option B. It introduces a new dispatch-to-park edge and additional lifecycle behavior without a demonstrated need.&#x20;

adv_review_prompt16.md



## **3. Additional decision: advisory-lock correction**

Include the §4 correction in Phase 1.

The corrected normative rule should state:

> Advisory locks are acquired outermost-first, before acquiring transactional row or table locks. An advisory lock must never be acquired while holding those transactional locks. Once acquired, it may remain held while the protected operation obtains its subsequent database locks, subject to the established release and failure semantics.

Do not merely replace the old sentence. Produce the no-cycle proof from the actual call graph.

In particular, investigate whether any path can acquire a row/table lock and subsequently request an advisory lock while another path holds the advisory lock and waits for that row/table lock. Examine Class A and every relevant caller, not only `classa.go`.

Record the correction in §13, including the reason the old wording was reversed. The exact lock terminology must distinguish advisory locks from the transactional locks they precede.

## **4. Authorization for Prompt 17**

PHASE 0 AUTHORIZED

Proceed in the specified order.

**Phase 0 — Baseline verification**

The following conditions are hard stops:

1. Recompute the `e20a4689…` acceptance file-set digest with the exact §6 command.
2. Reconcile the current `HEAD 982aecf` file set against that acceptance state.
3. Enumerate every delta. Review/decision documentation and register changes may be explainable; any source-code delta stops the work.
4. Verify the report hash and all WS-A review artifacts.
5. Record Go 1.27.3, DBOS SDK v1.9.0, local PostgreSQL 19.0, and the serial `-p 1` methodology.
6. Preserve the accepted state and create the frozen red baseline through isolated directory copies. No branches, commits, or pushes.
7. Establish the per-finding `miniapp_red` overlay procedure.

Do not begin implementation during baseline verification.

**If any discrepancy appears, stop and submit the exact discrepancy and evidence.** Do not repair, normalize, or explain away a source delta.

### **One additional diagnostic: the GitHub Actions failure**

There is an important connection to Prompt 16's O-05 observation.

The failed GitHub run executed `go test ./...`, which uses package-level parallelism by default. The two restart tests failed after 120 seconds without producing their `submitted` markers. The Prompt 16 review separately documented shared-database interference when test packages execute concurrently, and established serial `-p 1` execution as the supported methodology.&#x20;

adv_review_prompt16.md



Therefore, I am revising my earlier diagnosis: **shared-database cross-package contention is now the first hypothesis to test**, ahead of a DBOS worker-startup defect. It is not yet proven to be the cause of this particular GitHub run.

During Phase 0, compare:

- The failing default-parallel test invocation.
- The same full suite using `go test -p 1 ./...`.
- The two restart packages individually, with isolated disposable databases where appropriate.

Do not alter worker startup or increase timeouts until this comparison identifies the actual failure mechanism. Record the result. No CI changes are authorized merely to make the run green.

## **5. Conditional continuation**

Once Phase 0 is verified clean, the approved execution order is:

1. **Phase 1 — M-01 and contract corrections**

   Predicated lifecycle transitions, deterministic lock-gate races, advisory-lock correction, L-01 through L-04, and O-05.
2. **Phase 2 — Effect-authorization transaction**

   Canonical `authorizeAttempt`, pinned grant snapshot, `GenerationFence`, `approvals.ByInvocationTx`, and all eleven required persistence-level tests.
3. **Phase 3 — O-01 TOCTOU proofs**

   Six post-submission authorization-change tests, including the kill/restart replay case.
4. **Phase 4 — L-05/L-06 and exit evidence**

   Retry classification, denial audit, full verification, per-finding red/green, and the ten-section WS-B exit report.

The final exit report must stop for independent adversarial review. Passing tests alone does not constitute acceptance.

**Final owner decision:** Phase 0 may start now. All four implementation phases are authorized conditional on a clean Phase 0. Local PG19 only, mock effects only, forward-only migrations, no Neon, no OpenCode, no commits, and no pushes.

The governing requirement remains unchanged: **no invalid approval, invocation, authority, grant, lease, or effect state may cross the&#x20;**`ATTEMPTING`**&#x20;boundary.**