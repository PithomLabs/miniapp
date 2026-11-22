# Consolidated Adversarial Review Disposition — plan4.1.md

**Basis:** Combined DeepSeek, Gemini, and Z adversarial reviews, plus the subsequent architectural review of `plan4.1.md`.

**Disposition:** **Approved to begin Phase 0. No architecture-level blocker has been identified.** The reviews are substantially aligned. Their findings should be consolidated into the amendments below rather than treated as three separate sets of requirements.

This document is a consolidated review and amendment register. It does not itself modify `plan4.1.md`.

---

## 1. Consensus: decisions to preserve

The reviewers agree that the following are sound and should remain unchanged:

1. **Two independent dimensions:** execution semantics (Class A/B/C/D) and compute placement (Machine/Sprite/Hybrid/Local). Placement follows the trust profile of the code, not the identity of the agent caller.
2. **Selective DBOS adoption:** Class A uses ordinary Go/PostgreSQL operations; Class B uses River; Class C uses River by default and DBOS only when durable checkpointed recovery is justified; Class D uses DBOS subject to ADR-001.
3. **Trusted Go framework:** reference miniapps consume public framework APIs; the framework does not depend on reference miniapps. MCP, REST and CLI are thin adapters over one canonical invocation and authorization path.
4. **Manifest as governed artifact:** `miniapp.json` describes identity, version, tool contracts, execution class, capabilities, policy, fixtures, dependencies and deployment profile. Published artifacts are immutable.
5. **Two-principal authorization:** distinguish agent identity from the authority under which an operation is permitted. Support delegated human authority and explicitly approved service/trigger authority without granting agents implicit administrator power.
6. **External-effect honesty:** DBOS/River durability does not provide unconditional exactly-once effects at external providers. Unknown outcomes must be reconciled or quarantined, not blindly retried or compensated.
7. **Sprite/DBOS distinction:** Sprite checkpoints restore filesystem state; DBOS and Neon own authoritative business workflow state.
8. **Shared platform runtime by default:** use a central registry/router and shared Fly deployment initially. Dedicated Fly apps or process groups require a demonstrated isolation, scaling or customer need.
9. **Vertical-slice-first roadmap:** prove `crm-read` (Class A) and `lead-followup` (Class D) before expanding the catalog; add `contact-import` (Class C/River) next.
10. **Dependency honesty:** pin exact versions with dated evidence in Phase 0. No `latest` dependency policy and no assumptions that DBOS, River, MCP or Sprite SDKs expose unverified APIs.
11. **Lifecycle as a framework feature:** Design → Develop → Validate → Publish → Deploy → Operate → Maintain, with a developer CLI and CI/CD integration that use the same underlying services and policies.

## 2. Consolidated required amendments

The following findings recur across reviews or expose a material gap in the plan. They are consolidated into unique amendments and assigned to the phase where they must be resolved.

### A1 — Make the Sprite credential boundary strict

**Priority:** Critical; resolve in Phase 0 security contracts, implement before any untrusted Sprite code runs.

The current allowance for short-lived downscoped tokens inside Sprites is still unsafe for arbitrary or agent-authored code. A token remains a credential that code in the Sprite may inspect or exfiltrate, including to an allowlisted destination.

**Required amendment:** Untrusted or agent-authored code running in a Sprite must not receive SaaS credentials, refresh tokens, tenant master secrets or bearer tokens. External operations must go through the platform capability proxy. The proxy authenticates the scoped task identity, checks the approved capability, tenant/resource scope, risk policy and rate/budget limits, then performs the provider request itself. The Sprite receives only the permitted operation result.

Any exceptional direct-credential use must be limited to separately reviewed trusted code, not the general Sprite execution profile. Prefer removing the exception from the initial framework contract.

**Acceptance:** a Sprite process running untrusted code cannot read, enumerate or use tenant credentials, access another tenant, or call an unapproved provider operation. Network policy alone is not accepted as a credential boundary.

### A2 — Make Sprite task provisioning and cleanup crash-safe

**Priority:** High; design in Phase 0, prove before Phase 5 hybrid task release.

