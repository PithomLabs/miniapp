## deepseek

# Review: `prompt15.2_plan.md`

## Verification limits stated up front

I do not have `docs/prompt15.2.md` or `docs/prompt15_z.md` in this conversation, only the plan that cites them. I therefore cannot verify the plan's interpretation of the governing prompt directly. What I can verify:

- **Internal consistency** — whether the plan's phases, sequences, and boundaries are coherent with each other.
- **Coverage of the adversarial review** — whether every HIGH finding from `adv_review_prompt14.md` (X-01 through X-05) and every root-cause-sharing MEDIUM finding is addressed by some phase, test, or explicit deferral.
- **SDK claim plausibility** — whether the line-precise v1.9.0 citations are internally consistent with the behaviors the plan attributes to them.
- **Boundary preservation** — whether the standing prohibitions (lead-followup hard stop, no commits, forward-only migrations, owner-decision surfacing) are maintained.
- **Discipline inheritance** — whether the root-cause-before-fix principle is actually operationalized, or merely cited.

Where I cannot verify a claim because I lack the cited source, I note it as **NOT VERIFIABLE FROM THIS REVIEW'S EVIDENCE** rather than approving or rejecting it.

## Verdict

**APPROVED to proceed to P0 and P1 as written.** No structural blockers. Seven refinements recommended before P2 begins — all clarifications, none requiring re-planning.

The plan honors the structural discipline that the prior cycle failed to establish. It verifies SDK behaviors against the actual pinned source rather than against documentation claims. It sequences work by dependency rather than by finding list. It surfaces the two remaining owner decisions rather than silently resolving them. And it correctly rejects the universal `transition()` primitive approach from a competing prompt draft that 15.2 superseded.

The plan can start. It should not proceed from P1 to P2 until the seven items below are resolved — most are quick, but one (the P1 contract review authority) is substantive.

## What the plan gets right

**SDK verification is done properly.** The plan reads `/home/chaschel/Documents/go/dbos-transact-golang-1.9.0` and confirms byte-identical to the pinned v1.9.0 module source via `diff -rq` before citing it. The line citations for each behavior (`InsertWorkflowStatus` at `system_database.go:1471` with `RejectExisting` enforcement at `:1700`; `SysDB.Send` at `:5037` with no FK on `workflow_status` at `:2570`; `Shutdown` returning timeout enumeration including `"workflows"` at `dbos.go:1148`; `ReenqueueForRecovery` filtering on `application_version` at `:6349`; the v1.9.0 enum at `workflow_status.go:13–20`; `Recv` with timeout at `workflow.go:3976`) are the right kind of evidence — the plan is reading source, not summarizing it.

**RC-1/RC-2 are not pre-accepted.** P1 explicitly says "§1 above is the head start, not the verdict" and requires ratification, revision, or rejection in writing before P2 starts. This is the discipline the root-cause principle demands and the prior cycle did not follow.

**The universal `transition()` primitive is correctly rejected.** `docs/adv_review2_prompt14.md` mandated it and the two `classd-state-transition-*.md` documents; the plan explicitly forbids both. The alternative — six normative transition-integrity rules plus subsystem-specific transition operations — is the correct response to the actual problem (state-transition discipline, not a syntactic abstraction).

**Sequencing is dependency-driven, not finding-driven.** WS-C (X-07 mock measurement integrity) comes first because "later effect evidence depends on it." WS-A (X-01 through X-04) groups the `runtime/`-file findings with X-19 and X-22 because they touch the same files. WS-B (X-05) precedes WS-D because cancellation-vs-claim is the more fundamental race. This ordering is correct.

**Red/green evidence is specified.** "Red/green evidence for every HIGH finding and X-07 via an isolated pre-remediation snapshot (separate database names, both runs bound to exact source digests, active tree untouched)" is exactly the discipline that would have caught the prior cycle's "passed but not actually exercised" problem.

**Standing boundaries are preserved and named.** The plan explicitly lists: ADR-001 `lead-followup` hard stop, no commits or pushes, no protected-document edits, forward-only migrations, Neon/remote-CI honestly recorded as BLOCKED / unobserved when not run, stop only for genuine governing-spec conflicts, unresolved owner decisions, or unsafe/unauthorized mutations. This is the right articulation.

**The two owner decisions are surfaced, not invented.** P6 Open Questions correctly identifies (1) supersession of the prior prompt drafts and (2) OpenCode authorization as owner decisions. The plan does not assume either. Q2 explicitly offers the "mark OpenCode BLOCKED by owner decision" path if authorization is not forthcoming.

## Concerns (non-blocking)

### C1 — P0 digest provenance is not independently verified

