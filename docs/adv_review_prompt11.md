# Prompt 11 — Adversarial Review: `lead-followup` ADR-001

- **Reviewer role:** independent Principal Architect / distributed-systems reviewer (read-only; no code, test, manifest, ADR, or history modified except this report).
- **Primary document:** `docs/adr/miniapps/lead-followup-adr001.md` (DRAFT — NOT APPROVED, 97 lines).
- **Baseline:** `docs/plan4.2.md` (authoritative, immutable) + `docs/DBOS.md` gates/rubric + `docs/adr/001-dbos-adoption.md` port + G0-Core adjudication (PASS, owner Gani Mendoza) + `docs/phase1/open-spec-items.md` (OSI-1).
- **Review date:** 2026-11-22. HEAD `5cf9627` (merge) over `e6b5b7b` (G0 baseline); Phase 1 tree uncommitted.
- **Report path:** `docs/adv_review_prompt11.md` per review request (prompt11 §11 names `docs/adr/miniapps/lead-followup-adr001-review.md`; that file was not created — no other file touched).
- **This review confers no approval. Increment 6 remains blocked. The increment-5 hard stop stays in force.**

## A. Executive verdict

**Recommendation: DEFER the ADOPT decision — do not approve, do not reject the workload.**

For the business owner, in plain terms: the ADR proposes adopting the DBOS durable-workflow engine for sales-lead follow-up on the strength of a 12/14 complexity score and three claimed business consequences (lost revenue, customer spam, playbook violation). None of the business semantics behind that score exist in any authoritative source — the playbook (how many touches, when, on which channels, under what rules) is entirely `[PROPOSED]`, and `docs/phase1/open-spec-items.md` item OSI-1 explicitly records this as an open gap. Scored strictly on *established* requirements, the workload is **6/14**, which the governing rubric places in "evaluate existing queue and database capabilities first," not "adopt DBOS." The 12 is the ceiling *if the owner confirms everything proposed*; it is not a measured result.

For the engineer: three mechanism claims do not survive contact with the code. (1) "Exactly-once per touch" is unachievable as written — the framework's own effect ledger is honestly at-least-once with UNKNOWN quarantine, external sends cannot share a transaction with the touch record, and no reconciler exists to drain quarantine. (2) Per-lead serialization via deterministic workflow ID is half a design: under the SDK's default ID-reuse policy a completed sequence's ID returns the old result and ignores new input, so any ID scheme without a sequence-generation discriminator permanently bricks future campaigns for that lead — and generation is undefined. (3) The "recovery guard" and "versioned recovery" do not exist: `WorkflowDef.Version` is a string nothing reads, the executor sets no application version (the SDK derives one from the binary hash, so *every deploy* is a new version), and all workflow families share a single `AppName`, contrary to plan4.2 §11.2. Two further framework gaps directly endanger this workload: approval denial marks the invocation failed but never cancels the running workflow (an orphan sequence keeps sending), and approval decisions are never delivered to the waiting workflow (no decision→signal bridge; expiry is never even set).

None of this means DBOS is wrong for lead follow-up. A confirmed multi-day, multi-touch, cancel-safe sequence with intolerable-duplicate consequences is a natural Class D workload, and the Phase 1 substrate (canonical invocation, ledger, approvals table, tenant isolation, shared-pool transactions) is well-built for what it claims. But an ADOPT today would approve an engine for requirements that do not yet exist, with guarantees the machinery cannot keep. Resolve the owner-decision register (§E), apply the design corrections (§F/§H), and re-review: that is a short path to a defensible ADOPT.

## B. Findings

Severity: BLOCKER (adoption cannot be decided until resolved), HIGH (wrong behavior or missing load-bearing mechanism), MEDIUM (material gap or unproven claim), LOW (minor defect or polish). Each finding states whether it needs a human business decision or an engineering change.

### BLOCKER findings

**B1 — No authoritative playbook exists; every scored semantic is owner-unconfirmed (OSI-1 open)**
- Severity: BLOCKER (human business decision).
- Location: ADR lines 7-9, 15-20, 25, 32, 75-78; `docs/phase1/open-spec-items.md:7-18`.
- What is wrong/missing/unproven: Number/timing of touches, channels, quiet hours, eligibility, exit/escalation, approval scope, cancellation, and failure behavior have no approved source. The ADR is honest about this (`[PROPOSED]`, "nothing here invents an owner decision"), which is to its credit — but it then scores 12/14 and proposes ADOPT on those same unconfirmed semantics.
- Why it matters: DBOS.md §4 and prompt11 forbid awarding points for unconfirmed semantics. Gate 3 consequences (revenue loss, spam, violation) are assumptions with unanalyzed likelihood/severity/reversibility. An ADOPT now would be an engine commitment without a requirements baseline; any playbook the owner later confirms (e.g., a single touch, no approval) could strand the decision.
- Evidence: OSI-1 text ("No approved source … specifies the follow-up sequence"); ADR's own `[PROPOSED]` markers on the promise, stage list, mechanism, and entity identity.
- Required resolution: Owner completes decision register §E (at minimum: touches, timing, channels, eligibility, exits, approval yes/no, cancel semantics, failure behavior). Then rescore §D and re-review. Engineering cannot close this.

**B2 — Deterministic workflow ID per lead, as proposed, bricks future campaigns; sequence generation is undefined**
- Severity: BLOCKER (engineering design + owner confirmation of campaign semantics).
- Location: ADR lines 62-63 (§7 "stable run handle derived per invocation"), 75-78 (entity identity); `executor/dbos/executor.go:104` (random `run-uuid` IDs today); SDK `dbos/workflow.go:942-950,1134-1139` (v1.9.0, verified in module source).
- What is wrong/missing/unproven: The ADR offers "a deterministic workflow ID derived from the lead, or a partitioned DBOS queue" as interchangeable options with no generation concept. They are not equivalent, and the first is broken as specified: under the SDK default `WorkflowIDReusePolicyReturnExisting`, starting a workflow with a completed workflow's ID does **not** re-execute — it returns the recorded result and **ignores the new input**. A lead whose sequence completes can therefore never start a legitimate later campaign under the same derived ID; the submission silently returns stale success. Concurrent same-lead submissions with different invocation IDs also create two runtime invocations (claim keys differ, `runtime/store.go:38-58`) while attaching to one run — invocation records, audit attribution, and `getByRun` lookups diverge with no defined resolution. "Stable run handle derived per invocation" additionally contradicts the substrate, which mints random IDs per `Submit`.
- Why it matters: This is the serialization mechanism the coordination point and the "collapse into one sequence" promise rest on. As written it guarantees either duplicate concurrent sequences (random IDs) or permanent one-sequence-per-lead-ever (naive deterministic IDs).
- Evidence: SDK doc comment quoted above (read from `dbos-transact-golang@v1.9.0/dbos/workflow.go`); `executor.go:104`; `invocations.claim` keyed on `(tenant, miniapp, version, tool, idempotency_key)` only.
- Required resolution: Define sequence generation as a business concept (what starts a new generation: new submission after terminal state? new campaign ID? owner-confirmed), then derive workflow ID from `(tenant, lead, generation)` with a durable sequence record and partial unique index on the active generation (§F). Add the §G concurrency/generation tests. Do not start workflows by bare `lead_id`.

