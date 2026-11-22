The next task is **Prompt 9 — Owner Decision Resolution and G0-Core Final Acceptance**.

This is the transition from technical completion to formal authorization. The coding agent must present the six outstanding decisions, obtain your explicit answers, and only then update the acceptance record. It must not make decisions on your behalf.

# Prompt 9 — Owner Decision Resolution and G0-Core Final Acceptance

## Role

Act as the principal architect and technical acceptance coordinator for:

`github.com/PithomLabs/miniapp`

Your objective is to resolve the outstanding owner decisions required by G0-Core, obtain explicit owner acceptance, and establish whether the repository is ready for Phase 1.

This task follows Prompt 8.

Phase 0 implementation and technical evidence gathering are substantially complete. The remaining work concerns formal architectural acceptance, security accountability, harness selection, and source provenance.

**This is an owner-decision task, not a new implementation task.**

You must never invent, infer, or manufacture owner approval.

## 1. Governing documents

Read the current versions of:

- `docs/plan4.2.md`
- `docs/phase0/g0-evidence.md`
- `docs/phase0/g0-core-adjudication.md`
- `docs/evidence/README.md`
- `docs/adr/016-*.md`
- The relevant identity, invocation, registry, contract and security ADRs.
- The current threat model.
- `docs/prompt2.md` and plan4.2 Appendix D.
- The existing CLI specification and assessment.

The authoritative acceptance criteria are `docs/plan4.2.md` §23.1.

Do not modify the governing architecture baseline.

## 2. Starting position

The latest adjudication reports:

| Condition                                     | Status  |
| --------------------------------------------- | ------- |
| 1 — Core contracts reviewed/versioned         | PARTIAL |
| 2 — Phase 1 dependency pins                   | PASS    |
| 3 — DBOS/Neon shared-handle atomicity         | PASS    |
| 4 — DBOS family/version isolation             | PARTIAL |
| 5 — Tenant transaction context/RLS            | PASS    |
| 6 — Initial target harness                    | PARTIAL |
| 7 — Named security owner                      | BLOCKED |
| 8 — Prompt 2 truncation                       | BLOCKED |
| 9 — ADR-001 and decision responsibilities     | PASS    |
| 10 — A15/A16 design                           | PASS    |
| 11 — Dated spike evidence and source revision | PARTIAL |

Verify these statuses against the current adjudication report before proceeding.

The repository was reported to have zero commits. Reconfirm this.

## 3. Stage One — Prepare the six owner decisions

Prepare a clear decision packet containing the following six decisions.

Do not change repository files yet.

### Decision 1 — Condition 1: Formal core contract acceptance

The core architecture requires review and acceptance of:

- Class A and Class D contracts.
- Agent principal versus authority principal.
- Synchronous and asynchronous invocation semantics.
- Effect records and effect-state transitions.
- Entity-key and `no_entity` semantics.
- Static registry model.
- Non-circular manifest-to-build release binding.

The adjudication identified one specific issue:

Synchronous invocation results currently rely on per-tool `OutputSchema`; there is no shared envelope struct.

Present the actual implementation and its consequences.

Ask the owner to choose:

**A. Accept the current design.**

Per-tool output schemas are the authoritative output contracts, and the canonical invocation result contract does not require one shared Go output-envelope struct. Record this as an explicitly reviewed design choice.

**B. Require a shared output-envelope contract.**

Record the missing requirement and request separate implementation authorization. Do not implement it during this task.

**C. Request further review.**

Keep condition 1 PARTIAL until a specific review gap is resolved.

Also provide a checklist showing the status of every other required contract element. Do not let the output-envelope discussion obscure unrelated contract acceptance.

Required evidence for acceptance:

- Explicit owner decision.
- Reviewer identity.
- Date.
- Exact reviewed contract/ADR revision.
- Outstanding reservations, if any.

### Decision 2 — Condition 4: DBOS family/version and deployment topology

Present the four-part adjudication.

**Already proven**

- Stable DBOS `AppName` separation where supported by the recorded tests.
- Existing family-isolation spike results.

**Already documented**

- Per-family workflow-version strategy.
- Old-version retention and drain requirements.
- Same-image/process-group deployment approach.
- Separate Fly applications as the fallback if safe per-family release isolation cannot be achieved.

**Required now for G0**

