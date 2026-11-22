The prompt below authorizes **Gate A only**: M1, M2, M3, L1, L3 and L4. It explicitly preserves the `plan4.2.md` baseline, prevents scope expansion into Gates B/C, and does not authorize a Git commit.&#x20;

adv_review_prompt5.md



# Implementation Prompt — Miniapp Phase 0 Remediation, Gate A Only

## Role

Act as the senior Go engineer responsible for implementing a narrowly scoped remediation to the `github.com/PithomLabs/miniapp` repository.

You are implementing the approved **Gate A corrective actions** from the adversarial implementation review. This is an implementation task, not another general review.

Your job is to correct the underlying defects, preserve the existing architecture, add regression tests that demonstrate the required invariants, and provide verifiable execution evidence.

Do not merely patch symptoms. Investigate the existing implementation, understand why each failure occurs, and make the smallest durable correction that prevents recurrence.

## 1. Repository and governing authority

Repository:

`/home/chaschel/Documents/go/miniapp`

Authoritative architecture baseline:

`docs/plan4.2.md`

Adversarial review:

`docs/adv_review_prompt5.md`

The review's §15 defines the Gate A backlog. Its findings contain the detailed reproduction evidence, affected files, expected behavior, and proposed regression tests.

Read the relevant sections of both documents before making changes.

**`docs/plan4.2.md` is authoritative and must remain unchanged.**

Do not reopen settled architecture decisions, redesign the framework, or reinterpret the Phase 1 roadmap.

The owner has authorized implementation of Gate A only.

## 2. Exact scope

Implement these six findings and nothing beyond them:

| Finding | Required work                                                                                                               |
| ------- | --------------------------------------------------------------------------------------------------------------------------- |
| M1      | Make unknown `MINIAPP_POLICY` values fail closed to strict; add regression tests and correct `ENV_VARS.md`.                 |
| M2      | Prevent `PoolConfig.String()` from exposing passwords in keyword-format PostgreSQL DSNs; add comprehensive redaction tests. |
| M3      | Make both database readiness loops fail when PostgreSQL never becomes ready; add regression coverage.                       |
| L1      | Turn Go formatting checks into actual failing gates in the development Taskfile, CI and repository checks.                  |
| L3      | Correct stale spike documentation and skip messages that imply Neon is mandatory when local Docker is supported.            |
| L4      | Remove the stray `spikes/started` artifact and add a regression test for unexpected files in `spikes/`.                     |

### Explicitly excluded

Do not implement, partially implement, or opportunistically clean up:

- M4: PostgreSQL image-major grounding and volume checks.
- M5: production release identity verification.
- M6: production clean-tree and revision provenance.
- M7: RLS failure-mode regression tests or integration execution.
- M8: consolidation of the execution-category guide.
- M9: CI integration/Docker job.
- M10: startup configuration consumers, startup equivalence, and permission-model work.
- H1: production authority and Fly hosting boundary.
- O1–O7: owner and infrastructure actions.
- L2, L5, L6, L7, L8, L9 and L10.

Do not add `AGENTS.md`, pin Docker image digests, change Fly configuration, modify the production Taskfile, or perform unrelated contract cleanup.

Gate B and Gate C remain separate authorizations.

## 3. Repository safety rules

The repository was reported to have zero Git commits, with the existing files untracked. Therefore, do not assume that an empty `git diff` means there are no existing files or changes to preserve.

Before editing:

1. Inspect `git status --short` and the repository's current state.
2. Record the initial status and identify the exact files you intend to modify.
3. Read the relevant source and tests before deciding how to implement each correction.
4. Preserve all existing files and user work outside the authorized scope.

During implementation:

- Do not run `git reset`, `git clean`, destructive cleanup commands, or broad file deletion.
- Do not stage or commit anything.
- Do not create a Git commit, tag, release, or branch.
- Do not modify production configuration, Fly applications, Fly secrets, Neon resources, or remote infrastructure.
- Do not read, print, expose, or copy secret values.
- Do not modify `docs/plan4.2.md` or `docs/adv_review_prompt5.md`.
- Do not change package versions or introduce new dependencies unless a demonstrated technical necessity exists. If a dependency appears necessary, stop and report the reason rather than silently expanding scope.

The only authorized deletion is `spikes/started`, and only after verifying that the exact path is still a zero-byte regular file. If its state differs, preserve it and report the discrepancy.

If shell execution or a required tool is blocked, do not bypass sandbox restrictions. Continue only with work that can be performed safely, and report unavailable verification honestly.

## 4. Implementation requirements

### M1 — Unknown `MINIAPP_POLICY` must fail closed

Affected areas:

- `internal/config/config.go`
- `internal/config/config_test.go`
- `docs/ENV_VARS.md`

The review demonstrated that:

`MINIAPP_POLICY=bogus`, with `CI` unset, currently returns `warn`.

This contradicts the code comment and documentation, which promise that unknown values fail closed to strict.

Investigate the distinction between an unset environment variable and an explicitly supplied but invalid value.

Required behavior:

| MINIAPP_POLICY                      | CI unset                              | CI=true                               |
| ----------------------------------- | ------------------------------------- | ------------------------------------- |
| Unset                               | warn                                  | strict                                |
| `warn`                              | warn                                  | warn                                  |
| `strict`                            | strict                                | strict                                |
| `disabled`                          | Preserve existing documented behavior | Preserve existing documented behavior |
| Unknown value                       | strict                                | strict                                |
| Explicitly empty or malformed value | strict                                | strict                                |

Preserve the existing semantics of recognized policy values. Do not introduce a new policy mode.

In particular, an unknown value must never silently weaken validation because the environment is not CI.

Add table-driven regression tests covering at least:

- Unset.
- `strict`.
- `warn`.
- `disabled`.
- `bogus`.
- `STRICT`.
- `strict ` (trailing whitespace).
- Explicitly empty value.

Run the cases with `CI` unset and `CI=true`.

Use the appropriate environment lookup semantics so unset and explicitly set values are not accidentally conflated.

Correct `docs/ENV_VARS.md` to describe the actual, tested behavior precisely.

### M2 — `PoolConfig.String()` must not leak keyword-format DSN passwords

Affected areas:

- `internal/store/store.go`
- `internal/store/store_test.go`

The review demonstrated that URL-format redaction does not protect keyword-format PostgreSQL DSNs.

Example:

`host=db user=u password=s3cret dbname=x`

Currently, the password can appear verbatim in `PoolConfig.String()`.

Investigate the DSN formats accepted by the existing implementation and the exact behavior of its diagnostic formatter.

Required invariant:

**No password or other credential material may appear in the string representation of `PoolConfig`, regardless of DSN format or malformed input.**

Implement a durable redaction strategy.

Acceptable approaches include:

- A proper format-aware parser and sanitizer for the supported DSN formats.
- Omitting the raw DSN entirely from the diagnostic representation.
- Redacting the entire DSN whenever safe parsing cannot be guaranteed.

A fragile regular expression that handles only simple unquoted values is not sufficient.

Do not fall back to printing the original DSN when parsing fails.

Tests must cover at least:

1. URL DSN containing a password.
2. URL DSN without a password.
3. Keyword DSN containing an unquoted password.
4. Keyword DSN containing a quoted password.
5. Keyword DSN containing a quoted value with escaping, where supported.
6. `passfile` in keyword format.
7. Malformed DSN input.
8. A unique credential canary that must never occur in `PoolConfig.String()`.

Where the sanitizer preserves safe connection metadata, verify that host and username remain useful. Where safe parsing is not possible, prefer complete DSN redaction over credential leakage.

Keep the diagnostic method useful, but security takes precedence over preserving every DSN field.

Do not introduce connection-string logging elsewhere.

