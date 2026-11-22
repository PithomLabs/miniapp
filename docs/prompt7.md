The Neon sprint materially advances G0-Core. I agree with the **G0-Core NOT PASSED** verdict, but the remaining work is now much narrower.

APPROVED — TWO TEST-ONLY ADDITIONS

I approve the next two technical actions:

1. **Condition 3:** Add the missing DBOS shared-handle transaction-completion test.
2. **Condition 5:** Add the missing concurrent-tenant isolation test.

This authorization covers regression tests and their necessary disposable fixtures only. It does not authorize implementation changes to the framework. If either test exposes a defect, stop and report it for separate authorization.

## **Remaining owner decisions**

These cannot responsibly be invented or decided by the coding agent.

| Item       | Decision required                                                    |
| ---------- | -------------------------------------------------------------------- |
| Q7         | Owner sign-off on the CLI specification                              |
| Q8         | Accept Prompt 2 Appendix D or supply the missing remainder           |
| Q9         | Name the security owner                                              |
| G0-Core-1  | Provide contract-review sign-off                                     |
| G0-Core-4  | Adjudicate AppName isolation versus Fly retention deferral to G6     |
| G0-Core-6  | Name the first supported agent harness                               |
| G0-Core-11 | Authorize the first Git commit to establish source revision identity |

**Q4-Neon is resolved.** Keep the `us-east-2` versus Fly `sin` placement consideration for G6; it should not become a G0-Core blocker unless plan4.2 explicitly requires it.

## **Next implementation prompt**

# Prompt 7 — G0-Core Remaining Test Evidence

## Objective

Complete the two narrowly scoped test additions outstanding from the Neon G0-Core closure sprint:

- Condition 3: DBOS shared-handle transaction-completion behavior.
- Condition 5: concurrent-tenant isolation.

The owner has explicitly authorized these test additions only.

The purpose is to establish whether the existing implementation satisfies the relevant plan4.2 requirements. This is not authorization to repair, redesign, refactor, or extend framework functionality.

## Governing documents

Read:

- `docs/plan4.2.md`
- `docs/phase0/g0-evidence.md`
- Relevant DBOS and RLS spike runbooks.
- Existing DBOS, transaction-completion, and RLS spike implementations.
- The Neon execution evidence recorded in the tracker and evidence register.

Use the exact acceptance requirements in plan4.2. Do not infer missing semantics from test names.

## Scope A — Condition 3: DBOS shared-handle variant

The Neon sprint established that:

- DBOS basic transaction behavior passed against the pooled connection path.
- A separate-pool transaction-completion path executed.
- `dbos.transaction_completion` exists.
- The shared-handle variant was not covered.

Inspect the existing implementation and test suite to determine exactly what the untested shared-handle path is supposed to establish.

Add the smallest meaningful regression test covering that variant.

Requirements:

1. Exercise the actual shared-handle path, not a substitute connection or separate-pool implementation.
2. Demonstrate the transaction-completion behavior required by the governing specification.
3. Verify the resulting durable state, not merely a successful function return.
4. Include the relevant negative or rollback case if required by the specification.
5. Prevent accidental fallback to another database configuration.
6. Keep diagnostics free of credentials.
7. Ensure cleanup is restricted to disposable test objects.

Do not claim coverage of DBOS notification behavior unless the test actually exercises it.

Do not equate the existence of `dbos.transaction_completion` with proof that every shared-handle completion path works.

## Scope B — Condition 5: concurrent-tenant isolation

The Neon sprint established that the existing RLS path works for default-deny and tenant isolation, but the concurrent-tenant leg remains unproven.

Inspect the existing RLS fixture and the exact condition-5 acceptance criteria.

Add a minimal concurrent test that:

1. Uses distinct tenant identities.
2. Executes tenant-scoped operations concurrently.
3. Exercises the actual transaction-scoped tenant context mechanism.
4. Verifies that each tenant sees only its authorized data.
5. Detects context leakage between concurrent transactions.
6. Includes the failure or default-deny behavior required by the specification.
7. Uses an application role that does not bypass RLS.
8. Cleans up safely without broad database deletion.

The test must demonstrate isolation under concurrency, not simply run two sequential tenant queries.

Avoid timing-dependent assertions. Use explicit synchronization barriers or equivalent deterministic coordination.

## Execution requirements

First establish whether the existing Neon test branch remains available and whether the recorded connection paths are still valid.

Use the established safe test configuration.

Keep local PostgreSQL and Neon results separate.

Do not create, delete, or recreate Neon projects or branches. Do not change credentials or secrets.

Run the new tests locally when possible. Then run them against Neon only if the established test environment and non-destructive preflight succeed.

For each execution, record:

- Exact test name.
- Exact command.
- Environment class: local Docker or Neon.
- Test branch identity when applicable.
- Start and end timestamps.
- Exit status.
- Observed durable-state result.
- Skips or failures.
- Redacted evidence.

A compiled test or skipped test is not a behavioral pass.

## Strict implementation boundary

Permitted:

- New or modified test code necessary for these two authorized tests.
- Disposable test fixtures required by those tests.
- Minimal test-only helpers where existing helpers cannot express the necessary behavior.

Not permitted:

- Changes to production framework behavior.
- Changes to DBOS or River configuration.
- Changes to connection-pool construction.
- RLS policy redesign.
- Dependency changes.
- Changes to plan4.2 or ADRs.
- Changes to Gate A implementation.
- Changes to Fly, Sprite, or deployment infrastructure.
- Git staging, commits, tags, or pushes.

If either test reveals a framework defect, stop that workstream. Report the exact reproduction, expected invariant, actual result, and smallest proposed correction. Do not implement the correction.

## Verification

Run the two targeted tests and relevant existing package suites.

Use race testing where appropriate.

Confirm that the new tests would detect the failure they are intended to prevent. Explain the negative control or other evidence supporting that assertion.

Do not claim that the entire G0-Core spike suite has passed merely because these two tests pass.

## Final report

Provide:

### 1. Condition 3

Test added, invariant exercised, exact execution evidence, result.

### 2. Condition 5

Test added, invariant exercised, exact execution evidence, result.

### 3. Verification matrix

PASS, FAIL, BLOCKED, SKIPPED or NOT RUN for every relevant command and environment.

### 4. Change inventory

Exact changed paths. Confirm that changes are limited to the authorized test scope.

### 5. Findings

Any defects exposed, with no unauthorized fixes.

### 6. G0-Core impact

Explain whether the two conditions can now be promoted to PASS. Do not change other condition statuses.

### 7. Safety confirmation

Confirm no implementation redesign, dependencies, secrets, remote infrastructure, commits, staging, or unrelated documentation were changed.

**Final instruction:** Prove the missing behavior. Do not manufacture a passing result. Do not fix an implementation defect under this test-only authorization.

## **What happens after Prompt 7**

Once those two tests have genuine evidence, the technical portion of G0-Core should be substantially closer to closure.

The remaining decisions are yours: Q7, Q8, Q9, contract-review sign-off, harness selection, condition-4 adjudication, and first-commit authorization.

I recommend **not authorizing the first commit yet**. First complete Prompt 7, review its exact changed-file inventory, and settle the outstanding owner decisions. Then the first commit can establish a coherent, reviewed source revision rather than freezing an incomplete evidence state.