**B3 — "Exactly-once per touch" is unachievable as written; quarantine has no drain and the tx-step wording is incoherent**
- Severity: BLOCKER (engineering design + replacement language §H).
- Location: ADR lines 15-18 (promise), 29-30 (effects 2), 35-36 (Gate 1), 60-61 (§7 tx boundary), 79-81 (effect boundaries); `effects/effects.go:71-103`; `contract/run.go` effect machine.
- What is wrong/missing/unproven: (a) The framework's own ledger is honestly at-least-once: `Perform` sends outside any transaction (`effects.go:94`), then records the outcome in a later transaction — the provider-accepted/crash-before-confirm window is inherent, and `adopt()` correctly quarantines it as UNKNOWN rather than promising exactly-once. The ADR promises what its substrate refuses to promise. (b) "Transaction steps recording the touch first" cannot include the send: `RunAsTransaction` is DB-only; a send is not transactional. The honest shape is tx-record-intent → step-send-via-ledger → tx-record-outcome, with crash windows between each hop. (c) Nothing drains UNKNOWN: no reconciler, no sweeper, no provider status-lookup, no operator runbook, no alerting. Quarantine without a drain is a stuck touch wearing a nicer state name. (d) "Touch record + checkpoint commit atomically" conflates the app record (atomic with checkpoint inside `RunAsTransaction`, proven at G0-3) with the external outcome (never atomic with anything).
- Why it matters: The business promise "no lost, duplicated, or out-of-window touches" is the load-bearing claim of the whole ADR. As worded it overcommits on duplicates (window 2/4/6), underdesigns losses (windows 1/3 with no drain), and misdescribes the mechanism.
- Evidence: `effects.go:90-102` (three separate transactions around one `Send`); `TestPerformAdoptsStrandedAttempt` proves quarantine, not resolution; G0-3 evidence covers app-write/checkpoint atomicity only; grep finds no reconciler/sweeper implementation.
- Required resolution: Adopt the replacement guarantee in §H (at-least-once execution, at-most-once recorded attempts per touch identity, provider-idempotent where supported, reconcile-or-quarantine otherwise); design the UNKNOWN drain (reconcile-by-read for mock channels in Phase 1 + operator runbook + alerting); rewrite §7 test 3 to assert record/checkpoint atomicity and test the send windows separately per §G.

### HIGH findings

**H1 — No per-family AppName, no managed ApplicationVersion, no recovery guard (plan4.2 §11.2 unimplemented)**
- Severity: HIGH (engineering change; owner awareness via G0 D2).
- Location: ADR lines 47-48 ("versioned recovery" as covered), 64-66 (§7 versioning plan); `executor/dbos/executor.go:40-84` (one `AppName` for the whole registry; `dbos.Config` sets no version); `registry/registry.go:133-135` (`Version` required but never read anywhere — verified by grep); SDK `dbos/dbos.go:1212-1223` (`computeApplicationVersion` = sha256(binary hash + app name), automatic).
- What is wrong/missing/unproven: Every deployment changes the SDK application version (binary hash), and recovery of pending workflows across versions follows SDK defaults the team has neither configured nor tested. `WorkflowDef.Version "1"` is decorative. The executor binds every miniapp family to one AppName, while plan4.2 §11.2 and the G0-accepted D2 topology require a stable distinct AppName per Class D family (process group per family). "Recovery guard rejects unknown versions" describes code that does not exist.
- Why it matters: For a multi-day workflow that will routinely span deployments, version/recovery behavior is the durability story. Today a deploy's effect on pending lead-followup runs would be governed by unexamined SDK defaults.
- Evidence: `executor.go:54-58`; grep for `ApplicationVersion|recovery guard` in Go code returns only the ADR line itself; `TestDBOSFamilyIsolation` (G0-3) proves AppName namespace isolation, not version policy or single-AppName multi-family safety.
- Required resolution: Implement the §11.2/D2 shape before enablement (per-family AppName + explicitly managed application version + documented recovery/across-deploy policy + guard), or record an owner-accepted variance with its residual risk. §7 test 5 must run a real version transition with old runs pending (§G), not assert a string field.

**H2 — Approval denial/expiry marks the invocation failed but never cancels the running Class D workflow**
- Severity: HIGH (engineering change in existing code).
- Location: `runtime/service.go:202-228` (`pollApproval` decided-not-approve and expired branches); contrast `lifecycle.go:62-88` (`Cancel`).
- What is wrong/missing/unproven: Both branches call `store.update(... RunFailed ...)` and return — neither calls `Async.Cancel`. For a Class D tool submitted optimistically at `service.go:186-195`, denial leaves the DBOS workflow running: a lead-followup sequence denied by the approver would keep waiting, waking, and sending touches while the invocation record says failed.
- Why it matters: This directly breaks the ADR's approval-gating story and turns "approval denied" into a false negative with customer-facing sends continuing in the background.
- Evidence: Code read of `pollApproval`; `TestApprovalRejectFails` covers Class A only (no workflow exists to orphan), so the suite cannot catch it.
- Required resolution: On terminal non-approval for an invocation with a live run, cancel the run (best-effort with audit) before/while marking failed; add a Class D denial test asserting run cancellation; define touch-settlement semantics for deny-during-ATTEMPTING (B3 drain).

**H3 — No approval-decision→workflow signal bridge; expiry is never set and never swept**
- Severity: HIGH (engineering change + owner expiry values).
- Location: `runtime/lifecycle.go:93-113` (`DecideApproval`: records decision + audit, never signals); `runtime/service.go:168-196` (Class D submits immediately to "wait on the approval event"); `approvals/approvals.go:105-110` (expiry enforced lazily inside `Decide` only); sole `approvals.Request` construction at `service.go:171` sets no `Expiry`.
- What is wrong/missing/unproven: A Class D workflow parked on approval has no wake path: nobody sends the decision as a signal, so the workflow waits until its own `Recv` timeout (if any) or forever — "resumes on decision or expiry" (ADR line 33-34) has no mechanism on either half. `ExpiresAt` is always NULL (no runtime value, no sweeper), so expiry never occurs; the "expired" branch in `pollApproval` is reachable only via a `Decide` call that itself finds an expired row — which cannot exist.
- Why it matters: The human-involvement point and the "approval-pending recovery" claim depend on machinery that is entirely absent. A gated touch would hang past any playbook deadline with no timeout, no escalation, and no audit trail of the stall.
- Evidence: Code reads cited; `approvals_test.go:20` sets `Expiry` only in the test fixture, never production.
- Required resolution: Build the bridge (decision → signal to the waiting run, with redelivery/duplicate-decision safety), set per-playbook expiries at submit, add a sweeper or DBOS-timeout path that settles stale approvals, and cover decision/expiry/cancel races in §G. Owner must supply expiry periods and on-expiry behavior (§E).

**H4 — Run IDs are random per Submit; submit-then-record-failure orphans a live workflow with a broken handle**
- Severity: HIGH (engineering change in existing code).
- Location: `executor/dbos/executor.go:95-117` (`runID := "run-" + uuid.NewString()`); `runtime/service.go:236-244` and `:186-195` (record-after-submit without compensation); ADR line 62-63.
- What is wrong/missing/unproven: (a) "Stable run handle derived per invocation" is false of the substrate: every `Submit` mints a fresh random ID, so stability exists only via the runtime's invocation record — and only while that record's `run_id` write succeeds. (b) If `Async.Submit` succeeds and the subsequent `store.update` fails, the caller gets `retryable`; the agent re-invokes under the same key, `replay` returns a handle with `RunID == ""` (record status running, no run recorded), and the actually-running workflow is orphaned — reachable by nothing, cancellable by nothing, still sending. The same hole exists in the approval-submit path.
- Why it matters: The ADR's §7 "duplicate submission returns the live handle" test would fail against this substrate on the failure path that matters most (partial submit). For lead-followup, an orphan sequence is unmonitored customer contact.
- Evidence: Code reads; no test covers submit-success/record-failure (fakeAsync never fails after submit).
- Required resolution: Make submit-then-record atomic-or-compensated (record intent before submit, or cancel-on-record-failure with audit); add a fake-executor test proving replay-after-partial-submit returns the live handle and leaves no orphan; reconcile the "derived handle" wording with whatever ID scheme §F adopts.