P0 states "worktree digest `5564a37b` (247 files)" as a carried-over fact. The plan does not say:
- How that digest was computed.
- Whether the working tree has moved since it was computed.
- Whether any P0 evidence entry references it with a verifiable method.

If `5564a37b` was computed by the P0 agent using some ad-hoc method, and a different agent now computes it differently, "the digest holds" is unverifiable. Since P1 will not touch the tree, P0's digest remains the only bindable evidence baseline for the pre-remediation snapshot in P3/P4. It needs to be reproducible by someone who is not the P0 agent.

**Recommendation:** P1 documents the exact command(s) that produce `5564a37b` (or whatever digest is authoritative) and confirms reproducibility from a clean check.

### C2 — P1 contract review authority is unspecified

P1 produces `docs/phase1/invocation-lifecycle-contract.md` and says "No source edits until this document exists." That is necessary but not sufficient. A contract document authored by the same agent who will implement it is a discipline check, not a review check. The prior cycle's failure mode was not "no contract existed" — it was "the contract, the implementation, and the self-assessment were all authored by the same agent who was incentivized to consider the work complete."

Who reviews the contract before P2 begins? The plan says "reviewed" in the *root-cause principle* context (via `prompt15.2.md` §... which I don't have) but does not name a reviewer, a review method, or an acceptance criterion for P1 exit.

**Recommendation:** P1 explicitly names the P1 exit criterion as "contract document exists AND has been reviewed by a reviewer who is not the P2 implementer." If the same agent will do P1 and P2, that is a risk to surface explicitly in the plan, not to leave implicit.

### C3 — The "six normative transition-integrity rules" are not named or sourced

P1 references "The six normative transition-integrity rules" and "subsystem-specific transition operations (no universal `transition()`)." The six rules are not enumerated in the plan. If 15.2 defines them, the plan should cite the section. If the plan is drafting them in P1, that is a significant design decision that should be visible in the plan, not deferred to the P1 document.

The reason this matters: the six rules are the operational definition of what "the discipline" means. If they are wrong or incomplete, everything downstream is wrong or incomplete. The plan's own claim — that the remediation leaves behind a "structural mechanism that makes the class of finding impossible or caught" — rests entirely on those six rules being the right six rules.

**Recommendation:** Either cite 15.2 §... where the six rules are defined, or enumerate them in the plan's §3 so they are reviewable before P1 begins. The former is fine if the rules exist; the latter is necessary if P1 is authoring them.

### C4 — Submission choke-point contingency is absent

P1 requires a "submission choke-point specification with call-graph / grep / automated-test proof that every executor submission passes through it." That is the right audit. The plan does not say what happens if the audit finds a bypass.

The prior cycle's failure mode was "the audit was performed, found no bypass, and the conclusion was 'no bypass exists' — with no contingency if the audit was wrong or incomplete." The remediation should specify what P2 does if the choke-point audit in P1 produces a partial result, ambiguous evidence, or a genuine bypass on a code path the plan did not anticipate.

**Recommendation:** P1 documents the choke-point audit result including negative findings, and P2 treats "the audit found a bypass" as a stop condition requiring an owner decision, not as a problem P2 silently fixes.

### C5 — X-17 and X-18 phase scoping is ambiguous

The plan groups WS-H (X-17, CI with no postgres service) and WS-I (X-18, evidence digest) into a single bullet in P2 without specifying:
- Whether "add postgres service to CI" is the fix (bounded, single PR) or whether the fix is "document the limitation and gate G1 differently" (also bounded, but different).
- Whether "fix X-18" means retroactively re-digesting the prior cycle's evidence or only binding the current cycle's evidence.
- Whether WS-H and WS-I can be completed independently or whether WS-I depends on WS-H's CI being observed.

X-17 is load-bearing for the "CI enforces Class D rules" G1 row. X-18 is load-bearing for "evidence recorded and verifiable." Both need clear scoping before P2 starts.

**Recommendation:** P2's WS-H and WS-I each state their acceptance criterion in one sentence. For X-17, "CI runs the DB-backed tests against a postgres service, observed green on a remote runner" vs. "CI limitation documented and G1 row 12 marked BLOCKED" are very different outcomes; the plan should say which one this cycle targets.

### C6 — WS-D groups a policy decision with mechanical fixes

WS-D bundles X-06 (signal existence verification — mechanical), X-11 (bounded approval wait — policy), X-12 (settlement audit — mechanical), X-13 (refusal records — mechanical), X-14 (shutdown failure handling — mechanical).

X-11 requires choosing **one** enforceable mechanism from {framework-mandated Recv-timeout ceiling, reconcile sweeper, poll-driven renotify}. The plan says "P1 selects exactly one enforceable mechanism per 15.2 §8/D" — correct that the selection happens in P1, but the selection has governance implications. A Recv-timeout ceiling changes what miniapps must do; a reconcile sweeper adds infrastructure; a poll-driven renotify couples approval convergence to polling.

**Recommendation:** The P1 contract's X-11 section names the selected mechanism, the alternatives considered, and the reason the selected one is enforceable without requiring miniapp cooperation. If the selected mechanism requires any owner decision (e.g., "miniapps must use Recv with timeout <= N minutes"), that decision is surfaced in P1, not embedded in P2.

### C7 — P3/P4 "acceptance matrix (§14)" is underspecified

P3/P4 references "acceptance matrix (§14)" without enumerating it. The plan's own §14 is not included in `prompt15.2_plan.md` — the section numbering jumps from §13 to §6 in the Open Questions block, which suggests the plan's §14 reference is to `prompt15.2.md`'s §14, not its own. But this is not stated.

The consequence: P3/P4's acceptance criterion is defined by reference to a document the plan does not quote and the current reviewer cannot see. The plan must either quote the acceptance matrix or state explicitly which prompt section it references so that the P3/P4 completion can be verified against the actual text.

**Recommendation:** P3/P4 explicitly cite the section number in `prompt15.2.md` where the acceptance matrix is defined, and either quote it or note its exact scope in one paragraph so the P3/P4 exit gate is reviewable without re-opening the governing prompt.

## Structural assessment

The plan is structurally sound. It:

- **Establishes contract-before-code** as the P1 → P2 gate.
- **Verifies SDK behaviors** against pinned source, not documentation.
- **Correctly rejects** the universal `transition()` primitive from a superseded prompt draft.
- **Sequences by dependency** rather than by finding enumeration.
- **Requires red/green evidence** for the HIGH findings, not just passing tests.
- **Surfaces owner decisions** rather than resolving them silently.
- **Preserves all standing boundaries** including the lead-followup hard stop, no commits, forward-only migrations, and honestly-reported Neon/remote-CI status.

The seven concerns above are refinements to how P1 and P2 are executed, not challenges to whether the plan is the right plan. None of them requires reordering the phases or reopening a decision that the plan already makes.

## Conditions for P2 to proceed

P1 exit must include, at minimum:

1. **C1** — reproducible digest command documented, `5564a37b` (or its replacement) confirmed verifiable.
2. **C2** — contract review authority named; if the same agent will implement, that risk is stated explicitly in the plan.
3. **C3** — six normative transition-integrity rules either cited from `prompt15.2.md` or enumerated in P1's contract document.
4. **C4** — submission choke-point audit result documented, including the negative-finding case; if a bypass is found, P2 stops for owner decision.
5. **C5** — WS-H and WS-I each state their acceptance criterion as a single sentence.
6. **C6** — X-11 mechanism selection documented with alternatives and any implied owner decisions.
7. **C7** — acceptance matrix referenced with its actual source section, quoted or scoped.

If any of these cannot be resolved in P1 without touching a protected document or making a decision reserved for the owner, that blocker is documented in the P1 exit report and P2 does not begin until the owner resolves it. The plan's own "stop only for genuine governing-spec conflicts, unresolved owner decisions, or unsafe/unauthorized mutations" applies here: an unresolved C2 or C6 is an unresolved owner decision.

## Recommendation

**P0 and P1 can begin now.** P2 waits on P1 exit including the seven conditions above. P3 and P4 follow P2 completion per the plan's existing sequence.

The plan is the right plan. The refinements are how the plan should be executed, not what the plan should be. The next action is not a revision to this plan — it is the start of P0 work with P1's conditions surfaced in the P1 exit report.

The root-cause principle from the prior cycle — *diagnose before fixing, establish discipline not patches, demonstrate the discipline addresses the findings* — is honored in this plan's structure. If P1 produces the contract document it specifies, and P2's remediation is demonstrated against that contract rather than against the finding list, the class of defect that produced X-01 through X-05 cannot silently recur. That is what success looks like, and this plan is the shape that gets there.


## Z

# Review of prompt15.2_plan.md — Remediation Plan for the Prompt 14 Findings

| Field | Value |
|---|---|
| Reviewer role | Independent plan reviewer (adversarial, pre-implementation gate) |
| Object under review | `upload/prompt15.2_plan.md` (114 lines, PLAN ONLY — no implementation authorized) |
| Verifiable inputs held by this review | `docs/adv_review_prompt14.md` (full), the Prompt 15 root-cause lineage (`prompt15.md` §3/§6 — RC-1a–d, RC-2, six contract rules) |
| **Not verifiable here** | `docs/prompt15.2.md` (the governing prompt — every §-reference to it), `docs/prompt15_z.md`, `docs/prompt15.1.md`, `docs/prompt15_plan_review.md`, `docs/adv_review2_prompt14.md`, the SDK clone (`/home/chaschel/Documents/go/dbos-transact-golang-1.9.0`), HEAD `5cf9627`, digest `5564a37b` |
| Deliverable | This review only; no repo files touched |

## A. Verdict: CONDITIONALLY ACCEPTABLE — proceed to P1 after required amendments; P2 start is gated on the amended map

The plan is structurally sound and, on the evidence verifiable here, materially better discipline than the Prompt 13 round: contract-before-code is retained as a hard gate, the measurement instrument is built before the evidence that depends on it, red/green is planned via isolated pre-remediation snapshots bound to digests, the SDK claims are pinned to exact v1.9.0 citations, supersession is raised as an owner question instead of assumed, and Neon/remote-CI/OpenCode are honestly planned as BLOCKED/unobserved rather than manufactured. The plan's two governance choices (no universal `transition()` primitive; genuine-attachment default for X-08) are both defensible — the first is compatible with the Prompt 15 lineage (which mandated a per-row predicate table, never a primitive), the second is actually **more spec-aligned** than the lineage's either/or: `prompt13` §7 (per X-08 field 3) requires invocation records to preserve identity when attaching to an existing run.

It is not acceptable as-is, because one HIGH completeness gap drops a required class of work (the F-01–F-14 residuals that are not X-IDs), and several MEDIUM gaps let multi-part fixes silently degrade into their one-line WS summaries. All are amendable in the plan document; none reopens the plan's structure.

## B. What the plan gets right (verified against held inputs)

1. **§1 SDK table settles the design loci correctly.** X-03: `InsertWorkflowStatus` upserts `ON CONFLICT (workflow_uuid)` with `RejectExisting` self-write detection — so same-ID resubmit attaches, but two framework-minted run IDs defeat SDK dedup entirely; the predicated-adoption fix must live in `runtime/store.go`. This kills the tempting wrong answer ("the SDK already dedups") with a citation. X-06: `SysDB.Send` is a blind INSERT with no FK on `workflow_status`, and no atomic check-and-send API exists — so existence-verify via `RetrieveWorkflow` plus a **documented residual TOCTOU** is the honest resolution shape the review's field 9 allows. X-14: `Shutdown`'s timeout error enumerates pending components — detection exists; silent `pool.Close()` is indefensible. X-10: the plan pins the full v1.9.0 enum (including `SUSPENDED`, `MAX_RECOVERY_ATTEMPTS_EXCEEDED`), which tightens the review's "all non-terminal states" into a checkable list.
2. **Ordering is evidence-driven.** X-07 (the measurement instrument) precedes every effect-evidence fix — correct, since takeover/kill/restart/probe send-count claims are vacuous until the log records every attempt. Contract-before-code is preserved verbatim ("No source edits until this document exists").
3. **The lock-discipline line is the correct X-05 design constraint.** "Never hold a PostgreSQL transaction or lock across an external send" forces the right atomic scope: guard + claim INSERT + fossilizing `PENDING→ATTEMPTING` transition inside one predicated transaction, committed **before** the send — which is exactly the fossilizing-transition design the review's §F effects analysis verified as sound. It also rules out the wrong fix (holding a lock across the provider call).
4. **Digest discipline is real this time.** Worktree digest `5564a37b` (247 files) recorded at P0; both red/green runs "bound to exact source digests." Directly answers X-18.
5. **Governance honesty.** The supersession question (§6.1) is asked rather than assumed — a prior GO exists for a different structure, and the plan refuses to ride it. Neon/remote-CI recorded BLOCKED/unobserved matches the review's G1 rules (row 12 cannot pass until remote CI executes or the gap is owner-accepted). The "13/14 claim withdrawn" line matches §C19 (CONTRADICTED).
6. **Superseded-materials list is explicit** — including naming `adv_review2_prompt14.md` as a competing draft whose universal-`transition()` design was rejected. Rejecting the primitive while keeping per-row predicates is sound: predicates are what get audited; a primitive is only an implementation shape.

## C. Findings

Severity scale as in the review family: HIGH = required work dropped or invariant weakened; MEDIUM = plan gap that must be amended before the phase it affects; LOW = precision/hardening note.

### PR-01 (HIGH — plan completeness): the F-01–F-14 residuals are absent from the work list

The plan maps **X-01–X-24** (§3, §4) and rebuilds F-01–F-14 only as *reporting* (§5). But the review's §D and §F name work items that carry no X-ID and that the remediation lineage requires closed (Prompt 15 §11.C: "plus the F-01–F-14 residuals named in the review (§D)"). Concretely, from the review:

- **F-03:** stale-resume-after-takeover interleaving is untested; `Settle` is unaudited/unexposed.
- **F-10:** the signal-timing matrix is missing three cases — signal-after-timeout, signal-then-kill, signal delivered while the workflow's process group is unavailable.
- **F-02:** lazy-expiry end-to-end with a real workflow is missing; `settleDenied` "confirmed down" overclaims cooperative cancel.
- **§F cancellation:** the cooperative-cancel reality (in-flight steps complete by SDK design) is "correctly disclosed nowhere — add one line to the ops doc."

These are invariant-adjacent tests and disclosures, not prose. **Required:** add an explicit residual-closure list to the §3 map (each residual → WS → test/doc), and assign the four items above to WS-B/WS-D/WS-E scope. P2 start is gated on this.

### PR-02 (MEDIUM): the six state machines do not visibly own three findings' transitions

The §3 machine list (invocations, approvals, effects/attempts, workflow/run identity, sequences/attachments, cancellation) leaves the following without an obvious owner in the transition table: the **Signal/notification path** (X-06 — which machine's row makes "signal unknown run" an error?), the **kill-switch control plane** (X-15 — engage/disengage transitions, their authorization and audit), and **foreign-version visibility** (X-10 — enumeration rows per non-terminal state). An X-finding whose transitions have no owning machine tends to ship as a patch outside the contract. **Required:** either extend the machine list or add a machine→findings ownership row in the contract doc covering every X-ID; no X-ID may be absent from the ownership map.

### PR-03 (MEDIUM): the sanctioned serial invocation must be baked into CI, not just recorded

P0 records the real constraint: the parallel suite is red on the shared local DB; the sanctioned invocation is `go test -p 1 ./... -count=1` (per Prompt 13), and the plan correctly flags that 15.2 §6's bare `go test ./...` needs this qualification. But WS-H (X-17) is where the CI workflow gets written — if CI is built around bare `go test ./...`, remote CI repeats the parallel red forever and row 12 stays unpassable for a second reason. Relatedly, the parallel-red **root cause** (cross-package interference on the shared DB — the review's O-7 "parallel truncation" disposition) should be named in the contract/ops doc so `-p 1` is a documented, understood constraint rather than folklore. **Required:** WS-H must specify CI uses the sanctioned invocation (or an explicitly re-engineered parallel-safe fixture set), and record the interference mechanism.

### PR-04 (MEDIUM): the standing-boundaries list is missing three constraints

§5's boundary list (hard stop, no commits/pushes, protected docs, forward-only migrations, honest BLOCKED recording, stop conditions) omits: **mock-only channels / no real provider writes**, **no weakening of existing tests** (skip-fast gating only), and **disposable-DB-only** (no shared/production Neon mutations — §2's "production untouched" says it once for P0 but not as a standing rule). A plan that will be executed over days should carry its fences with it. **Required:** restate the three, or cite the governing 15.2 sections verbatim.

