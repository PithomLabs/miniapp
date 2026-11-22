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