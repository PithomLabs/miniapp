# **Consolidated independent review — Prompt 15.2 implementation plan**

Structurally sound

P2 entry conditional

**Consolidated verdict: ACCEPTABLE TO PROCEED WITH P0 AND P1, SUBJECT TO THE CONDITIONS BELOW. P2 MUST NOT BEGIN UNTIL THE P1 EXIT GATE IS SATISFIED.**

The two attached reviews substantially agree. The DeepSeek review finds no structural blocker and recommends seven refinements. The second review identifies one more serious completeness issue: the plan accounts for X-01–X-24 but omits several residual requirements from F-01–F-14 that have no X-finding identifier.

These conclusions are compatible. The plan does not need to be redesigned, but its implementation map and P1 exit criteria need strengthening.

The consolidated review below retains the valid findings from both reviewers and removes duplication. It also distinguishes changes required before P2 from documentation improvements that can be made during P1.&#x20;

prompt15.2_plan_review\.md



## **1. Overall assessment**

The plan gets the most important things right:

- It requires root-cause diagnosis before implementation.
- It uses the pinned DBOS v1.9.0 source rather than relying on assumptions about SDK behavior.
- It rejects a universal `transition()` primitive in favor of subsystem-specific transition operations with common guarantees.
- It fixes mock measurement integrity before relying on later effect evidence.
- It groups related runtime changes and sequences work according to dependencies.
- It requires red/green testing against isolated pre-remediation snapshots.
- It preserves the `lead-followup` hard stop, forward-only migrations, no-commit/no-push restriction, and honest treatment of unobserved CI and Neon results.

The six normative transition-integrity rules and the RC-1/RC-2 diagnosis are the correct foundation, subject to independent validation in P1.

The principal weakness is not the architecture or overall sequence. It is **traceability completeness**: every review requirement, including requirements without their own X-ID, must have an owner, an implementation task, and evidence.

## **2. Consolidated findings and required amendments**

I have grouped the reviews' concerns into seven implementation gates and three supporting refinements.

### **Gate 1 — Restore all F-01–F-14 residual requirements (HIGH)**

The most significant omission is that Prompt 15.2 maps X-01–X-24, but some important F-01–F-14 residual requirements appear only in the final reassessment rather than the implementation plan.

These are not optional documentation details.

Add the following explicit work items:

| Origin       | Missing requirement                               | Required disposition                                                                                                                                    |
| ------------ | ------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| F-03         | Stale worker resumes after lease takeover         | Add a genuine interleaving test proving the stale worker cannot send or overwrite the new owner's state                                                 |
| F-03         | `Settle` is unaudited/unexposed                   | Determine the required audit and operator-visible settlement semantics; test them                                                                       |
| F-02         | Lazy approval expiry lacks a real workflow E2E    | Add a real workflow test proving expiry is observed and settled                                                                                         |
| F-02         | `settleDenied` overstates "confirmed down"        | Correct the semantics and reporting to reflect actual cooperative cancellation                                                                          |
| F-10         | Signal after timeout                              | Add a timing regression test                                                                                                                            |
| F-10         | Signal followed by process kill                   | Add a kill/recovery regression test                                                                                                                     |
| F-10         | Signal when workflow process group is unavailable | Add a routing/recovery regression test                                                                                                                  |
| Prompt 14 §F | Cooperative cancellation reality is undocumented  | Explain that in-flight steps may complete according to SDK behavior; cancellation prevents new reservations, not an external effect already in progress |

Assign these explicitly:

- Effect lease and settlement items to WS-C / WS-B as appropriate.
- Approval expiry to WS-D.
- Signal timing cases to WS-D or WS-E, with one named owner.
- Cooperative-cancellation documentation to the relevant operational documentation.

**Acceptance condition:** every F-01–F-14 residual named in Prompt 14 has an explicit task, test or documentation disposition. A final matrix update alone is not sufficient.

### **Gate 2 — Make state-machine ownership exhaustive**

The six proposed state machines are sensible:

1. Invocation.
2. Approval.
3. Effect and effect attempt.
4. Workflow/run identity.
5. Sequence and attachment.
6. Cancellation.

