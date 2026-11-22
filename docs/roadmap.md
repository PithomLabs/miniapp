My recommendation is to **stop adding infrastructure and test spikes for now, finish G0-Core formally, and then move into Phase 1: the actual framework vertical slice.**

You have reached an important transition. The difficult Phase 0 engineering work is largely behind you. The next major value is no longer another isolated proof: it is turning the contracts and proven execution boundaries into a working miniapp framework.

There is one distinction to preserve: conditions 3 and 5 now have strong fresh evidence, but the tracker has not yet been updated and G0-Core has not formally passed.

# **1. Where we are today**

# **Phase 0 — Core architecture and enabling proofs**

Implementation complete; G0-Core pending

The foundation, contracts, dependencies, security design, execution tests and integration evidence are substantially established.

# **3, 5**

Fresh technical evidence for G0 conditions

# **2, 9, 10**

Previously reported PASS

# **1, 4, 6, 11**

Still require adjudication or evidence closure

# **7, 8**

Owner-dependent blockers

Here is the current interpretation:

| G0 condition                             | Current position | Remaining work                                               |
| ---------------------------------------- | ---------------- | ------------------------------------------------------------ |
| 1. Core contracts                        | Partial          | Formal review and sign-off                                   |
| 2. Dependency pins                       | Pass             | None reported                                                |
| 3. DBOS shared-handle atomicity          | Pass eligible    | Record new evidence in tracker                               |
| 4. DBOS family/version isolation         | Partial          | Adjudicate version-retention policy and safe worker topology |
| 5. Tenant isolation/RLS                  | Pass eligible    | Record concurrent test evidence                              |
| 6. Initial harness                       | Partial          | Name first target harness and verify its round trip          |
| 7. Security owner                        | Blocked          | Q9: name owner and obtain threat-model signature             |
| 8. Prompt 2 resolution                   | Blocked          | Q8: accept Appendix D or supply missing remainder            |
| 9. ADR-001 and decision responsibilities | Pass             | None reported                                                |
| 10. A15/A16 design                       | Pass             | None reported                                                |
| 11. Core spike evidence                  | Partial          | Source revision and complete dated evidence                  |

**My assessment:** you have substantially reduced the technical uncertainty. The remaining G0 work is predominantly about architecture acceptance, accountability, and evidence provenance.

One important clarification concerning condition 4: plan4.2 requires an explicit workflow-version and old-version retention/drain strategy at G0. The actual Fly deployment-overlap and graceful-shutdown drill belongs to G6. Do not defer the G0 design requirement merely because the operational Fly drill happens later.

# **2. The roadmap — from foundation to production**

The governing roadmap is `docs/plan4.2.md`, §19, with its readiness conditions in §23.

## **Miniapp implementation roadmap**

1. **Phase 0 — Core architecture contracts and enabling proofs**

   Current: closure pending

   Establish contracts, identity and authority, execution semantics, dependency pins, DBOS integration, tenant isolation, security boundaries, and evidence.

   Exit: G0-Core.
2. **Phase 1 — Core vertical slice and lifecycle bootstrap**

   Next development phase

   Build the actual framework: canonical invocation, Class A and Class D executors, authorization, audit, static registry, lifecycle adapters and one working harness.

   Exit: G1.
3. **Phase 2 — Provider security and read-only integrations**

   Establish provider identity, token broker, capability proxy, SSRF protection, OAuth and real read-only integrations.

   One SaaS sandbox and one controlled custom REST destination. No autonomous external writes.

   Exit: G2.
4. **Phase 3 — River jobs and Class C substrate decision**

   Implement Class B background jobs, investigate batch workloads, and compare River against DBOS for Class C.

   Choose a substrate through measured evidence and a human-reviewed ADR-001 decision.

   Exit: G3.
5. **Phase 4 — Effects, approvals, first external write and Sprite workspace preview**

   Implement effects ledger, reconciliation, UNKNOWN quarantine, approvals, separation of duties, compensation, and entity serialization.

   Independently investigate Sprite workspace preview and publish.

   Exit: G4.
6. **Phase 5 — Catalog expansion, bounded Sprite tasks and promotion**

   Expand the business-outcome miniapp catalog. Implement bounded Sprite tasks, provisioning, policy enforcement, trusted-build artifacts and promotion/rollback.

   Exit: G5.
