I recommend making the next task a **G0-Core Closure Sprint**. Its purpose is to turn the remaining partial and blocked conditions into verified evidence, not to add more framework functionality.

The prompt below preserves the Phase 0 implementation, requires the agent to distinguish owner-dependent blockers from executable tests, and prevents it from declaring G0-Core passed prematurely.

# Prompt 6 — Miniapp G0-Core Closure and Evidence Verification

## Role

Act as the principal architect and verification engineer responsible for closing Phase 0 of:

`github.com/PithomLabs/miniapp`

Your objective is to establish whether **G0-Core can genuinely pass**, using the existing implementation, owner decisions, and fresh execution evidence.

This is an evidence-closure task, not a new implementation phase.

Phase 0 implementation is complete to the extent supported by existing evidence. Gate A has been accepted and closed. Do not reopen completed work without concrete evidence of a regression, violated invariant, or material security issue.

The outstanding objective is to resolve the G0-Core conditions that remain PARTIAL or BLOCKED.

## 1. Governing documents

Before doing anything, inspect:

- `docs/plan4.2.md` — authoritative architecture baseline.
- `docs/phase0/g0-evidence.md` — authoritative G0-Core evidence tracker.
- `docs/adv_review_prompt5.md` — adversarial review and findings.
- `docs/prompt3.md` or the exact governing Prompt 3 document found in the repository — original task requirements and seven deliverables.
- The Neon-spike runbook and Sprite runbook referenced by the evidence tracker.
- `docs/evidence/` — existing evidence register.
- Relevant ADRs and spike implementations.

Do not assume paths or names for documents that have not been located. Discover the actual repository paths first.

Preserve Prompt 3's seven-deliverable structure for the final report.

**Precedence:** `plan4.2.md` governs architecture and acceptance criteria. The evidence tracker governs the current status and required proof. Earlier plans are historical unless explicitly referenced by the authoritative baseline.

## 2. Current known state

The previous Phase 0 execution report established:

### Implementation

- Repository audit completed.
- Contract enforcement and JSON Schemas updated to match plan4.2.
- ADRs 001, 003, 008 and 016 rewritten; ADRs 005, 007 and 009 amended.
- Miniapp ADR template and inventory created.
- DBOS family/version plan and determinism rules documented.
- SSRF test design and data classification documented.
- Evidence register and runbooks created.
- DBOS family-isolation spike added.
- `scripts/checks.sh` integrated into CI.
- No Phase 1 implementation.
- No Sprite implementation.
- `plan4.2.md` remains authoritative and must not be modified.

### Previously verified evidence

- `gofmt`, `go vet`, full Go test suite, race tests and repository checks passed.
- MCP in-process round-trip passed.
- DBOS atomic commit and rollback passed once against live Neon at approximately 16:19 on November 22, 2026.
- The Neon run produced actual observed logs; retain the distinction between that historical pass and a fresh verification.

### G0-Core status from the previous report

| Condition | Previous status |
| --------- | --------------- |
| 1         | PARTIAL         |
| 2         | PASS            |
| 3         | BLOCKED         |
| 4         | PARTIAL         |
| 5         | BLOCKED         |
| 6         | PARTIAL         |
| 7         | BLOCKED         |
| 8         | BLOCKED         |
| 9         | PASS            |
| 10        | PASS            |
| 11        | PARTIAL         |

Treat these as historical starting statuses. Read the current tracker and establish whether anything has changed before relying on them.

### Outstanding owner actions

The previous report identified:

- **Q4 — Neon:** restore account access. At approximately 16:20 on November 22, 2026, the Neon API showed zero projects and direct authentication returned `28P01` under an unchanged, unexpired OAuth identity. No project mutation was issued from the previous environment. The root cause was undetermined.
- **Q7 — CLI:** obtain the required owner sign-off.
- **Q8 — Prompt 2:** owner must accept Appendix D or supply the missing remainder.
- **Q9 — Security:** name the security owner.
- **First harness:** identify the initial supported agent harness, needed for G0-Core-6.
- Q1, Q2, Q3, Q5 and Q6 remain associated with their specific tracker gates. Do not pull later-phase requirements into G0-Core without evidence that they are dependencies.

