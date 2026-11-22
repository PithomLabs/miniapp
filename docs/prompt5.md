I recommend making this a **read-only, end-to-end adversarial review**, not another implementation exercise. The reviewer should challenge the actual code, configuration, tests, and operational assumptions, then give you a defensible go/no-go recommendation before proceeding to the next development phase.

One discrepancy is worth explicitly checking: the reported development Taskfile has 17 tasks, but the enumerated inventory currently accounts for 16. The review should reconcile that against the actual `task --list` output rather than inventing another task.

# Prompt 5 — Adversarial Architecture and Implementation Review

## Miniapp Framework: Phase 0, Taskfile, Docker PostgreSQL, Fly Secrets, and Business Execution Categories

### 1. Role and objective

Act as an independent principal software architect, senior Go engineer, infrastructure security reviewer, and production-readiness auditor.

Conduct a rigorous adversarial review of the implementation in:

`/home/chaschel/Documents/go/miniapp`

The governing architecture baseline is:

`docs/plan4.2.md`

Your objective is to determine whether the implementation is sufficiently correct, secure, coherent, and operationally complete to proceed to the next implementation phase.

Review the actual repository, not previous implementation reports or claims.

The review must cover the entire implementation journey:

1. Phase 0 contracts, DBOS/River/Neon architecture, and evidence.
2. Go configuration and persistence boundaries.
3. Miniapp execution-category design and developer guidance.
4. Separate development and production Taskfiles.
5. Local Docker PostgreSQL environment.
6. Environment-specific secret delivery.
7. Fly development and production configuration.
8. Security, authorization, deployment, and operational safety.
9. Documentation, regression tests, and developer experience.

This is an adversarial review, not a request to implement another architecture.

**Do not modify source files, configuration, Taskfiles, documentation, migrations, credentials, cloud resources, or Git history.**

Produce findings and a prioritized corrective-action plan. Implementation of identified corrections requires a separate authorization.

---

### 2. Governing rules

- `docs/plan4.2.md` is authoritative.
- Do not reopen or rewrite the frozen baseline.
- Do not silently substitute earlier plan4.1 requirements for plan4.2.
- Preserve ADR numbering and approved decisions.
- Do not introduce new dependencies or architecture merely because an alternative exists.
- Do not treat documentation assertions as proof of implementation.
- Do not treat compilation as proof of runtime behavior.
- Do not treat skipped tests as passes.
- Do not assume an existing cloud resource is correctly configured simply because it exists.
- Do not invent credentials, infrastructure state, owner decisions, or test results.
- Do not weaken security controls to produce green test output.
- Do not commit changes.
- Do not implement Phase 1 functionality during this review.

When the implementation differs from plan4.2, determine whether the difference is a defect, a justified implementation choice, or an explicitly deferred requirement.

---

### 3. Repository reconnaissance and evidence integrity

Inspect:

- `AGENTS.md` and repository instructions.
- `docs/plan4.2.md`.
- ADR-001 through ADR-017.
- `contract/`.
- `registry/`, `runtime/`, `executor/`, and other existing Go packages.
- `internal/config/`.
- `internal/store/`.
- `deploy/`.
- `miniapps/`, if present.
- `spikes/`.
- `infra/`.
- `migrations/`.
- `scripts/`.
- `Taskfile.dev.yml`.
- `Taskfile.prod.yml`.
- `compose.yaml`.
- `compose.host.yaml`.
- Dockerfile and `.dockerignore`.
- `.gitignore`.
- `docs/phase0/`.
- `docs/evidence/`.
- All development, security, deployment, and execution-category guides.
- CI workflows.
- Test fixtures and generated JSON Schemas.

Establish the actual repository state before making judgments.

Run read-only inspection of Git status and the working diff. The repository was initialized but no commit was authorized. Do not stage or commit anything.

Verify the reported implementation:

- 17 ADRs.
- 11 generated JSON Schemas.
- Separate development and production Taskfiles.
- Local PostgreSQL 19.0 implementation.
- Seven development/Fly scripts, subject to actual inventory.
- Go `_FILE` configuration contract.
- Regression and infrastructure tests.
- Phase 0 spike harnesses.
- Execution-category developer guide.
- Secret-management and environment documentation.

