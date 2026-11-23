Now that **G0-Core has passed and its acceptance record is committed to GitHub**, the next task is officially Phase 1.

I recommend naming it **Prompt 10 — Phase 1 Core Vertical Slice and Lifecycle Bootstrap**.

The objective is to move from architectural contracts to a functioning framework, demonstrating two real business outcomes: `crm-read` (Class A) and `lead-followup` (Class D).

# Prompt 10 — Phase 1: Core Vertical Slice and Lifecycle Bootstrap

## Role

Act as the principal Go architect and implementation engineer for:

`github.com/PithomLabs/miniapp`

G0-Core has passed. Its eleven acceptance conditions have been adjudicated, the source baseline and acceptance evidence have been committed to GitHub, and Phase 1 is now authorized.

Your mission is to implement the first working vertical slice of the Miniapp framework.

This is the transition from architecture and enabling proofs to an executable, reusable framework.

The result must demonstrate two meaningful business capabilities:

- `crm-read`: synchronous Class A execution.
- `lead-followup`: durable Class D execution using DBOS.

The framework must expose the same canonical business execution path through its supported interfaces. It must not become a collection of unrelated handlers, adapters, and workflow implementations.

## 1. Governing authority

Read before implementation:

- `docs/plan4.2.md` — authoritative architecture.
- `docs/phase0/g0-evidence.md` — accepted G0 evidence.
- `docs/phase0/g0-core-adjudication.md` — final G0 acceptance.
- `docs/adr/001-dbos-adoption.md` and relevant ADRs.
- `docs/adr/016-*` and `docs/adr/017-*`.
- Contract definitions and generated JSON Schemas.
- DBOS family/version plan and determinism rules.
- Dependency/version register.
- Accepted Neon, DBOS shared-handle, RLS and OpenCode spike evidence.
- `docs/app_category_guidelines.md` only after verifying that the consolidated version is authoritative. Do not rely on contradictory historical drafts.

**`docs/plan4.2.md` is immutable and authoritative.**

Do not redesign the architecture or reopen settled decisions without concrete evidence of an incompatibility, violated invariant, or material security problem.

## 2. Baseline verification

Before editing:

1. Identify the current branch and Git HEAD.
2. Verify the published G0 acceptance commit and its ancestry.
3. Confirm the working tree is clean.
4. Inspect the complete repository and identify implemented versus placeholder packages.
5. Verify Go dependency pins against the accepted version register.
6. Verify the Gate A and G0 regression tests remain intact.
7. Confirm the current DBOS SDK and its supported APIs from the pinned version.
8. Record the exact baseline source revision.

Do not assume empty package directories contain working implementations.

Do not discard, overwrite or reorganize existing work.

## 3. Architectural invariants

Every implementation decision must preserve these principles.

### Framework architecture

- One Go module: `github.com/PithomLabs/miniapp`.
- One shared Go runtime.
- Statically registered miniapp implementations.
- Independently versioned manifests.
- One canonical invocation service.
- Thin REST, MCP, CLI and Web adapters.
- No duplicate business execution or authorization implementations across transports.

Miniapps represent meaningful, typed and permissioned business outcomes.

Do not turn every route, CRUD helper, database query or provider endpoint into a miniapp.

### Identity and authority

Keep agent principal and authority principal distinct.

Authorization must happen before dispatch.

Tenant identity, authenticated human authority, permissions and approval context must originate from trusted server-side context.

Never trust client-provided tenant IDs, role assignments, authority IDs, or approval flags.

### Execution model

Preserve the agreed category model:

- Class A: request-bound synchronous execution.
- Class D: durable business workflow execution using DBOS.

Do not introduce River execution into Phase 1. Its Class B and Class C work belongs to Phase 3.

Do not introduce a home-grown workflow engine.

Execution category and compute placement remain separate dimensions.

### Effects and security

- Preserve the effect state machine.
- Do not retry an UNKNOWN external outcome blindly.
- Use stable business identities and entity serialization for consequential writes.
- Keep approval, idempotency, audit and blast-radius controls server-side.
- Never claim exactly-once external provider effects.
- No real provider write operations in Phase 1.

