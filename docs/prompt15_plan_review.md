GO — approved with guardrails

I approve **single-pass execution through the Prompt 15 completion report**, using the three recommended choices:

1. **Environment:** local Docker/PostgreSQL and real process-kill tests, with normal host-level permissions. Best-effort Neon against the existing `phase0/g0-spikes` branch only, subject to the safeguards below.
2. **Checkpoint:** single GO through implementation, adversarial validation, and close-out. Do not pause after the root-cause matrix.
3. **OpenCode:** repeat the Prompt 13 scope—transport, MCP, and an agent turn. Improve the evidence artifact rather than expanding into a larger multi-turn scenario.

Send the following to the coding agent.

**GO — Begin Prompt 15 remediation.**

Approved decisions:

**1. Execution environment**

- Use local Docker/PostgreSQL with the necessary host permissions for integration tests, concurrency tests, and genuine process-kill/restart tests.
- Do not weaken host-wide security controls, change Docker daemon security settings, or access unrelated data to make tests run.
- Neon is best-effort. First perform the read-only availability and branch-identity probe.
- Use `phase0/g0-spikes` only if it actually exists and can be confirmed as the authorized test branch. Do not create another project or branch.
- Do not drop, recreate, or destroy Neon resources. Do not run destructive fixtures, migrations, truncations, or cleanup against a branch whose test-only status and data-safety are not established.
- If Neon is unreachable, missing, or unsuitable for safe testing, record it as BLOCKED. Continue with local PostgreSQL; do not let Neon availability block the rest of the work.

**2. Checkpoint**

- One GO through P0–P4 and the final `docs/prompt15_imp.md` report.
- Do not stop after the root-cause matrix or after fixing only the five HIGH findings.
- Complete the full X-01–X-24 remediation pass, including all X-24 subfindings.
- Stop for owner review only if you encounter a genuine conflict with `plan4.2.md`, an accepted G0 decision, an unresolved owner-policy choice, or a proposed change that would weaken a governing invariant or require unauthorized access or mutation.
- A discovered code defect, failed regression test, or newly identified root cause is not by itself a reason to stop. Investigate and repair it within scope.
- Never invent an owner decision to keep implementation moving. Preserve genuinely unresolved decisions as open.

**3. OpenCode E2E**

- Re-run the same breadth as Prompt 13: transport, MCP, and one genuine agent turn.
- Do not expand into a multi-turn scenario.
- Improve the evidence artifact to include the OpenCode version, model/version, source worktree digest, invocation/run identifiers, and corresponding server-side audit evidence. Redact credentials.
- State precisely what this demonstration proves and what it does not prove.

**Additional implementation instructions**

- Your P0 baseline and P1 root-cause matrix are mandatory. No source changes before the matrix is written.
- Treat Prompt 15 sections and finding IDs as the authoritative traceability structure. The lettering discrepancy is a documentation quirk; follow the section numbers and X-IDs.
- Explicitly include **X-14** in the lifecycle remediation work. Prompt 15's acceptance matrix includes it, but its implementation needs a concrete task: inspect shutdown error handling in `cmd/miniapp/main.go`, `testkit/stack/stack.go`, and relevant tests. A failed or timed-out DBOS shutdown must never be followed by a silent `pool.Close()` while workers may still be using the shared pool. Preserve diagnostics and terminate through a clearly reported non-zero failure path or another demonstrably safe shutdown strategy. Add fault-injection tests proving the pool is not closed prematurely.
- Keep implementation sequential at shared state-machine boundaries. Do not have independent agents or parallel edits modify `runtime/service.go`, `runtime/store.go`, approval transitions, or effect-claim logic concurrently.
- Treat source digests as part of every important evidence run, not merely a final documentation task.
- Preserve all existing dirty and untracked files, user-owned documents, G0 evidence, and governing documents.
- Migrations remain forward-only. Test fresh installation and upgrades. Never rewrite an applied migration.
- No commits, pushes, remote mutations, or `lead-followup` implementation, manifest, registration, execution, or ADR approval.

**Engineering standard**

For every finding, establish the violated invariant, trace the root cause across the relevant state machine and failure boundaries, repair the shared mechanism, and add adversarial regression tests that fail against the original behavior.

Do not merely patch the reported line. Do not make tests pass by weakening assertions. Do not claim PASS without executed regression evidence.

Proceed through the complete remediation and report the actual outcome, including anything that remains PARTIAL, FAIL, NOT RUN, or BLOCKED.

One important point: **X-14 needs explicit ownership in the implementation plan.** Your plan's lifecycle grouping is appropriate, but Prompt 15's dedicated workstream descriptions do not give X-14 the same detailed treatment as the other findings. The instruction above closes that gap.

The intended sequence is now clear: baseline → root causes → complete remediation → adversarial proof → evidence-bound G1 reassessment. No architecture redesign and no premature G1 PASS.