Record discrepancies rather than assuming the report is accurate.

**Inventory arithmetic check:** The reported development Taskfile lists:

- `doctor`
- Six `dev:db:*` tasks
- Two `dev:*` tasks
- `test`
- `test:integration`
- Five `fly:*` tasks

That list totals sixteen named tasks, despite the report claiming seventeen. Inspect actual Taskfile parsing and task inventory. Identify the discrepancy without inventing an extra task.

---

### 4. Review A — Architecture and Phase 0 correctness

Review the existing Phase 0 contracts and implementation against plan4.2.

#### 4.1 Execution classification and contracts

Verify that the contract package correctly represents:

- Class A, B, C, and D execution semantics.
- Independent compute placement.
- Manifest identity and immutable version.
- Static registration and implementation identity.
- `entityKey` and explicit `noEntityReason`.
- Capability scope and attribution.
- Blast-radius limits.
- Typed input and output contracts.
- Run, effect, approval, and task records.
- Release-envelope integrity.
- Content-addressed deployment profiles.
- Non-circular artifact matching.

Investigate whether the implementation validates the invariants or merely exposes corresponding struct fields.

Test for:

- Invalid categories.
- Missing entity identity on consequential writes.
- Invalid no-entity justifications.
- Missing or excessive limits.
- Manifest/implementation mismatch.
- Incorrect artifact digest.
- Mutable profile references.
- Release-envelope tampering.
- Duplicate or conflicting identities.

Check that JSON Schemas are generated consistently and that no manual schema drift exists.

#### 4.2 ADR-001 and DBOS

Inspect the Go-specific ADR-001 port.

Verify:

- The original DBOS decision logic has been translated accurately into the Go/PostgreSQL/River architecture.
- Per-miniapp Class C/D decision records use the intended template.
- Class C River-versus-DBOS comparison is a genuine workload decision, not a predetermined result.
- Class D adoption is supported by a real durability and orchestration requirement.
- Human architectural judgment is not falsely represented as an automatically enforceable code rule.

Inspect the DBOS family/version isolation strategy:

- Distinct application identity for workflow families.
- Explicit `AppName`.
- Explicit workflow `ApplicationVersion`.
- Stable workflow IDs.
- Version compatibility and patching rules.
- Old workflow-version retention.
- Deployment and drain behavior.
- Prevention of incompatible replay.
- Schema and executor ownership boundaries.

Challenge the assumption that independently versioned manifests automatically imply independently versioned workflow implementations.

Review the family-isolation spike and distinguish the evidence it actually proves from the properties that still require a Fly deployment test.

#### 4.3 Determinism analyzer

Inspect the proposed or implemented determinism analyzer.

Verify that it detects the actual prohibited patterns in DBOS workflow control code, including:

- `time.Now`.
- Uncontrolled random sources.
- Uncontrolled goroutines.
- Nondeterministic `select`.
- Nondeterministic map iteration.
- Direct external I/O in workflow control flow.
- Forbidden DBOS calls inside durable step bodies.

Check for false negatives involving helper functions, aliases, package-qualified calls, and indirect invocation.

Do not claim complete determinism enforcement from a simple textual grep.

Determine whether the analyzer is implemented, tested, wired into CI, or still only documented.

#### 4.4 Effect and approval correctness

Verify:

- The effect state machine rejects illegal transitions.
- Stable idempotency keys are enforced.
- UNKNOWN is not treated as FAILED.
- Unknown outcomes are not blindly retried.
- Provider reconciliation has an explicit owner.
- Compensation is registered before the forward effect.
- UNKNOWN effects do not trigger unsafe automatic compensation.
- Approval decisions bind to canonical material inputs.
- Approval expires or invalidates when relevant inputs change.
- An agent cannot approve its own consequential action.
- Approval authority is independent from agent authority.

Check the actual transaction boundaries, not only the data structures.

#### 4.5 PostgreSQL tenant isolation

Review `internal/store.WithTenant` and its consumers.

Verify:

- `SET LOCAL` or equivalent transaction-local context is applied within the same transaction used for tenant queries.
- The helper cannot accidentally execute tenant queries outside the transaction.
- Context does not leak across pooled connections.
- Missing tenant context fails closed.
- RLS policies correctly handle absent or invalid tenant context.
- Application roles cannot bypass policies through superuser, table-owner, or `BYPASSRLS` privileges.
- Concurrent tenants cannot observe each other's data.
- Tests exercise the actual roles used by the application.

The previously discovered RLS superuser-bypass and autocommit defects must have regression tests that genuinely reproduce the failure mode.

---

### 5. Review B — Business execution-category guidance

Inspect the developer-facing execution-category guide and verify that it is consistent with plan4.2 and ADR-001.

The guide must clearly distinguish:

| Category | Business execution requirement        | Default mechanism                   |
| -------- | ------------------------------------- | ----------------------------------- |
| A        | Ordinary request-bound operation      | Go handler + PostgreSQL transaction |
| B        | One-shot independently retryable job  | River                               |
| C        | Bounded batch or multi-step operation | River or DBOS, decided per workload |
| D        | Durable business workflow             | DBOS                                |

Challenge both under-engineering and over-engineering.

#### Required questions

**Class A**

- Does the guide preserve synchronous result delivery?
- Does it distinguish request-bound work from external-effect safety?
- Does it prohibit unnecessary queues and workflows?

**Class B**

- Does it distinguish a retryable job from a durable multi-stage workflow?
- Are uniqueness and idempotency treated as separate concerns?
- Are terminal failures observable?

**Class C**

- Does it avoid automatically selecting River?
- Does it require a meaningful River-versus-DBOS comparison?
- Does it explain per-record progress, resume, bounded concurrency, and provider rate limits?
- Does it avoid creating a shadow workflow engine?

**Class D**

- Does it require a genuine durable business promise?
- Does it explain deterministic replay and DBOS durable boundaries?
- Does it account for AppName and workflow-version isolation?
- Does it explain external-effect and recovery responsibilities that DBOS does not solve automatically?

Verify that approval, Saga, entity serialization, idempotency, reconciliation, and blast-radius control are described as cross-cutting patterns rather than additional execution categories.

Verify that compute placement is explicitly independent from execution category.

Check that examples organize miniapps around business outcomes rather than one miniapp per provider API endpoint.

Check that developers are told when NOT to use each category, not just when to use it.

Flag any misleading example, contradiction, or statement that encourages developers to introduce unnecessary infrastructure.

---

### 6. Review C — Two Taskfiles and environment isolation

Review `Taskfile.dev.yml` and `Taskfile.prod.yml` independently.

Verify:

- Both parse using the pinned Task version.
- The `--taskfile` invocation convention works.
- Each displays its exact intended task inventory.
- Production does not import development tasks.
- Development does not import production tasks.
- Shared scripts do not silently introduce a cross-environment execution path.
- There is no implicit production default.
- No task interpolates or prints secret values.
- No task hides a deployment or destructive mutation under a harmless name.
- All referenced scripts and configuration files exist.
- Every declared Task variable is used safely.
- Dry-run output is safe and does not expose secret material.

Inspect the development target:

`miniapp-demo`

Inspect the production target:

`miniapp-prod`

The existence of `miniapp-prod` does not by itself establish that the owner has approved it as the production target. Confirm that production remains unconfigured or gated unless explicit confirmation is recorded.

#### Development Taskfile

Check all of these operations:

- `doctor`
- `dev:db:pull`
- `dev:db:up`
- `dev:db:down`
- `dev:db:logs`
- `dev:db:status`
- `dev:db:reset`
- `dev:up`
- `dev:down`
- `test`
- `test:integration`
- `fly:status`
- `fly:secrets:list`
- `fly:secrets:stage`
- `fly:secrets:apply`
- `fly:deploy`

Verify the actual inventory count against the implemented Taskfile.

#### Production Taskfile

Check:

- `doctor`
- `fly:status`
- `secrets:list`
- `secrets:stage`
- `secrets:apply`
- `deploy`
- `release:verify`
- `rollback:prepare`

Production operations must validate:

- Explicit production application identity.
- Expected Fly organization.
- Authenticated operator identity.
- Required validation checks.
- Immutable build/release identity.
- Explicit confirmation.
- Appropriate authorization.

No production mutation should occur during the review.

---

### 7. Review D — Local Docker PostgreSQL and data safety