**H5 — Effect ledger is race-unsafe under concurrent duplicate attempts; touch identity must embed tenant scope**
- Severity: HIGH (engineering change).
- Location: `effects/effects.go:108-176` (`claim`→`load`→`transition` in separate transactions; `UPDATE … WHERE business_op_id` with no state predicate); `migrations/000001_framework_init.up.sql:61-73` (`business_op_id TEXT PRIMARY KEY` — global, not per-tenant); ADR lines 75-78 (`lead_id` alone as entity).
- What is wrong/missing/unproven: Two concurrent `Perform` calls for one op: the loser loads the winner's PENDING/ATTEMPTING row and `adopt()`s it to UNKNOWN while the winner is mid-send; both then `UPDATE` unconditionally, so last-writer-wins can overwrite CONFIRMED with UNKNOWN (or vice versa) depending on interleaving. No `SELECT … FOR UPDATE`, no advisory lock, no state-predicate guard. Separately, the global PK means two tenants' identical touch IDs collide: the second tenant's `Perform` adopts the first tenant's record (RLS `USING` would hide the row for non-bypass roles and turn this into a confusing load error; for bypass roles it cross-reads). `load`/`transition` carry no tenant predicate and rely solely on RLS.
- Why it matters: This is precisely the failure mode per-lead serialization exists to prevent — and the ledger underneath it does not hold under the concurrent submissions §5 posits. A duplicate-send quarantine that corrupts under concurrency is worse than none: it reports certainty it does not have.
- Evidence: Code reads; `effects_test.go` covers sequential repeats only (single-threaded, one tenant).
- Required resolution: Serialize per-op attempts (row lock on the claimed record held across the attempt decision, or state-predicated updates with affected-row checks); scope touch identity as `(tenant, lead, generation, touch_index)` (§F) so the global PK is safe by construction; add concurrent-duplicate and cross-tenant-ID tests (§G).

**H6 — Gate 4 estimates are ungrounded and Alternative B omits its largest cost centers**
- Severity: HIGH (engineering analysis + owner cost tolerance).
- Location: ADR lines 43-52, 82-87 ("~2–3 weeks" vs "~1 week", no breakdown).
- What is wrong/missing/unproven: Neither estimate decomposes into design/test/operate/recover/upgrade/incident cost. Alternative B's "residual ~1 week" omits: version management across binary-hash versions (H1); the approval→signal bridge + expiry/sweeper (H3); per-lead serialization mechanism selection and proof (B2); UNKNOWN reconciliation design/operation (B3); monitoring/alerting/runbooks for stuck workflows (§7.7 requires these); determinism enforcement (no analyzer exists — `docs/phase0/determinism-rules.md` is spec-only); durable-sleep memory/topology analysis for multi-day waits (M5); and the optimistic-submit orphan path (H4). Alternative A is framed as a maximal bespoke engine rather than the realistic middle design (River delayed jobs per touch + lead-state table + the existing effect ledger + reaper), which for a small fixed touch count may be genuinely simpler — that comparison is not performed.
- Why it matters: Gate 4 is one of the two gates requiring human judgment (plan4.2 §10.1). A 3:1 ratio with no breakdown and omitted cost centers cannot support it.
- Evidence: ADR text (no decomposition); missing-implementation inventory from B2/B3/H1/H3–H5.
- Required resolution: Re-estimate both alternatives line-by-line against the §F minimum design (including 12 months of operation), cost the realistic River middle explicitly, and present the comparison — not the conclusion — to the human reviewer.

### MEDIUM findings

**M1 — 12/14 is scored on proposed semantics; established-requirements score is 6/14 ("evaluate simpler alternatives first")**
- Severity: MEDIUM (analysis correction; flips on owner input, not engineering).
- Location: ADR lines 21-34; DBOS.md §4 rubric (0/1/2 scale + bands).
- What is wrong/missing/unproven: Per-dimension (full rationale in §D): lifetime 2→1 (durability established, multi-day span not); stages 2→1 (proposed stage list; "independently checkpointed" unconfirmed); waiting 2→1 (durable waits established, "long" unconfirmed); recovery 2→1 ("resume from previous progress" depends on the unconfirmed no-duplicate promise); effects 2→1 (touch count unconfirmed — a one-touch playbook is rubric-1 "one retryable integration"); coordination 1 held (cancel/operator races are established via the cancel-safe descriptor; per-lead trigger concurrency is not); human 1→0 (no source requires approval). Total 6/14 → band 5–7.
- Why it matters: The score is presented as the quantitative backbone of PROPOSED ADOPT. At 6/14 the governing rubric recommends exactly what has not been done: evaluate existing queue/database capabilities first. The 12 remains valid as the *ceiling if the owner confirms everything* — the ADR must say that.
- Evidence: Rubric text (DBOS.md:260-293); OSI-1; §D breakdown.
- Required resolution: Rescore per §D with flip conditions; do not present 12/14 as measured.