Do not assume these owner decisions have since been completed.

## 3. Scope and authorization

Authorized work:

1. Audit the current G0-Core status against plan4.2.
2. Identify exact owner decisions and dependencies.
3. Verify Neon account and project access without changing remote resources.
4. Execute the existing DBOS, River and RLS spike suite when its prerequisites are satisfied.
5. Verify that each test actually exercised its intended behavior.
6. Update the G0 evidence tracker and evidence register with truthful results.
7. Produce an updated readiness recommendation.

### Not authorized

Do not:

- Implement Phase 1.
- Implement Sprite functionality.
- Redesign framework architecture.
- Rewrite or amend ADRs.
- Modify `docs/plan4.2.md`.
- Modify `docs/adv_review_prompt5.md`.
- Perform broad source-code refactoring.
- Change Go module or dependency versions.
- Create or delete Neon projects or branches without explicit owner approval.
- Reset Neon credentials, change account security, or attempt account recovery on the owner's behalf.
- Modify production infrastructure, Fly applications, or Fly secrets.
- Commit, stage, tag, or push Git changes.
- Delete, recreate, or migrate databases or volumes as a testing shortcut.

The only routine documentation mutations permitted are evidence-status updates to:

- `docs/phase0/g0-evidence.md`
- Existing evidence-register files under `docs/evidence/`
- A narrowly scoped new G0-Core closure report, if the existing reporting structure requires one.

Any source-code change required to fix a discovered defect is outside this task's authorization. Document the defect, reproduction, impact, and proposed minimal correction, then stop that workstream for separate approval.

## 4. Workstream A — Establish the real starting state

Before execution:

1. Capture the current branch and `git status --short`.
2. Record the current repository file inventory.
3. Identify any existing owner or agent modifications and preserve them.
4. Inspect the exact current content of the G0 tracker and evidence register.
5. Map every G0-Core condition to:
   - Its governing plan4.2 requirement.
   - Its present status.
   - The required evidence.
   - The existing implementation or test that supplies that evidence.
   - Outstanding owner decisions.
   - Its next executable action.

Do not confuse implementation existence with behavioral proof.

In particular, distinguish:

- Source code exists.
- Test compiles.
- Test was skipped.
- Test executed and passed.
- Test executed and failed.
- Cloud-specific behavior was independently verified.

These are different evidence states.

Produce a condition-by-condition readiness matrix before running tests.

## 5. Workstream B — Resolve owner-dependent blockers

Extract the exact wording and acceptance requirements for Q4, Q7, Q8 and Q9 from the governing documents.

Also establish the first-harness decision required by G0-Core-6.

Do not invent answers or silently treat a proposed choice as an owner decision.

For each owner action, record:

| Item          | Required owner decision or action                          | Evidence of completion                                                                 | Blocking condition                                |
| ------------- | ---------------------------------------------------------- | -------------------------------------------------------------------------------------- | ------------------------------------------------- |
| Q4            | Restore Neon access                                        | Authenticated project visibility and successful connection to the intended test branch | Neon-backed tests cannot run                      |
| Q7            | CLI sign-off                                               | Explicit recorded approval against the governing specification                         | Relevant G0-Core condition remains unresolved     |
| Q8            | Accept Prompt 2 Appendix D or supply the missing remainder | Explicit owner decision and referenced artifact                                        | Relevant G0-Core condition remains unresolved     |
| Q9            | Name security owner                                        | Named owner and recorded acceptance of responsibility                                  | Security ownership requirement remains unresolved |
| First harness | Name the first supported harness                           | Explicit owner decision recorded against G0-Core-6                                     | G0-Core-6 cannot be closed                        |

For Q1, Q2, Q3, Q5 and Q6, inspect their exact tracker definitions. Only include those that genuinely block G0-Core.

### Critical Neon restriction

The previous Neon incident remains unexplained.