### PR-05 (MEDIUM): X-08's genuine-attach default must not absorb the fix's unconditional parts

Defaulting to genuine run-attach is spec-aligned (see §B.6 here) and better than the lineage's either/or — but three parts of X-08's resolution are unconditional regardless of attach-vs-detect: the **supersede→cancel linkage** (superseded probe runs currently keep sending), the **barrier-synced claim race test** (replacing the sequential `TestProbeConcurrency`), and the **loser-status semantics** (no `Succeeded` while the sequence is active). The plan's WS-E one-liner names only "genuine attachment default." **Required:** state the unconditional parts in the WS-E scope, and record the attach decision as an owner-decision item in the §17.2 report even if 15.2 pre-decided it.

### PR-06 (MEDIUM): multi-part fixes must survive the one-line WS summaries

Several review resolutions have sub-parts the plan's compact WS entries can silently drop. The §3 map must carry each finding's **full sub-fix list** from the review's field 9 — specifically: **X-01** (adopt the dangling pending approval row from the failed park; exactly-one `approval-requested` audit), **X-02** (all three convergence paths: grant, denial, expiry on a never-submitted run), **X-04** (predicate discipline extends to expiry-marking *and* `CancelPending`, not just `Decide`), **X-11** (three parts: enforced bound, commit-time audit — currently audited only after successful signal — and striking the nonexistent "durable mailbox" phrase), **X-07** (the false comment and surviving exactly-once phrasing, not just the log schema). **Required:** amend the map to per-part fidelity before P2.