- Explicit workflow-version policy.
- Old-version retention and drain strategy.
- Safe deployment topology.
- Owner acceptance of the selected architectural strategy.

**Deferred to G6**

- Live Fly deployment-overlap tests.
- Graceful shutdown and drain drills.
- Actual production recovery/version interaction.

Present the alternatives:

A. Accept the same-image/process-group topology, subject to G6 behavioral verification.

B. Require independently deployable Fly applications per family because the shared topology cannot be adequately justified.

C. Request a specific architectural revision.

Do not claim G0 version isolation is complete merely because the AppName test passed.

Do not require the full G6 operational drill at G0.

Required evidence:

- Explicit topology decision.
- Explicit workflow-version compatibility policy.
- Retention/drain policy.
- Named decision-maker.
- Dated acceptance record.

### Decision 3 — Q9: Name the security owner

The core threat model requires an accountable security owner.

Ask the owner to provide:

- Full name.
- Role.
- Accountability for reviewing capability, privilege, Sprite-profile and approval-boundary changes.
- Whether the person accepts responsibility.
- Whether the person approves the current core threat model.

Do not infer the owner from repository contributions or previous discussions.

Required evidence:

- Named individual.
- Explicit acceptance of responsibility.
- Exact threat-model revision.
- Dated signature or approval record.
- Recorded reservations.

Condition 7 cannot pass until this evidence exists.

### Decision 4 — Q8: Resolve Prompt 2 truncation

`docs/prompt2.md` ends at §7.1.

The missing remainder has not been supplied.

Appendix D of plan4.2 contains architect-authored completions D1–D4. These are not original Prompt 2 text.

Ask the owner to choose:

**A. Accept Appendix D.**

Explicitly accept D1–D4 as the governing completion replacing the missing instructions.

**B. Supply the original remainder.**

Provide the missing Prompt 2 material and subject it to a compatibility review against plan4.2.

**C. Reject both pending further review.**

Keep condition 8 BLOCKED.

Required evidence:

- Explicit acceptance or supplied remainder.
- Exact document/revision reference.
- Date.
- Any exclusions or reservations.

Silence or prior informal discussion is not acceptance.

### Decision 5 — Initial target harness

Condition 6 requires a working initial MCP/REST invocation round trip with the first target harness.

The two-family compatibility matrix is a G6 requirement and must not be pulled into G0.

Present the available harness candidates grounded in the existing project records. Do not invent capabilities or claim that a candidate was already tested.

Ask the owner to select one initial harness.

Then define the smallest required evidence:

- Harness name and version.
- Transport and configuration.
- Authentication identity.
- Invocation request.
- Framework execution and authorization path.
- Actual result or run handle.
- Dated execution report.

If the chosen harness has not been tested, retain condition 6 as PARTIAL until that test runs.

Do not substitute the in-process MCP SDK round trip for an external harness round trip unless plan4.2's requirement is demonstrably satisfied by that exact environment.

### Decision 6 — First Git commit authorization

The repository has been reported to have zero commits. Condition 11 requires core spike reports to include a source revision.

Prepare a proposed first-commit manifest.

Before requesting authorization:

1. Inventory the complete source tree.
2. Identify the exact source and test files to include.
3. Identify documentation and evidence files to include or exclude.
4. Verify no secrets, local credentials, temporary files or unintended artifacts are tracked.
5. Run formatting, vet, tests, race tests and repository checks.
6. Prepare an exact proposed commit message.
7. Record the expected source tree and evidence relationships.

Ask the owner to approve or reject the proposed scope.

**Do not stage or commit in this task.**

A separate explicit owner authorization is required for the actual commit operation.

After authorization, a separate task may perform the commit and bind spike reports to that source revision.

Do not claim that committing automatically makes historical test executions reproducible. Evidence must still identify the source state actually tested.

## 4. Stage Two — Classify all other owner questions

Read plan4.2 Appendix B and classify Q1–Q7 individually.

Use this table:

| Question                                  | Actual requirement                                                         | Governing phase                                              | G0 blocker?              |
| ----------------------------------------- | -------------------------------------------------------------------------- | ------------------------------------------------------------ | ------------------------ |
| Q1 — First SaaS                           | First live provider selection                                              | Phase 2                                                      | No                       |
| Q2 — Two harness families                 | Multi-family compatibility selection                                       | Phase 6                                                      | No                       |
| Q3 — DBOS Enterprise budget               | Conditional decision if open-source wake behavior proves insufficient      | As required by later wake evidence                           | No                       |
| Q4 — Infrastructure organizations/regions | Neon G0 resources resolved; Fly and Sprite resources belong to later gates | G0/G4/G6 as applicable                                       | Neon component resolved  |
| Q5 — `plan3_review.md`                    | Whether the review is owed or document may be removed                      | Housekeeping, subject to any additional governing obligation | No by default            |
| Q6 — `agent-toolchain`                    | Whether an early use case requires it                                      | Catalog/implementation planning                              | No                       |
| Q7 — CLI assessment and keep-list         | Approval of CLI freezing basis                                             | Phase 1 preparation                                          | Not a G0 §23.1 condition |

Verify every classification against plan4.2 before finalizing.

Do not elevate later-phase decisions into G0 blockers.

## 5. Stage Three — Wait for actual owner decisions

Present all six decisions clearly and collect the owner's responses.

Do not use assumed defaults.

If a decision is not supplied:

- Leave it pending.
- State its exact G0 impact.
- Do not update the tracker to PASS.

For a decision requiring implementation, such as selecting option B for condition 1, record the required work and request separate implementation authorization.

Do not silently turn a review task into an implementation task.

## 6. Stage Four — Record accepted decisions

Only after the owner explicitly responds:

1. Record each decision in the appropriate owner-decision register or adjudication document.
2. Record the exact decision wording, decision-maker, date and affected plan requirement.
3. Preserve reservations and conditions.
4. Update the applicable G0 condition status only when every part of its acceptance requirement is met.
5. Keep historical evidence unchanged.
6. Do not modify `docs/plan4.2.md`.

Use append-only decision records where practical.

## 7. Stage Five — First-commit preparation

If the owner authorizes the proposed commit scope, prepare a separate commit task.

Do not perform the commit as an automatic continuation of this task.

The subsequent commit task must:

- Recheck the working tree.
- Verify the exact approved file list.
- Run the agreed pre-commit checks.
- Confirm no unintended artifacts or credentials.
- Create one source revision.
- Record the resulting revision identifier.
- Bind every applicable spike report to the revision actually tested.
- Preserve evidence that was gathered before the commit as historical evidence, not pretend it ran against the new revision.

If the owner does not authorize a commit, condition 11 remains PARTIAL.

## 8. Final report

Use these sections:

### A. Owner decision packet

Present all six decisions, their alternatives and exact completion evidence.

### B. Decisions received

For each decision, identify:

- Decision supplied.
- Who supplied it.
- Date.
- Exact accepted wording.
- Remaining conditions.

Do not list a decision as received if the owner has not provided it.

### C. Appendix B classification

Report Q1–Q7 and confirm their actual phase dependencies.

### D. G0-Core condition matrix

Report all eleven conditions with their latest evidence state.

### E. Remaining gaps

Identify any technical, design-review, signature, harness, source-provenance or implementation requirements that remain open.

### F. Commit readiness

Report the proposed commit inventory and whether authorization was granted. Do not claim a commit exists unless one was actually created in a separately authorized task.

### G. Final readiness verdict

Exactly one of:

- **G0-Core PASS**
- **G0-Core NOT PASSED**

A PASS requires all eleven conditions to be satisfied by current evidence and recorded owner decisions.

## 9. Safety and integrity

- No source implementation.
- No new tests.
- No dependency changes.
- No Fly, Neon or Sprite mutations.
- No credential or secret operations.
- No staging, commit, tag or push in this task.
- No architecture redesign.
- No owner decision inferred from silence.
- No acceptance criterion weakened.
- No baseline modifications.

**Final instruction:** Obtain explicit decisions, document them faithfully, and establish exactly what remains before G0-Core can pass. The purpose is to authorize the next engineering phase responsibly, not to force a green status.

## **What happens after this**

There are two possible paths:

- **All six decisions accepted:** prepare the separately authorized first-commit task, then perform final G0 evidence adjudication.
- **Some decisions remain pending:** record the decisions received and keep G0-Core NOT PASSED. Do not begin Phase 1.

The first real Phase 1 coding task should only be issued after the formal G0-Core verdict becomes PASS.