### DBOS and durability

- One stable DBOS `AppName` per Class D family.
- Explicit workflow-version registration.
- Retain old workflow artifacts until pending executions drain.
- Keep nondeterminism and external I/O behind appropriate durable boundaries.
- Preserve transaction-boundary correctness proven in Phase 0.
- Stop and drain DBOS workers before closing shared database pools they depend on.

The final point is particularly important: Phase 0's shared-pool shutdown observation must become an explicit executor lifecycle requirement.

## 4. Phase 1 implementation scope

Implement the Phase 1 components specified in plan4.2.

| Component                | Responsibility                               |
| ------------------------ | -------------------------------------------- |
| `registry/`              | Static registration and manifest resolution  |
| `runtime/`               | Canonical invocation and execution lifecycle |
| `executor/classa`        | Synchronous request-bound execution          |
| `executor/dbos`          | Durable Class D execution                    |
| `auth/`                  | Principal and authorization enforcement      |
| `effects/`               | Core effect records and state handling       |
| `approvals/`             | Core approval mechanisms                     |
| `audit/`                 | Durable, attributable audit records          |
| REST/MCP/CLI adapters    | Thin invocation interfaces                   |
| `testkit/`               | Reusable framework test infrastructure       |
| `miniapps/crm-read`      | Class A reference miniapp                    |
| `miniapps/lead-followup` | Class D reference miniapp                    |

Implement only meaningful packages and interfaces. Avoid empty abstractions and placeholder APIs introduced for hypothetical future functionality.

### 4.1 Canonical invocation service

Build the single execution entry point used by every transport.

It must own:

- Invocation validation.
- Principal and authority resolution.
- Tenant and permission enforcement.
- Manifest and implementation resolution.
- Authorization before execution.
- Execution-class dispatch.
- Idempotency handling.
- Run/result handling.
- Audit attribution.
- Consistent error semantics.

Class A returns its result inline.

Class D returns a stable run handle and exposes the appropriate lifecycle operations.

Transport adapters may translate HTTP, MCP or CLI input into canonical invocation requests, but must not duplicate policy or execution logic.

### 4.2 Class A — `crm-read`

Build the first synchronous business capability.

Use a deterministic test connector and controlled data fixtures.

Demonstrate:

1. Invocation through the canonical service.
2. Trusted identity and tenant resolution.
3. Authorization before business execution.
4. Tenant-scoped data access.
5. Business-result validation.
6. Attributed audit recording.
7. Inline result returned to the caller.

Test unauthorized access, cross-tenant access, invalid input, and repeated requests.

Do not introduce a polling or run-handle lifecycle for an ordinary synchronous read.

### 4.3 Class D — `lead-followup`

Build a durable business workflow using DBOS.

First prepare a workload-specific `adr001.md` decision record using the accepted template.

The record must justify Class D and identify the durable business requirements, persistence boundaries, entity identity, retries, external-effect boundaries, recovery behavior and version policy.

**Obtain the required human review of this decision record before enabling DBOS execution for `lead-followup`.**

Do not silently treat a prepared ADR as an approved ADR.

Use mock channels and test connectors only.

Demonstrate:

- Durable workflow submission.
- Stable run handle.
- Durable stage progression.
- Run/status/signal/cancel operations.
- Restart and replay.
- Duplicate submission protection.
- Entity-level serialization where required.
- Approval-pending recovery using mock approval channels.
- Complete audit attribution.
- Version-safe recovery.

The workflow must represent a meaningful business process, not simply a DBOS demonstration that sleeps and returns a string.

Do not invent business requirements that are absent from approved contracts or the miniapp specification. Record unresolved semantics as Open Spec Items.

### 4.4 Authentication, authorization and audit

Implement authorization before dispatch.

Test:

- Missing authentication.
- Invalid authority context.
- Unauthorized capability invocation.
- Cross-tenant access.
- Attribution of human and agent principals.
- Approval-pending execution.
- Recovery after a process restart.

Every consequential invocation must have a traceable audit record.

### 4.5 Effects and approvals

Implement the core Phase 1 record and service boundaries needed for Class A/D execution.

Use mock channels and test-only effects.

