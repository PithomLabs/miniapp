## chatgpt

# **Adversarial review: Consolidated Implementation Plan v2**



Architecture is viable

Conditional approval



**The plan is substantially coherent, and I do not see a reason to abandon its overall architecture.** In particular, separating execution semantics from compute placement is the right architectural decision. It resolves the earlier ambiguity between long-running business processes and agent-oriented isolated execution.

I would approve it as the basis for Phase 0, but I would not yet call it an implementation-ready specification. There are several important gaps around Sprite security, hybrid task recovery, artifact promotion and lifecycle operations that must be closed before the affected components are implemented.

My main concern is not DBOS versus River, or Machines versus Sprites. It is whether the platform can reliably move a miniapp from development to a trusted, deployed and maintainable production artifact without weakening its security or execution guarantees.

## **1. What the plan gets right**

There are several decisions I would explicitly preserve.

### **The two-dimensional architecture**

This is the most important improvement:

- Dimension A: execution semantics — Class A, B, C or D.
- Dimension B: compute placement — Machine, Sprite, hybrid or local.

The plan correctly says a durable workflow can orchestrate an isolated Sprite task without making the Sprite the owner of business state.&#x20;

plan4.1(5).md



This resolves an important conceptual ambiguity:

**An agent-invokable miniapp is not automatically a Sprite workload.**

A trusted Go miniapp that exposes a business operation to Codex or Hermes is ordinarily a Machine workload. An environment running agent-authored, generated or otherwise untrusted code is a Sprite workload.

That distinction should remain.

### **The execution-class allocation**

The Class A/B/C/D decision is consistent with the DBOS adoption policy:

- Class A: ordinary Go handler and PostgreSQL.
- Class B: River.
- Class C: River by default, DBOS when checkpointed recovery is justified.
- Class D: DBOS.

The plan also correctly keeps per-entity serialization and Saga compensation as composable patterns rather than creating unnecessary execution classes.&#x20;

plan4.1(5).md



### **The Go framework boundary**

The proposed dependency direction is appropriate. Reference miniapps depend on public framework interfaces; the framework does not import reference miniapps; transport adapters do not own business logic.&#x20;

plan4.1(5).md



The distinction between `miniapp.json` as a contract/permission manifest and Go code as the implementation is also a sound decision. It avoids accidentally creating an unrestricted LLM-generated code deployment system.

### **The development and operations lifecycle**

The seven-stage lifecycle is a valuable part of the architecture:

Design → Develop → Validate → Publish → Deploy → Operate → Maintain.

The unified CLI, immutable publication, deployment profiles, status inspection and rollback are all appropriate foundations for the developer and operator experience you want.&#x20;

plan4.1(5).md



I would retain this as a core framework deliverable, not an optional developer-tools project.

## **2. Adversarial findings requiring corrections**

These are the issues I would resolve before their respective implementation phases.

### **Finding 1 — Sprite credential isolation is not strict enough**

Critical security correction

Section 14 permits a Sprite to receive short-lived, downscoped tokens when the manifest allows them.&#x20;

plan4.1(5).md



I would remove this exception for untrusted or agent-authored code.

A short-lived token is still a credential. Code running inside the Sprite can inspect its process environment, filesystem and memory. A domain allowlist does not guarantee that a credential cannot be transmitted to an allowed destination.

Fly's own documentation confirms that network access begins permissively and that an external network policy must be applied. Its policies restrict outbound network access, but do not replace API-level authorization.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://fly.io\&sz=32)

Fly+1



**Required correction:**

The default Sprite execution model must be:

```
Untrusted code inside Sprite
           |
           v
     Capability proxy
           |
           v
Platform authorization
           |
           v
Approved provider operation
           |
           v
External SaaS API
```

The Sprite never receives SaaS access tokens, refresh tokens, unrestricted API keys or tenant master credentials.

Instead, the capability proxy performs approved operations after checking the task identity, permission manifest, authority, endpoint and operation scope.

The Sprite receives the result, not the credential.

This also makes the external-effect protocol applicable consistently to both trusted Go workflows and isolated Sprite execution.

**Acceptance requirement:** demonstrate that code with full root privileges inside a Sprite cannot extract or use a tenant's SaaS credentials, access another tenant, or execute an operation outside the task's approved capability contract.

### **Finding 2 — Hybrid DBOS/Sprite task creation needs stronger recovery semantics**

High priority

The plan introduces stable `task_id` values and says the orchestrator stores task identity in Neon. It also has explicit task lifecycle states.&#x20;

plan4.1(5).md

plan4.1(5).md



However, it does not fully specify the crash window around Sprite provisioning.

Consider:

1. DBOS calls the Sprite provisioning API.
2. Fly creates the Sprite.
3. The DBOS worker crashes before it records the Sprite ID.
4. DBOS retries the step.
5. Another Sprite is created.

You now have duplicate compute, potentially duplicated code execution, and an orphaned Sprite.

This matters because DBOS steps execute at least once. A process can crash after an external side effect but before the step's completion is durably recorded.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://docs.dbos.dev\&sz=32)

DBOS Docs



**Required correction:** define Sprite task provisioning as a recoverable, idempotent resource lifecycle.

I recommend a durable task registry with:

| Field              | Purpose                                                     |
| ------------------ | ----------------------------------------------------------- |
| `task_id`          | Stable unique business task identity                        |
| `parent_run_id`    | Parent execution                                            |
| `workflow_id`      | DBOS workflow                                               |
| `sprite_id`        | Actual Fly resource                                         |
| `state`            | Provisioning, policy verified, executing, completed, failed |
| `lease_expires_at` | Cleanup and orphan detection                                |
| `result_ref`       | Authoritative Neon result                                   |
| `last_error`       | Recovery diagnostics                                        |

The operation should first persist a unique task intent in PostgreSQL, before attempting provisioning.

On retry, the runner must recover the existing task rather than blindly create another Sprite. Provisioning failures with an unknown result need reconciliation, and abandoned resources need a sweeper.

The Phase 0 Sprite proof of concept must test worker termination at every provisioning transition, not merely a successful create/run/delete sequence.

### **Finding 3 — Sprite-to-Machine promotion is not yet a complete release pipeline**

High priority