But three findings do not visibly have an owning state machine:

- X-06: signal and notification lifecycle.
- X-10: foreign-version enumeration and operational visibility.
- X-15: kill-switch control-plane state.

Add a machine-to-finding ownership table to `invocation-lifecycle-contract.md`.

It must account for every X-01–X-24 finding, not just the five HIGH findings. Some findings belong to operational disciplines rather than application state machines—for example, CI, analyzer enforcement, and evidence provenance. Give those explicit owners too.

**Acceptance condition:** no finding is ownerless or represented only as a reporting obligation.

### **Gate 3 — Make P1 independently reviewable**

The contract-before-code requirement is correct, but there is a risk in having the same agent author the contract, implement it, and declare it correct.

Require:

- Contract authored and completed during P1.
- Review by someone other than the P2 implementer.
- Explicit review comments or an adjudication record.
- Confirmation that the six rules cover every applicable finding and F-residual.

If the same agent must perform both roles, disclose that as a self-review limitation. Do not call it independent verification. Obtain explicit owner acceptance before treating that review as an independent P1 exit.

The six rules should be enumerated or explicitly referenced in the plan so that their content is reviewable before implementation:

1. Predicated single-writer transitions.
2. Guard-and-act consistency.
3. Authorization at the execution boundary.
4. No unreachable operational states.
5. Truthful lifecycle and audit reporting.
6. Falsifiable evidence.

The subsystem-specific implementation approach is correct. Do not reintroduce the universal transition primitive.

### **Gate 4 — Tighten CI and X-17 acceptance**

The baseline has an important constraint: serial testing is green, while parallel testing is red against the shared local database.

The plan correctly identifies:

```
go test -p 1 ./... -count=1
```

as the sanctioned invocation.

This qualification must carry into the CI implementation.

WS-H must either:

- Use the sanctioned serial invocation in CI, including the appropriate database-backed test configuration; or
- Make the affected integration fixtures genuinely parallel-safe and prove that with execution.

Do not configure remote CI around bare `go test ./...` and reproduce the known parallel failure.

Also investigate and document the precise cause of the existing parallel-red result. If shared test database state is the culprit, document how truncation, fixtures, schemas, or package execution interact. Do not simply label it "interference" without tracing it.

There are two different acceptance outcomes:

**X-17 implementation acceptance:** the CI workflow provisions a pinned PostgreSQL service, uses the correct test command, and its configuration and tests are validated locally.

**G1 row 12 acceptance:** remote CI actually executes the database-backed tests and passes.

Because the current no-push boundary prevents observing a new remote run, G1 row 12 must remain PARTIAL until an authorized remote run is observed. This does not prevent implementing the CI fix.

### **Gate 5 — Preserve X-08's unconditional requirements**

Genuine run attachment should remain the default. Prompt 13 §7 requires invocation identity to be preserved when attaching to an existing run, which supports genuine attachment rather than detect-and-exit.

The following three requirements apply regardless of the precise attachment implementation:

1. Superseded active generations must not continue sending effects.
2. The sequence claim race must have a barrier-synchronized concurrency test.
3. The losing invocation must not report `Succeeded` while its target sequence remains active.

Make these explicit in WS-E.

The final report must describe the actual attachment semantics, including status propagation, cancellation, target failure, and cross-tenant boundaries.

Do not silently re-scope attachment to detect-and-exit.

### **Gate 6 — Preserve every sub-requirement of multi-part findings**

The plan's compact workstream descriptions must not cause individual requirements to disappear.

Expand the implementation map with these specific sub-requirements:

| Finding | Required details that must remain explicit                                                                                                                         |
| ------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| X-01    | Adopt a dangling pending approval row where legitimate; prevent bypass after park failure; exactly one appropriate `approval-requested` audit event per invocation |
| X-02    | Grant, denial, and expiry must all converge, including when the workflow was never submitted                                                                       |
| X-04    | Predicates apply to `Decide`, expiry marking, and `CancelPending`                                                                                                  |
| X-07    | Log every attempt independently; correct the false mock comment and surviving exactly-once language                                                                |
| X-11    | Enforce a recovery bound; audit the durable decision at commit; remove the nonexistent "durable mailbox" description                                               |