Inspect the Docker Compose configuration, image-selection logic, and volume-safety scripts.

#### PostgreSQL image selection

Verify that the selected image is:

- An official PostgreSQL image.
- A stable general-availability release.
- Not RC, beta, or development.
- Recorded with a verified version and digest.
- Resolved consistently by the Taskfile and Compose configuration.

Verify that the documentation distinguishes a mutable tag from an immutable image digest.

Review the image-selection script for tag parsing, release-status verification, error handling, and network failure behavior.

Do not assume that `latest` always means the newest stable GA release.

#### PostgreSQL data directory and volume safety

Verify:

- The actual image's `PGDATA` is used.
- Volume layout matches the selected official image.
- Persistent volumes are isolated by PostgreSQL major version.
- The running server version is checked against the expected volume version.
- An incompatible volume causes startup refusal.
- Existing data is preserved.
- No automatic volume deletion occurs.
- No major upgrade is attempted without an explicit migration.
- Reset requires a deliberate confirmation.
- The confirmation cannot accidentally target an arbitrary Docker volume.

Challenge whether the mismatch check can be bypassed by stale or contradictory metadata.

Inspect behavior after:

- Normal stop/start.
- Image patch update.
- Major-version change.
- Incomplete initialization.
- Interrupted upgrade.
- Missing or unreadable volume metadata.

#### Docker Compose security

Inspect:

- Dedicated network.
- Port exposure.
- Loopback-only optional host binding.
- Health checks.
- Restart behavior.
- Service users and privileges.
- Secret mounts.
- Bind mounts.
- Docker build context.
- `.dockerignore`.

Verify that no database credential is embedded in a Dockerfile, image layer, build argument, Compose literal, or tracked configuration.

Do not run destructive volume operations.

---

### 8. Review E — Go configuration and `_FILE` contract

Inspect the actual configuration resolver and its use by `internal/store`, application startup, and DBOS setup.

Required contract:

```
DATABASE_URL
DATABASE_URL_FILE

DBOS_SYSTEM_DATABASE_URL
DBOS_SYSTEM_DATABASE_URL_FILE
```

Verify:

1. Environment variable resolution.
2. File-based resolution.
3. Conflict rejection when both forms are supplied.
4. Missing-value rejection.
5. Empty-file rejection.
6. Whitespace handling.
7. File permission behavior and error reporting.
8. Newline handling.
9. Secret redaction in errors.
10. No connection string logging.
11. No unintended fallback to another environment.
12. Same configuration semantics for local Docker, Fly development, and Fly production.

Check the rule that local Docker uses file-based credentials while Fly supplies the same logical credentials through ordinary environment variables.

Verify that the application does not require environment-specific Go code or different Docker images.

Inspect whether the file resolver can read arbitrary paths when untrusted configuration controls the path. Determine whether that is acceptable within the declared threat model.

Test error wrapping and third-party database errors to ensure connection URLs cannot leak indirectly.

The configuration contract must not silently prefer one credential source over a conflicting second source.

---

### 9. Review F — Secrets management and threat boundaries

This is a critical review area.

The intended arrangement is:

| Environment     | Secret authority                 | Application delivery              |
| --------------- | -------------------------------- | --------------------------------- |
| Local Docker    | Owner-only local state directory | Compose secret files and `*_FILE` |
| Fly development | Fly secrets for `miniapp-demo`   | Environment variables             |
| Fly production  | Fly secrets for `miniapp-prod`   | Environment variables             |

Verify that the same Go configuration contract is used in all three cases.

#### Local secret storage

Inspect `local-secrets-init.sh` and determine the actual default storage directory and filenames from the code. Do not infer them from earlier discussion or documentation.

Verify:

- The actual default path follows the documented OS convention.
- Directory permissions are restrictive.
- Secret files are owner-readable only.
- Files are generated using a cryptographically secure random source when appropriate.
- Initialization is idempotent.
- Existing credentials are not overwritten.
- Local secrets are excluded from Git.
- The entire host secret directory is not mounted into containers.
- Only required files are mounted into each service.
- The local secret lifecycle does not silently copy credentials into Fly or Neon.

