The next task should be **Phase 0 evidence completion and G0 closure preparation**, not another architecture review or an early start on Phase 1. The prompt below directs the coding agent to verify the reported work against the actual repository, complete everything possible, and distinguish genuine infrastructure blockers from missing owner decisions.

# Coding Agent Task: Phase 0 Completion, Evidence Verification, and G0 Closure

## Role

Act as the senior Go platform architect and implementation engineer for `github.com/PithomLabs/miniapp`.

You are continuing work in:

`/home/chaschel/Documents/go/miniapp`

The authoritative architecture baseline is `docs/plan4.1.md`, which is frozen. Phase 0 is underway and has received explicit approval.

Your mission is to complete and substantiate Phase 0, eliminate remaining implementation defects, prepare the outstanding infrastructure spikes for execution, and maintain an honest G0 readiness record.

This is an implementation and evidence task, not another architecture-review exercise.

## 1. Current implementation state

The following work has reportedly been completed:

- Go module initialized at Go 1.26 with six pinned, source-verified dependencies:
  - DBOS Transact Go v1.9.0
  - River v0.51.2
  - Official Go MCP SDK v1.8.0
  - sprites-go v0.2.2
  - pgx v5.13.0
  - jsonschema v0.14.0
- `contract/`: frozen contracts for manifests, capabilities, policy, runs, effects, approvals, Sprite tasks, promotion records, and deployment profiles.
- Nine generated JSON Schemas, with validation, drift, effect-state-machine, and content-addressed digest tests.
- `deploy/`: deployment profiles and interfaces.
- `internal/store/` and `internal/config/`: foundational shapes and tests.
- `spikes/`: harnesses for MCP, River transactions and uniqueness, DBOS atomicity and crash recovery, RLS context, and Sprite lifecycle.
- ADR-001 through ADR-017.
- Threat model, CLI specification, retention, availability and migration notes, dependency version register, CI skeleton, and migrations README.
- G0 evidence tracker: `docs/phase0/g0-evidence.md`.

Reported verification:

- `gofmt` clean.
- `go vet` clean.
- `go test ./...` green.
- Contract tests green, including A4 profile-digest binding.
- MCP in-process round-trip passes.
- Infrastructure spike harnesses compile and skip honestly because the required infrastructure access is unavailable.

Treat this as a status report to verify, not as proof that every acceptance condition has passed.

## 2. Absolute scope boundaries

You MUST:

- Continue only Phase 0 as defined in `docs/plan4.1.md`.
- Preserve `docs/plan4.1.md` as the frozen authoritative baseline.
- Preserve ADR-001 through ADR-017 and their numbering.
- Preserve A1–A16 and the existing execution-class and placement architecture.
- Keep the shared Go runtime, static miniapp registration, and independently versioned manifests as the default per ADR-016.
- Keep the lifecycle golden path and CLI/GitHub Actions parity design per ADR-017.
- Keep DBOS, River, Neon, Fly, and Sprite responsibilities within their existing phase boundaries.
- Preserve the G0 pass conditions and failure dispositions.

You MUST NOT:

- Implement Phase 1 miniapps, invocation runtime, or production execution paths.
- Pull Sprite workspace implementation, Sprite task execution, or production promotion forward.
- Introduce a new architecture, module, workflow engine, or dependency without a demonstrated requirement and an approved decision.
- Rewrite the frozen architecture document to accommodate implementation convenience.
- Convert skipped infrastructure tests into passes.
- invent owner decisions, organization access, provider credentials, security approvals, or successful spike results.
- weaken security requirements to get a green test suite.

Do not initiate another general adversarial review. Reopen a decision only when implementation evidence exposes an actual incompatibility, violated invariant, or material security problem.

## 3. First: establish the actual repository state

Before changing code:

1. Inspect the repository tree, Git status, module file, ADRs, Phase 0 documents, schemas, migrations, spike harnesses, and CI workflow.
2. Compare the actual implementation with the status report above.
3. Identify incomplete, duplicated, contradictory, or untested artifacts.
4. Run the existing validation suite using the correct environment.

This environment has known constraints:

- Default GOPATH and GOCACHE locations may be read-only.
- A parent `go.work` file interferes with module isolation.
- `flyctl` and Docker are unusable.
- No local PostgreSQL server exists.
- Neon is the only available path for live PostgreSQL integration spikes.

Use writable GOPATH/GOCACHE overrides and `GOWORK=off` when appropriate. Do not modify global machine configuration merely to make the project build.

Record the actual commands, environment requirements, exit codes, and test results.