These are not separate architecture decisions. They are required details of the existing fixes.

### **Gate 7 — Make X-11 mechanism selection explicit**

The SDK supports timed `Recv`, but that does not automatically make approval waiting framework-bounded.

P1 must compare the available choices:

- Framework-mandated bounded receive through a controlled approval-wait primitive.
- A reconcile/sweeper mechanism.
- Poll-driven notification redelivery.

For the chosen mechanism, specify:

- Why it is enforceable.
- How direct use of an unbounded receive is prevented from bypassing it.
- How expiry is observed after a lost signal.
- What recovery bound is actually guaranteed.
- What happens if the decision commits immediately before process termination.

A required miniapp behavior that changes the published implementation contract must be surfaced as an owner decision. Do not introduce that obligation without making it explicit.

**Acceptance condition:** the contract proves bounded recovery without depending solely on a well-behaved miniapp or a human retrying the decision.

## **3. Supporting refinements**

These are valid, but do not require restructuring the implementation phases.

### **C1 — Reproduce the baseline digest**

The reported `5564a37b` digest covering 247 files must be reproducible.

Record the full digest, hashing methodology, sorted file manifest, and exact command. Include tracked modifications and relevant untracked files.

Do not rely on an unexplained abbreviated digest.

### **C2 — Name the pre-remediation snapshot method**

Use an isolated Git worktree at the recorded pre-remediation state, or an equivalent preserved snapshot.

Run red and green tests against separate database names.

Record both source digests. Leave the active tree untouched.

### **C3 — Bind SDK verification to a portable source identity**

The plan reports that the SDK source at `/home/chaschel/Documents/go/dbos-transact-golang-1.9.0` is byte-identical to pinned v1.9.0.

P1 should independently confirm the comparison and record the source digest or a module-source hash in the evidence register.

Avoid making evidence depend on a machine-local absolute path. Cite the pinned SDK version and source digest.

Treat the line-specific SDK claims as reported findings until P1 reproduces the source comparison.

### **C4 — Improve the X-06 signal semantics**

`RetrieveWorkflow` followed by `Send` is reasonable, but it does not make the two operations atomic.

The contract must distinguish:

- Workflow existence was verified.
- Signal submission was accepted by DBOS.
- Workflow actually processed the signal.

Do not report a successful notification request as proof that the workflow received or processed it.

The durable approval decision remains authoritative. The workflow must be capable of recovering by rereading durable state.

### **C5 — Specify X-10 status classification**

Enumerate DBOS v1.9.0 statuses explicitly and distinguish active/nonterminal work from terminal statuses.

In particular, verify the actual semantics of `MAX_RECOVERY_ATTEMPTS_EXCEEDED`. Do not blindly treat it as ordinary pending work or omit it from operational visibility.

### **C6 — Specify X-14 failure-path shutdown behavior**

A timed-out `Shutdown` must not be followed by a silent `pool.Close()` while workers may still use the pool.

Require the implementation to report pending worker components, preserve diagnostics, and use a demonstrably safe process-termination strategy.

Test both fault-injected shutdown failure and actual timeout with an unresponsive worker.

### **C7 — Correct the X-04 workstream rationale**

Grouping X-01–X-04 is appropriate because the findings share invocation/approval lifecycle integrity.

However, X-04 lives in `approvals/approvals.go`, not the same runtime files as X-01 and X-03.

Describe the grouping as shared lifecycle ownership, not shared source-file ownership.

## **4. Standing boundaries to restate explicitly**

The plan should repeat these requirements in its permanent scope section, not rely on scattered references:

- No `lead-followup` implementation, manifest, registration, execution, or ADR approval.
- No commits, pushes, or remote mutation.
- No edits to protected architecture and review documents.
- No real provider writes or integrations. Mock-only channels.
- Do not weaken existing tests. X-17 must be fixed with skip-fast prerequisite checks, not deletion or broad skipping.
- Use disposable databases only. No destructive experiments against shared or production Neon resources.
- Migrations remain forward-only.
- Unresolved owner decisions must be surfaced, not invented.