**M2 — ApproverScope is stored but never enforced; Decide-after-cancel is possible**
- Severity: MEDIUM (engineering change + owner roles).
- Location: `approvals/approvals.go:82-132` (`Decide` checks self-approval, tenant, hash, expiry — never `ApproverScope` or role); `runtime/lifecycle.go:93-113` (no invocation-status check before `Decide`).
- What is wrong/missing/unproven: Any authenticated non-requester in the tenant can approve anything, including cross-role (an agent approving another agent's consequential send). `ApproverScope: "tenant-ops"` in tests is decorative. A `Decide` landing after the invocation was cancelled still records "decided," producing audit evidence of an approval for a dead run.
- Why it matters: Separation of duties is the control the human point exists to serve; half of it (self-approval) is enforced, the other half (authorized role) is not.
- Evidence: Code reads; `TestApprovalSelfDecisionRejected` proves only the enforced half.
- Required resolution: Enforce scope/role at `Decide` (owner defines roles, §E); refuse decisions on non-waiting invocations; test both.

**M3 — Same-key resubmission with different input replays the first result without comparing inputs**
- Severity: MEDIUM (engineering change in existing code).
- Location: `runtime/service.go:129-131, 142-163` (`replay` uses the record only); `runtime/store.go:38-58` (claim ignores input on conflict).
- What is wrong/missing/unproven: Two submissions sharing an idempotency key but carrying different leads/inputs: the second receives the first lead's invocation ID, result, or run handle with no error. The key namespace already scopes `(tenant, miniapp, version, tool, key)` — but not input.
- Why it matters: For an agent-driven API where keys may be reused across retries with mutated payloads, this converts a caller bug into cross-lead result confusion — silent wrong-lead attribution for a follow-up tool.
- Evidence: Code read; `TestInvokeIdempotentReplay` replays identical input only.
- Required resolution: On claim-conflict, compare a canonical input hash; mismatch → terminal `idempotency-key-reuse` error naming the conflict (standard practice). Test same-key/different-input explicitly.

**M4 — Entity identity omits tenant scope; cross-tenant lead-ID collision is unaddressed**
- Severity: MEDIUM (design correction; owner confirms lead-ID uniqueness or not).
- Location: ADR lines 75-78 ("the sales lead (`lead_id`)"); `migrations/000001_framework_init.up.sql:62` (global PK); `effects.go:132-142` (tenant-predicateless load).
- What is wrong/missing/unproven: Nothing establishes that `lead_id` is globally unique. If lead IDs are per-tenant (the normal CRM shape — cf. `app.crm_contacts` keyed `(id, tenant_id)`), then bare-`lead_id` workflow IDs, touch IDs, and effect PKs collide across tenants: the second tenant's sequence attaches to, adopts, or corrupts the first tenant's records (compounding H5).
- Why it matters: Tenant isolation is a G0-proven property of the substrate; the proposed identity scheme would silently void it for this workload.
- Evidence: Schema reads; `crm_contacts` migration uses per-tenant IDs, establishing the house pattern the ADR ignores.
- Required resolution: Owner confirms lead-ID scope (§E); design carries `(tenant_id, lead_id, …)` in every identity regardless (§F); add the different-tenants-same-lead-ID test (§G).

**M5 — Durable Sleep across restart is never demonstrated; multi-day-wait topology is unanalyzed**
- Severity: MEDIUM (substrate proof + owner scale input).
- Location: ADR lines 26, 70-72 (§7 crash recovery "kill mid-wait"); `executor/dbos/dbosexec_test.go` (no `Sleep` usage — verified by grep); `spikes/dbos_test.go` (crash probe blocks in a step, not a sleep); DBOS.md:622 (sleep-suspend is Enterprise; open-source keeps sleepers resident).
- What is wrong/missing/unproven: No test in the repo sleeps durably and recovers across a process restart — yet multi-day waits are the workload's core. Open-source sleep residency implies worker memory/topology scales with concurrent sleeping sequences; no capacity model, no alerting, no stuck-sleeper detection exists (§7.7 requires all three).
- Why it matters: Gate 1's central claim ("durable waits") rests on a primitive with zero local proof and a known Enterprise boundary on the efficient form.
- Evidence: Grep for `Sleep(` across Go code returns no workflow usage; G0-3 crash evidence covers step/transaction recovery, not sleep recovery.
- Required resolution: Add a sleep-across-restart probe (short sleep, kill worker mid-sleep, assert wake + continuation) before enablement; owner supplies expected concurrent-sequence scale (§E) for the capacity note.

**M6 — Optimistic Class D submit diverges invocation and run lifecycles (approval path)**
- Severity: MEDIUM (engineering change).
- Location: `runtime/service.go:168-196` (submit-then-park); `pollApproval` Class D branch `:214-219` (returns status, never forwards the decision into the run).
- What is wrong/missing/unproven: Beyond H2/H3's missing cancel/bridge: the approved path never verifies the workflow actually resumed on the approval — `pollApproval` reports executor status, which for a workflow still blocked on a never-delivered event reads `waiting_event` forever while the invocation record claims progress. Invocation status and run status can disagree indefinitely with no reconciliation.
- Why it matters: Operators and agents will read invocation status; silent divergence is silent stall.
- Evidence: Code read; `TestClassDLifecycle` uses the fake executor with no approval gating, so the intersection is untested.
- Required resolution: Couple the lifecycles (decision signal + status reconciliation on poll, §F); test approve-then-observe-resumption end to end against the fake, then the real executor.

**M7 — Increments 4–5 report "complete" with zero evidence-register entries; all DB tests skip in this review**
- Severity: MEDIUM (process/evidence, not code quality).
- Location: `docs/evidence/README.md` (no Phase 1 entries — verified by grep); this review's V-run: `go test ./...` green at `5cf9627`+uncommitted tree, but 16 SKIP / 2 PASS across runtime/effects/approvals/dbosexec (no `TEST_DATABASE_URL`, no Docker).
- What is wrong/missing/unproven: The prompt11 implementation summary (submit/status/signal/cancel, tenant steps, lifecycle paths) is *code-reviewed* here and the code is good — but no dated execution (revision, env, command, result) is recorded anywhere, and I could not execute the DB-backed paths. Per review discipline, every behavioral substrate claim below is therefore NOT PROVEN *by this review* (some were proven at G0 on spike shapes — cited where applicable).
- Why it matters: The ADR's "reusing the tested Phase 1 DBOS substrate" leans on tests nobody has filed evidence for. G0 earned its PASS with dated entries; Phase 1 should hold the same bar before a workload bets on it.
- Evidence: Evidence-register grep; test-run counts above.
- Required resolution: Run the DB suites once from a daemon host (`test:integration` or Neon), file dated entries per the register format, then re-run at the enablement revision. Owner/operator action, not a code change.

### LOW findings

**L1 — `invocations.update` carries no tenant predicate (relies on RLS alone, inconsistently)**
- Severity: LOW (engineering hardening).
- Location: `runtime/store.go:103-115` (`WHERE id = $5::uuid` only) vs `getByRun` `:83-101` (explicit tenant check with a comment explaining privileged roles bypass RLS).
- What is wrong/missing/unproven: The file's own comment establishes that RLS cannot be relied on under privileged roles — yet `update` relies on exactly that. Reachable impact is limited (callers pass IDs previously loaded under tenant scope), but the layering is inconsistent: one confused-deputy path or future caller breaks the assumption silently.
- Why it matters: Defense-in-depth consistency for the invocation record every lifecycle path mutates.
- Evidence: Code read of both functions.
- Required resolution: Add `AND tenant_id = $6` to the UPDATE (and to effects `transition`'s UPDATE for the same reason); test cross-tenant update refusal under a bypass role.

**L2 — `Status` swallows executor errors and serves the possibly-stale record**
- Severity: LOW (engineering hardening).
- Location: `runtime/lifecycle.go:28-34` (`if live, err := …; err == nil` — error silently ignored).
- What is wrong/missing/unproven: During a DBOS outage, `Status` returns the last recorded status (e.g., `running`) with no indication it is stale. Operators cannot distinguish "running" from "executor unreachable."
- Why it matters: Status lies by omission exactly when accurate status matters most (incidents).
- Evidence: Code read.
- Required resolution: Surface staleness (return the record with a `stale` indicator, or a retryable error naming the executor failure); test the executor-down path with a failing fake.

**L3 — `IntentHash` hashes raw input bytes, so serialization variants spuriously invalidate approvals**
- Severity: LOW (engineering hardening).
- Location: `runtime/service.go:75-78` (`string(input)` hashed verbatim); consumed at `approvals.go:111-113` (mismatch fails closed).
- What is wrong/missing/unproven: The decider must byte-reproduce the agent's JSON serialization (`{"a":1}` vs `{"a": 1}` hash differently). Semantically identical intents with different whitespace/key order fail as "intent changed," forcing re-approval for a non-change.
- Why it matters: Fail-closed is the right direction, but spurious invalidation trains approvers to re-approve reflexively — eroding the control H2/H3/M2 are trying to make meaningful.
- Evidence: Code read; `TestIntentHashStability` covers stability, not canonicalization.
- Required resolution: Canonicalize (unmarshal + re-marshal with sorted keys, or hash a canonical form) before hashing on both sides; test whitespace/key-order equivalence.

### What the ADR and implementation get right (no finding)

Credit where due: the DRAFT marking and hard-stop language are exemplary; `[PROPOSED]` discipline plus the OSI-1 linkage is exactly how an honest ADR behaves; Alternative B's "still needs us to build" list (manifest, connectors, eligibility, approval parking, audit, serialization wiring) anticipates H3/B2 rather than hiding them; the §7 checks name the right eight areas even where the mechanisms are missing; the runtime's auth-before-dispatch, server-derived tenant, intent-hash binding, self-approval prevention, and audit-mandatory paths are well-constructed; the effect ledger's adopt-to-UNKNOWN (rather than blind retry) is the correct discipline; and the executor's single-datasource/shared-pool design with documented shutdown ordering correctly implements the prompt11 "provisioning belongs to the executor" decision (`executor.go:62-66`, `WorkflowEnv`).

## C. Four-gate assessment

Governing definitions: DBOS.md §3 (business questions authoritative; Hono/pg-boss references ported to Go/River per plan4.2 §10.1). Thresholds: Gate 1 — at least one genuine durable requirement; Gate 2 — at least two substantial characteristics; Gate 3 — failure breaks a business promise or creates expensive reconciliation (human judgment); Gate 4 — Alternative B genuinely simpler with stronger guarantees (human judgment).

| Gate | Threshold | ADR claim | Assessment | Evidence / unresolved conditions |
|---|---|---|---|---|
| 1 — Durable need | ≥1 of: recovery after termination, durable waits, checkpoint resumption, cross-deploy progress, multi-stage execution, durable event/human wait | Multi-day waits + exactly-once touches across restarts/deploys | **PARTIAL PASS** — kind established, magnitude not | PASS half: "durable waits" + "cancel-safe recovery" are governing plan4.2 descriptors for this workload (cited ADR line 12-13), and cross-deploy progress follows from any multi-hour sequence. UNCONFIRMED: multi-day span (B1), that stages are dependent rather than schedulable (no stage analysis vs Class B chain), that restart recovery is a business requirement vs an engineering preference. "Exactly-once touch semantics" is not a cognizable Gate 1 requirement (B3). Missing analysis: why N River delayed jobs + lead-state table + effect ledger fails — the ADR's Alt A never costs this middle design (H6). |
| 2 — Characteristics | ≥2 substantial from the §3 list | 4: waiting, recovery, effects, coordination | **THRESHOLD MET at 2; claimed 4 NOT established** | Established: (a) delays/long waits — durable waits in the governing descriptor; (b) cancellation/resumption/operator intervention — cancel-safe recovery in the descriptor. NOT established: multiple independently committed stages (proposed stage list), multiple consequential external services (touch count/channels unconfirmed; one mock channel exists), fan-out/concurrency management (trigger topology unknown; per-lead serialization is a proposed mechanism, not a requirement), human approval (no source requires it), compensation (no saga design). The threshold is met without them — but only just, and only on the two descriptor-backed items. |
| 3 — Business consequence | Breaks a business promise or creates expensive reconciliation | Lost revenue attribution; customer spam; playbook violation | **NOT ESTABLISHED — owner judgment required** | All three consequences are plausible and well-chosen, but none is sourced: no owner statement, no incident history, no quantified likelihood/severity/reversibility. Likelihood unanalyzed (how often do leads enter follow-up? how often do agents double-submit?). Severity unanalyzed (is one duplicate touch a shrug or a churn event? does it depend on channel?). Reversibility unanalyzed (a sent touch is irreversible; a lost touch may be recoverable by re-entry — the asymmetry matters and cuts toward different mitigations). Missing consequences the owner must weigh: consent/opt-out/suppression violations (regulatory, not just rude), wrong-person/wrong-channel sends, send costs at scale, and the operational cost of quarantined touches with no drain (B3). Gate 3 cannot pass on architect prose; it needs the owner. |
| 4 — Net complexity | Alt B genuinely simpler with stronger guarantees | 2–3 weeks bespoke vs ~1 week DBOS | **NOT DEMONSTRATED** | See H6: no breakdown, eight omitted Alt-B cost centers (versioning, approval bridge, serialization mechanism, reconciliation ops, monitoring/runbooks, determinism enforcement, sleep topology, orphan paths), and the realistic River middle never costed. Directionally, DBOS *may* still win — durable waits + cancel-safe recovery + signals are real machinery to hand-roll — but "may" is not a Gate 4 pass. The comparison must be redone against the §F minimum design with 12-month operations included, and the human reviewer must judge the redone numbers, not these. |

**Gate roll-up:** Gate 1 partial, Gate 2 threshold-met, Gate 3 blocked on owner, Gate 4 blocked on analysis. No gate affirmatively fails — but only Gate 2 passes as drawn, and the two human-judgment gates (3, 4) have nothing yet for a human to judge. This is the profile of a DEFER, not a REJECT: the workload leans durable, the proof is not there.

## D. Revised score

Rubric: DBOS.md §4 (0 = negligible, 1 = limited, 2 = important; bands 0–4 ordinary code/job, 5–7 evaluate queue+DB first, 8–10 strong DBOS candidate, 11–14 very strong). Rule applied: no points for unconfirmed semantics (B1/M1).

| Dimension | ADR | Revised | Rationale |
|---|---|---|---|
| lifetime (short request / background / hours-days+) | 2 | **1** | Durable waits are descriptor-established, so this is not request-bound; but no source establishes hours/days span. *Flips to 2 when the owner confirms any multi-hour/day interval.* |
| stages (one op / few dependent / multiple checkpointed) | 2 | **1** | The stage list is explicitly `[PROPOSED]`; "independently checkpointed stages" presumes the touch/eligibility decomposition the owner has not confirmed. A few dependent operations is the most the concept establishes. *Flips to 2 with a confirmed multi-touch decomposition.* |
| waiting (none / short delay / durable long wait or event) | 2 | **1** | Durability established, "long" not (same evidence as lifetime). An external-event wait (reply detection) would also qualify for 2 — currently unconfirmed. *Flips to 2 with long intervals or confirmed event waits.* |
| recovery (full retry safe / partial retry / resume from progress) | 2 | **1** | Resume-from-progress presumes completed touches must never repeat — i.e., the unconfirmed no-duplicate promise. Safe-crash behavior is required (cancel-safe descriptor), which exceeds "full retry safe." *Flips to 2 when the owner confirms duplicates intolerable.* |
| effects (none-or-one-simple / one retryable / multiple consequential) | 2 | **1** | Touch count unconfirmed; a one-touch playbook is literally rubric-1. The one established send is retryable-through-the-ledger. *Flips to 2 with ≥2 confirmed consequential touches.* |
| coordination (none / basic concurrency / fan-out or competing workers) | 1 | **1 (held, re-grounded)** | The ADR's grounding (per-lead trigger collapse) is unconfirmed mechanism, not requirement — but cancel/operator intervention concurrent with execution IS established via the cancel-safe descriptor, which is basic concurrency. *Stays 1; reaches 2 only with confirmed fan-out/rate-limit/competing-worker requirements.* |
| human (none / occasional action / approvals part of process) | 1 | **0** | No source requires approval or any manual action for this workload. The approval design is proposed machinery in search of a requirement. *Flips to 1 with any confirmed manual step, 2 if approvals gate the process.* |
| **Total** | **12** | **6** | Band 5–7: "Evaluate existing queue and database capabilities first." |

**Ceiling analysis (for the re-review):** if the owner confirms the full proposed shape — multi-day span, multi-touch decomposition with eligibility gates, long durable waits, intolerable duplicates, concurrent trigger topology, and approval-gated touches — the honest score is 2+2+2+2+2+1+1 = **12/14**, and the ADR's number is vindicated. The defect is presenting the ceiling as the measurement. The ADR should carry both numbers: **6/14 established, up to 12/14 conditional on §E confirmations**.

## E. Business-owner decision register

Every row is OWNER DECISION REQUIRED unless noted. No defaults are proposed — where the sources support no choices, the owner must supply the value. "Blocks" names what cannot proceed without it.

| # | Decision | Questions the owner must answer | Consequence of the choice | Exact input required | Blocks |
|---|---|---|---|---|---|
| E1 | Number of touches | How many follow-ups constitute one sequence? Fixed or variable? | 1 touch collapses effects to rubric-1 and weakens the DBOS case; ≥2 with gates strengthens it (§D). Determines touch-identity cardinality. | Integer or rule; variability conditions. | Score, Gate 2/3, B3 scope |
| E2 | Timing | Exact intervals between touches, total sequence deadline, timezone for all timestamps. | Multi-hour/day intervals flip lifetime/waiting to 2; short delays keep the River-middle competitive (H6). Out-of-window semantics depend on this. | Per-touch delay values + units; overall deadline; timezone. | Score, Gate 1, window enforcement |
| E3 | Channels | Which channels, in what order/fallback, what happens when a channel is unavailable or a touch fails. | Determines connector set, per-channel idempotency availability (B3: provider-supported or not?), and failure behavior. Mock channels in Phase 1 must mirror the confirmed set. | Ordered channel list; fallback/skip/abort rule per channel. | Effects design, B3 |
| E4 | Quiet hours | Which timezone (lead-local? tenant?), days, holidays, local restrictions; what happens to a touch scheduled inside quiet hours (defer? skip?). | Affects wait computation (not plain sleep — constrained wake) and out-of-window definition. Lead-local quiet hours require lead timezone data. | Timezone rule; quiet windows; defer-vs-skip rule. | Wait design, window enforcement |
| E5 | Eligibility | Conditions for entering a sequence; conditions re-checked before each touch; what happens on mid-sequence ineligibility (pause? exit? escalate?). | Determines eligibility-gate count (stage decomposition) and exit accounting. | Entry predicate; per-touch predicate; ineligibility disposition. | Stage design, exits |
| E6 | Exit conditions | Which events terminate: conversion, opt-out, invalid contact, ineligibility, bounce, reply? Is exit terminal or pausing? Who is notified? | Defines terminal states the workflow and tests must implement; opt-out handling has regulatory weight (see E11). | Exhaustive exit-event list with per-event disposition. | Workflow design, tests |
| E7 | Escalation | Trigger conditions, escalation target (whom), mechanism (task? notification? approval?), and what happens to the sequence during/after escalation. | Unscoped escalation becomes an unbounded human-in-the-loop requirement; determines whether escalation is a workflow branch or an external process. | Trigger rule; target role; mechanism; sequence disposition. | Human scope, Gate 2 |
| E8 | Approval | Which touches (if any) require approval; who may approve (roles); expiry period; on-expiry behavior; on-rejection behavior (cancel sequence? skip touch?); can approval be re-requested? | "No approval" zeroes the human dimension and deletes H3/M6 work from scope; any "yes" requires H2/H3/M2 fixes. Expiry/rejection semantics are currently unimplementable (H2/H3). | Per-touch approval yes/no; approver roles; expiry duration; expiry disposition; rejection disposition. | Human dimension, H2/H3/M2 |
| E9 | Cancellation | Who can cancel (agent? owner? lead via opt-out?); effect on in-progress ATTEMPTING sends (settle? abandon?); effect on scheduled-but-unsent touches; audit requirements. | Cancel-during-send is the sharpest race in the design (B3 window 7); the answer determines touch-settlement semantics and the cancel path tests. | Cancellor roles; in-flight disposition; scheduled-touch disposition. | Cancel design, B3/H2 |
| E10 | Failure behavior | Per failure class — channel unavailable, send rejected, outcome unknown, eligibility-check error, approval stall — retry? quarantine? escalate? abort? With what limits and whose attention? | This IS the UNKNOWN drain design (B3c): without it, quarantine is terminal-by-neglect. Determines reconciler scope and alerting. | Per-class disposition + retry caps + alert targets. | B3 drain, monitoring |
| E11 | Consent & suppression | Consent basis for each channel; suppression-list source and precedence; opt-out propagation delay tolerance; duplicate-contact (same person, two lead records) limits. | Regulatory exposure (spam law treats unconsented outreach differently from poor timing); suppression checks are per-touch gates. Not mentioned anywhere in the ADR. | Consent rule per channel; suppression source; dedup rule. | Compliance, eligibility |
| E12 | Lead ownership & preferences | What happens on lead reassignment mid-sequence (new owner notified? sequence continues?); contact-preference changes mid-sequence; lead merge/dedup during flight. | Mid-flight entity mutation breaks naive per-lead identity (B2/M4); determines whether generation must also key on owner/version. | Reassignment disposition; preference-change handling; merge rule. | Identity design (B2) |
| E13 | Sequence generation & campaigns | What starts a new generation (new submission after terminal state? explicit campaign ID? time-based re-entry?); can two generations overlap; who may start one. | Without this, B2's ID scheme cannot be built at all — generation is the missing key component. | Generation rule; overlap rule; initiation authority. | B2, identity |
| E14 | Lead-ID scope | Are lead IDs globally unique or per-tenant? | Per-tenant (expected) forces `(tenant, lead)` composite identity everywhere (M4); global simplifies but must be asserted, not assumed. | Uniqueness scope statement. | M4, identity |
| E15 | Duplicate tolerance | Is exactly one duplicate touch a defect, and of what severity — per channel? Is a lost touch worse than a duplicate, or vice versa? | Sets the recovery target (resume-vs-restart), the Gate 3 severity input, and the monitoring thresholds. The ADR asserts intolerability without sourcing it. | Severity ranking: lost vs duplicate vs out-of-window, per channel if it varies. | Gate 3, recovery target |
| E16 | Scale & cost envelope | Expected concurrent sequences, touches/day, peak bursts; per-send cost tolerance; stuck-sequence alert latency. | Sizes the sleep-residency capacity model (M5), the UNKNOWN-drain staffing, and pool/connection budgets (§7.8). | Orders of magnitude + alert latency target. | M5, ops design |
| E17 | Gate 3 consequence confirmation | Confirm/deny/quantify: lost follow-up → lost revenue attribution; duplicate touch → customer-facing spam; out-of-window touch → playbook violation. Likelihood, severity, reversibility of each. | This is the Gate 3 pass/fail input; engineering cannot supply it. | Per-consequence confirm + severity + reversibility note. | Gate 3 |
| E18 | Gate 4 cost tolerance | Accept the re-estimated Alt A vs Alt B comparison (H6 redo) including 12-month operations; state DBOS operational-ownership acceptance. | The Gate 4 human judgment; must be rendered on redone numbers, not the ADR's 3:1. | Signed comparison acceptance. | Gate 4 |

E1–E10 are the prompt11 §3 table, completed with consequence + input columns. E11–E18 are additions this review requires (consent, ownership, generation, ID scope, duplicate tolerance, scale, and the two human-gate judgments). E13/E14 are engineering-blockers disguised as business questions: no identity design is possible without them.

## F. Technical design recommendations

Minimum viable design for the confirmed requirements, sequenced so each layer is provable before the next is built. Framework behavior vs miniapp behavior is separated explicitly. Nothing here authorizes implementation — the hard stop holds until the ADR is signed.

**F1. Identity spine (framework + miniapp; closes B2/M4, needs E13/E14).** Miniapp owns a durable `lead_sequences` record keyed `(tenant_id, lead_id, generation)` with a partial unique index on `(tenant_id, lead_id)` where `status = 'active'` — the single serialization point, claimed transactionally at submission. Workflow ID derives deterministically from `(tenant, lead, generation)`; touch/effect IDs from `(…, touch_index)`; the runtime invocation keeps its own UUID/key identity and *references* the sequence (never equals it). Second concurrent submission for a live generation attaches to the existing run handle (same run, distinct invocation records with explicit "attached" audit); submission after terminal state opens generation N+1. Participating in a partitioned DBOS queue is then an optimization (ordering/fairness per plan4.2 §11.2), not the correctness mechanism — correctness lives in the unique index.

**F2. Honest effect protocol (framework hardening + miniapp usage; closes B3/H5, needs E3/E10/E15).** Framework: close the ledger race (row-lock or state-predicated updates, H5); add the UNKNOWN drain shape (reconcile-by-read hook per channel + quarantine listing + operator settle/abort with audit + stuck-UNKNOWN alerting). Miniapp: each touch is tx-record-intent → step-send-via-`effects.Perform` → tx-record-outcome; step code never sends except through the ledger; touch step inputs carry `(tenant, lead, generation, touch_index, idempotency_key)`; no provider SDK inside workflow control flow. Guarantee language per §H.

**F3. Approval coupling (framework; closes H2/H3/M2/M6, needs E8).** Framework: decision→signal bridge (decision recorded → signal delivered to the waiting run with redelivery safety; duplicate decisions idempotent); terminal non-approval cancels the live run (H2); `ExpiresAt` set from playbook at submit + sweeper/timeout settlement path; `ApproverScope`/role enforcement at `Decide` (M2); invocation/run status reconciliation on poll (M6). Miniapp: workflow waits on the approval topic with the playbook timeout; on expiry/rejection follows the owner-confirmed disposition (skip touch vs cancel sequence, E8).

**F4. Version and family discipline (framework; closes H1, needs owner variance-or-build decision).** Per-family `AppName` (lead-followup gets its own family from day one), explicitly managed application version tied to the workflow compatibility line (not the binary hash), documented across-deploy recovery policy, and a real guard (refuse to launch/recover unknown versions). Plus the §G version-transition test with old runs pending. Until this exists, every deploy is an unplanned version migration for any pending sequence.

**F5. Submit atomicity and replay correctness (framework; closes H4/M3, plus L1–L3 opportunistically).** Record-before-submit or cancel-on-record-failure with audit; canonical-input-hash comparison on claim conflict; tenant predicates on mutation statements; staleness surfacing in `Status`.

**F6. Substrate proofs before enablement (closes M5/M7).** Sleep-across-restart probe; full DB-suite execution with dated evidence entries (M7); capacity note from E16 scale input. The G0-3 spike evidence (shared-handle atomicity, crash recovery, AppName isolation) carries over as established substrate behavior and need not be re-proven — but sleep recovery, version transitions, and the approval bridge are new proofs, not corollaries.

**What stays miniapp-specific:** playbook interpreter (waits computed from E2/E4, not bare sleeps), eligibility predicates (E5), channel connector mapping onto `effects.Channel` (mock in Phase 1), touch content assembly (IDs-only inputs per payload discipline), and the per-touch approval-requirement flag (E8). Everything else above is framework or shared-protocol work that lead-followup merely exercises first.

## G. Acceptance-test matrix

Each row: the invariant, the adversarial test, the expected result, and when its evidence is due. "Approval" = needed before the human signs ADOPT; "enablement" = needed before any `lead-followup` workflow executes. Environment: L = local Docker PG via `test:integration`, N = Neon branch, K = controlled process kill/restart, U = pure unit/static. The eight planned §7 checks are rescoped where their current wording asserts a nonexistent mechanism.

| # | Invariant (maps to ADR §7 #) | Adversarial test | Expected result | Env | Due |
|---|---|---|---|---|---|
| G1 | Determinism (§7.1): same input ⇒ same step sequence | Determinism lint over the workflow package + replay test (run twice, diff step history) | Lint clean; identical step sequences and results | L,U | Enablement (lint exists? NO — analyzer is spec-only; replay test at enablement, lint when the analyzer lands) |
| G2 | Idempotency (§7.2): duplicate submission ⇒ live handle, one sequence | Same-key resubmit mid-flight; different-key concurrent resubmit same lead; same-key different-input resubmit | Same-key: same invocation + live handle; different-key: attach to one run with "attached" audit (F1); different-input: terminal key-reuse error (M3) | L | Approval (design proof) + enablement |
| G3 | Tx boundary (§7.3): touch intent record + checkpoint atomic | Kill worker between intent-tx commit and send; kill between send-accept and outcome-tx | First case: PENDING adopted UNKNOWN, no send on recovery, drainable to NOT_APPLIED→retry; second: UNKNOWN quarantined, never blind-retried (B3) | L,K | Enablement |
| G4 | Stable handle (§7.4): invocation ⇔ run binding survives partial failure | Submit succeeds, `run_id` record write fails (fault-injected store) | Compensated (cancel + audit) or attached on retry; never an orphan run with an empty handle (H4) | L,U (fake) | Approval |
| G5 | Versioning (§7.5): old runs survive deploy under policy | Start runs on version N, deploy N+1, assert drain-or-migrate per policy; launch with unknown version | Old runs drain/recover per the written policy; unknown version refused (H1/H4) — NOT a string-field assertion | L,K | Enablement |
| G6 | Payload discipline (§7.6): IDs-only, bounded, no secrets | Static input/output schema audit + runtime size/secret scan on recorded workflow inputs | No customer records, credentials, URLs, or tokens; sizes bounded | L,U | Enablement |
| G7 | Crash recovery (§7.7): kill at every boundary ⇒ resume without repeating CONFIRMED touches | Kill mid-wait (sleep), mid-tx-step, mid-send-step, mid-signal-wait; relaunch; diff touch ledger before/after | Resume from last completed touch; zero duplicate CONFIRMED touches; zero lost intents (all UNKNOWN-or-better with drain path) | L,K | Enablement |
| G8 | Op cost (§7.8): bounded, owned, alerted | Capacity note from E16 + stuck-sleeper/stuck-UNKNOWN alert drill + connection-budget accounting | Documented budgets, owners, alert paths; drill observed (not described) | L | Approval (budgets) + enablement (drill) |
| G9 | Concurrency: same lead, two agents, different keys | Parallel submit ×2 (barrier-synced) + parallel submit during active send | Exactly one active generation; loser attaches (audit shows both invocations → one run); effect ledger shows no double-send and no UNKNOWN-clobber (H5) | L | Approval |
| G10 | Tenant scope: identical lead IDs, two tenants | Same `lead_id` in tenant A and B, parallel sequences | Fully independent sequences/handles/records; no attach, no adopt, no cross-read (M4) | L | Approval |
| G11 | Approval races: decide/expire/cancel/reject interleavings | Decide-after-cancel; cancel-during-approval-wait; double-decide; decide with mutated intent; expiry sweep | Post-cancel decide refused (M2); cancel settles the approval; double-decide idempotent-or-rejected consistently; mutated intent fails closed; swept approvals settle runs (H2/H3/M6) | L | Enablement |
| G12 | Denial settles the run | Approve-path test, then reject a gated Class D invocation with a live run | Run cancelled (not merely invocation-failed); no further touches; audit shows denial→cancel chain (H2) | L | Approval (framework fix proof) |
| G13 | UNKNOWN drain: every quarantine resolves | Drive each failure class (E10) to UNKNOWN, then run the drain | Each UNKNOWN reaches CONFIRMED/FAILED/NOT_APPLIED via reconcile or operator settle with audit; none lingers past the alert latency (B3) | L | Enablement |
| G14 | Generation lifecycle: terminal ⇒ new generation possible | Complete a sequence, submit again for the same lead; cancel mid-flight, submit again | New generation opens with a fresh workflow (no stale-result return, B2); cancelled generation's touches settle per E9 before/while the new one starts | L | Approval |
| G15 | Sleep recovery: kill mid-sleep ⇒ wake on time after relaunch | Short durable sleep, SIGKILL worker mid-sleep, relaunch | Wake fires (within tolerance), sequence continues, exactly-once wake accounting (M5) | L,K | Enablement |
| G16 | DB tx failure inside a step ⇒ step/record coherent | Fault-inject tx failure in intent-record and outcome-record steps | Step error propagates; no half-recorded touch; retry/recovery converges to a ledger terminal-or-UNKNOWN state | L | Enablement |

Gaps in the ADR's planned eight that this matrix adds: G4 (orphan path), G9/G10 (the §5 scenarios), G11/G12 (approval races/denial), G13 (the drain the ADR never names), G14 (generation), G15 (sleep proof). G1's lint half is blocked on the nonexistent analyzer — say so in the ADR rather than listing it as plannable now.

## H. Proposed ADR corrections

Exact replacement wording. Line references are to the current 97-line draft. These are proposals for the ADR author/owner — the original ADR is untouched by this review.

**H-a. Business promise (lines 15-20).** Replace:
> every sales lead receives its follow-up touches exactly per the confirmed playbook — no lost, duplicated, or out-of-window touches
with:
> every sales lead receives its follow-up touches per the confirmed playbook, with at-most-once CONFIRMED sends per touch identity, no silently lost touch intents (every intent reaches CONFIRMED, FAILED, or operator-settled quarantine), and window compliance measured against owner-confirmed intervals
Reason: B3 — the original promises exactly-once delivery the machinery cannot provide.

**H-b. Score block (lines 21-34).** Replace the single score with:
> Score (established requirements only): lifetime 1, stages 1, waiting 1, recovery 1, effects 1, coordination 1, human 0 = 6/14 (band: evaluate existing queue/database capabilities first). Conditional ceiling if the owner confirms the full proposed playbook (multi-day span, multi-touch decomposition, long waits, intolerable duplicates, concurrent triggers, gated touches): 12/14. Per-dimension flip conditions in review §D.
Reason: M1 — no points for unconfirmed semantics.

**H-c. Gate 1 (lines 35-36).** Replace "with exactly-once touch semantics" with "with at-most-once CONFIRMED sends per touch identity and no silently lost intents". Add: "The realistic non-DBOS middle (River delayed jobs + lead-state table + effect ledger) is costed in Gate 4; Gate 1 does not presume its inadequacy." Reason: B3/H6.

**H-d. Gate 2 (lines 37-38).** Replace "waiting, recovery, effects, coordination — four present" with "waiting and cancellation/resumption — two present on established requirements (threshold met); multi-stage effects and per-entity coordination require owner confirmation of touch count and trigger topology (E1/E13)." Reason: §C Gate 2.

**H-e. Gate 3 (lines 39-42).** Append: "Consequences are architect-proposed and owner-unconfirmed (E17); likelihood/severity/reversibility unanalyzed. Regulatory (consent/suppression, E11) and cost-at-scale (E16) consequences are not yet weighed." Reason: §C Gate 3.

**H-f. Alternatives and estimates (lines 43-52, 82-87).** Replace the week-estimates with a pointer: "Estimates withdrawn pending line-by-line re-estimation against the minimum design (review §F) including 12-month operations, the River middle, version management, approval bridge, serialization mechanism, and reconciliation operations (review H6)." In Alternative B "covers," strike "versioned recovery" and "workflow identity" as covered; move both to "still needs us to build" with the designs in review §F (F1/F4). Reason: H1/H6/B2.

**H-g. §7 checks (lines 53-74).** (1) Determinism: append "lint half blocked until the Phase 1 analyzer exists; replay test proceeds." (2) Idempotency: append "covers same-key, different-key-attach, and different-input-reject cases (review G2)." (3) Tx boundary: replace with "touch *intent record* + checkpoint commit atomically; send-window crash cases tested separately (review G3)." (4) Workflow ID: replace with "run handle derived from (tenant, lead, generation) with generation lifecycle per review F1; orphan-path compensated (review G4)." (5) Versioning: replace with "per-family AppName + managed application version + real recovery guard; proven by a live version transition with old runs pending (review G5)." (8) Op cost: append "budgets, owners, and alert drills per review G8; sleep-residency capacity from owner scale input (E16)."

**H-h. Entity identity (lines 75-78).** Replace the paragraph with: "Entity identity is `(tenant_id, lead_id, generation)` pending owner confirmation of lead-ID scope (E14) and generation semantics (E13). Serialization is enforced by a durable sequence record with a partial unique index on the active generation; workflow-ID derivation and queue partitioning are consequences of that record, not alternative correctness mechanisms (review F1)." Reason: B2/M4.

**H-i. External-effect boundaries (lines 79-81).** Replace "channel sends happen only inside transaction steps recording the touch first" with: "each touch is tx-record-intent → step-send-through-`effects.Perform` → tx-record-outcome; sends never occur inside transaction steps or workflow control flow; UNKNOWN outcomes route to the reconcile-or-settle drain (review F2), never to blind retry." Reason: B3.

**H-j. Decision line (line 88).** Replace "PROPOSED ADOPT (pending human review)" with "PROPOSED DEFER — re-review for ADOPT after §E owner decisions and §§F–G design/test corrections (review §I)." The reviewer/date lines stay blank.

## I. Approval recommendation

**The ADR is not ready for human ADOPT review. It should be returned as PROPOSED DEFER with this report attached.**

More precisely: the document is ready to serve as the *deferral record* — its DRAFT marking, `[PROPOSED]` discipline, OSI-1 linkage, and hard-stop language are correct and should be preserved. What is missing is everything an ADOPT signature would attest to: owner-confirmed requirements (§E), achievable guarantee language (§H), a buildable identity design (F1), a drain for quarantine (F2), approval coupling (F3), and version discipline (F4).

Path back to ADOPT (all required, in this order): (1) owner completes E1–E18; (2) ADR revised per §H and rescored per §D; (3) framework gaps H2–H5/M2/M3/L1–L3 fixed with §G tests green; (4) substrate proofs M5/M7 filed; (5) re-review confirms the gates on the revised record. Only then should a human reviewer sign.

Reviewer (human): <blank — awaiting owner>   Date: <blank>

**Increment 6 remains blocked. The hard stop stays in force: no `lead-followup` workflow code, manifest, registration, or DBOS execution until the owner reviews these findings, resolves §E, and explicitly signs the ADR.**

## Change inventory (review discipline, prompt11 §12)

- HEAD at review time: `5cf9627` (merge) over `e6b5b7b` (G0 baseline). First-commit + G0 binding predate this review.
- Pre-existing uncommitted state (untouched): modified `docs/phase0/version-register.md`, `go.mod`, `go.sum`; untracked `approvals/ audit/ auth/ effects/ executor/ miniapps/ registry/ runtime/ testkit/`, `migrations/00000[12]*`, `docs/phase1/`, `docs/prompt10.md`, `docs/prompt11.md`, `docs/adr/miniapps/lead-followup-adr001.md`.
- This review added exactly one file: `docs/adv_review_prompt11.md`. No code, test, manifest, ADR, workflow registration, or execution configuration was created, modified, deleted, or enabled. No commit or push performed.
- Verification runs (read-only): `go build ./...` clean; `go test ./...` green (all suites pass-or-skip; 16 SKIP / 2 PASS across the five DB-backed packages — no `TEST_DATABASE_URL`, no Docker daemon); SDK-source reads from the pinned `dbos-transact-golang@v1.9.0` module cache; no Fly/Neon/production contact.