A DBOS step that provisions a Sprite can succeed externally and crash before its completion is checkpointed. Retrying the step must not create duplicate Sprites or duplicate execution. A runner can also die while a Sprite continues running and consuming resources.

**Required amendment:** define a durable Sprite task registry in Neon with a unique stable `task_id`, parent run/workflow identity, provisioning state, Sprite ID, policy digest, lease/fencing information, execution state, result reference and cleanup state. Provision/reconcile operations must be idempotent on the task identity. Add a lease-based reaper that detects abandoned tasks, revokes task authority, terminates/decommissions the Sprite, records an unknown or failed outcome as appropriate, and lets the parent workflow reconcile safely.

Test crashes after task intent creation, after Sprite creation, after policy application, during execution, after execution but before result persistence, and during cleanup.

### A3 — Enforce a policy-confirmed barrier before Sprite execution

**Priority:** Critical; define in Phase 0 and implement before any Sprite `exec` capability is exposed.

Sprite networking starts permissively. Provisioning, applying the policy and executing code are separate operations, so a partial failure or race could permit execution under an unconfirmed or stale policy.

**Required amendment:** the Sprite adapter must enforce a state machine in which execution is impossible until the expected restrictive network/resource policy has been applied and read back successfully. The policy digest must match the immutable deployment profile bound to the task. If policy application, read-back, or digest verification fails, fail closed and do not execute. Re-check the policy before each new task execution where configuration could have drifted.

### A4 — Bind deployment profiles and policy contents into immutable artifacts

**Priority:** Critical; resolve in Phase 0 contract freeze (G0).

A manifest that references a mutable profile by name can pass review and later run under different network, resource or security settings without changing its manifest hash.

**Required amendment:** content-address all referenced deployment profiles and transitive policy inputs at publish time, including Sprite network policy, resource limits, execution identity/profile and any security-relevant configuration. Record their digests in the published artifact and promotion record. `validate`, `publish`, `promote` and task admission must reject unresolved, unbound, stale or mismatched profile references. A profile change requires a new immutable version and fresh validation/review.

### A5 — Assign ownership for all deployment adapters

**Priority:** Required plan correction before the relevant phases are scheduled.

The roadmap defines `deploy/machine` and the Sprite workspace adapter but does not assign both to implementation phases. Phase 4 preview needs Sprite workspace operations; Phase 6 production deployment needs the Machines adapter.

**Required amendment / phase ownership:**

- **Phase 4:** implement the Sprite *workspace* adapter required for development preview (`ProvisionWorkspace`, policy, exec/dev services, snapshot/restore and decommission). Do not implement the full bounded Sprite task runner yet.
- **Phase 5:** implement the Fly Machines deployment adapter and immutable promotion/deployment integration before Phase 6 consumes it. Extend the Sprite adapter with bounded task execution, leases, cleanup and task reconciliation.
- **Phase 6:** consume and operationally harden both adapters; do not defer their initial implementation to this phase.

This explicitly separates Sprite workspaces for preview from Sprite tasks for hybrid workflow execution.

### A6 — Make Sprite-to-Machine promotion a trusted build/release process

**Priority:** High; define before Phase 4 publish and implement by Phase 5.

A Sprite checkpoint is not a trusted production release artifact. It is a filesystem snapshot and may contain mutable state, installed tools, secrets or unrelated files.

**Required amendment:** promotion must build from reviewed source in a controlled, trusted build environment. The release artifact should be an immutable OCI image or equivalent content-addressed build artifact with a dependency lock, build provenance, SBOM where feasible, test and security results, source revision, manifest digest, profile digests and reviewer/approval record. Deploy the immutable artifact to Machines. Sprite snapshots may be retained as workspace provenance or restore points, but must not be promoted directly as production images.

Rollback redeploys a previously approved immutable artifact and uses the documented workflow compatibility strategy.

### A7 — Name a security owner

**Priority:** Required Phase 0 task.

Name an individual accountable for signing the threat model and reviewing capability additions, privilege changes, Sprite security profiles and approval-boundary changes. G0 threat-model sign-off must name its approver. Phase ownership alone is not security ownership.

