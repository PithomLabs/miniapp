I reviewed all six uploaded architecture documents. **The package is substantial, internally well organized, and captures the intended Miniapp architecture.** Most importantly, it treats Security, Observability, and Developer Experience as three coordinated aspects of one runtime rather than separate libraries developers must assemble.

However, I found several cross-document inconsistencies that I would correct before formally approving F-0 and authorizing implementation.

Targeted revision recommended before F-0 approval

This is not a rejection or a request for architectural redesign.

## **1. Corrections required**

### **A. Exactly-once guarantees are overstated**

Two statements need correction:

- The unified architecture says the kernel dispatches exactly once per invocation.&#x20;

  foundational-architecture.md

- The Workflow Kit says transactions and the effect ledger confer exactly-once business semantics.&#x20;

  developer-experience-enforcement.md



These statements are too broad given Miniapp's actual recovery model, including at-least-once execution, attempt fencing, and UNKNOWN quarantine.

Specify the distinctions explicitly:

- One logical invocation and stable business-operation identity.
- Durable, idempotent dispatch intent.
- Potentially repeated job delivery and worker execution.
- At-most-one committed transactional business execution within its defined boundary.
- External effects protected by authorization, ledger, attempt fencing, provider idempotency where available, and UNKNOWN quarantine.

Never promise exactly-once arbitrary external provider delivery.

### **B. Separate audit from telemetry health events**

The Observability Contract adds `telemetry-degraded` and `exporter-down/recovered` to the **audit action vocabulary**.&#x20;

observability-contract.md



This conflicts with the same contract's principle that audit records authoritative business/security transitions and that exporter failures must not affect the business path.&#x20;

observability-contract.md

observability-contract.md



My recommendation:

- Keep exporter health in structured logs and metrics.
- Keep durable audit for security-relevant and consequential state transitions.
- Define explicitly which retries, worker pickups, and recovery events merit durable audit records versus operational telemetry.

We must not accidentally turn PostgreSQL into a high-volume operational logging system.

### **C. Correct the sampling architecture**

The Observability Contract proposes head sampling for routine spans and Collector-side tail sampling to retain all error, recovery, and UNKNOWN paths.&#x20;

observability-contract.md



There is a technical problem: a Collector cannot retain an error trace whose spans were already discarded by the SDK's head sampler.

Define a compatible strategy. Either preserve candidate traces for Collector-side tail decisions or explicitly accept that some errors may be sampled out.

Do not promise that all error traces will be retained without an architecture that actually guarantees it.

### **D. Fix roadmap-to-conformance traceability**

There are two issues here.

First, the roadmap's F-1 exit criterion requires Security acceptance criteria 1–5 on A/D, but Security criterion 3 requires B/C/D worker-checkpoint proof. B and C do not exist until F-5 and F-6.&#x20;

foundational-roadmap.md



Split the criterion into its A/D obligations for F-1 and B/C obligations for their respective adapter stages.

Second, the Developer Experience Contract requires three distinct Class D fixtures to reuse the Workflow Kit, and F-7 includes that proof. But the conformance matrix does not give that specific requirement its own identifiable conformance row. Its CDX-05 currently refers to the adversarial demo.&#x20;

developer-experience-enforcement.md

foundational-conformance-matrix.md

foundational-roadmap.md



Add an explicit Workflow Kit reuse conformance requirement.

## **2. Additional decisions to clarify**

These are not reasons to redesign the package, but they matter before implementation.

**Delegation revocation semantics.** Security says delegation revocation is immediate, but Class B checks at worker pickup while Class D checks before consequential effects. Define whether an already-running B job may continue after revocation and how the effect boundary handles that situation without violating C1–C7.&#x20;

security-contract.md

security-contract.md



**Milestone naming.** The roadmap uses M1–M12 for new milestones while also referring to existing lifecycle M1–M6 and the M5 sequence-verification requirement. Prefix new milestone identifiers, for example `F-M01` through `F-M12`.&#x20;

foundational-roadmap.md

foundational-roadmap.md



**Go enforcement boundary.** DX correctly acknowledges that trusted Go code can bypass a library boundary, but the demo matrix says a direct provider call has no runtime path. Clarify that it has no *supported framework-mediated path* and that static analysis detects known violations. Hostile in-process code requires actual isolation.&#x20;

developer-experience-enforcement.md

developer-experience-enforcement.md



## **3. What I approve**

I strongly endorse the core architecture:

- The Security Contract's separation of identity issuance from authorization enforcement.
- The common Runtime Kernel and Business Capability proposal, with honest recognition of existing registry and transaction constraints.
- The four-pillar Observability model and producer-not-platform principle.
- The framework-owned conformance suite as a production-readiness gate.
- The separation of A/B/C/D execution semantics.
- The explicit carry-forward of P31-001, P29-005, the revocation drill, M5, and P24-008.

The inventory is particularly important: it distinguishes implemented, partial, documented, and absent capabilities rather than presenting proposed architecture as existing functionality.&#x20;

foundational-architecture.md



## **4. Recommended next action**

I recommend a small **Prompt32.1 — Foundational Architecture Reconciliation** work order.

Its scope should be only:

1. Correct exactly-once terminology.
2. Separate audit records from telemetry-pipeline health.
3. Resolve the sampling contradiction.
4. Align roadmap exit criteria with actual conformance coverage.
5. Specify delegation revocation checkpoints.
6. Resolve milestone naming and Go-boundary wording.
7. Re-run cross-document reference and requirement-coverage audits.

No implementation, source modifications, migrations, or dependency changes.

**My overall assessment:** The architectural direction is sound. The package is close to owner-approval quality, but these corrections matter because they concern guarantees the framework will eventually make to developers and operators.

Once reconciled, I would move to F-0 approval, then authorize F-1 and F-2 as coordinated foundational implementation workstreams. Class B and C should remain behind the shared conformance gates.