### PR-07 (LOW): name the pre-remediation snapshot mechanism

"Isolated pre-remediation snapshot … active tree untouched" is the right idea; leave the mechanism named — a git worktree at the recorded digest or an equivalent snapshot — with **both** digests (red run, green run) in the report. Otherwise the remediation round reproduces inside itself the exact X-18 ambiguity it exists to fix.

### PR-08 (LOW): bind the SDK clone to evidence

`diff -rq` clean is asserted, not recorded. Put the clone's digest (or module-cache hash) in the evidence register, and keep the machine-local path (`/home/chaschel/...`) out of the register — cite the pinned module version and digest so the evidence survives the machine.

### PR-09 (LOW): WS-A's file rationale is loose

X-04 lives in `approvals/approvals.go`, not the `runtime/` store/service files that justify grouping X-01–X-04 with X-19/X-22. Keep the grouping (one workstream is fine); fix the stated rationale so the dependency-ordering logic stays honest.

### PR-10 (OBSERVATION): unverifiable inputs — accepted as recorded claims, surfacing at P1

This review cannot open `prompt15.2.md` or the other superseded/competing documents, the SDK clone, or the repo. Every §-reference to 15.2 (§5/§6/§8/§12/§14/§17.2), the supersession claims, the byte-identity claim, and the P0 numbers (HEAD, digest, preflight counts, Neon probe) are accepted as recorded claims. The plan's own design makes this safe: P1 requires ratifying RC-1a–d/RC-2 and authoring the contract **in writing against source + pinned SDK**, which is exactly where any contradiction between 15.2's text and the plan's structure would surface — before code. No action needed beyond keeping that gate intact.

