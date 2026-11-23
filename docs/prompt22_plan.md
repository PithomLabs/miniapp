Goal

  Implement WS-B Phase 2 (Prompt 22): a canonical, atomic effect-time authorization boundary
  (authorizeAttempt) that revalidates authoritative persisted state and commits PENDING → ATTEMPTING
  before any provider call, under binding constraints C1–C7, starting with P21-002 as the entry task.

  Success Criteria

  Phase 2 acceptance criteria 1–15 hold: P21-002 resolved with a discriminating PG19 test; Class A
  lifecycle rows in the contract; authorizeAttempt canonical with all checks atomic in its own ordered
  transaction; C1–C7 respected; both cancellation race orders tested; no provider-path bypass;
  post-commit provider calls only; no blind retries of ambiguous outcomes; C6 Class A recovery
  contract verified; PG19 digest-pinned local/CI parity; full suite, DBOS suite, race subset, and
  checks green with no hidden skips; every guarantee linked to source, test, and evidence.

  Approach

  Follow prompt22's five stages in order, carrying forward the Phase 1 evidence discipline (frozen
  discriminating tests, identical red/green bytes, per-finding pre-fix copies, exact commands and
  digests, never-silent contract amendments). Key decisions:

  • Stage 0 re-verifies rather than trusts. Reconnaissance (navigation only, not evidence) found the
    tree drifted to HEAD 1a67ba2 (digest b0a4771d…, owner "prompt21" commit), plus four probable gaps
    Stage 0 must confirm or refute: (1) Settle still transitions through RECONCILING in two
    transactions despite the accepted atomic-settlement record, and the
    TestSettleCrashConverges/TestSettleAudited tests do not exist; (2) no production
    sequence/generation tables exist (probe_sequences is test-fixture-only), which the Stage 2 design
    must resolve for §7.5 fencing; (3) no RWMutex/pinned snapshot, /healthz config hash, or
    verify-revocation tooling exists despite §7.3/§4.1; (4) CI and compose float postgres:19.0 by tag.
    Gaps become recorded findings with minimal corrections or explicit deferrals — never silent
    reinterpretations.
  • P21-002 first, no authorization code before it closes. Reproduce the pending-plus-attached stuck
    state through the real dispatch/restoration path, then choose between predicated pending → running
    at adoption versus pending-with-run liveness, against the stated criteria (Class A pending
    untouched, restoration eligible, atomic predicated writes, terminal/quarantined excluded, no new
    sweeper). The decision is recorded in the contract before authorizeAttempt depends on it.
  • Design note gates implementation. Stage 2 maps every §7 check to source, lock, mode, and test,
    including the lock/transaction interaction graph against the Class A attempt tx; any genuine
    contradiction with C1–C7 stops work per §16 instead of bending constraints.
  • No-bypass by inventory, not by audit. Enumerate every production provider-initiation path
    (Perform, takeover/adopt, retry/reconcile, Channel implementors, DBOS workflow call sites,
    faultwrap/test-only wiring excluded by construction) and route each through the boundary; prove
    with a test per path that authorization failure means no provider call.
  • Error taxonomy extends, never forks. Align new refusal reasons with existing terminal/retryable
    kinds and ledger errors (ErrAttemptInProgress, superseded); a new code only where no existing one
    fits, justified in the design note.

  Steps

  1. Stage 0 — Baseline and source audit. Verify repo/branch/worktree/digest state; read all seven
     governing sources; inventory effects, ledger, approvals, grants/policy, kill switches, lifecycle
     predicates, channels/adapters, DBOS paths, fault hooks; verify PG19 parity (record
     SELECT version(), resolve the full immutable image digest via inspect, diff local vs CI); confirm
     or refute each recon-flagged gap; emit the source map and baseline manifest. Deliverable:
     baseline record in the evidence register.
  2. Stage 1 — P21-002. Reproduce via the real path with mock effects; demonstrate the failure
     (paralyzed Perform, misleading cancel, dead status); select and document the semantics; implement
     the smallest shared-boundary fix; add restoration/Perform/cancel/sync regression tests plus live
     and terminal controls; update the contract's pending semantics and every affected predicate row;
     verify on PG19. Gate: closed before any authorization decision treats pending as live.
  3. Stage 2 — Contract maintenance + authorization design. Add the P21-003 Class A lifecycle and
     attempt-tx lock rows (matching shipped code and tests); resolve P21-001 (retarget boundary tests
     to live completeTx or remove dead completeIfLive with explanation); fold P21-004 one-liners into
     touched files only. Then write the authorization design note (§7.1–§7.6 mapped, lock order/modes,
     separate-tx boundary, error/retry semantics, cancellation linearization, lease/generation
     resolution, grant-snapshot assumptions, interaction graph, per-invariant tests).
  4. Stage 3 — Implement the boundary. Smallest shared authorizeAttempt enforcing all checks plus the
     predicated transition atomically (C1: no advisory lock; C2: RLock → 1 → 2 → 3 → 4 → 5; C3: own
     tx/checkout); wire every production effect path through it with post-commit provider calls;
     implement the §4.1 grant-snapshot RLock plus setter API if Stage 0 confirms it missing; extend
     the INFO-2 guard or add a checks.sh raw-SQL gate if new SQL paths appear (C4); pin the PG image
     digest in compose and CI (§11).
  5. Stage 4 — Adversarial verification. Authorization matrix (§10.1), deterministic race tests on
     real PG19 sessions (both cancellation orders, worker-vs-worker same-effect and cross-generation,
     lease turnover, restoration-vs-dispatch, shared-row contention), crash tests at all six
     boundaries (§10.3) with retry/adopt/reconcile/UNKNOWN dispositions, P21-002 regression, C6 Class
     A Perform fixture (stable BusinessOpID, bounded recovery, no indefinite tx hold), bypass
     inventory re-verified; then full serial suite with DBOS URLs set, required -race subset,
     red/green staging per §10.6 with honest classification.
  6. Stage 5 — Closeout. Update lifecycle contract, authorization design docs, operational/revocation
     docs, evidence register; write docs/phase2/prompt22-phase2-implementation-report.md (§15, 16
     sections, honest claim classification); confirm worktree state, no commits/pushes/deployments.
     Stop; Phase 3 needs separate instruction.

  Non-goals (per §12/§16): River/Class B/C, X-12/X-10 implementation, Sprites/Fly/production, real
  providers, redesigns, UI/generators, commits or pushes.

  Validation Plan

  • Stage gates: Stage 1 exit needs the P21-002 red/green pair plus contract rows; Stage 3 needs the
    approved design note; Stage 4 needs the matrix plus race/crash evidence before closeout.
  • Commands (CWD repo root, GOWORK=off, disposable PG19 DBs, TEST_DBOS_DATABASE_URL always set for
    DBOS): gofmt -l ., go vet ./..., go test -p 1 ./... -count=1,
    go test -race -p 1 ./runtime/ ./approvals/ ./effects/, bash scripts/checks.sh, plus focused -run
    invocations per test with exit status recorded. Highest-risk validation: the T6-class
    crash-at-commit-boundary tests and the cancellation-vs-authorization both-orders races —
    rendezvous must be on persisted PG state (row/blocking evidence), never sleeps or goroutine
    counts.
  • Environment proof: SELECT version(), image reference plus full digest, Go version/arch, redacted
    env, commands with working directories, counts/skips/durations; any non-PG19 environment marked
    BLOCKED, never PASS.

  Risks / Open Questions

  • Q1 — Settle/RECONCILING contract-vs-code gap. If Stage 0 confirms Settle still settles via two
    RECONCILING transitions against the accepted atomic record, should Phase 2 include the minimal
    atomic-Settle correction as enabling ledger work (my recommendation, since crash/recovery evidence
    depends on settlement atomicity), or record it as deferred with the crash tests scoped around it?
    Awaiting owner direction; default if unanswered: verify precisely, implement the minimal
    correction only if a Phase 2 invariant requires it, otherwise file as deferred with evidence.
  • Risk: §7.5 fencing has no production backing table today; the Stage 2 design may need a
    forward-only migration (tested fresh and upgraded) or a ledger-grounded alternative — flagged
    early so it surprises no one.
  • Risk: owner-side concurrent doc edits (observed in prior phases) drifting live-tree digests;
    mitigated by frozen per-finding copies and manifest-anchored claims.

  Reply go to start, or tell me what to adjust.


◆ Recap: Plan WS-B Phase 2 effect-time authorization per prompt22, starting with P21-002. Next action:
         owner reviews the plan and replies go.
