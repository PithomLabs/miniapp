Gate 1 CLEAR — RELEASED

I accept the independent review. **The remaining Stage 3A work is authorized to resume.** The two HIGH findings are closed, C2 is restored, and the remaining issues are LOW-level disposition and traceability work—not foundational blockers.&#x20;

adv_review_prompt26.md

adv_review_prompt26.md



I am making the two outstanding owner decisions now:

- **P26-001 — Confirmed.** The restriction against bypassing a hostile approval applies when a caller attempts to change the trust claim of an invocation that was persisted as Confirm. A genuinely persisted Autonomous invocation may proceed without approval linkage when the admitted manifest explicitly permits Autonomous. An approval record that cannot legitimately belong to that Autonomous invocation does not create an approval requirement.
- **P26-002 — Accepted as a bounded residual.** Empty `TrustModes` retains its legacy unrestricted interpretation for admission, but does not grant the explicit Autonomous approval-linkage exception. Pin the resulting gated-Autonomous behavior with a test. Before any empty-mode production manifest enables gated tools, submit-side validation must prevent admission of an invocation that authorization cannot perform.
- **P26-003 — Required editorial correction.** Restore the original C1–C7 definitions. C6 remains mechanism-sound but integration-unproven; P24-010 did not close it.

**My next instruction is Prompt27: close these three LOW findings and establish a clean, reconciled baseline.** Then we proceed with the remaining Stage 3A implementation. This keeps the next evidence gate free from known documentation ambiguities.

# Prompt27 — P26 LOW-Finding Dispositions and Gate 2 Baseline Preparation

## Role

Act as the implementation owner for `github.com/PithomLabs/miniapp`, following the independent Prompt26 Gate 1 review.

Gate 1 is CLEAR. The owner has authorized resumption of the remaining Stage 3A work, but this work order is deliberately limited to closing P26-001, P26-002, and P26-003 and reconciling the working-tree baseline.

Do not begin the remaining Stage 3A implementation in this task.

## 1. Governing decisions

### P26-001 — Confirm the P24-005 owner interpretation

The owner confirms:

The prohibition against bypassing denied, expired, mismatched, or cancelled approval through a caller-supplied trust claim applies to an invocation whose authoritative persisted trust is Confirm.

A genuinely persisted Autonomous invocation may proceed without approval linkage only when the admitted manifest explicitly permits Autonomous.

A synthetic, stale, or otherwise inapplicable approval record does not create an approval requirement for a legitimate persisted Autonomous invocation. The exception must never permit a caller to change a persisted Confirm invocation into Autonomous.

Record this confirmation as an explicit owner disposition in a durable document. Reference the Prompt26 finding and the relevant authorization design and contract sections.

Do not change the implementation merely to make a synthetic approval record veto a legitimate Autonomous invocation.

### P26-002 — Pin the empty-TrustModes legacy behavior

The owner accepts the following bounded residual:

- An empty `TrustModes` list retains its existing legacy unrestricted behavior for persisted-trust admission.
- An empty list does not explicitly permit Autonomous for the gated approval-linkage exception.
- Therefore, a gated Autonomous invocation admitted through the legacy empty-mode submission path can be refused at authorization with `approval-missing`.

Add a focused matrix leg that exercises the actual submit-to-authorization behavior with an empty-mode manifest.

The test must establish:

1. What submit admits.
2. What persisted trust is recorded.
3. What authorization subsequently decides.
4. The exact refusal reason.
5. That the refusal does not cause an unauthorized send, indefinite retry loop, or misleading success response.

Classify this test honestly. Do not claim red/green unless the pre-fix tree actually produces a different result.

Record the activation trigger:

Before any empty-mode production manifest enables gated tools, submit-side validation must prevent admission of trust combinations that cannot pass the authorization boundary. The preferred durable direction is explicit trust-mode declaration for gated tools, or an equivalent manifest-admission rule that preserves existing legacy behavior outside that boundary.

Do not introduce a broad manifest migration or silently redefine empty-list semantics in this task.

### P26-003 — Correct C1–C7 traceability

Correct the Prompt25 closeout report's §11. It relabels several constraints and obscures the still-unproven C6 integration requirement.

Restore the original Prompt21 definitions:

| Constraint | Original meaning                                                                                                           | Correct status                                                       |
| ---------- | -------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------- |
| C1         | `authorizeAttempt` takes no advisory lock                                                                                  | Holds                                                                |
| C2         | Authorization follows the global lock order                                                                                | Holds; P24-003 restored Settle's effect-lock-only footprint          |
| C3         | Authorization uses its own transaction, not the attempt's `pgx.Tx`                                                         | Holds                                                                |
| C4         | No new advisory-lock acquirers                                                                                             | Holds                                                                |
| C5         | Explicit pending semantics                                                                                                 | Holds                                                                |
| C6         | Class A handlers that call `Perform` handle `ErrAttemptInProgress` with bounded recovery and correct operation-ID behavior | Mechanism sound; integration unproven; fixture and guidance deferred |
| C7         | O-01 orphan window closes at the effect boundary                                                                           | Holds; strengthened by P24-001                                       |

P24-010 corrected contention guidance. It did not complete C6.

Do not describe C6 as closed. Preserve its explicit deferral and required future evidence.

Use the Prompt26 §7 table as the authoritative corrected mapping.

## 2. Worktree reconciliation

Prompt26 identifies additional worktree changes after the Prompt25 evidence manifest:

- `docs/prompt26.md`
- An outside-activity append to `docs/faq.md`
- The Prompt26 independent review report itself

Inventory these changes against the current HEAD and worktree.

Preserve the `docs/faq.md` append. Do not reset, discard, overwrite, or attribute it to the Prompt25 or Prompt26 implementation without evidence.

Record its exact diff, current hash, and out-of-scope status. Identify the Prompt26 review report and prompt document as separate review artifacts.

Do not silently include these late changes in historical Prompt25 digests. Establish a new, explicit baseline for the next workstream.

## 3. Required implementation order

1. Freeze the current HEAD and complete worktree inventory.
2. Reconcile the three late-arriving files above.
3. Record the P26-001 owner disposition.
4. Add the P26-002 empty-mode integration matrix leg and activation-trigger documentation.
5. Correct Prompt25 report §11 for P26-003.
6. Update the relevant design/contract/evidence references without rewriting historical evidence.
7. Run focused tests and required repository checks.
8. Write the closeout report and record the new baseline digests.

## 4. Validation

Use the exact pinned PostgreSQL 19 image:

`postgres@sha256:bb6e70add742cd868fb62cb3b2f67d1dd2bdd2b65a521e778420748753687c60`

Re-resolve it through Docker inspection. Use disposable databases only. Do not touch the long-lived development database.

Run the new empty-mode integration test against PostgreSQL 19, the affected authorization/runtime package suites, `gofmt -l .`, `go vet ./...`, and `bash scripts/checks.sh`.

Run the full serial test suite if the changed contract or test fixture affects broader package behavior.

Classify evidence accurately: execution-proven, source-proven, characterization-only, or accepted residual.

## 5. Deliverable

Create:

`docs/phase2/prompt27-p26-low-findings-closeout.md`

Include:

- P26-001 owner confirmation and its exact scope.
- P26-002 test behavior, results, and production activation trigger.
- P26-003 corrected C1–C7 table and explicit C6 status.
- The worktree reconciliation, including the `docs/faq.md` delta.
- New source/tree digests and manifest identity.
- Exact PostgreSQL image digest and server version.
- Test commands and observed results.
- Any residuals and their activation triggers.

## 6. Hard boundaries

Do not:

- Begin the remaining Stage 3A implementation.
- Implement the C6 fixture.
- Wire production Settle or operator authentication.
- Implement revocation tooling.
- Add provider bypass or full crash matrices.
- Introduce migrations or dependency changes.
- Commit, stage, push, deploy, or invoke real provider writes.

Stop after the closeout report.

The next work order will resume the remaining Stage 3A implementation under a newly frozen baseline. Gate 1 is already CLEAR; Prompt27 is bounded housekeeping, not a request to reopen Gate 1.

**After Prompt27:** the next substantial task will be the remaining Stage 3A implementation, followed by Gate 2 independent review. C6's fixture, revocation tooling, Settle wiring and operator authentication, full authorization/crash matrix, and bypass suite remain in that work—not in this cleanup.