## D. Phase-by-phase assessment

| Phase | Assessment |
|---|---|
| P0 (baseline) | Sound; digest recorded; serial-invocation qualification correctly flagged (see PR-03). OpenCode preflight correctly held for owner authorization. |
| P1 (contract) | Correct gate and contents; strengthen with PR-01 (residual map), PR-02 (machine ownership), PR-06 (per-part fidelity). "§1 is the head start, not the verdict" is the right posture for ratification. |
| P2 (remediation) | Ordering is evidence-driven and dependency-correct; amend per PR-01, PR-05, PR-06; name the snapshot mechanism (PR-07). |
| P3/P4 (proof, close-out) | Matches the required battery: barrier-synced `-race` suites, kill/restart, security re-verification, acceptance matrix, G1 rebuild from `prompt10.md` §9 with the 13/14 claim withdrawn. Ensure per-finding red/green **runs** (not just the snapshot plan) land in §17.2's closure records. |
| §6 (owner questions) | Both questions are the right ones and correctly block the affected scope only (OpenCode → G1 row 10; supersession → plan authority), not the whole plan. |

## E. Finding-coverage check (X-01–X-24 → plan location)

| Findings | Plan location | Gap |
|---|---|---|
| X-01, X-02, X-03, X-04 | WS-A (§4.2), SDK locus §1 | Per-part fidelity (PR-06) |
| X-05 | WS-B (§4.3), lock discipline §3 | None — design constraint is correct |
| X-06 | WS-D (§4.4), SDK §1 | Machine ownership (PR-02) |
| X-07 | WS-C first (§4.1) | Phrasing/comment sub-fixes (PR-06) |
| X-08, X-09, X-10 | WS-E (§4.5), enum §1 | Unconditional parts (PR-05) |
| X-11, X-12, X-13, X-14 | WS-D (§4.4), SDK §1 (X-11, X-14) | X-11 three-part fidelity (PR-06) |
| X-15 | WS-F (§4.6) | Machine ownership (PR-02) |
| X-16, X-17, X-18 | WS-G/H/I (§4.7) | CI invocation constraint (PR-03); snapshot/digest mechanics (PR-07, PR-08) |
| X-19–X-24 | WS-A (X-19, X-22) + WS-J remainder | Explicit dispositions per the lineage §9 (fix or document-and-defer; none silent) |
| F-residuals (no X-ID) | **absent** | PR-01 (HIGH) |