### M3 — Database readiness loops must fail when PostgreSQL is unavailable

Affected area:

- `Taskfile.dev.yml`

The review identified two defective readiness loops:

- `dev:db:up`
- `test:integration`

Both use a retry loop in which a failed `pg_isready` is followed by `sleep`. When every attempt fails, the final `sleep` exits successfully. The Taskfile therefore reports success even though PostgreSQL never became ready.

Investigate the exact Taskfile commands and shell behavior.

Required invariant:

**A readiness task must return success if and only if the database readiness condition was actually established.**

After the retry loop, perform a final readiness check whose failure propagates as a nonzero task exit.

Do not allow a successful `sleep`, logging command, or cleanup command to overwrite the readiness failure status.

Apply the correction to both tasks.

Regression coverage must demonstrate that persistent readiness failure cannot produce a successful task result.

Prefer a deterministic test using stubbed readiness and sleep commands if it can be implemented without introducing unnecessary infrastructure. Otherwise, add a meaningful structural test that ensures the final readiness gate exists outside the loop and is not masked by another command.

Do not require a live Docker daemon for this Gate A regression test.

Do not alter Docker volume behavior, PostgreSQL image selection, or the integration suite's architecture.

### L1 — Go formatting checks must actually fail

Affected areas:

- `Taskfile.dev.yml`
- `.github/workflows/ci.yml`
- `scripts/checks.sh`
- Relevant `infra/` tests

The review established that:

`gofmt -l .`

returns exit status zero even when it prints unformatted files.

Consequently, the current Taskfile and CI checks are decorative rather than enforceable.

Correct this at every relevant gate.

Required behavior:

- `task test` fails when Go source files are not formatted.
- CI fails when Go source files are not formatted.
- `scripts/checks.sh` checks the entire intended Go source set, including `infra/`, rather than silently omitting it.

Use a construct that makes non-empty formatting output fail the check and gives the developer a useful list of offending files.

Also ensure that a genuine `gofmt` execution error is not converted into success by shell command substitution or pipeline behavior.

Do not automatically rewrite files inside the check itself. Formatting checks should be checks, not silent mutation.

Add a regression test or static assertion that would fail if the non-failing `gofmt -l .` pattern is reintroduced.

Keep the development task inventory unchanged.

### L3 — Correct stale spike skip messages

Affected areas:

- `spikes/doc.go`
- `spikes/dbos_test.go`
- `spikes/river_test.go`
- `spikes/rls_test.go`
- Any additional exact occurrences discovered during inspection.

The review found that the spike documentation and skip messages still say that a Neon branch is required.

That is stale: the integration tests can use the local Docker PostgreSQL environment through `test:integration`; Neon is an alternative, not a universal prerequisite.

Update the wording to explain the supported execution choices accurately.

The wording should direct developers to `test:integration` for the local Docker path and explain that an appropriately configured Neon branch is an alternative.

Do not remove legitimate Neon-specific requirements from tests that genuinely need Neon.

Do not change test execution semantics, environment variables, or the integration architecture.

Add a lightweight regression test or static assertion that prevents the misleading mandatory-Neon wording from returning and verifies that the local integration path is discoverable.

### L4 — Remove the stray `spikes/started` artifact

Affected areas:

- `spikes/started`
- A suitable repository-hygiene test, likely under `infra/`.

The review identified a zero-byte file named:

`spikes/started`

It appears to be a stray crash-probe marker rather than a legitimate source or fixture.

First inspect the exact path and verify that it remains a zero-byte regular file.

Only then remove that exact file.

Add a regression test that checks the contents of `spikes/` against an explicit allowlist or expected file policy.

The test should permit legitimate Go source and any intentionally approved files, but reject unexpected artifacts such as `started`.

Make the repository path resolution robust; do not make the test depend on whichever directory happens to be the process working directory.

Do not broadly delete files or change the DBOS crash-probe implementation.