These boundaries are particularly important because the remediation spans multiple implementation packages and may take several stages.

## **5. P1 exit gate**

P1 must not be considered complete merely because `invocation-lifecycle-contract.md` exists.

The following conditions must be satisfied before P2 begins:

| Condition              | Evidence required                                                                          |
| ---------------------- | ------------------------------------------------------------------------------------------ |
| Root-cause assessment  | RC-1a–d and RC-2 ratified, revised, or rejected with reasoning                             |
| Full coverage          | X-01–X-24 and every F-01–F-14 residual assigned                                            |
| Machine ownership      | Every finding mapped to a state machine or operational discipline owner                    |
| Transition rules       | All six rules explicitly enumerated and enforceable                                        |
| Contract review        | Independent review or explicit owner acceptance of self-review risk                        |
| Submission choke point | All submission call sites audited; bypasses documented and resolved                        |
| X-11 mechanism         | Exactly one enforceable recovery mechanism selected, with alternatives evaluated           |
| X-08 semantics         | Genuine attachment requirements and unconditional cancellation/status requirements defined |
| X-17/18 acceptance     | CI and evidence criteria separately specified                                              |
| Evidence provenance    | Baseline digest method reproduced; SDK source identity bound to evidence                   |

A discovered submission bypass is not automatically an owner-level architecture conflict. Fix it if the correction remains within the governing specification. Escalate only if the correction genuinely requires a new architecture or policy decision.

Where the independent review, X-11 mechanism, or another issue requires an owner decision, stop P2 at that boundary rather than treating an unresolved decision as approval.

## **6. Final phase assessment**

| Phase | Consolidated verdict                                                                                              |
| ----- | ----------------------------------------------------------------------------------------------------------------- |
| P0    | Accept. Reproduce the baseline digest and investigate serial/parallel test differences.                           |
| P1    | Accept with the expanded traceability, contract, and independent-review requirements.                             |
| P2    | Do not begin until P1 exit conditions are satisfied.                                                              |
| P3    | Accept. Real PostgreSQL, synchronized barriers, fault injection, and genuine process-kill recovery are essential. |
| P4    | Accept. Full X-matrix, residual F-matrix, capability matrix, G1 rebuild, and digest-bound evidence required.      |

The P3/P4 acceptance matrix should cite its exact governing location, for example `docs/prompt15.2.md §14`, after checking the actual file. The report should state the scope of that matrix so that a reviewer does not need to infer which document's section numbering is intended.

## **7. Review limitations**

The attached reviews are plan reviews, not implementation verifications. They do not independently establish that the actual Prompt 15.2 text, repository, SDK clone, HEAD, or worktree digest have the properties claimed in the plan.

P1 must inspect the actual governing files and pinned SDK source directly, verify all section references, and resolve any discrepancy.

Do not treat a plan review's acceptance as proof that the implementation has been executed or that the environment is ready.

## **Final recommendation**

Accept P0/P1

**The plan is sound and should not be redesigned.** The required amendments add completeness and independent review discipline without changing its dependency order.

The six most important additions are:

1. Restore the missing F-01–F-14 residual work.
2. Give every finding an explicit state-machine or operational owner.
3. Make the P1 contract independently reviewable.
4. Make serial CI execution and X-17 acceptance explicit.
5. Preserve all unconditional X-08 requirements.
6. Expand multi-part findings so none of their requirements are lost.

The remaining refinements—digest provenance, SDK-source identity, signal semantics, shutdown handling, status classification, and exact acceptance references—should be incorporated in the same plan revision.

**Recommended authorization boundary:** P0 and P1 may proceed under the agreed environment safeguards. P2 remains gated by the complete P1 exit criteria. The separate owner questions about Prompt 15.2 supersession and OpenCode model preflight remain unresolved until explicitly authorized.

The governing engineering principle stays unchanged:

**Diagnose the structural failure. Derive the contract. Repair the shared mechanism. Prove the correction with adversarial evidence.**