## F. Required amendments before GO (minimal set)

1. Add the F-residual closure list to the §3 map and P2 scope (PR-01 — gates P2).
2. Add machine→findings ownership so no X-ID is ownerless in the contract (PR-02).
3. Bind WS-H/CI to the sanctioned serial invocation and record the parallel-red root cause (PR-03).
4. Restate the three missing standing boundaries (PR-04).
5. Make X-08's unconditional parts explicit and record the attach decision as an owner item (PR-05).
6. Upgrade the §3 map to per-part fidelity for X-01/X-02/X-04/X-07/X-11 (PR-06).

PR-07/08/09 are amendments to make while editing, not GO gates.

## G. Scope limits of this review

Plan-only review; no execution, no repo access, no access to the governing 15.2 document or SDK clone (see PR-10). All verdicts about 15.2-specific requirements are conditional on that document matching the plan's description of it; all verdicts about findings, ordering, SDK design loci, and evidence discipline are grounded in `docs/adv_review_prompt14.md` and the Prompt 15 root-cause lineage held by this review.


## chatgpt

# **Review of Prompt 15.2 implementation plan**

Strong plan

Minor revisions required

**My verdict: ACCEPT WITH AMENDMENTS.**

This is a substantial improvement over the earlier implementation plans. It addresses the important architectural and evidentiary weaknesses rather than simply rearranging the finding list.