Do not release real external writes.

Preserve the distinction between an attempted effect and a confirmed external outcome.

### 4.6 Static registry and lifecycle

Implement static miniapp registration using independently versioned manifests.

Validate that:

- Manifest and implementation identity match.
- Registration failures fail closed.
- Invalid manifests are rejected.
- Duplicate registration cannot silently overwrite an existing miniapp.
- Runtime lookup uses immutable version identity.
- Release-envelope verification remains non-circular.

Implement the Phase 1 lifecycle developer commands:

`init → dev → validate → test`

Do not implement the full production lifecycle CLI or promotion system.

## 5. OpenCode integration

The G0-Core initial harness selection is OpenCode.

Use the accepted G0 evidence for the OpenCode 1.19.6 stdio integration.

Build one end-to-end harness demonstration using the supported MCP transport.

Demonstrate:

1. OpenCode connects to the framework.
2. Authentication is established.
3. Available miniapps are discoverable through the approved interface.
4. `crm-read` executes successfully.
5. An unauthorized invocation is refused.
6. `lead-followup` can be submitted and its stable run handle observed, after its ADR is approved.
7. Audit attribution is retained.

Keep the transport adapter thin.

Do not introduce a second harness implementation. The multi-family harness matrix belongs to G6.

Do not modify the user's global OpenCode configuration. Use isolated configuration for repeatable tests.

## 6. DBOS determinism analyzer

Implement the pinned-SDK-aware Go analyzer specified by the Phase 0 determinism rules.

At minimum, investigate and enforce the rules covering:

- Goroutines in durable workflow bodies.
- Select and nondeterministic channel behavior.
- Map iteration order.
- Direct time and random access.
- Mutable global state.
- Direct I/O in workflow bodies.
- Durable calls inside inappropriate step boundaries.

Use `go/analysis` where appropriate.

Build positive and negative analyzer fixtures.

Do not use broad grep-based checks as a substitute for semantic analysis when Go syntax and type information can establish the actual violation.

The analyzer must run in strict mode for Class D packages in CI before any Class D production release.

Report unsupported cases explicitly rather than claiming complete nondeterminism detection.

## 7. Required tests and failure scenarios

Implement tests for:

**Class A**

- Inline successful result.
- Invalid request.
- Unauthorized invocation.
- Cross-tenant denial.
- Audit attribution.

**Class D**

- Successful submission.
- Duplicate submission.
- Restart and replay.
- Pending old workflow during version transition.
- Version isolation and drain behavior.
- Approval-pending recovery.
- No duplicate confirmed effects.

**Shared framework**

- Static registration.
- Manifest validation.
- Authorization before dispatch.
- Idempotency.
- Entity serialization where required.
- Audit completeness.
- Worker shutdown ordering.
- Database pool shutdown safety.

**Analyzer**

- Valid workflow fixtures.
- Each prohibited nondeterministic construct.
- CI strict-mode enforcement.

Include adversarial tests that reproduce realistic failure modes. A test must fail when its invariant is deliberately violated.

Do not weaken existing tests to obtain a green result.

## 8. Delivery and verification

Work in coherent, reviewable increments.

At each increment:

1. Identify the required invariant.
2. Trace the implementation path through relevant packages.
3. Implement the smallest durable correction.
4. Add regression tests.
5. Verify local behavior.
6. Record evidence before moving to the next increment.

Use the existing local PostgreSQL development environment and test connectors.

Use Neon only where real database behavior requires it and only through the established disposable test environment.

No production deployment is required in Phase 1.

Run:

- Formatting checks.
- `go vet ./...`.
- Full Go tests.
- Race tests.
- Repository checks.
- Analyzer tests.
- OpenCode end-to-end tests.
- Real PostgreSQL integration tests.
- DBOS crash/replay and version-recovery tests.

Keep unit, local PostgreSQL and Neon evidence separate.

Bind all major test reports to the exact source revision tested.

## 9. Phase 1 exit gate — G1

Phase 1 is complete only when:

