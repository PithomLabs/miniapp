## Goal

Close the five LOW Gate 2 findings from `docs/adv_review_prompt29.md` per prompt30: harden refusal evidence (P29-001), add the guard negative control (P29-002), strengthen trust-name detection (P29-003), verify the supersession record (P29-004), document the abandoned-PENDING boundary (P29-005), and genuinely attempt-or-honestly-block the operational revocation drill — without touching Stage 3A architecture, guarantees, or history.

## Success Criteria

- Six policy-refusal legs assert PENDING-untouched row state plus exact `effect-refused` counts (or the register claim is narrowed with contract justification).
- Bypass guard has a planted-violation fixture + negative control mirroring P24-009; a broken matcher fails loudly.
- Trust-name guard detects normalized case-insensitive `trust` with adversarial-variant tests and zero false positives on ordinary fields.
- Supersession chain verified complete; closeout verdict untouched.
- Abandoned-PENDING boundary documented in guide + contract pointer with no execution claims; ops doc gains the hash-sourcing line; ten-minute semantics unchanged.
- Drill either executed with full evidence or marked BLOCKED/NOT EXECUTED naming the exact missing prerequisite.
- Closeout report complete; all gates green on disposable pinned PG19.

## Approach

Research established every mechanism in advance. The six P29-001 legs live in `effects/p28_matrix_policy_test.go` and currently assert only typed error + zero sends; the same package already has `p28ReadRow`/`p28RefusedAudits` helpers to reuse, and refusal-audit semantics check out (kill-switch/grant-denied refusals occur after invocation load, so `inv` is attributable → exactly one `effect-refused`; row stays PENDING since refusal precedes step 5). P29-002 mirrors the P24-009 pattern exactly (`auth/testdata/policy_guard_violations.go` + `// want:` markers + control test; the inventory test already skips `testdata/` dirs), which requires extracting the guard's per-file matcher into a callable function — test-only refactor, production untouched. P29-003 becomes a pure `isTrustBearingName()` function (normalized case-insensitive `trust` substring, per the review's own suggestion) with a table test covering the five required adversarial names plus the seven real `Intent` fields as negatives (verified: none contains "trust", so zero false positives). P29-004 is verify-only (memo addendum and register rows observed to exist; confirm completeness). P29-005 is two doc insertions with explicit no-execution language. The drill environment genuinely appears absent (DB-only compose, no replica tooling, no named/authorized Fly app, no sprite-env) — but the plan inspects rather than assumes.

Key decisions: (1) New tests are green-first characterization; no red/green claims (no defects expected; if one appears, freeze bytes and reclassify honestly). (2) Strengthen-don't-narrow for P29-001 unless a leg's audit semantics legitimately differ — then correct the register claim per prompt30 §2. (3) No manifest artifact — prompt30 §9 specifies report + register + doc updates only; digests live in the report. (4) Drill defaults to honest BLOCKED unless inspection finds an authorized isolated multi-replica environment.

## Steps

1. Freeze baseline (HEAD, status, source/tree digests) and snapshot the tree; record crash-test and `effects.Settle` hashes for the byte-identical proofs.
2. P29-001: add row-state + audit assertions to the six legs; run focused tests.
3. P29-002: extract guard matcher, add `effects/testdata/bypass_guard_violations.go` fixture (prohibited Send impl + call site, `// want:` markers), add negative-control test; verify it fails if the matcher is neutered (temporary local check, not committed).
4. P29-003: add `isTrustBearingName()`, rewire the struct guard to it, add adversarial + negative table test.
5. P29-004: verify the five checklist items against memo/register/plan/closeout; record result, change nothing unless incomplete.
6. P29-005: boundary paragraph in `docs/development/effect-authoring.md` + contract pointer at the C6 rule; one hash-sourcing line in the ops revocation section; no execution language.
7. Drill: inspect for isolated multi-replica environment + authorization; execute with full evidence if present, else record BLOCKED/NOT EXECUTED with the precise missing prerequisite. Never touch prod.
8. Evidence register rows, `docs/phase2/prompt30-residual-closeout.md` (§9 contents), full validation battery, stop.

## Validation Plan

- `go test ./effects/ -run 'TestP28KillSwitch|TestP28ToolDeclared|TestP28GrantAllowDeny|TestP28ProviderSend|TestP28VerifyGuard|TestP28Intent' -count=1 -v` on disposable DBs at `postgres@sha256:bb6e70ad…` (re-resolved via `docker inspect`).
- Affected suites `go test -p 1 ./effects/ ./runtime/`, then full `go test -p 1 ./... -count=1`, `go test -race` on touched concurrency surface, `gofmt -l .`, `go vet ./...`, `bash scripts/checks.sh`.
- Byte-identity: `sha256sum -c` on pre-recorded Settle crash-test + `effects.go` hashes; `git diff --stat` review proving only intended files changed.
- Highest-risk validation: the P29-001 audit-count assertions — if any leg legitimately audits zero or twice, the test must encode the contract truth and the register row gets corrected instead of forcing `×1`.

## Risks / Open Questions

- Risk: drill environment inspection is the one step whose outcome I cannot fully prejudge; the plan treats BLOCKED as a first-class honest result with a named prerequisite.
- Open questions: None — all material choices resolve from prompt30, the governing docs, and the audited sources above.