In particular, I agree with its use of the pinned DBOS v1.9.0 source as the authority for SDK behavior, its root-cause-first approach, and its decision to repair mock measurement integrity before relying on subsequent effect tests.&#x20;

prompt15.2_plan.md



I would not start implementation until the following points are incorporated.

## **1. What the plan gets right**

**P0 — Baseline discipline**

The plan records HEAD `5cf9627`, the 247-file worktree digest, serial test results, the Neon branch-identity probe, and the absence of implementation changes. That gives us a concrete starting point.&#x20;

prompt15.2_plan.md



**P1 — Design before code**

This is essential. The six state machines, six transition-integrity rules, retry-as-re-entry, submission choke-point proof, and finding-to-test mapping provide an actual engineering contract rather than a collection of implementation intentions.

**P2 — Dependency order**

I particularly agree with this sequence:

1. Repair provider-attempt measurement.
2. Correct invocation and approval state together.
3. Repair cancellation versus effect claiming.
4. Correct lifecycle and shutdown behavior.
5. Establish real run attachment and version affinity.
6. Secure the kill-switch control plane.
7. Finish analyzer, CI, evidence, and remaining hardening.

The ordering makes sense because later proof depends on earlier measurement and state-transition integrity.&#x20;

prompt15.2_plan.md



**P3/P4 — Independent proof**

The use of isolated pre-remediation snapshots for red/green testing is particularly important. It allows the agent to demonstrate that a regression test actually detects the old defect without damaging the active working tree.

The withdrawal of the 13/14 G1 claim is also correct.

## **2. Required amendments**

These are the changes I would make before authorizing execution.

### **A. P0: Strengthen the worktree digest evidence**

The plan records `5564a37b` and 247 files. That is a useful reference, but the evidence register should include:

- Full SHA-256 digest, not merely an abbreviated identifier.
- The deterministic hashing algorithm.
- The sorted file manifest or a reproducible way to regenerate it.
- Confirmation that tracked modifications and relevant untracked files are included.
- The HEAD and branch associated with the digest.

Also, record exact test counts, not just "serial suite green."

### **B. Do not hide the parallel test failure**

The plan reports that serial tests pass but parallel tests fail against the shared local database.

Using `go test -p 1 ./... -count=1` as the sanctioned suite command is reasonable if that is the established Prompt 13 execution requirement.

However, establish why the parallel suite fails.

Distinguish shared-fixture interference from actual framework concurrency defects. Do not let serial execution substitute for the barrier-synchronized concurrency tests that Prompt 15 requires.

The official suite may remain serial, but concurrent framework behavior must independently be proven.

### **C. X-05: Specify the exact cancellation/claim ordering**

The plan correctly places cancellation protection at the claim boundary.

I would additionally require the contract to define a consistent database lock order for:

- Invocation cancellation.
- Effect reservation.
- Effect settlement.
- Stale lease takeover.

This matters because fixing one race must not introduce deadlocks in another.

The critical rule remains:

**The effect reservation and cancellation must have an unambiguous database order. Never hold the PostgreSQL lock across the external provider send.**

An attempt reserved before cancellation is in flight. An attempt reserved after cancellation must be refused.

### **D. X-06: Do not describe a verified signal as guaranteed delivery**

The plan confirms that DBOS `Send` does not verify workflow existence and that `RetrieveWorkflow` is available. It correctly recognizes the residual check-then-send race.&#x20;

prompt15.2_plan.md



There is an important semantic distinction:

`RetrieveWorkflow` followed by `Send` can verify existence before attempting notification, but it cannot guarantee that the workflow remains active until notification is inserted.

Therefore, define the result as an accepted notification request, not proof that a waiting workflow received or processed the signal.

The durable approval decision must remain authoritative, and the workflow must be able to recover by rereading that state.

### **E. X-08: Make attachment acceptance criteria more explicit**