## 4. Second: conduct a root-cause implementation audit

Review the implemented packages against the contracts and ADRs.

Pay particular attention to these invariants:

### Contract and manifest integrity

- Manifest identity and immutable version resolution must be unambiguous.
- Manifest implementation identity must not silently drift from its statically registered implementation.
- Deployment profiles and their transitive policy inputs must be content-addressed.
- A stale, unresolved, or mismatched profile must be rejected.
- The published JSON Schemas must remain consistent with Go contracts.
- Invalid manifests must fail closed with useful typed errors.

### Execution and external-effect semantics

- The effect state machine must reject illegal transitions.
- Idempotency identity must be stable and explicit.
- UNKNOWN outcomes must never be blindly retried.
- Approval records must bind decisions to the material operation and its inputs.
- No code should imply that a database transaction makes an external SaaS side effect exactly-once.
- Compensation must not be automatically invoked for an UNKNOWN effect without a safe basis.

### Identity and authorization

- Agent identity and authority identity must remain separate.
- Trust modes must be enforced server-side.
- A miniapp's declared permission must not exceed the approved capability and tenant scope.
- Missing authority must fail closed.
- Tenant context must be transaction-safe and concurrency-safe.
- RLS must remain defense in depth, not a substitute for application authorization.

### DBOS and River

- DBOS workflow determinism requirements must be explicit.
- Workflow IDs and payload contracts must be stable.
- The DBOS application-version recovery strategy must not assume that independently versioned manifests automatically isolate workflow code.
- River enqueue and uniqueness guarantees must be proven at the actual transaction boundary.
- Class C must not silently grow into a home-made workflow engine.

### Sprite security contracts

- Untrusted Sprite code must never receive SaaS credentials, refresh tokens, API keys, or bearer tokens.
- External operations must pass through the capability proxy.
- The network-policy confirmation barrier must fail closed.
- Sprite filesystem snapshots must never be treated as authoritative workflow state.
- Sprite task intent and result ownership must remain with Neon and the parent workflow.

Do not merely check that interfaces and structs exist. Investigate whether the implementation actually enforces the stated invariants.

For every defect found, fix the underlying cause and add a regression test demonstrating the invariant. Do not use superficial workarounds.

## 5. Third: prepare and execute G0 spikes

Inspect every existing spike harness and classify it accurately:

- Implemented and locally verified.
- Executed against a real dependency or infrastructure.
- Compiled but skipped.
- Blocked by missing credentials or organization access.
- Failed with a documented failure disposition.

The distinction matters. A compiling harness is not proof of production behavior.

### Required spike evidence

**DBOS + Neon**

Prove the actual application-write/checkpoint transaction boundary using the pinned DBOS version and real Neon PostgreSQL connection configuration.

Include crash injection at relevant transaction boundaries. Verify the persisted state after restart. Demonstrate that no atomicity claim exceeds what the tested pool handle actually guarantees.

**River + PostgreSQL**

Prove transactional enqueue with the actual domain transaction. Verify uniqueness behavior under concurrent and repeated submissions. Record the selected connection mode and any limitations.

**RLS and tenant context**

Verify transaction-local tenant context under the selected connection mode. Test concurrent tenant transactions, context reset, role behavior, and cross-tenant access.

**Sprite round-trip**

At the pinned SDK version, verify provisioning, policy application, policy read-back, digest comparison, execution, and teardown.

Never execute untrusted code unless the restrictive policy is confirmed first.

**MCP**

Distinguish the existing in-process round-trip from a genuine transport-level round-trip. Verify the contract actually required by G0 and document exactly which transport and client/server boundary was exercised.

Do not overstate in-process evidence as multi-process or deployed-harness compatibility.

### Infrastructure restrictions

Do not repeatedly attempt to use Docker, flyctl, or a local PostgreSQL process when the environment cannot support them.

Where Q4 access is missing:

- Keep the harnesses executable and independently testable.
- Ensure their configuration and setup instructions are complete.
- Make skips explicit, deterministic, and reasoned.
- Document the precise credentials, organizations, resources, and commands required to execute each live spike.
- Do not fabricate live results.

Where a spike cannot run, produce a reproducible execution runbook rather than a fictional pass.

## 6. Fourth: resolve G0 owner actions without inventing decisions

Maintain the G0 evidence tracker as the single operational record.

Outstanding owner decisions:

- **Q9 — Security owner:** obtain the named owner's identity and explicit threat-model sign-off. Do not nominate someone without authorization.
- **Q8 — prompt2 remainder:** determine whether the missing remainder exists. Otherwise obtain explicit owner acceptance of Appendix D. Do not reconstruct missing requirements and label them original instructions.
- **Q4 — Infrastructure:** obtain the actual Fly, Neon, and Sprite organization, region, quota, credentials, and resource access required by the applicable spikes.
- **Q1 — First SaaS:** keep the first real SaaS decision visible and request the required owner decision. Preserve its formal relationship to the Phase 2 integration gate in `docs/plan4.1.md`; do not silently rewrite G0's formal pass conditions.

Prepare a concise owner-action section in `docs/phase0/g0-evidence.md` showing exactly what is needed, why, who must provide it, and what evidence will close it.

No fake sign-offs, placeholder approvals, or invented infrastructure facts.

## 7. Fifth: verify ADR and documentation consistency

Check that:

- ADR-001 through ADR-017 exist and match the approved decisions.
- The implementation does not contradict the frozen baseline.
- A15 and A16 are reflected in the actual contracts and Phase 0 evidence, not merely documented.
- The repository tree and package ownership match the Phase 0 implementation.
- The dependency register contains exact pins, source links, verification dates, and licenses.
- The threat model has a clear approval boundary.
- The migration notes distinguish application-owned schema migrations from DBOS- and River-managed schema migrations.
- Retention and availability notes have explicit ownership and operational consequences.
- The CLI specification does not create a second implementation of business or deployment policy.

Correct outdated references and documentation inconsistencies within Phase 0 scope. Do not create duplicate documents just to record the same fact.

## 8. Sixth: automated quality and regression checks

Run the checks that are possible in this environment:

- `gofmt`
- `go vet ./...`
- `go test ./...`
- contract validation and JSON Schema drift tests
- race tests where supported
- module dependency and license checks
- static checks for import-boundary violations
- spike harness compilation and skip behavior
- CI workflow validation where tooling is available

Add regression tests for any defects uncovered in the audit.

A test that skips must report why it skipped. A skip must never be counted as evidence that the external invariant passed.

Do not alter a test to make it green while weakening the requirement it was designed to verify.

## 9. Required deliverables

At completion, provide:

1. **Repository audit report** — actual state versus the reported state.
2. **Change summary** — files changed, defects fixed, and tests added.
3. **Verification report** — exact commands, environment configuration, results, and skipped checks.
4. **G0 evidence tracker update** — every G0 condition marked with its actual evidence and status.
5. **Owner-action checklist** — precise remaining decisions and infrastructure access needed.
6. **Spike execution runbooks** — only where live execution remains blocked.
7. **Final readiness recommendation** — whether Phase 0 is complete, whether G0 has passed, and what specifically prevents the next phase.

Use these statuses consistently:

- `PASS`: acceptance condition demonstrated by reproducible evidence.
- `BLOCKED`: cannot be demonstrated because a required external dependency or owner action is unavailable.
- `FAIL`: attempted and did not satisfy the acceptance condition.
- `NOT STARTED`: no meaningful implementation or evidence exists.
- `NOT APPLICABLE`: justified by the approved architecture.

Do not use `PASS` for a compiling harness, a skipped test, or a documentation assertion.

## 10. Completion criteria

This task is complete when:

- The actual Phase 0 implementation has been audited.
- All defects within Phase 0 scope have been fixed or explicitly dispositioned.
- Automated tests and static checks pass in the supported environment.
- Infrastructure spike harnesses are ready for reproducible live execution.
- Evidence and owner-action records are accurate.
- No Phase 1 or deferred Sprite implementation has been introduced.
- The remaining G0 blockers are explicit and actionable.

**Do not declare G0 passed unless every formal condition in `docs/plan4.1.md` §23.1 has been satisfied with evidence.**

Once G0 genuinely passes, stop. Do not begin Phase 1 in this task. Report that Phase 1 is now eligible to start under the existing approved roadmap.

## Operating principle

The objective is not to make the tracker green. It is to make the system's claims true.

Investigate root causes, test actual failure boundaries, preserve security invariants, and implement durable fixes. Distinguish what works from what merely compiles, what has been simulated from what has been proven against real infrastructure, and what the agent can complete from what requires explicit owner action.

**No more architecture review cycles. Execute Phase 0, gather evidence, and let G0 decide when implementation may advance.**

One detail worth preserving: under `plan4.1.md`, Q1 is formally a Phase 2 integration dependency, while Q8, Q9 and Q4 affect G0 readiness. The prompt keeps Q1 visible as an owner action without silently changing the frozen gate definition.