The plan describes immutable promotion records containing a manifest hash, dependency lock, tests, provenance and a source Sprite snapshot ID.&#x20;

plan4.1(5).md



That is good governance, but the production artifact-creation mechanism is not sufficiently specified.

A Sprite checkpoint is a snapshot of the Sprite's filesystem, not a trusted, reproducible production release artifact. Fly documents that checkpoints can capture the entire writable environment, including installed packages and database changes.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://fly.io\&sz=32)

Fly+1



**Required correction:**

Promotion must mean:

```
Sprite development workspace
          |
          v
Reviewed source changes
          |
          v
Clean build in trusted build environment
          |
          v
Tests + security checks
          |
          v
Immutable OCI image
          |
          v
Image digest + SBOM + provenance
          |
          v
Authorized release decision
          |
          v
Fly Machine deployment
```

The Sprite snapshot can be retained as development provenance or restored for workspace recovery. It must not be treated as the production artifact.

The release must be built from reviewed source in a controlled build environment, with a pinned dependency lock and a verifiable artifact digest.

Rollback should redeploy a previously approved artifact, not attempt to reconstruct a previous production state from a Sprite checkpoint.

This is particularly important because the platform's selling point is that agents can create isolated work and promote only trusted results.

### **Finding 4 — The Go packaging and runtime model is still ambiguous**

Architecture clarification

The plan chooses one repository and one Go module. It also describes a shared runtime by default and separately deployable miniapp binaries.&#x20;

plan4.1(5).md



These choices can coexist, but the build and registration model needs to be explicit.

There are at least two possible designs:

**Model A — Shared host**

One trusted Go runtime includes registered miniapps. Each has its own manifest, permission model and version, but they share the platform deployment.

**Model B — Independent binaries**

Each miniapp has its own `main` package and binary, depending on the same framework library. A central invocation gateway routes calls to the appropriate service.

The plan currently describes both without fully specifying how the registry, MCP router, release system and deployment topology work together.

I recommend Model A for the initial reference catalog. It minimizes deployment complexity and provides one consistent invocation path.

Keep the framework capable of supporting Model B later, for customers that require independent deployment, resource isolation or release cadence.

Do not rely on Go's plugin mechanism as a general-purpose hot-loading solution.

Phase 0 should freeze the registration, build and binary model.

### **Finding 5 — Deployment UX needs a stronger end-to-end acceptance test**

The plan contains a good CLI proposal and lifecycle model, but deployment operations are spread across Phases 4, 5 and 6.&#x20;

plan4.1(5).md

plan4.1(5).md



The user experience should not be judged by whether individual commands exist.

It should be judged by whether a developer can:

1. Scaffold a new miniapp.
2. Develop and run it locally.
3. Connect test integrations.
4. Validate contracts and policies.
5. Run fixtures and failure tests.
6. Preview against an isolated environment.
7. Publish an immutable artifact.
8. Deploy to the correct Fly target.
9. Inspect execution and deployment health.
10. Upgrade, roll back or retire the miniapp safely.

The implementation plan should demonstrate this journey with a complete working example.

I would add two things:

- **CI/CD integration:** document a standard GitHub Actions workflow for validation, test, build, publish and deploy, using the same framework services and policies as the CLI.
- **Environment management:** explicitly cover local, preview, staging and production environments, connection references, credential rotation, Neon branches, migrations and deployment configuration.

The CLI should hide routine Fly API details without hiding important operational choices.

### **Finding 6 — Phase sequencing needs one small correction**

Phase 1 includes worker-kill recovery and an acceptance criterion involving a machine stop. Actual Fly deployment and operational validation, however, are concentrated in Phase 6.&#x20;

plan4.1(5).md

plan4.1(5).md



Clarify the acceptance environments:

- G1: real PostgreSQL, local/container process termination and restart.
- G6: actual Fly Machine termination, replacement, wake-up and recovery.

Alternatively, introduce a minimal real Fly test deployment in Phase 1.

The important point is not to claim real Fly recovery based on a local process-kill test.

## **3. Additional technical checks**

These are important, but the plan already anticipates most of them.

| Area                                | My assessment                                                                                           |
| ----------------------------------- | ------------------------------------------------------------------------------------------------------- |
| DBOS and Neon transaction semantics | Correctly identified as a proof-of-concept requirement                                                  |
| Direct vs pooled Neon connections   | Correctly separated; must validate the pinned configuration                                             |
| DBOS sleep suspension               | Correctly treated as an Enterprise-dependent capability, not assumed                                    |
| Single DBOS executor                | Reasonable initial topology                                                                             |
| River and DBOS ownership            | Correct; one authoritative executor per workload                                                        |
| RLS                                 | Good direction; must test missing tenant context, pooled connection reuse and privileged database roles |
| MCP router                          | Correct to begin with generic tools and introduce namespaced tools after compatibility testing          |
| SDK gaps                            | Good to isolate `sprites-go` and the REST client behind interfaces                                      |
| Reference miniapp prioritization    | Good forcing set                                                                                        |
| Agent-toolchain                     | Sensibly deferred pending scope justification                                                           |

One DBOS detail worth preserving is the difference between durable steps and datasource transactions. The official Go documentation explains that a correctly configured datasource transaction can atomically commit application writes with the DBOS durability record; a normal step does not provide that same exactly-once database-write guarantee.&#x20;