Check that Compose's file-backed secret mechanism is described accurately. It is a file mount, not an encrypted vault or protection against a privileged host user or Docker daemon administrator.

#### Fly secret workflows

Review:

- `secret-input.sh`
- `app-preflight.sh`
- `secret-apply.sh`
- `release-verify.sh`
- All Taskfile call sites.

Verify:

- Hidden terminal input.
- Shell tracing disabled.
- No secret values in command arguments.
- Standard-input transfer where supported.
- Strict secret-name allowlists.
- Strict Fly application allowlists.
- No echoing or command interpolation.
- No unprotected temporary credential files.
- Safe shell cleanup.
- Correct error handling and signal behavior.
- Stage/apply separation.
- Production confirmation.
- No automatic production deployment.
- Metadata-only listing.
- No attempts to retrieve Fly secret values.

Assess whether `unset` and shell redirection are sufficient for the claims made. Identify the remaining process-memory and host-privilege limitations honestly.

#### Fly runtime threat boundary

Explicitly verify and document:

- Fly injects application secrets into the process environment.
- Application code can read those values.
- Code deployed to a secret-bearing application can potentially exfiltrate them.
- Anyone with effective deployment authority may therefore have indirect access to runtime secrets.
- Docker image contents and build arguments must never contain runtime credentials.
- Development and production credentials must never be reused.

Check whether the development coding agent has a path to production Fly credentials or deployment authority through environment variables, local CLI authentication, shared scripts, or inherited process context.

Do not inspect or print actual credentials.

Do not run `env`, `printenv`, `set`, process-environment dumps, unrestricted Docker inspection, or other commands that could reveal secret values.

The desired guarantee is no accidental or unauthorized disclosure through the approved workflow. Do not claim that an application process can use a credential while being cryptographically prevented from reading that same credential.

---

### 10. Review G — Fly and Neon operational correctness

Review all existing configuration and evidence without mutating cloud infrastructure.

#### Fly applications

Verify the development and production app identities.

Check:

- Correct app selection.
- Correct organization checks.
- Region configuration.
- Process configuration.
- Secret metadata operations.
- Deployment preflight.
- Release identity verification.
- Rollback preparation.
- Production authorization boundary.

Do not deploy, create Machines, rotate secrets, or modify Fly configuration remotely.

#### Neon database separation

Verify the intended separation:

- Local PostgreSQL is independent of Neon.
- Remote development uses development credentials.
- Production uses production credentials.
- The same database role/password is not reused across environments.
- Application and migration roles have distinct privileges where required.
- DBOS uses a supported connection mode.
- Transaction and RLS context assumptions are tested.

The local PostgreSQL success evidence must not be represented as proof of Neon-specific behavior.

Preserve the distinction between:

- Local PostgreSQL tests.
- Previously executed Neon-specific tests.
- Tests still blocked by Neon access.

Check for accidental fallback from development to production or vice versa.

---

### 11. Review H — CI, scripts, and supply-chain boundaries

Check:

- `scripts/checks.sh`.
- CI workflows.
- Shell scripts and permissions.
- Dependency and license register.
- Go module pins.
- Docker image digest.
- Generated schema drift checks.
- Import-boundary checks.
- Enterprise DBOS module exclusion.

Verify that no CI workflow has acquired unexpected Fly, Neon, or production credentials.

Verify that dependency updates are deliberate and that production release evidence can identify the actual source revision and build artifact.

Check whether test outputs and generated artifacts can contain connection strings, credentials, or personally identifiable data.

Inspect scripts for:

- Unsafe `eval`.
- Unquoted variables.
- Command injection.
- Unbounded input.
- Incorrect exit handling.
- Race conditions.
- Signal cleanup errors.
- Accidental destructive operations.
- Environment-variable leakage.
- Commands that silently select production.

---

### 12. Review I — Documentation and developer experience

Review the actual developer documentation.

At minimum:

- Execution-category guide.
- Local PostgreSQL guide.
- Secrets-management guide.
- Fly development guide.
- Fly production guide.
- Secret rotation guide.
- Taskfile documentation.
- Phase 0 evidence tracker.
- ADRs.
- CLI specification.

Check whether a developer can answer, without asking an architect:

1. How do I start local PostgreSQL?
2. Where are local credentials stored?
3. What are their actual default filenames?
4. How do I know which environment I am targeting?
5. How does Go resolve `_FILE` versus ordinary environment variables?
6. How do I run local tests without Neon?
7. How do I use Fly development?
8. How do I provision a development secret?
9. How is production kept separate?
10. What does each execution category mean?
11. When should I choose A, B, C, or D?
12. When should I NOT use DBOS?
13. Where should business rules, persistence, provider calls, and execution logic live?
14. Who authorizes production changes?

Look for contradictions, undocumented assumptions, stale command names, inaccurate security claims, and duplicate sources of truth.

Documentation must describe what the code actually does, not what the authors intended it to do.

---

### 13. Verification procedure

Run safe, non-destructive checks freshly.

#### Static and structural

- Git status and diff inventory.
- `task --taskfile Taskfile.dev.yml --list`.
- `task --taskfile Taskfile.prod.yml --list`.
- Taskfile parsing.
- `bash -n` on scripts.
- `gofmt -l .`.
- `go vet ./...`.
- `go test ./...`.
- Race tests on supported packages.
- JSON Schema drift checks.
- `scripts/checks.sh`.
- License checks.
- Import-boundary checks.
- Enterprise-module exclusion.
- Documentation/task inventory cross-check.

#### Docker

Where the daemon is accessible, execute non-destructive validation:

- Compose configuration validation without printing secret values.
- Image identity and digest inspection without dumping container environments.
- PostgreSQL health.
- `pg_isready`.
- PostgreSQL server version.
- Stop/start persistence.
- Wrong-major volume refusal using a disposable test volume.
- Reset confirmation behavior using disposable test resources only.
- Local integration suite.

Never delete or modify an existing user database.

#### Fly

Read-only inspection only:

- App status.
- Secret metadata listing.
- Taskfile dry-run.
- Preflight validation.

Do not:

- Deploy.
- Launch.
- Scale.
- Create Machines.
- Apply or rotate secrets.
- Release or promote.
- Execute production rollback.
- Run database migrations remotely.

Classify each test based on what actually executed.

Use these labels:

- `PASS`: the invariant was demonstrated by an executed test.
- `FAIL`: an executed test violated the requirement.
- `BLOCKED`: the necessary infrastructure or authorization was unavailable.
- `NOT RUN`: the check was not attempted.
- `PARTIAL`: only part of the expected behavior was demonstrated.

---

### 14. Required adversarial scenarios

At a minimum, examine the following attack and failure cases.

**Secret exposure**

- Shell tracing accidentally enabled.
- Secret value passed as an argument.
- Fly CLI failure includes sensitive context.
- Secret path printed in debug output.
- Local secret file accidentally tracked by Git.
- Local container process reads more secrets than it requires.
- Agent obtains Fly authentication context.
- Production credentials accidentally used by development.
- Malicious code deployed to a Fly app reads runtime secrets.

**Environment confusion**

- Development Taskfile targets production.
- Production Taskfile targets development.
- Missing production app configuration defaults to another app.
- Production confirmation can be bypassed.
- Incorrect organization is accepted.
- Development and production Neon URLs are swapped.

**Database safety**

- Stale PostgreSQL volume.
- Incompatible major-version volume.
- Partial database initialization.
- Concurrent startup.
- Reset without explicit confirmation.
- Container recreated with new image.
- Database credential rotation.
- RLS context lost through autocommit.
- Superuser or table-owner RLS bypass.
- Connection string leaked by error serialization.

**Go configuration**

- Both ordinary and file variables set.
- Empty file.
- Unreadable file.
- Trailing newline.
- Whitespace-only value.
- Malformed connection string.
- Symlinked secret file.
- File path supplied from untrusted configuration.
- Third-party error embeds a credential.

**Execution category**

- Class A incorrectly made asynchronous.
- Class B used for a multi-stage durable process.
- Class C implemented as an unbounded custom workflow engine.
- Class D used without a durable business requirement.
- Approval incorrectly treated as a category.
- Entity concurrency controlled only by workflow ID.
- External effect retried after UNKNOWN.
- Agent-facing miniapp exposes raw provider endpoints rather than a business outcome.

---

### 15. Findings format and severity

Every finding must have:

```
ID:
Title:
Severity: CRITICAL / HIGH / MEDIUM / LOW / INFO

Affected files:
Exact line references:

Requirement / invariant:
Observed implementation:

Attack or failure scenario:
Why the current behavior is unsafe or incorrect:

Reproduction:
Evidence:
Root cause:

Recommended correction:
Required regression test:
Owning package / phase:

Whether this blocks the next development phase:
```

Severity definitions:

- **CRITICAL:** credential exfiltration, cross-environment production access, data loss, arbitrary production mutation, or a fundamental authorization bypass.
- **HIGH:** broken durability, unsafe external effects, tenant isolation failure, unrecoverable migration/volume behavior, or incorrect deployment target selection.
- **MEDIUM:** material functionality gap, incomplete failure recovery, incorrect category guidance, or operationally unsafe behavior with constrained impact.
- **LOW:** documentation, ergonomics, consistency, or maintainability issue.
- **INFO:** valid observation or improvement without a defect.

Do not inflate severity to make the review appear thorough.

Do not downgrade a serious security issue because an existing test suite is green.

---

### 16. Required deliverables

Produce a complete review report with these sections:

1. Executive verdict: GO / CONDITIONAL GO / NO-GO.
2. Actual repository inventory and baseline verification.
3. Architecture and Phase 0 correctness.
4. Execution-category guidance review.
5. Development and production Taskfile review.
6. Docker PostgreSQL and volume-safety review.
7. Go configuration and `_FILE` review.
8. Secret-management and threat-boundary review.
9. Fly and Neon isolation review.
10. CI, supply-chain, and script review.
11. Documentation and developer-experience review.
12. Verification matrix with exact commands and observed results.
13. Findings sorted by severity.
14. Unresolved risks and owner actions.
15. Prioritized corrective-action backlog.
16. Readiness recommendation for the next development phase.

Create the review report at:

`docs/reviews/adversarial-implementation-review.md`

Do not modify any other file.

The report must contain no actual secrets, database URLs, access tokens, or private credentials.

---

### 17. Decision criteria

Recommend **NO-GO** if any of these remain:

- A credible route for an untrusted coding agent to obtain production credentials or production deployment authority.
- Development/production target confusion.
- A tested tenant isolation bypass.
- Incompatible PostgreSQL volumes can be silently reused or destroyed.
- Credentials can leak through normal scripts, logs, diagnostics, or test output.
- External effects can be blindly retried after UNKNOWN.
- DBOS workflows cannot reliably retain their application/version identity.
- A critical Taskfile operation can bypass its authorization boundary.

Recommend **CONDITIONAL GO** when there are no critical or high-severity security/data-correctness blockers, but material non-critical implementation gaps remain with clear owners and gates.

Recommend **GO** only when the reviewed scope satisfies its stated requirements with reproducible evidence and remaining limitations are explicitly accepted.

Do not interpret GO as permission to deploy production or perform cloud mutations.

---

### 18. Final operating instruction

Be skeptical, specific, and evidence-driven.

Do not praise an architecture simply because it is well documented or a test suite is green.

Challenge the actual failure boundaries.

Investigate root causes rather than symptoms.

Require regression tests for demonstrated defects in any future corrective implementation.

Preserve the approved architecture unless concrete evidence demonstrates that an invariant cannot be satisfied.

The objective is not to produce the longest list of possible improvements.

The objective is to identify anything important that could cause this platform to fail, leak credentials, corrupt business state, misclassify business logic, or become operationally unmanageable once developers begin building real miniapps.

**Review everything implemented so far. Then tell us, with evidence, whether it is safe and sound to move to the next phase.**

### **My recommendation**

Run this review before starting Phase 1. Pay particular attention to three things:

1. **The secret boundary:** Fly protects secrets at rest, but deployed application code can read them. The coding agent's deployment authority is therefore part of the credential security model.
2. **The Go configuration boundary:** local `_FILE` and Fly environment-variable delivery should be behaviorally identical, with no silent fallback or cross-environment leakage.
3. **The business execution taxonomy:** developers must understand that A/B/C/D classify execution semantics, not code complexity or deployment location.

The review should produce a prioritized correction backlog and a clear go/no-go—not trigger another open-ended architecture redesign.