7. **Phase 6 — Production falsification and full lifecycle acceptance**

   Test the guarantees under actual production-like conditions: Fly deployment overlap, DBOS recovery, Neon PITR, two harness families, retention and erasure, cost, connections, kill switches, and the complete lifecycle.

   Exit: G6.

The important architectural principle is that later phases are not prerequisites for unrelated earlier work. A failed Sprite extension, for example, should defer that extension rather than automatically block the Machine-based execution path.

# **3. What each phase actually delivers**

This is where I would concentrate attention.

## **Phase 1 — Build the framework, not another collection of proofs**

Highest development priority

The Phase 1 objective is to demonstrate that the framework can actually execute meaningful business operations.

The planned packages and capabilities include:

| Component             | Responsibility                                       |
| --------------------- | ---------------------------------------------------- |
| `registry/`           | Static registration of miniapps and manifests        |
| `runtime/`            | Canonical invocation and execution lifecycle         |
| `executor/classa`     | Synchronous request-bound execution                  |
| `executor/dbos`       | Durable Class D execution                            |
| `auth/`               | Principal resolution and authorization               |
| `effects/`            | Core effect recording and state handling             |
| `approvals/`          | Core approval mechanisms                             |
| `audit/`              | Attributed, durable audit records                    |
| REST/MCP/CLI adapters | Multiple interfaces into the same invocation service |
| `testkit/`            | Reusable test infrastructure                         |

The first two illustrative business capabilities are:

- `crm-read`: a synchronous Class A operation.
- `lead-followup`: a durable Class D operation.

The lead-followup workflow should have its own reviewed `adr001.md` decision record before enabling DBOS execution.

The Phase 1 developer experience should include:

`init → dev → validate → test`

The framework should also support:

- Inline synchronous results for Class A.
- Stable run handles for Class D.
- Authentication before dispatch.
- Audit attribution.
- DBOS per-family `AppName` and workflow-version registration.
- Run, status, signal and cancel.
- Restart and replay.
- Duplicate submission protection.
- Version isolation and drain testing.
- A pinned-SDK-aware Go determinism analyzer.

**The G1 success criterion is not merely a successful demo.** Class A and Class D must work on real PostgreSQL with no duplicate confirmed effects, a complete audit trail, and version-safe recovery. Real external provider writes remain disabled.

## **Phase 2 — Connect to real business systems safely**

This is where the framework starts becoming useful to customers with actual SaaS accounts.

The work includes OAuth and PKCE, permission intersection, token refresh/revocation, reauthentication, webhook security, SSRF protection, provider rate limiting and kill switches.

The first SaaS choice (Q1) belongs here, not in Phase 1.

## **Phase 3 — Add the job and batch execution model**

This phase establishes:

- Class B jobs using River.
- Transactional enqueue.
- Retry and unique-job behavior.
- Per-record idempotency and progress.
- A measured River-versus-DBOS comparison for Class C.

Do not select River for Class C merely because it works for Class B. That decision requires workload-specific evidence.

## **Phase 4 — Make external writes safe**

This is a particularly consequential phase.

Before enabling the first external write, you need:

- A durable effects ledger.
- Reconciliation by reading the external provider.
- UNKNOWN outcome quarantine.
- Authenticated out-of-band approvals.
- Separation of duties.
- Idempotent retry and compensation.
- Entity-level serialization.
- Blast-radius limits.

Sprite workspace preview is an independent extension. Its failure should not automatically block the Machine-based write path.

## **Phase 5 — Expand the catalog and introduce controlled untrusted execution**

The roadmap names additional miniapps such as:

- `data-sync`
- `appointment-journey`
- `hybrid-task-demo`

It also introduces bounded Sprite tasks and trusted-build promotion.

This is deliberately late because arbitrary agent-authored code requires strong credential isolation, policy confirmation, resource controls, lifecycle management and trusted artifact provenance.

## **Phase 6 — Try to falsify the production claims**

This is where you prove the entire system survives realistic failure:

- Fly deployment overlap and graceful shutdown.
- DBOS wake/restart and workflow-version drain.
- Neon point-in-time restore and external-effect reconciliation.
- Two-harness compatibility.
- Retention and erasure.
- Kill-switch operation.
- Full preview-to-publish-to-deploy-to-upgrade-to-rollback-to-retire lifecycle.

You should not claim HA from one executor per DBOS application/family, or scale-to-zero without the relevant wake test.

# **4. Prioritized next tasks**