### A8 — Clarify why and when Sprites are in scope

**Priority:** Phase 0 plan-text clarification; not an architectural blocker.

State explicitly that Sprites have two distinct uses:

- **Preview/development workspaces:** in scope for the developer lifecycle, introduced in Phase 4.
- **Untrusted/generated code execution:** a later capability, introduced through bounded tasks in Phase 5 only after the security, provisioning and promotion gates pass. AI miniapp generation remains out of initial scope.

The deployment profile and placement contracts may be defined in Phase 0, but full task execution and promotion infrastructure should not be pulled forward merely because the interfaces exist.

### A9 — Exercise the complete hybrid failure matrix

**Priority:** Required for Phase 5 hybrid-task-demo acceptance.

`hybrid-task-demo` must test more than its happy path. It must exercise:

- Sprite provisioning or policy rejection;
- task timeout while the Sprite may still be running;
- Sprite loss or runner termination mid-task;
- result-schema mismatch;
- worker crash after Sprite creation but before task ID persistence/checkpoint completion;
- worker crash after task completion but before result persistence;
- cleanup/reaper behavior and repeated invocation.

The demo must prove idempotent task identity, bounded resource use, no credential leakage, authoritative Neon results and recoverable parent DBOS state. A separate failure demo is unnecessary if the same reference implementation's automated suite covers these cases.

### A10 — Define retention as a first-class policy

**Priority:** Define in Phase 2 or Phase 4; required before multi-tenant GA.

“No arbitrary active-handle expiry” must not become indefinite retention of every terminal record. Define retention and deletion/archive policies separately for active runs, terminal run history, DBOS history, River jobs, effects, audit events, Sprite tasks, promotion records, logs and traces. Specify owner, legal/security constraints, redaction, access, archival and deletion behavior. Surface retention configuration and usage in the operations view.

### A11 — Define DBOS executor availability and recovery ownership

**Priority:** Define in Phase 0; implement/validate in Phase 1 and Phase 6.

A single dedicated DBOS executor is a reasonable initial topology, but it is not by itself a high-availability design.

**Required amendment:** document Fly restart/health-check behavior, graceful drain and replacement, executor-down alerting, stuck-workflow detection, recovery ownership and operator procedure. Test deploy/restart/drain while workflows are active. Define explicit measured thresholds for adding a standby, multiple executors or DBOS Conductor. Do not claim HA from a single always-on Machine.

### A12 — Make phase/catalog and harness scope explicit

**Priority:** Phase 0 plan-text correction.

Add explicit phase numbers to the reference miniapp catalog, consistent with the roadmap. Recommended mapping:

- Phase 1: `crm-read`, `lead-followup`.
- Phase 2: connector/security integration; `crm-write` only after write-authorization tests.
- Phase 3: `notify-send`, `report-generate`, `contact-import`.
- Phase 4: `approval-flow` and preview workflow.
- Phase 5: `data-sync`, `appointment-journey`, `hybrid-task-demo`; `campaign-orchestration` only when its provider-effect gate passes.
- Deferred/assessed: `agent-toolchain`, with trusted Go execution defaulting to Machine placement and hybrid Sprite use requiring separate justification.

Clarify harness criteria: Phase 1 needs one harness to prove the end-to-end invocation path. Phase 6 proves interoperability across at least two distinct harness families and the transport/approval compatibility matrix. Do not describe Phase 1 as broad harness compatibility validation.

### A13 — Define Phase 0 failure dispositions

**Priority:** Phase 0 exit-gate requirement.

G0 must permit a negative result. For each load-bearing proof of concept, specify the consequence of failure:

- **DBOS transaction/pool-handle atomicity fails:** stop the affected adapter design; evaluate the supported DBOS transaction pattern or revise the persistence/transaction boundary. Do not claim atomic business-write/checkpoint behavior without proof.
- **Neon connection mode fails:** use a supported direct/session-compatible connection strategy or revise topology; do not silently substitute transaction pooling.
- **Sprite lifecycle/policy round-trip fails:** defer Sprite preview/task execution or revise the provider adapter; do not run untrusted code without policy verification.
- **Pinned SDK lacks required APIs:** select a verified compatible version, implement a typed REST gap only where supported, or revise/defer the feature. Do not code against imagined APIs.
- **MCP harness round-trip fails:** select a supported transport/adapter and narrow the compatibility claim.
- **RLS context design fails under pooling/concurrency:** revise transaction-local context and role model before real multi-tenant writes.

A failed spike triggers a documented architecture revision and renewed review; it is not waived by adding an undocumented workaround.

### A14 — Resolve the remaining lifecycle and operations seams

**Priority:** Phase 0 contracts and later implementation phases.

Consolidate the remaining reviewer notes into the following tasks:

- Freeze a minimal CLI command surface; do not duplicate `flyctl`, `psql` or existing platform administration tools. Every command maps to a lifecycle stage and calls shared services.
- Specify app-schema migration ownership separately from DBOS/River-managed schemas, including ordering, compatibility and CI/deploy gates. Never race application migrations against library-managed migrations.
- Re-verify `sprites-go` coverage at the pinned version. Use the SDK for supported lifecycle/exec/filesystem/checkpoint/policy operations; use a typed framework REST client only for verified API gaps. Both remain behind interfaces.
- Bind the truncated `prompt2.md` issue to G0: contracts are frozen subject to resolving the missing remainder or explicit owner acceptance of the labeled completions in Appendix D.
- Clean up the `TaskStatus` type/function naming collision during the public API signature freeze.

---

## 3. Findings merged or rejected as duplicates

The reviews contain several overlapping observations. They should not become separate work items:

| Overlapping findings | Consolidated treatment |
|---|---|
| Sprite credential leakage (prior review, Gemini C, Z C2) | A1: untrusted Sprite code uses the capability proxy; do not inject SaaS tokens. Token TTL/scope controls apply to proxy-issued task authority, not direct provider credentials. |
| Sprite task duplicate provisioning, mid-task runner death and orphan compute (Gemini A, Z C1, prior review) | A2 + A9: durable task registry, idempotent provisioning, leases, reaper and crash matrix. |
| Network default-open race (Gemini B, Z K15, prior review) | A3: policy-confirmed state barrier plus digest verification before execution. |
| Profile mutation bypasses published immutability (Z B2) | A4: content-address profiles and transitive security policy. |
| Adapter phase ownership (Z B1) | A5: Sprite workspace in Phase 4; Machine deployment adapter and Sprite task adapter by Phase 5; Phase 6 hardens. |
| Sprite snapshot mistaken for production artifact (prior review, Gemini promotion note) | A6: clean trusted build and immutable artifact promotion; snapshot is provenance/restore only. |
| Security owner (DeepSeek) | A7. |
| Sprite MVP justification (DeepSeek) | A8. |
| Hybrid demo coverage (DeepSeek) | A9. |
| Retention and no-expiry handles (Z C3) | A10: active handle stability, bounded terminal/audit retention. |
| Single DBOS executor and HA alerting (Gemini D, Z C4) | A11: restart/health, alerting, drain, stuck-workflow ownership and measured scale threshold. |
| Catalog phase ambiguity and harness scope (DeepSeek) | A12. |
| Phase 0 spike failure outcomes (DeepSeek) | A13. |
| CLI sprawl, migrations, SDK gaps, truncated prompt and signature naming (Gemini, Z N1–N4) | A14. |

**Do not adopt direct Sprite bearer-token injection as a compromise.** The more restrictive capability-proxy approach resolves the concern more safely and is consistent with the product's credential boundary.

**Do not add a second hybrid reference miniapp solely for failure coverage.** Expand the `hybrid-task-demo` automated acceptance suite instead.

**Do not move all Sprite task/promotion implementation into Phase 0.** Phase 0 freezes contracts and proves the basic provider round-trip; workspace preview and bounded tasks are implemented in their dependency-owning later phases.

---

## 4. Revised phase ownership and gates