I agree with genuine run attachment as the default.

Specify the following acceptance tests in the plan:

- Two concurrent invocations resolve to one canonical workflow.
- The losing invocation receives a valid handle linked to the winner.
- Its status follows the actual target workflow.
- Target failure is not reported as success.
- Cancellation behavior is explicitly defined.
- A superseded generation cannot continue sending effects.

This is important because a successful observation record is not equivalent to attachment.

### **F. X-10: Explicitly classify DBOS statuses**

The SDK enum lists:

`PENDING`, `ENQUEUED`, `DELAYED`, `SUSPENDED`, `SUCCESS`, `ERROR`, `CANCELLED`, and `MAX_RECOVERY_ATTEMPTS_EXCEEDED`.

The implementation should classify these explicitly rather than simply removing the PENDING filter.

In particular, establish the correct terminal classification of `MAX_RECOVERY_ATTEMPTS_EXCEEDED` from the actual SDK behavior. Do not treat every enumerated status as active work.

### **G. X-11: Make the approval timeout genuinely enforceable**

The SDK's `Recv` timeout is a promising mechanism. But merely requiring miniapps to supply a timeout does not establish a framework guarantee.

The contract must establish how a miniapp is prevented from bypassing the required bound by using an unrestricted receive operation.

Possible enforcement includes a framework-owned approval-wait primitive and analyzer restrictions on prohibited direct calls.

Test both the compliant path and an attempted bypass.

### **H. X-14: Define process termination behavior**

The plan correctly identifies that DBOS `Shutdown` returns an error when components remain pending.&#x20;

prompt15.2_plan.md



Clarify the shutdown failure contract:

- Do not close the pool while workers may still use it.
- Do not swallow the shutdown error.
- Report the components that did not stop.
- Terminate through an explicit, non-zero failure path where necessary.
- Verify that deferred cleanup does not subsequently close the pool unsafely.

A fault-injection test is necessary, but also test the actual timeout behavior with an unresponsive worker.

### **I. Migration ownership needs one small correction**

The plan puts X-19 and X-22 together with the invocation workstream, which is sensible because they touch runtime identity and persistence.

However, its final WS-J item also says X-19–X-24 remainder. This creates duplicate ownership.

Change that allocation to:

| Workstream | Findings                  |
| ---------- | ------------------------- |
| WS-A       | X-01–X-04, X-19, X-22     |
| WS-J       | X-20, X-21, X-23, X-24a–f |

Also make explicit that X-22 may require a forward-only migration, not just a service-layer validation. X-19 must be assessed for persisted identity compatibility before changing the encoding.

## **3. Two owner decisions**

The plan appropriately leaves two questions open.

**Decision 1 — Supersession**

I recommend explicitly confirming that `prompt15.2.md` supersedes the older Prompt 15 drafts and that the earlier GO applied to the old structure, not automatically to this new implementation plan.

**Decision 2 — OpenCode**

The earlier GO approved the Prompt 13-scope transport, MCP, and one-agent-turn E2E.

I recommend also authorizing the isolated-HOME model preflight, provided it does not modify global OpenCode configuration or introduce unapproved credentials, installations, or costs. If the required model cannot be used under existing authorization, mark the E2E BLOCKED rather than silently substituting another model.

## **4. Recommended final implementation sequence**

1. **P0 — Verify baseline**

   Reproduce the full worktree digest and baseline test evidence. Resolve the parallel-test discrepancy.
2. **P1 — Ratify the diagnosis**

   Complete `invocation-lifecycle-contract.md`, including the transition table and test mapping. No source changes before this.
3. **P2 — Repair shared mechanisms**

   Fault injection and mock integrity → invocation and approval → cancellation/claim → lifecycle and shutdown → attachment and versioning → kill-switch controls → analyzer, CI, evidence, and remaining hardening.
4. **P3 — Adversarial proof**

   Barrier-synchronized PostgreSQL tests, injected failures, genuine process kills, and red/green comparisons.
5. **P4 — Acceptance**

   Full X-01–X-24 matrix, F-01–F-14 reassessment, G1 14-row reassessment, and `prompt15_imp.md`.

## **Final assessment**

The plan is technically sound and has the right implementation order. Its strongest feature is the distinction between the runtime transition-integrity problem and the evidence-integrity problem.

I would make the amendments above, particularly the X-05 lock-order specification, X-06 signal semantics, X-11 enforcement against bypass, and X-22 migration ownership.

**Recommended status: Ready for final owner authorization after amendments.**

No implementation authorization should be inferred from this review alone. The earlier GO was for the previous Prompt 15 structure; the new Prompt 15.2 plan explicitly supersedes it.