## 5. Testing and verification

Use the repository's documented Go environment overrides where necessary, including the `GOWORK=off` and writable Go cache settings described in the Phase 0 evidence.

Do not change the module's Go version or dependency versions to work around environmental problems.

Run all checks that are available without external infrastructure.

At minimum, attempt:

1. Targeted tests for `internal/config`.
2. Targeted tests for `internal/store`.
3. Tests for `infra`.
4. Tests for `spikes` in the default environment, recording skips accurately.
5. `go test ./...`.
6. `go vet ./...`.
7. `go test -race ./contract/ ./internal/...` where the environment permits.
8. `bash scripts/checks.sh`.
9. `task --taskfile Taskfile.dev.yml test`.
10. `task --taskfile Taskfile.dev.yml --list`.
11. Taskfile and CI formatting-gate verification.
12. `bash -n` on any shell scripts that were changed (if any).

Use static tests and unit tests to prove the Gate A fixes.

A test that skips because infrastructure is unavailable is **not a passing behavioral test**. Report it as skipped or blocked.

Live Docker and Neon checks are not Gate A prerequisites. Do not attempt production or remote infrastructure operations.

Do not claim the integration suite passed merely because `go test ./...` compiled the spike package.

## 6. Change discipline

Keep the patch small and directly traceable to the six authorized findings.

Do not consolidate unrelated documentation, reorganize directories, rename packages, introduce abstractions for hypothetical future use, or rewrite existing architectural guidance.

Do not change the dev/prod environment separation.

Do not add new Taskfile tasks unless a concrete requirement makes one unavoidable; the expected development task inventory remains 16.

Do not turn Gate A into a general cleanup pass.

When a proposed correction touches a file outside the expected areas, explain why it is necessary and keep the change limited to the relevant invariant.

## 7. Completion criteria

Gate A is complete only when:

- M1 is corrected and its CI/non-CI regression matrix passes.
- M2 cannot expose a credential canary through `PoolConfig.String()` for any tested DSN form.
- M3 readiness tasks fail when PostgreSQL never becomes ready.
- L1 formatting gates return nonzero when Go formatting violations exist.
- L3 documentation and skip messages correctly explain local Docker versus Neon execution.
- L4's verified stray artifact is removed and directory hygiene is regression-tested.
- All relevant tests and checks have been run, with results honestly recorded.
- No excluded finding has been implemented.
- `docs/plan4.2.md` remains unchanged.
- `docs/adv_review_prompt5.md` remains unchanged.
- No Git commit, staging operation, deployment, secret mutation, or remote infrastructure change occurred.

## 8. Final response

When finished, provide a concise but evidence-based implementation report with these sections:

### A. Changes made

For each of M1, M2, M3, L1, L3 and L4, identify the files changed and the invariant now enforced.

### B. Regression tests

List the new or modified tests and explain which failure mode each test proves.

### C. Verification results

Use a table with:

- Check or command.
- Result: PASS, FAIL, BLOCKED, or SKIPPED.
- Concise evidence or reason.

Do not label a skipped or blocked test as a pass.

### D. Scope confirmation

Explicitly confirm:

- Only Gate A was implemented.
- Gates B and C were not started.
- `docs/plan4.2.md` was not modified.
- No Git commit was created.
- No deployment, Fly mutation, Neon mutation, or secret mutation occurred.

### E. Remaining work

List any Gate A item that remains incomplete, along with the exact blocker and next action.

Do not declare Gate A complete if any required invariant or regression test remains unresolved.

---

**Final instruction:** Solve the underlying defects, not merely the reproductions. Preserve the architecture. Prove the fixes with regression tests. Do not exceed the authorization boundary. Report exactly what was changed and exactly what was verified.

This prompt leaves the agent with a narrow implementation mandate and explicit evidence requirements. The next decision after its report should be whether Gate A is genuinely complete—not whether to quietly begin Gate B.