[image](https://www.google.com/s2/favicons?domain=https://docs.dbos.dev\&sz=32)

DBOS Docs+1



The plan is right to make this a Phase 0 proof rather than assume that sharing a Neon database URL is sufficient.

## **4. Reference miniapp assessment**

I agree with the implementation sequence in §16.

## **Initial proving set**

[Meisterst du schon CRM für KMU? Mache den Test!](https://images.openai.com/static-rsc-4/HXUfkyqS1ILu5ttLNO2gOGKdIUvnLkaNArY-iIcJJSmnEIcMcciTq3C4QA77JnMesofskfJkQo4zvvaxP-eEct-IH_3ostLv1BEpX8UQYZAV-zUyk7WRNztveYVGxTlcE50T_ChhJmi0i1O31qEKlaodSWeInoc3PsmaexunyJ0?purpose=inline)

`crm-read`**&#x20;— Class A / Machine**

Proves contract validation, agent identity, capability authorization, tenant isolation and audit.

First

[Appsavio](https://images.openai.com/static-rsc-4/aNli4V2coKhQWfR_vmzzmigvF_i5ueQcMcthqyvRw990O-swXfr7lfD1Kgv0JrlW0PQbS55GIWnQYhA7rgDaQWlsqXl58g7Nj2xoa0_lUMdHk9nDR77dJSqdu6gqex3ME-aCRwFwU5nSkOb3TXaI9rpFtggd1nqgEu-HWUoRs1E?purpose=inline)

`lead-followup`**&#x20;— Class D / Machine**

Proves durable waits, recovery, approval, provider effects and execution status.

First

[Importing and Exporting Customers & Contacts with a spreadsheet - shopVOX Pro - HelpDocs](https://images.openai.com/static-rsc-4/ykAPEx3L0VuY6Bek5i6BsJtO4SRtV43VQWZBbITA6ROLDxIfrgHhlex4W5d0h-QkEXDnVtGfTUwefc1zpgea9M-JeoJRJqsQlb5SvbRAEY3uvZ6-x68_KeGGX_S34qIymC6Uo-Bu5kzCYWDSH6GztHqkPG84B-OMSPm381hrnIM?purpose=inline)

`contact-import`**&#x20;— Class C / Machine**

Proves River, per-record idempotency, bounded concurrency and resumable batch progress.

Second

[APS + Remote Development Environments | Autodesk Platform Services](https://images.openai.com/static-rsc-4/e_wHG0iIdqp9S0NcB_448_VD1bM7UBMbLgN5Pnwq_8su4pSIOxX3i115dVsd3NUYVw278nxtkz-7SMOxIJVuL-3gOA3H5GU8m_ARCsrJVdbQmd0iZvih8f_8QT3arpziRYTZI9YfX7n91EFF6kHNdyA3gUYoNzc42XWpaPAXRMo?purpose=inline)

`hybrid-task-demo`**&#x20;— Class D + Sprite**

Proves an isolated task can be orchestrated by DBOS without giving up authoritative state or credential isolation.

After Sprite security and recovery proof

I would retain `crm-write`, `notify-send`, `report-generate`, `data-sync`, `approval-flow`, `appointment-journey` and the gated `campaign-orchestration`.

The `agent-toolchain` deserves a separate scope decision. It could be valuable, but it risks expanding the framework into a general-purpose agent execution engine.

The reference catalog should demonstrate the framework's capabilities, not force the framework to absorb every conceivable automation feature.

## **5. Approval assessment**

**Overall architecture**

Approved

The product thesis, execution taxonomy, compute-placement distinction, Go framework approach and Fly/Neon deployment direction are coherent.

**Phase 0**

Approved to proceed

Repository investigation, contract design, dependency verification, security model and technical spikes can begin immediately.

**Phase 1**

Conditional

Proceed once the runtime and packaging model are frozen and the first real recovery tests pass.

**Sprite/hybrid implementation**

Additional gates

Do not implement production Sprite task execution before credential isolation, idempotent resource provisioning and promotion integrity are explicitly designed and tested.

**My verdict: CONDITIONAL GO.**

There are no fundamental architectural reasons to reject this plan. Its core decisions are sound, and its two-dimensional execution/placement model is a meaningful improvement over the earlier proposals.

I do require the following amendments to the plan:

1. Remove direct credentials from untrusted Sprites; use the capability proxy.
2. Define crash-safe, idempotent Sprite task provisioning and cleanup.
3. Define a genuine trusted build and release pipeline from Sprite workspace to Machine deployment.
4. Freeze the Go binary and registry model.
5. Make the whole developer-to-operations lifecycle a tested product capability.

These do not require another architecture redesign. They are refinements to make the current design safe and implementable.

## **6. Recommended disposition**

I would update the plan to record the five amendments and then freeze it as the architecture baseline.

The first three are mandatory before the Sprite/hybrid phase. The fourth should be resolved in Phase 0. The fifth should be reflected in the lifecycle acceptance tests and implemented incrementally.

One final observation: the plan is unusually clear about what it does not know. It admits that the repository is essentially greenfield, that some earlier reliability evidence is unavailable, and that all production dependency versions remain unpinned.&#x20;

plan4.1(5).md

plan4.1(5).md



That honesty is a strength. Keep it. Do not let the next coding agent turn unverified assumptions into architectural facts simply because they appear in a polished plan.

**I approve Phase 0. I would approve the implementation roadmap as a whole once the five amendments are incorporated, with the Sprite and hybrid work remaining explicitly gated by evidence.**


## deepseek

# Adversarial Review: `plan4.1.md` Consolidated Implementation Plan

**Verdict: APPROVED for Phase 0.** No structural blockers. The plan is ready to start Phase 0. Five non-blocking corrections should be resolved before Phase 1 begins. Phase 0 itself is where the corrections can be made, because the corrections are all contract-level or sequencing-level decisions.

---

## 1. What survives adversarial review

**The two-dimension taxonomy (R16) is the strongest decision in the plan.** Separating execution semantics (A/B/C/D) from compute placement (Machine/Sprite/Hybrid/Local) is genuinely necessary, and the plan's insistence that "placement follows the trust profile of the code, not the identity of the caller" (R17) is the correct discipline. Conflating these would have been a fundamental error; the plan avoids it cleanly.

**The reconciliation table (R1–R23) is disciplined.** Each row names the disagreement, the sources, the resolution, and the rationale. The load-bearing seven (R2 effect semantics, R3 wake topology, R4 two principals, R7 Go library, R16 two dimensions, R18 checkpoint scope, R20 SDK/REST split) are correctly identified as the ones that matter most. This is what an ADR-driven plan should look like.

**The dependency register is honest.** Every version is UNPINNED and every pin is deferred to Phase 0 with verification date and source link. The plan explicitly states "`latest` is never a strategy." This is the right posture — it prevents the plan from being invalidated by dependency drift.

**The exit gates are testable.** G0 through G6 each have concrete criteria. Phase 6's acceptance criteria ("wake test passes under the documented recovery model; same contract works across MCP/REST/CLI and both harnesses; server-side approvals hold where harnesses lack elicitation") are the kind of evidence that can be independently verified, not just asserted.

**The risk register (K1–K20) is genuinely self-critical.** K1 (DBOS Go SDK maturity), K3 (wake topology unproven), K4 (pool-handle atomicity), K14 (checkpoint semantics confusion), K15 (Sprite network default-open), K19 (promotion integrity) — these are exactly the risks that would kill the project if unaddressed. The plan names them, maps them to mitigations, and does not soften the language.

**The scope exclusions are load-bearing.** §22 lists AI miniapp generator, unrestricted browser automation, arbitrary LLM code, BPMN/rules engine, rulez, per-miniapp Fly apps, and multi-connector support as out of scope. R11 correctly retires rulez from the product foundation. This is the discipline that prevents the configuration graveyard.

---

## 2. Genuinely weak points (not blockers)

### 2.1 The Sprite dimension's MVP justification is thin

The plan adopts Sprites for two reasons: (a) isolated preview environments in Phase 4, and (b) isolated execution for untrusted/generated code in Phase 5+. The first is a real MVP-adjacent need. The second serves a capability that §22 explicitly defers ("AI miniapp generator," "arbitrary LLM code deployment").

The plan builds the full Dimension B infrastructure (ADR-012, ADR-013, §11.3, `deploy/sprite`, `deploy/tasks`, `deploy/promotion`, `hybrid-task-demo`) for a use case that isn't in the MVP. This is real scope. A tighter plan would defer `deploy/tasks` and `deploy/promotion` to a later phase and keep only `deploy/sprite` for preview environments.

The counter-argument — which the plan makes implicitly but never states — is that Dimension B's interfaces must exist from Phase 0 so the manifest schema, the placement admission check, and the deployment profile abstraction are stable from the start. That is correct. But the *implementation* of `deploy/tasks` and `deploy/promotion` could land in Phase 5 without the intermediate phases needing them. The plan does put them in Phase 5, which is the right phase, but it also puts `deploy/sprite` scaffolding in Phase 0 and the manifest's `deployment` section in the Phase 0 contract freeze. This is defensible but should be named explicitly.

**Recommendation (non-blocking):** In §2 or §22, add one sentence naming the two distinct justifications for Sprites — preview environments (Phase 4, in scope) and untrusted code execution (Phase 5+, deferred from product scope but built as infrastructure). The reader currently has to infer this.

### 2.2 The hybrid-task-demo is the only hybrid reference

§16 has twelve miniapps; exactly one (`hybrid-task-demo`) exercises hybrid placement. The entire Dimension B thesis — "durable workflows may orchestrate bounded Sprite tasks" — is validated by a single demo. If that demo's design has a blind spot (e.g., it never exercises a mid-task timeout with a subsequent retry, or a policy rejection with a subsequent fallback path), the pattern is proven only for the specific path the demo takes.

The §18 test architecture does list the hybrid failure matrix ("task timeout, policy rejection, Sprite loss, and result-schema mismatch"), which is good. But the demo miniapp needs to *exercise all four* of those outcomes, not just the happy path. The plan says this in §18 but §16's description of `hybrid-task-demo` is one line and doesn't commit to it.

**Recommendation (non-blocking):** Expand §16's `hybrid-task-demo` description to state explicitly that it must exercise all four failure modes from §18, or add a second hybrid reference (`hybrid-task-failure-demo`) that does.

### 2.3 Security owner is a prompt requirement not carried into Phase 0

The original implementation-plan prompt (§Phase 0) required "Name a security owner." The plan's Phase 0 tasks list (ADR formalization, contracts, pins, spikes, threat model, RLS context, CLI freeze, branching, CI) does not include naming the security owner. K6 (MCP approval semantics) and K15 (Sprite network default-open) are security-critical and are assigned as risks but not as owned work.

**Recommendation (non-blocking):** Add "Name a security owner who signs the threat model and reviews every capability addition" to Phase 0's task list. This is a one-line fix but it matters — the plan's own G0 exit criterion is "threat model signed off," and without a named signer, "signed off" has no meaning.

### 2.4 The catalog-to-phase mapping is vague

§16 says "Sequence: First (vertical slice)," "Second," "Early," "After first durable slice," "With Phase 5 Sprite work," "Gated on real-provider effect-protocol evidence," "Later." Only the first three map cleanly to phases:

- `crm-read` → Phase 1 (confirmed in §19 Phase 1)
- `lead-followup` → Phase 1 (confirmed in §19 Phase 1)
- `contact-import` → Phase 3 (implied by §19 Phase 3, since it's Class C / River)
- `notify-send`, `report-generate` → Phase 3 (implied)
- `data-sync` → Phase 5 (implied — it's listed in §19 Phase 5)
- `approval-flow` → Phase 4 (implied — approval hardening is Phase 4)
- `appointment-journey` → Phase 5 (implied — listed)
- `hybrid-task-demo` → Phase 5 (confirmed)
- `campaign-orchestration` → Phase 5 (confirmed)
- `agent-toolchain` → deferred (R13, confirmed)
- `crm-write` → Phase 5 (implied)

The mapping is inferable but not explicit. This will cause confusion when Phase 3 starts and someone has to decide whether `data-sync` belongs there or in Phase 5.

**Recommendation (non-blocking):** Add a column to §16's table with the phase number, or reorder §19's phase tasks to explicitly enumerate which §16 entries are in scope for each phase.

### 2.5 Phase 1 claims "callable from Claude Code and Hermes/OpenClaw" but harness verification is Phase 6

§19 Phase 1 exit gate says "agent invokes both tools." §19 Phase 1 acceptance says "callable from Claude Code and from a Hermes/OpenClaw sandbox." But §18 says harness compatibility is tested at the "two-harness compatibility" layer, and §19 Phase 6 says "invoke one SaaS miniapp... from two harness families via MCP plus REST and CLI."

Phase 1 clearly means "we can demo it from two harnesses," not "we've proven harness-independent compatibility." But the wording invites confusion.

**Recommendation (non-blocking):** Clarify in §19 Phase 1 that the exit criterion is "callable from at least one harness end-to-end with audit trail; two-harness proof is Phase 6" — or leave "callable from two harnesses" as Phase 1 but explicitly note that Phase 6 does the *compatibility matrix* (transports, elicitation, task augmentation).

---

## 3. What the plan gets right that most plans get wrong

**The vertical slice comes first.** Phase 1 builds `crm-read` (Class A) and `lead-followup` (Class D) end-to-end via MCP + REST + CLI on test connectors. This proves the thesis before any catalog expansion, Sprite work, or generator work. The single worst failure mode for this kind of product — building an impressive substrate that no one can actually use — is explicitly prevented by the phase ordering.

**DBOS adoption is gated, not assumed.** ADR-001 and the §10.1 table make DBOS the explicit Class D-only substrate. R1 prevents Class C from creeping into DBOS by default. R2 corrects the acceptance language from "zero duplicate effects" to "no duplicate confirmed effects, no silently lost operations." This is the discipline that separates a real plan from an aspirational one.

**The wake test is deferred to Phase 6 but acknowledged as a real risk (K3).** The plan doesn't pretend it's solved. It says "external scheduler only if the baseline fails." That fallback is thin — what does "external scheduler" actually mean operationally? — but the plan is honest that this is unproven.

**R20 (SDK vs REST client split) is honest about SDK gaps.** The plan inspected `sprites-go` v0.2.2, catalogued which areas are covered (Appendix C), and identified the gaps (usage/logs/connectors/privilege). It proposes a framework-owned typed REST client for the gaps. This is the correct posture: use the SDK where it's authoritative, write our own client where it isn't, and wrap both behind interfaces.

**The CLI assessment (R19, ADR-014, §13) applies a clear rule.** "Every command must map to exactly one lifecycle stage in §6.4 and own no policy." One binary, two command groups. This prevents CLI sprawl while keeping the operator experience coherent.

**Prompt truncation is acknowledged and labeled (Appendix D, K20).** The plan doesn't silently fill in the missing sections; it labels them as "architect's completion, needing owner confirmation." This is exactly the right way to handle incomplete requirements.

---

## 4. What will determine Phase 0's success

Phase 0 is the gate. If Phase 0 succeeds, the rest of the plan is feasible. If Phase 0 reveals a problem with the load-bearing spikes, the plan will need real revision. Watch for:

**1. Pool-handle atomicity spike (K4, R9).** The plan says "atomicity depends on pool-handle configuration, not on sharing a URL." This is correct and honest. But the spike must actually verify `RunAsTransaction` (or version equivalent) with a crash probe. If it fails, Class D workflows can't atomically enqueue River jobs or DBOS workflows alongside business data changes, and the transactional-enqueue claim unravels. **This is the single most consequential spike in Phase 0.**

**2. Sprite round-trip spike.** Provision → policy application → exec → teardown → result-in-Neon. If this fails or the API is unstable at the pinned version, Dimension B's implementation path needs revision. The plan's fallback (REST client for gaps) doesn't cover "Sprites unavailable entirely."

**3. Version pinning with evidence.** Every dependency in §20 must be pinned with a verification date and source link. If any pin reveals a version that doesn't have the API the plan assumes (e.g., DBOS Go SDK doesn't expose `Recv` or `RunAsTransaction` at the pinned version), the plan's substrate patterns (§6 shapes) need correction before Phase 1.

**4. Threat model sign-off.** Needs a named owner (see §2.3 above). Without one, G0's "threat model signed off" is unenforceable.

**5. RLS context mechanism design.** The plan says "session variable, roles, `FORCE ROW LEVEL SECURITY`." That is a real pattern but needs concrete design in Phase 0. If the pattern requires session variables that don't survive connection pooling, or requires a specific pool mode, that interacts with K5 (Neon pooled endpoint) in ways that need to be resolved before Phase 2.

If any of these fail, the plan should stop at Phase 0 and revise, not proceed to Phase 1 hoping the failure is recoverable. The plan is structured this way — G0 is a real gate — but the plan should be clearer that Phase 0 *can* produce a "this doesn't work" outcome that requires re-planning.

**Recommendation (non-blocking):** Add a "Phase 0 failure modes" subsection to §19 or §23 stating what happens if each load-bearing spike fails. Specifically: pool-handle atomicity fails → what changes? Sprite round-trip fails → what changes? Version pin reveals API mismatch → what changes?

---

## 5. Approval

**Approved for Phase 0.** The plan can start immediately. No structural blockers.

**Non-blocking corrections to address before Phase 1:**

| # | Correction | Where |
|---|---|---|
| 1 | Add explicit note on Sprite dimension's two justifications (preview in-scope, untrusted code deferred) | §2 or §22 |
| 2 | Expand `hybrid-task-demo` to commit to exercising all four failure modes, or add a second hybrid demo | §16 |
| 3 | Add "Name security owner" to Phase 0 task list | §19 Phase 0 |
| 4 | Add phase column to §16's catalog table, or enumerate each §16 entry in its owning phase in §19 | §16 and §19 |
| 5 | Clarify Phase 1's harness scope (one harness in Phase 1, two-harness matrix in Phase 6) | §19 Phase 1 |
| 6 | Add "Phase 0 failure modes" subsection naming what changes if each load-bearing spike fails | §19 or §23 |

None of these change the plan's architecture. All six can be resolved in Phase 0 alongside the contract freezes and spikes. The plan is structurally sound and ready to execute.

---

## 6. What to watch in Phase 1

Once Phase 0 clears G0, the plan proceeds. The next adversarial checkpoint is **G1 (vertical slice on real Postgres)**. Specifically:

- Does `lead-followup` survive a mid-run machine stop and resume from the correct step *with no duplicate confirmed effects*? This is the load-bearing Class D claim.
- Does the audit trail from a MCP tool call → DBOS workflow → external capability → response allow an operator to answer "what did agent X do on behalf of user Y in tenant Z on date D?" If not, the audit design (§8) needs correction.
- Does the CLI's `init`/`validate`/`test` actually scaffold and check the Phase 1 miniapps themselves, or does the plan dogfood the lifecycle only superficially? §19 Phase 1 says "the slice dogfoods the lifecycle from day one" — verify this claim.

If G1 clears, the plan is on solid ground. If G1 fails, the failure mode will be informative — it will point to either a substrate problem (DBOS, River, or the transaction boundary) or a contract problem (`miniapp.json`, tool descriptors, run envelope). Both are correctable without re-architecting, because Phase 0's contracts are the right contracts.

---

## Summary

**Verdict:** Approved for Phase 0. No structural blockers. Six non-blocking corrections, all resolvable in Phase 0. The plan's strongest features are its two-dimension taxonomy, its DBOS-adoption discipline, its vertical-slice-first sequencing, and its willingness to name its own weaknesses. Its weakest features are the thin MVP justification for the Sprite dimension and the single hybrid reference miniapp. Neither is fatal, and both are addressable with documentation additions rather than architectural changes.

**Proceed to Phase 0.**


## gemini

### Adversarial Review: `plan4.1.md`

**Verdict: APPROVED WITH CONDITIONS (No Structural Blockers for Phase 0).**

The architecture in `plan4.1.md` is coherent, honest about its assumptions, and structurally sound for starting Phase 0. Separating **Dimension A (execution semantics: Class A/B/C/D)** from **Dimension B (compute placement: Machine/Sprite/Hybrid/Local)** prevents the most dangerous failure mode of mixing durable workflows with isolated sandboxes. Treating Sprite checkpoints strictly as filesystem snapshots rather than workflow state, enforcing network policy before executing untrusted code, and keeping `miniapp.json` + typed Go contracts as the governed surface give the repository a clean foundation.

That said, an adversarial pass surfaces **real operational hazards and design seams** that must be tightened during Phase 0–2 so they do not turn into structural blockers later.

---

### 1. Structural & Architectural Stress-Tests

#### A. Hybrid DBOS + Sprite Execution Across Crash & Timeout Windows

* **The Seam:** §11.3 states a DBOS step calls `deploy.RunTask`, which provisions/reuses a Sprite, stages inputs, applies network policy, runs a bounded command, and copies results to Neon.


* **Adversarial Failure Mode:** What happens if a Sprite task *succeeds* externally or writes partial artifacts, and the `RunTask` step or result-copy step crashes before DBOS records step completion? Or what if a Sprite task outlives the DBOS step's execution timeout while the Sprite itself is still running?
* **Condition for Phase 0/1:** Make `RunTask` idempotent on `TaskID` (`tenant:workflow:step:attempt`), store task execution state (`STAGED` → `RUNNING` → `RESULT_PERSISTED`) in Neon *outside* ephemeral Sprite memory, and ensure `CancelTask` / lease expiration tears down orphaned Sprites so leaked sandboxes do not silently burn quota or leave live network access.

#### B. Network Policy Race Condition ("Starts Open")

* **The Seam:** §14 notes that outbound networking on Sprites starts open and the framework applies a restrictive network policy before running agent-supplied code.


* **Adversarial Failure Mode:** If provisioning, policy application, and command execution are separate API calls without a hard state barrier or atomic pre-exec gate, a race or partial failure could expose an unguarded workspace or run code under a stale policy.
* **Condition for Phase 0/1:** Enforce a strict state-machine gate in `deploy/sprite` (`POLICY_CONFIRMED` required before `CommandContext` / exec starts) and verify in Phase 0/6 that unconfirmed policy states fail closed and never execute commands.



#### C. Sprite Credential Boundary vs. Outbound API Calls

* **The Seam:** §14 allows "short-lived downscoped tokens for explicitly listed endpoints — resolved at dispatch, never tenant master credentials" when the manifest permits.


* **Adversarial Failure Mode:** If a Sprite receives a bearer token directly in environment variables or staged files, untrusted/agent-generated code inside the Sprite could exfiltrate that token to an allowlisted or compromised endpoint.
* **Condition for Phase 2:** Prefer **brokered/proxied egress** (where the Sprite talks to a controlled gateway/proxy that injects credentials at the boundary) wherever feasible, or strictly scope token TTL, audience, and network egress rules together, and audit/redact any secret injection path.



#### D. Single-Executor DBOS Bottleneck & Scale-to-Zero Tension

* **The Seam:** §17 specifies a single dedicated `dbos` Machine kept always-on until the Phase 6 wake test proves otherwise.


* **Adversarial Failure Mode:** A single always-on DBOS Machine is a single point of availability for Class D workflows during Machine replacements, deploys, or traffic spikes. Conversely, if `dbos` is kept always-on, cost optimization from idle scale-to-zero applies to River/workloads rather than DBOS unless proven safe.


* **Condition for Phase 0/6:** Keep the single-executor baseline for simplicity and deterministic recovery in early phases, but explicitly test rolling restarts/drain behavior and define when a second standby/worker or Conductor becomes mandatory.



#### E. Truncated `prompt2.md` & Unpinned Dependencies

* **The Seam:** `prompt2.md` is truncated at §7.1 (Appendix D), and all external dependency versions in §20 are currently `UNPINNED`.


* **Adversarial Failure Mode:** Building schema or CLI contracts before locking exact SDK versions (`dbos-transact-golang`, `river`, MCP Go SDK, `sprites-go`) risks rework if method signatures, transaction hooks, or context types differ.
* **Condition for Phase 0:** Treat pinning exact versions + verification dates + source links + running spikes as a hard prerequisite to writing production adapters in Phase 1.



---

### 2. Minor Plan Hygiene & Consistency Notes

1. **CLI Command Count vs. "No-Proliferation" Rule:** §13 proposes 10 lifecycle items (12 if `status`, `runs`, `logs` are separate subcommands) on top of the `invoke` group. Freeze a tight, minimal command set in Phase 0 and enforce the rule that every subcommand maps cleanly to one lifecycle stage without duplicating existing `fly` or `psql` workflows.


2. **Generated Code vs. Trusted Runtime:** §2 prohibits unreviewed LLM-generated Go/shell/JS inside the trusted production runtime, routing untrusted code to Sprites and governed promotion to Machines. Ensure `publish` and `promote` in §8/§13/§19 enforce static analysis, dependency lock verification, fixture execution, and human/policy approval before any Sprite-derived artifact can be promoted to a Machine target.


3. **Neon Schema & Migration Ownership:** §7 and §12 place `app`, `river`, and `dbos` in Neon while noting migrations for DBOS/River come via pins. Phase 0 should explicitly document how schema migrations for `app` vs. library-managed tables are ordered and gated in CI/CD so deploys never race schema changes.



---

### 3. Approval & Recommended Next Step

**Phase 0 is approved to begin** under the conditions above. There are no structural blockers preventing contract definition, ADR formalization (`ADR-001` through `ADR-015`), dependency pinning, or integration spikes.

To lock down the start of Phase 0 cleanly, which of these two items do you want to tackle first:

1. **Confirm the open decisions in Appendix B** (specifically first SaaS integration, target harnesses, DBOS Enterprise stance, and whether `prompt2.md` has a missing remainder), **or**


2. **Draft the Phase 0 contract & ADR baseline** (`miniapp.json` schema, Dimension A × B admission rules, `SpriteTask` / promotion records, and the Phase 0 spike checklist)?


## Z

# Adversarial Review — plan4.1.md

**Reviewed document:** `Consolidated Implementation Plan v2 — Go Miniapp Framework, References, and Fly Deployment Platform` (`upload/plan4.1.md`, 975 lines)
**Review type:** Adversarial structural review (attack the plan; find what breaks, not what pleases)
**Reviewer basis:** Full read of plan4.1.md; ADR-001 (`docs/DBOS.md`) as governing policy; rea.md patterns; external corroboration of stack claims (§6 of this review)
**Date:** 2026-11-22

---

## Verdict

> **APPROVED — no architecture-level structural blockers.**
> Two required plan-text revisions (**B1, B2**) must be folded into the plan before Phase 4–6 work is scheduled. Neither affects the architecture, the security model, or Phase 0. **Phase 0 may start immediately**, exactly as §23 states.

The architecture holds under attack: the two-dimension model (execution class × compute placement) is coherent and consistently applied across §10, §16, and §17; ADR-001 remains mandatory and enforced; the security perimeter fails closed; the acceptance language is honest about at-least-once effects; and every load-bearing unknown is assigned to a falsifiable gate with dated evidence. The two findings below are **completeness defects in the plan text** — an unowned adapter pair and a hole in the published-artifact immutability guarantee — not design contradictions.

**What would flip this verdict** (all already gated, none currently true): prompt2's missing remainder contradicting Appendix D on placement/security rules; Phase 0 spikes falsifying K4/K5 (DBOS/app atomicity, pooled endpoints) in a way that breaks §12; sprites-go at the pinned version lacking policy-before-exec capability.

---

## 1. Required plan revisions (must fix — structural)

### B1 — `deploy/machine/` and the `deploy/sprite/` workspace adapter are owned by no phase

**Location:** §7 (layout), §6.2 (dependency diagram), §8 (workspace API), §19 (phase packages) vs §13/§19 Phase 4–6 tasks.

**Attack:** Follow the roadmap as written. Phase 0 builds `deploy/` as *profiles + interfaces* only. Phase 1–3 own no `deploy/` packages. Phase 4 owns `deploy/` only for *preview, publish*. Phase 5 owns `deploy/` only for *tasks, promotion, REST gap client*. Phase 6 builds `fly/` (fly.toml config) and "deploys the shared runtime." Result:

- **`deploy/machine/`** (the Fly Machines target adapter, §7) is never assigned an implementing phase, yet Phase 6's deployment of the gateway/river/dbos runtime requires it.
- **`deploy/sprite/`** workspace adapter (`ProvisionWorkspace` / `Snapshot` / `Restore` / `Decommission`, §8) is likewise unowned, yet **Phase 4's `preview` command ("disposable Neon branch + Sprite preview") cannot be delivered without it** — G4 is unsatisfiable as written.

**Required fix:** Name the owners. E.g., add `deploy/sprite` (workspace adapter) to Phase 4 packages and `deploy/machine` to Phase 4 or Phase 5, with Phase 6 consuming both. Also state explicitly that Phase 4 deliberately introduces Sprite *workspaces* before Phase 5's Sprite *tasks* — §19's preamble ("Sprite integration enters at the phases where their dependencies are proven") implies this ordering is intentional; make it explicit so a coding agent executing Phase 4 does not stall or improvise.

### B2 — Published-manifest immutability is bypassable through referenced deployment profiles

**Location:** §9.1 ("Immutable once published"), §9.6 (promotion record), §14 (network policy), §17 ("network-policy templates per profile").

**Attack:** The manifest binds placement and Sprite policy **by name**; the promotion record binds manifest hash + dependency lock + test results + provenance — but **not the content of the referenced profiles**. Sequence: publish v1 with profile `agent-sandbox` (allowlist A) → review and promote → edit the named profile (allowlist B) → the promotion record still validates, the manifest hash is unchanged, and production now executes under a network policy **that was never reviewed**. The immutability guarantee — the whole point of R22/K19 — is bypassed without touching the manifest, and §14's enforcement story inherits the drift.

**Required fix (lands in G0 contract freeze):** Content-address deployment profiles at publish time: hash each referenced profile into the manifest at `publish`; the promotion record embeds those hashes; `validate` refuses any manifest carrying an unbound or stale profile reference; profile edits force a new manifest version. One contract clause plus one admission rule.

---

## 2. Concerns (fix during execution; none block approval)

### C1 — Orphaned Sprite tasks: the inverse of the handled failure (high priority)
§11.3 proves "a lost Sprite never strands the workflow" — the workflow side is handled. The **platform side** is not: if the task runner dies between `RUNNING` and result-copy, the task row is stuck `RUNNING` and the Sprite keeps running and billing (§9.6 has no transition for this; §14 leases expire *identity*, not the task; §17 alerts on lease violations but specifies no resolution). **Fix:** task-level lease + reaper that decommissions the workspace and marks the result `UNKNOWN`, letting the orchestrator retry with a new task ID per §11.3; add a §18 test row "runner killed mid-task."

### C2 — Downscoped Sprite tokens lack explicit limits (medium)
§14 says "short-lived downscoped tokens for explicitly listed endpoints." Tighten before Phase 2's design: TTL, minimal scope ⊆ manifest allowlist, single-use or per-task budget/rate caps, per-task issuance audit, and an allowlist-quality rule (no generic domains that double as exfiltration destinations). The pieces exist; make them mandatory and testable.

### C3 — Retention policy has no owner (medium)
R14/§9.4 defer "terminal-history retention" to a "separate policy," and §7 gives `audit/` a retention responsibility — but no phase task ever *defines* retention for runs, effects, audit events, Sprite-task records, or promotion records. Unbounded run history under R14's "no handle expiry" is a slow structural leak. **Fix:** assign the retention policy as a named artifact of Phase 2 or 4, surfaced in the §17 operate view.

### C4 — DBOS executor HA posture is underspecified (medium)
ADR-008 mandates a "single dedicated executor, always available." The wake test (G6) covers waits, not executor *loss*. Specify: Fly restart-on-exit / health-check replacement for the `dbos` process group, an **executor-down alert** (missing from §17's alert list), and stuck-workflow detection with recovery ownership per ADR-001 §7.7. "Always available" must not silently mean "available after someone notices."

### C5 — K5 asserts an unverified mechanism at High impact (low)
"Neon pooled endpoint incompatible with DBOS (`LISTEN/NOTIFY`)" — the mechanism claim is unverified in the plan and by this review. The direct-connection default is safe regardless, but the Phase 0 spike should actually test whether the DBOS Go SDK tolerates pooled endpoints; if it is polling-only, §12's connection budgets simplify materially.

---

## 3. Non-blocking notes

- **N1 — Carried-forward content is unverifiable here.** plan4.md, Review 12, and OpenLevel code are absent from the workspace (§4 admits this, K12 mandates re-derivation). Treat the ADR-001..010 and K1–K13 numbering as provisional until the Phase 0 ADR files exist.
- **N2 — Appendix D is good practice; make it bind on G0.** The labeled-completions treatment of the truncated prompt2 (K20, Q8) is exemplary. Add "contracts frozen subject to Appendix B Q8 resolution" to the G0 exit wording so a late remainder cannot silently reopen frozen contracts mid-phase.
- **N3 — §16 `agent-toolchain` placement "Assessed"** should default-resolve to a §10.2 row (trusted Go code → Machine) with hybrid as the exception path, so admission never has a dangling cell.
- **N4 — Cosmetic:** §8's `TaskStatus(ctx, taskID) (TaskStatus, error)` shadows the type name; irrelevant now, but the Phase 0 signature freeze should clean it.

---

## 4. Attacks the plan survives (verification of strengths)

| Attack | Where it should have broken | What held |
|---|---|---|
| "DBOS everywhere" creep | §10, §16 | ADR-001 adopted verbatim, executor admission check, rubric-gated C→D promotion; R13 defers `agent-toolchain` |
| "Zero duplicate external effects" overclaim | §15, §18, G-acceptance | **R2's corrected language** matches the provider-complete/checkpoint-missing crash window — the single most important honesty fix in the plan |
| Scale-to-zero during durable waits | §17, §19 P6 | R3/K3: always-on baseline; wake topology demoted to a Phase 6 falsification test with external-scheduler fallback |
| Agent identity ⇒ Sprite per tool call | §10.2 | R17: placement follows code trust, not caller identity; per-call Sprites correctly rejected |
| Sprite checkpoint mistaken for workflow durability | §11.3, §18, K14 | R18 + dedicated checkpoint-semantics tests ("restore returns files only") |
| Default-open Sprite networking | §14, K15 | R21: policy-before-exec with read-back confirmation; task never starts unconfirmed; verify-default-open-then-lock drill in Phase 6 |
| Silent promotion of unreviewed code to production | §9.6, §14, K19 | Promotion record with manifest hash, dependency lock, provenance, review gate; refusal tests — **held, modulo B2's profile binding** |
| Duplicate workflow on agent retry / webhook replay | §6.3, §9.4, §11.1 | Stable operation IDs as idempotency keys; unique River jobs; no arbitrary handle expiry (R14) |
| Conflating execution class with compute placement | §1, §10 | R16/ADR-011: two independent dimensions; §10.2 matrix consistent with §16 catalog and §17 topology (checked row by row) |
| Evidence by anecdote | §3, §20 | R10 kills the 150ms claim (measure instead); §20 all-UNPINNED with Phase 0 pinning + dated verification; Appendix C inspected file-level evidence with re-verify-at-pin discipline |
| Truncated governing input poisoning the plan | §4, Appendix D, K20 | Labeled architect completions + owner question Q8; nothing smuggled in as fact |
| Hybrid flow strands the workflow on Sprite loss | §11.3 | Typed failure mapping (timeout / policy rejection / Sprite loss / schema mismatch), Neon-held authoritative results, new-task-ID retries |
| ADR-001 §7 obligations dropped under new complexity | §11.2, §17 | Determinism, identity, versioning + deployment guard, payload discipline, recovery policy all restated and test-bound (§18) |
| Phase/gate inconsistencies | §16 vs §19 | Catalog sequence cross-checked against phases: crm-read/lead-followup→P1, contact-import→P3, approval-flow→P4, hybrid→P5, campaign gated on K8 evidence — consistent |

---

## 5. External corroboration performed by this review

| Claim in the plan | Result |
|---|---|
| DBOS Transact Go SDK (§20, K1) | **Confirmed** — official repo `github.com/dbos-inc/dbos-transact-golang`, v1.x actively maintained through 2026 |
| River as Postgres-backed Go queue (§11.1, §20) | **Confirmed** — `github.com/riverqueue/river` / riverqueue.com; transactional enqueue, uniqueness, retries, scheduling |
| Fly Sprites + sprites-go (Appendix C, K16–K17) | **Confirmed at product level** — `github.com/superfly/sprites-go` ("computers for agents", hardware-isolated Linux machines) and fly.io/sprites ecosystem exist; a Jan 2026 design post matches the plan's characterization. Appendix C's file-level coverage claims remain the plan's own inspected evidence — the re-verify-at-pin requirement is the right control and stays mandatory |
| Go 1.26.0+ toolchain floor (§20) | Plausible for 2026-11; consistent with sprites-go's go.mod per Appendix C |

---

## 6. Reviewer's closing assessment

The plan's load-bearing quality is that **nothing consequential is claimed without a gate that can falsify it**: wake topology (G6), atomicity (G0 spikes), provider effect semantics (K8/G4), SDK coverage (K17/R20). Its load-bearing weakness was incompleteness, not design: two adapters without a phase owner (B1) and an immutability guarantee with a side door (B2). Both are small, precise revisions; both are specified above; neither reopens any decision in §3.

**Approved.** Apply B1 and B2 to the plan text, fold C1–C5 into their named phases, and start Phase 0.
