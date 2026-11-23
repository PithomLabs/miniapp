# **WS-A adversarial exit review**

WS-A conditionally acceptable — WS-B remains gated

This is substantial progress. The implementation addresses the reported X-01–X-04 defects, and the red/green report is unusually candid about where the evidence is weaker.

I would not reject the implementation. I would, however, withhold WS-A exit acceptance until two issues are resolved.

## **1. Strong work accepted**

The following changes are architecturally consistent with the accepted contract:

| Area                                            | Assessment                                       |
| ----------------------------------------------- | ------------------------------------------------ |
| R3 pending-aware claim and replay               | Sound direction                                  |
| Atomic Park                                     | Correct transaction boundary                     |
| `submitChecked` choke point                     | Correct architectural enforcement point          |
| Submit-if-missing on poll and approval decision | Closes important stranded-run paths              |
| Predicated Decide and expiry                    | Correct single-writer discipline                 |
| Typed `ErrUnknownRun`                           | Better than string-based error classification    |
| Canonical `ScopeID`                             | Good shared identity primitive                   |
| Migration 000006                                | Appropriate coverage for identity and attachment |

The X-01 and X-02 red/green results are especially encouraging. The report records predicted failures on the original snapshot and successful remediated behavior.&#x20;

wsc_exit_a.md



## **2. HIGH: Rule 3 authorization interpretation needs owner acceptance**

This is the most important architectural question.

The original contract explicitly says that a gated tool requires a persisted approval linkage in `decided/approve` status before submission.&#x20;

invocation-lifecycle-contract(1).md



But the same contract also requires the framework to submit a workflow while its approval is still pending, so that the workflow can wait for the decision.&#x20;

invocation-lifecycle-contract(1).md



These requirements conflict.

The WS-A implementation resolves the conflict by accepting a valid live linkage in either `pending` or `granted` state. This is a reasonable interpretation, but it changes the normative authorization rule.&#x20;

wsc_exit_a.md



**My recommendation: accept this interpretation, but narrow its meaning.**

There must be two separate permissions:

- **Workflow-start permission:** A valid pending approval linkage permits submission solely to establish and execute the durable approval-wait workflow.
- **Business-execution permission:** Consequential work remains prohibited until the persisted approval is granted and its intent hash, tenant, authority, and other required conditions are revalidated at the execution boundary.

A pending approval must never be treated as a granted approval.

The revised Rule 3 should explicitly state:

> A gated workflow may be submitted with a valid pending approval linkage only to enter its durable approval-wait path. It may proceed to consequential business execution only after the framework verifies a persisted granted decision with matching authority and intent. The effect-authorization boundary independently revalidates that decision. Missing, denied, expired, cancelled, or mismatched approval linkages fail closed.

Also require tests proving that all entry paths—first submission, replay, polling, and approval decision—cannot bypass the approval wait or execute consequential work while approval remains pending.

**Owner decision required:** Do you approve this narrow amendment to Rule 3? I recommend approval because it reconciles the atomic Park design with the durable approval-wait requirement without weakening the actual execution authorization boundary.

## **3. HIGH: X-03 and X-04 red tests are not sufficiently deterministic**

The report itself acknowledges that the pre-remediation failures are probabilistic in principle, although failures were observed on round one.&#x20;

wsc_exit_a.md



This distinction matters.

A barrier that releases two goroutines together does not prove they both read the same prior database state before either writes.

For these defects, the test must synchronize at the vulnerable boundary:

- **X-03:** Both submissions observe the empty `run_id` before either attempts adoption.
- **X-04:** Both decisions observe `pending` before either commits its decision.

Use a deterministic database/test-harness synchronization mechanism that works against both snapshots. The identical test source must reliably expose the old behavior and pass against the corrected behavior.

Do not count a race that happened to reproduce once as a deterministic regression test.

The green implementation may be correct by construction, but the pre-remediation evidence must reliably demonstrate the defect it claims to fix.

## **4. Findings that do not block WS-A**

**Self-approval and expiry tests passing on both snapshots:** The report appropriately distinguishes these as guard tests rather than discriminating red/green tests. Keep that distinction in the evidence register.

**Unknown-run cancellation:** The typed SDK error matching is a good improvement. Preserve the distinction between a genuinely nonexistent run and unavailable executor-family infrastructure. An infrastructure failure must never be silently absorbed as an already-absent workflow.

**Two contract-driven test updates:** Atomic rollback expectations and lease-takeover retry behavior are legitimate changes, provided the updated tests continue to prove the original invariants.

## **5. Final disposition**

| Workstream condition | Decision                                                  |
| -------------------- | --------------------------------------------------------- |
| X-01                 | Accept based on reported evidence                         |
| X-02                 | Accept based on reported evidence                         |
| X-03                 | Remediation accepted; deterministic red proof outstanding |
| X-04                 | Remediation accepted; deterministic red proof outstanding |
| X-19 / X-22          | Accept based on reported evidence                         |
| Rule 3 divergence    | Owner decision and contract amendment required            |
| Standing boundaries  | No reported violation                                     |

**WS-B must remain gated** until the Rule 3 decision is recorded and X-03/X-04 have satisfactory discriminating evidence.

I would not reopen the completed X-01/X-02 work or redesign the implementation. The next work is limited to resolving the authorization wording and strengthening the two concurrency proofs.

The important principle is unchanged: the framework must make safe workflow execution the easiest supported path, not rely on every miniapp developer to understand and reproduce the framework's concurrency guarantees.