Do not interpret `28P01` as proof of a particular account failure. Do not infer that projects were deleted. Do not assume the incident is resolved simply because authentication succeeds in a later session.

The owner must restore or confirm account access through the Neon console or the established account recovery process.

The agent may perform non-destructive verification of authenticated access after the owner indicates restoration.

If authentication still fails or the intended project is not visible:

- Stop Neon execution.
- Preserve the error and timestamp without exposing credentials.
- Mark the relevant evidence BLOCKED.
- Do not create a replacement project or branch.
- Do not switch to another account or silently use another database.

## 6. Workstream C — Re-run the DBOS, River and RLS spikes

After Neon access is restored, inspect the existing Neon runbook and the spike tests.

Identify the exact intended test branch, database, and connection paths. Verify that they belong to the Phase 0 test environment and not production.

The previous report identifies the `phase0/g0-spikes` branch. Confirm its exact meaning and current existence from the tracker and runbook before connecting.

Do not create or delete a branch unless explicitly authorized.

### C1. Safety preflight

Before executing anything against Neon:

- Confirm authenticated project visibility.
- Confirm the intended test branch.
- Confirm the app and DBOS connection targets resolve to the intended environment.
- Confirm that test setup and cleanup are scoped to disposable spike objects.
- Inspect SQL and test cleanup behavior for destructive operations.
- Ensure no production or unrelated database can be selected by fallback configuration.
- Confirm that diagnostic output does not print connection strings, passwords or access tokens.

Abort the cloud test run if any of these conditions cannot be established.

### C2. Execute the relevant tests

Run the existing spike suite that covers:

**DBOS atomicity**

- Transaction commit behavior.
- Transaction rollback behavior.
- Atomicity of the relevant application and workflow state changes.

**DBOS crash recovery**

- Worker termination at the documented crash point.
- Recovery after restart.
- Correct durable-state behavior.
- No false assertion that the process survived when it was actually killed.

**DBOS family isolation**

- Independent AppName families.
- Independent workflow completion.
- Correct isolation of the tested workflow state.

**River**

- Transactional enqueue behavior.
- Atomicity between application writes and job enqueue.
- Unique-job behavior where covered by the existing spike.

**RLS**

- Tenant context enforced by the explicit transaction.
- Default-deny behavior.
- No cross-tenant visibility.
- Correct use of an application role that does not bypass RLS.

Do not claim that version isolation is fully proven when only AppName isolation was exercised. The family/version plan explicitly retains additional version-strategy work for a later phase.

### C3. Evidence discipline

For every test, report:

- Exact test name.
- Exact execution command.
- Database environment class.
- Test branch identifier, where applicable.
- Start and completion timestamps.
- Exit status.
- Observed result.
- Any skipped subtests.
- Relevant redacted log evidence.
- Any cleanup performed.

A test that compiled but did not execute is not a pass.

A test that ran against local PostgreSQL is not Neon evidence.

A previous successful run is historical evidence, not a substitute for the requested fresh execution.

If a test exposes a real defect, stop and produce a minimal defect report rather than silently changing implementation.

### C4. Local execution

Run the local integration path when useful and available.

Keep local Docker results separate from Neon results.

Local execution may validate general application behavior, but it does not automatically satisfy a Neon-specific G0-Core requirement.

Do not claim the complete spike suite passed if any required test was skipped.

## 7. Workstream D — Review remaining G0-Core conditions

After owner actions and available test executions:

Revisit every G0-Core condition, not just conditions 3 and 5.

In particular, independently assess the previously PARTIAL/BLOCKED conditions 1, 3, 4, 5, 6, 7, 8 and 11.

For each, identify:

1. The exact acceptance requirement.
2. The owner decision, artifact, implementation or execution evidence that satisfies it.
3. Whether that evidence is current and independently verifiable.
4. Whether the condition can now be marked PASS.
5. If not, the precise remaining gap.

Do not promote a condition to PASS because a related test succeeded.

Do not lower or reinterpret an acceptance requirement to obtain a green tracker.

Do not merge later phase gates with G0-Core.

## 8. Workstream E — Update evidence and tracker