Here is the order I recommend, based on impact and dependencies rather than convenience.

**Priority 1 — Close G0-Core**

Immediate

Produce a final G0 adjudication and owner decision package.

Update conditions 3 and 5 with the newly proven evidence. Resolve conditions 1, 4, 6, 7, 8 and 11. Keep all remaining requirements honest.

**Outcome:** a genuinely accepted G0-Core gate.

**Priority 2 — Establish source provenance**

Before Phase 1

Obtain authorization for the first Git commit. Review the complete tree, commit the accepted Phase 0 baseline, then run the necessary evidence-producing checks against that exact revision.

**Outcome:** reproducible test evidence tied to a real source revision.

**Priority 3 — Phase 1 core vertical slice**

Next major build

Implement canonical invocation, Class A, Class D, registry, authorization, audit, and lifecycle adapters. Demonstrate `crm-read` and `lead-followup` end to end.

**Outcome:** a usable, working framework rather than just contracts and spikes.

**Priority 4 — Phase 2 provider security**

Select the first SaaS provider and establish secure read-only integrations.

**Priority 5 — Phase 3 River and Class C**

Add background jobs and prove the right batch-processing architecture.

**Priority 6 — Phase 4 effects and first writes**

Enable real external actions only when approval and reconciliation controls are proven.

**Priority 7 — Phase 5 and 6 expansion**

Introduce Sprite execution, trusted promotion, production falsification and full lifecycle acceptance.

# **5. Owner decisions to settle now**

I would consolidate these decisions rather than opening another technical investigation.

| Decision      | What is needed                                                           | Priority            |
| ------------- | ------------------------------------------------------------------------ | ------------------- |
| Q9            | Name the security owner who signs the core threat model                  | G0 blocker          |
| Q8            | Accept Appendix D or provide the missing Prompt 2 remainder              | G0 blocker          |
| Condition 1   | Formal core contract review/sign-off                                     | G0 blocker          |
| Condition 4   | Adjudicate workflow versioning, retention/drain and safe worker topology | G0 blocker          |
| First harness | Name the initial harness for the G0 invocation round trip                | G0 blocker          |
| Condition 11  | Authorize first commit and source-revision evidence                      | G0 blocker          |
| Q7            | Approve or revise the Phase 0 CLI assessment and keep-list               | Phase 1 preparation |

A subtle but important distinction from Appendix B:

- The initial harness for G0 is needed now.
- The two-family harness compatibility matrix is a Phase 6 requirement (Q2). Do not force the complete multi-harness exercise into G0.

Other owner decisions can wait:

| Question                                 | Roadmap dependency                                                                    |
| ---------------------------------------- | ------------------------------------------------------------------------------------- |
| Q1: First SaaS                           | Phase 2                                                                               |
| Q2: Two harness families                 | Phase 6                                                                               |
| Q3: DBOS Enterprise budget               | Revisit if the Phase 6 open-source wake test fails                                    |
| Q4: Infrastructure organizations/regions | Neon resolved; Fly before G6; Sprite organization/quota before Phase 4 workspace work |
| Q5: `plan3_review.md`                    | Low-priority Phase 0 housekeeping                                                     |
| Q6: `agent-toolchain`                    | Catalog roadmap decision                                                              |

# **6. What I would do next, concretely**

I would assign the next task this name:

**Prompt 8 — G0-Core Final Adjudication, Owner Sign-Off and Evidence Freeze.**

It should be documentation and evidence work only, not another round of coding.

Its deliverables should be:

1. Updated G0 tracker with conditions 3 and 5 supported by the two new test reports.
2. Exact unresolved owner decision list.
3. A precise condition-4 interpretation separating G0's required version/topology strategy from G6's live deployment-overlap proof.
4. First-harness selection and evidence requirements.
5. A reviewed source tree ready for the first authorized commit.
6. A final G0 pass/not-passed assessment.

Only after G0-Core passes should the next substantial implementation prompt become **Prompt 9 — Phase 1 Core Vertical Slice**.

My strongest recommendation is to resist the temptation to jump directly into Phase 2, Sprite or the first SaaS integration. The most valuable work now is to get the core execution engine working.

**The strategic shift:** Phase 0 proved that the architecture can be implemented safely. Phase 1 must prove that the architecture can deliver a real business outcome. That is the milestone that turns this repository from a promising framework skeleton into a credible product foundation.