1. Class A works synchronously.
2. Class D works durably.
3. Both use the same canonical invocation service.
4. Authorization occurs before dispatch.
5. Audit attribution is complete.
6. DBOS workflow recovery is version-safe.
7. There are no duplicate confirmed effects.
8. Entity identity and serialization are demonstrated where required.
9. Approval-pending recovery works with mock channels.
10. The OpenCode integration works end to end.
11. The determinism analyzer works against its positive and negative fixtures.
12. CI enforces the required Class D rules.
13. Real provider writes remain disabled.
14. All required evidence is recorded and independently verifiable.

The G1 exit criterion is **Class A and Class D working on real PostgreSQL, with no duplicate confirmed effects, complete audit trail and version-safe workflow recovery.**

A successful demo alone does not satisfy G1.

## 10. Explicit exclusions

Do not implement:

- Phase 2 OAuth, token broker, custom REST provider, or real SaaS integrations.
- Phase 3 River workers or Class C substrate selection.
- Phase 4 first real external writes, Sprite workspace preview or publishing.
- Phase 5 Sprite tasks, catalog expansion and trusted-build promotion.
- Phase 6 production Fly deployment, deployment-overlap drills, multi-harness matrix or complete lifecycle acceptance.
- UI generation or a frontend framework.
- BPMN or a general-purpose workflow engine.
- An AI miniapp generator.
- Arbitrary agent-authored code execution in the trusted runtime.

Do not advance work merely because a package name appears in the roadmap.

## 11. Root-cause and scope discipline

When implementation exposes a problem:

- Investigate the underlying cause.
- Examine the complete system boundary and failure modes.
- Add a regression test that proves the required invariant.
- Implement a durable correction within Phase 1 authorization.
- Avoid superficial workarounds and symptom-only patches.

If a genuine architectural incompatibility or material security issue requires reopening a settled decision, stop the affected workstream and present evidence before expanding scope.

If a specification is incomplete, produce an Open Spec Items table containing the exact ambiguity, affected component, competing interpretations and consequence. Do not invent normative business behavior.

Do not turn this task into a repository-wide cleanup.

## 12. Git and delivery discipline

The G0 acceptance baseline has been committed to GitHub.

- Verify the actual current remote and local HEAD.
- Preserve the accepted G0 revision.
- Work from the approved mainline baseline or an explicitly recorded working branch.
- Do not force-push.
- Do not rewrite history.
- Do not include secrets or local state.
- Do not commit or push implementation changes unless separately authorized.

Prepare a precise changed-file inventory and a proposed commit boundary for review.

## 13. Final report

Deliver:

### A. Architecture and implementation summary

What actually exists now, not merely what was planned.

### B. Business vertical slices

Separate results for `crm-read` and `lead-followup`.

### C. Execution and security

Canonical invocation, identity, authorization, effects, approvals and audit behavior.

### D. DBOS

Transaction boundaries, workflow versioning, shutdown, recovery and replay.

### E. OpenCode integration

Exact harness version, configuration, calls and observed results.

### F. Determinism analyzer

Supported rules, fixture coverage, limitations and CI integration.

### G. Verification matrix

Exact commands, environments, exit statuses, and PASS/FAIL/BLOCKED/SKIPPED results.

### H. G1 readiness

A condition-by-condition assessment of every G1 exit requirement.

### I. Change inventory

Every added, modified and deleted path, with justification.

### J. Remaining work

Precise blockers, design decisions and recommended next action.

**Final instruction:** Build the smallest complete framework vertical slice that demonstrates real Class A and Class D business execution. The framework, not the demo, is the product. Preserve the architecture, use real evidence, and do not advance to Phase 2 or later work.

## **My recommendation**

Do not let the coding agent implement all packages independently and integrate them at the end. The most important implementation sequence is:

1. Canonical invocation service and static registry.
2. Authentication, authority and audit.
3. `crm-read` — prove Class A.
4. DBOS executor and workflow version registration.
5. Reviewed `lead-followup` ADR, then Class D execution.
6. Lifecycle and OpenCode integration.
7. Determinism analyzer and G1 acceptance tests.

That sequence keeps the central architectural invariant intact: **all callers invoke one governed execution service; miniapps provide business outcomes; DBOS provides durability where the business actually requires it.**