Update `docs/phase0/g0-evidence.md` only after the underlying evidence exists.

Preserve historical incident records, particularly the November 22 Neon incident and the earlier 16:19 DBOS atomicity observation.

Never erase or rewrite historical failure evidence to make the latest status appear cleaner.

The tracker should clearly distinguish:

- Historical PASS.
- Fresh PASS.
- PARTIAL.
- BLOCKED.
- NOT RUN.
- Owner action required.

Add evidence-register entries with stable references to the actual execution records.

Do not put secrets, passwords, access tokens or full connection strings into evidence files.

Do not fabricate logs, timestamps, project names, branch names or approval statements.

If the current environment prevents a required test, keep that requirement BLOCKED and state exactly what the owner or operator must do next.

## 9. Workstream F — Final G0-Core verdict

There are only two acceptable readiness outcomes.

### Outcome A — G0-Core PASS

Use this only when every mandatory G0-Core condition is independently satisfied by the evidence required in plan4.2.

The final report must explicitly identify the proof for each condition.

Do not require later phase gates such as Sprite implementation or production deployment merely to claim G0-Core completion unless plan4.2 explicitly makes them G0-Core requirements.

### Outcome B — G0-Core NOT PASSED

If any mandatory condition remains PARTIAL or BLOCKED, preserve that status.

Identify the exact remaining owner action, evidence requirement, or technical blocker.

A partial pass is not a G0-Core pass.

Do not change the acceptance criteria.

## 10. Final report — Prompt 3's seven deliverables

Preserve the seven-deliverable structure required by the original Prompt 3.

### 1. Repository audit

Record the current repository state, baseline integrity, relevant implementation and test inventory, and any drift found.

### 2. Change summary

List only evidence-document changes and any separately authorized work. Explicitly state if no source implementation changed.

### 3. Verification

Provide a command-by-command table with PASS, FAIL, BLOCKED, SKIPPED or NOT RUN.

Separate local Docker, Neon, and ordinary Go test evidence.

### 4. G0 tracker

Provide the final condition-by-condition status matrix, with direct evidence references.

### 5. Owner actions

List completed decisions and unresolved owner actions, with named responsibility and the exact evidence needed.

### 6. Runbooks

Confirm whether the Neon and Sprite runbooks remain executable and accurate. Do not execute Sprite work unless it is explicitly required by G0-Core.

### 7. Readiness

Give one unambiguous verdict:

- G0-Core PASS, or
- G0-Core NOT PASSED.

Explain the decisive evidence and list the exact next action.

## 11. Final safety and integrity requirements

- Preserve `docs/plan4.2.md`.
- Preserve `docs/adv_review_prompt5.md`.
- Preserve Gate A's accepted implementation.
- Do not start Phase 1.
- Do not implement Sprite.
- Do not modify dependencies.
- Do not mutate Neon or Fly resources outside the explicitly authorized, non-destructive test operations.
- Do not change secrets or account configuration.
- Do not commit, stage, tag or push.
- Preserve all owner changes.
- Never claim tests passed when they skipped.
- Never infer a successful cloud test from local execution.
- Never report G0-Core as passed while a mandatory condition remains unresolved.

**Final instruction:** The objective is not to make the tracker green. The objective is to establish, with independently verifiable evidence, whether the existing Phase 0 implementation satisfies every G0-Core requirement. If Neon access or owner decisions remain unavailable, report the blocker precisely and stop rather than manufacturing a substitute result.

### **My recommendation**

Keep this task separate from Phase 1. The critical path is:

1. Owner resolves Q4, Q7, Q8, Q9 and identifies the first harness.
2. Neon access is verified without changing resources.
3. DBOS, River and RLS spikes are freshly executed against the intended test branch.
4. The G0 tracker is adjudicated condition by condition.
5. Only then decide whether G0-Core has passed and authorize Phase 1.

One particularly important safeguard: **the earlier DBOS atomicity pass at 16:19 remains valid historical evidence, but it does not prove that the outstanding crash-recovery, family-isolation, River and RLS requirements have passed.**