| Phase | Consolidated scope additions | Exit evidence to add |
|---|---|---|
| **0 — Contracts and proofs** | Name security owner; freeze deployment profile digests; define Sprite task registry/state machine; define authority and credential boundaries; set retention requirements; define DBOS availability/recovery; assign adapter owners; specify migration ordering; resolve prompt2 remainder; document spike failure dispositions. | Dated, reproducible evidence for DBOS/Neon transaction semantics, River transactional enqueue, Sprite policy-before-exec round-trip, pinned SDK/API coverage, RLS context and MCP transport. A failed spike causes an explicit re-plan. |
| **1 — Vertical slice** | One harness end-to-end; DBOS worker health/restart baseline; audit and duplicate-submit proof. | Real PostgreSQL recovery; explicit distinction between local crash tests and real Fly tests. No broad two-harness claim yet. |
| **2 — Security and SaaS** | Strict capability proxy; no credentials in untrusted Sprites; retention design; OAuth/token and tenant controls. | Credential isolation, cross-tenant probes, revocation and scope enforcement; zero secrets in Sprite/task payloads, workflow checkpoints and logs. |
| **3 — River B/C** | Explicit catalog mapping; schema migration ownership and River lifecycle/retention. | B/C retry, batch progress, duplicate webhook and migration-order tests. |
| **4 — Approval and preview/publish foundations** | Sprite workspace adapter for preview; profile digest binding; initial publish metadata; CLI minimal set. | Preview refuses stale/unbound policy profiles; publish artifact and profile hashes are immutable. |
| **5 — Catalog and hybrid tasks** | Machine deployment adapter; Sprite task runner, idempotent provisioning, leases/reaper, policy barrier, hybrid failure demo, trusted build/promotion pipeline. | All hybrid failure cases pass; no orphan compute; no direct credential exposure; promotion requires immutable artifact, tests, provenance, digests and review. |
| **6 — Deployment/harness falsification** | Harden Machine/Sprite operations; executor-down and stuck-workflow alerts; full lifecycle and multi-harness tests. | Real Fly recovery/wake evidence; two harness families; deploy/rollback drills; operational runbooks and measured connection/latency/cost budgets. |

---

## 5. Phase 0 go/no-go conditions

Phase 0 may begin immediately. It must not automatically pass.

**G0 passes only when:**

- Core contracts and both classification dimensions are reviewed and versioned.
- Exact dependency versions are pinned and verified against source/docs.
- DBOS/Neon transaction semantics are proven with a crash probe at the actual pool-handle boundary.
- River transactional enqueue semantics are proven against the selected PostgreSQL setup.
- Sprite provisioning, policy application/read-back, execution and teardown work at the pinned API/SDK version.
- The credential boundary is capability-proxy based; untrusted Sprite code receives no SaaS credentials.
- Deployment profiles and referenced policy are content-addressed in the published artifact contract.
- Tenant context and RLS behavior are verified under the selected connection modes.
- The security owner signs the threat model.
- Every critical spike has a documented failure disposition.
- The truncated `prompt2.md` is resolved or its labeled completion is explicitly accepted by the owner.

If any load-bearing spike fails, stop before production adapters are implemented. Record the evidence, revise the architecture/contract and obtain renewed approval.

## 6. Final disposition

**Approved for Phase 0. No architecture-level blockers.** The reviewers independently affirm the two-dimension taxonomy, selective DBOS policy, shared invocation path, declarative manifests, vertical-slice-first sequence and evidence-gated treatment of operational unknowns.

The consolidated review identifies two required plan-text corrections from Z (adapter phase ownership and profile immutability) and elevates several security/reliability concerns from Gemini and the earlier review into explicit contract or phase-gate requirements. These corrections do not require a third architecture redesign.

The implementation baseline should be updated with A1–A14 above. Phase 0 can start now. Phase 1 should not begin until G0 passes. Sprite task execution and promotion should not begin until A1–A6 are implemented and tested in their owning phases.

**Approval statement:** Phase 0 is approved. The overall roadmap is conditionally approved subject to the amendments and evidence gates in this consolidated disposition.
